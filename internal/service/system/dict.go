package system

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	daoSystem "snowgo/internal/dao/system"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcache"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"time"
)

// DictRepo 定义opt dict相关db操作接口
type DictRepo interface {
	GetDictById(ctx context.Context, dictId int32) (*model.SystemDict, error)
	GetDictList(ctx context.Context, condition *daoSystem.DictListCondition) ([]*model.SystemDict, int64, error)
	IsCodeDuplicate(ctx context.Context, code string, dictId int32) (bool, error)
	TransactionCreateDict(ctx context.Context, tx *query.Query, dict *model.SystemDict) (*model.SystemDict, error)
	TransactionUpdateDict(ctx context.Context, tx *query.Query, dict *model.SystemDict) (*model.SystemDict, error)
	TransactionDeleteById(ctx context.Context, tx *query.Query, id int32) error
	TransactionDeleteItemByDictID(ctx context.Context, tx *query.Query, dictId int32) error
	TransactionUpdateItemByDictID(ctx context.Context, tx *query.Query, dictId int32, dictCode string) error
	GetItemListByDictCode(ctx context.Context, dictCode string) ([]*model.SystemDictItem, error)
	IsCodeItemDuplicate(ctx context.Context, dictId int32, itemCode string, dictItemId int32) (bool, error)
	TransactionCreateDictItem(ctx context.Context, tx *query.Query, item *model.SystemDictItem) (*model.SystemDictItem, error)
	GetDictItemById(ctx context.Context, itemId int32) (*model.SystemDictItem, error)
	TransactionUpdateDictItem(ctx context.Context, tx *query.Query, item *model.SystemDictItem) (*model.SystemDictItem, error)
	TransactionDeleteItemByID(ctx context.Context, tx *query.Query, id int32) error
}

type DictService struct {
	db         *repo.Repository
	cache      xcache.Cache
	dictRepo   DictRepo
	logService *OperationLogService
}

func NewDictService(db *repo.Repository, cache xcache.Cache, dictRepo DictRepo, logService *OperationLogService) *DictService {
	return &DictService{
		db:         db,
		cache:      cache,
		dictRepo:   dictRepo,
		logService: logService,
	}
}

type DictListCondition struct {
	Name      string `json:"name" form:"name"`
	Code      string `json:"code" form:"code"`
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`
	Offset    int32  `json:"offset" form:"offset"`
	Limit     int32  `json:"limit" form:"limit"`
}

type DictInfo struct {
	ID          int32
	Code        string
	Name        string
	Description *string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

type DictParam struct {
	ID          int32  `json:"id"`
	Code        string `json:"code" binding:"required,max=64"`
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description"`
}

type DictList struct {
	List  []*DictInfo
	Total int64
}

type ItemInfo struct {
	ID          int32      `json:"id"`
	ItemName    string     `json:"item_name"`   // 枚举显示名称
	ItemCode    string     `json:"item_code"`   // 枚举值编码
	Status      *string    `json:"status"`      // 状态：Active 启用，Disabled 禁用
	SortOrder   int32      `json:"sort_order"`  // 排序号
	Description *string    `json:"description"` // 描述
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type DictItemParam struct {
	ID          int32  `json:"id"`
	DictID      int32  `json:"dict_id"`
	ItemName    string `json:"item_name" binding:"required,max=128"`
	ItemCode    string `json:"item_code" binding:"required,max=64"`
	Status      string `json:"status"`
	SortOrder   int32  `json:"sort_order"`
	Description string `json:"description"`
}

var (
	ErrDictCodeExist        = errors.New(e.DictCodeExistError.GetErrMsg())
	ErrDictCodeNotFound     = errors.New(e.DictNotFound.GetErrMsg())
	ErrDictItemCodeExist    = errors.New(e.DictCodeItemExistError.GetErrMsg())
	ErrDictCodeItemNotFound = errors.New(e.DictItemNotFound.GetErrMsg())
)

// GetDictList 获取字典列表数据
func (d *DictService) GetDictList(ctx context.Context, condition *DictListCondition) (*DictList, error) {
	var startTimePtr *time.Time
	var endTimePtr *time.Time
	if condition.StartTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", condition.StartTime, time.Local)
		if err != nil {
			return nil, errors.New("start_time格式错误，应为yyyy-MM-dd HH:mm:ss")
		}
		startTimePtr = &t
	}
	if condition.EndTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", condition.EndTime, time.Local)
		if err != nil {
			return nil, errors.New("end_time格式错误，应为yyyy-MM-dd HH:mm:ss")
		}
		endTimePtr = &t
	}
	dictList, total, err := d.dictRepo.GetDictList(ctx, &daoSystem.DictListCondition{
		Name:      condition.Name,
		Code:      condition.Code,
		StartTime: startTimePtr,
		EndTime:   endTimePtr,
		Offset:    condition.Offset,
		Limit:     condition.Limit,
	})
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取系统字典列表异常: %v", err)
		return nil, errors.WithMessage(err, "系统字典列表查询失败")
	}
	dictInfoList := make([]*DictInfo, 0, len(dictList))
	for _, dict := range dictList {
		dictInfoList = append(dictInfoList, &DictInfo{
			ID:          dict.ID,
			Name:        dict.Name,
			Code:        dict.Code,
			Description: dict.Description,
			CreatedAt:   dict.CreatedAt,
			UpdatedAt:   dict.UpdatedAt,
		})
	}
	return &DictList{List: dictInfoList, Total: total}, nil
}

// CreateDict 创建字典数据
func (d *DictService) CreateDict(ctx context.Context, param *DictParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	// 检查code是否存在
	isDuplicate, err := d.dictRepo.IsCodeDuplicate(ctx, param.Code, 0)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "查询code是否存在异常: %v", err)
		return 0, errors.WithMessage(err, "查询字典编码是否存在异常")
	}
	if isDuplicate {
		return 0, ErrDictCodeExist
	}

	// 创建字典
	var dict *model.SystemDict
	err = d.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 创建字典
		dict, err = d.dictRepo.TransactionCreateDict(ctx, tx, &model.SystemDict{
			Code:        param.Code,
			Name:        param.Name,
			Description: &param.Description,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "字典创建失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "字典创建失败")
		}

		// 创建操作日志
		err = d.logService.CreateOperationLog(ctx, tx, &OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceDict,
			ResourceID:   dict.ID,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionCreate,
			BeforeData:   nil,
			AfterData:    dict,
			Description: fmt.Sprintf("用户(%d-%s)创建了字典(%d-%s)",
				userContext.UserId, userContext.Username, dict.ID, dict.Code),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}
		return nil

	})
	if err != nil {
		return 0, err
	}
	xlogger.InfofCtx(ctx, "用户(%d)创建字典成功: %+v", userContext.UserId, dict)
	return dict.ID, nil
}

// UpdateDict 更新字典数据
func (d *DictService) UpdateDict(ctx context.Context, param *DictParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	if param.ID <= 0 {
		return 0, ErrDictCodeNotFound
	}
	// 获取dict信息
	oldDict, err := d.dictRepo.GetDictById(ctx, param.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrDictCodeNotFound
		}
		xlogger.ErrorfCtx(ctx, "获取字典(%d)信息异常: %v", param.ID, err)
		return 0, errors.WithMessage(err, "字典信息查询失败")
	}

	// 检查code是否存在
	isDuplicate, err := d.dictRepo.IsCodeDuplicate(ctx, param.Code, oldDict.ID)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "查询code是否存在异常: %v", err)
		return 0, errors.WithMessage(err, "查询字典编码是否存在异常")
	}
	if isDuplicate {
		return 0, ErrDictCodeExist
	}

	// 更新字典
	err = d.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 更新字典
		dict, err := d.dictRepo.TransactionUpdateDict(ctx, tx, &model.SystemDict{
			ID:          param.ID,
			Code:        param.Code,
			Name:        param.Name,
			Description: &param.Description,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "字典更新失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "字典更新失败")
		}

		// 如果更新了dict code，还需要更新item表对应的dict code
		if oldDict.Code != param.Code {
			err = d.dictRepo.TransactionUpdateItemByDictID(ctx, tx, param.ID, param.Code)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "字典枚举更新失败: %+v err: %v", param, err)
				return errors.WithMessage(err, "字典枚举更新失败")
			}

			// 清除code对应item缓存
			oldCacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, oldDict.Code)
			newCacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, param.Code)
			if _, err := d.cache.Delete(ctx, oldCacheKey, newCacheKey); err != nil {
				xlogger.ErrorfCtx(ctx, "清除code对应item列表数据缓存失败: %v", err)

			}
		}

		// 创建操作日志
		err = d.logService.CreateOperationLog(ctx, tx, &OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceDict,
			ResourceID:   param.ID,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionUpdate,
			BeforeData:   oldDict,
			AfterData:    dict,
			Description: fmt.Sprintf("用户(%d-%s)修改了字典(%d-%s)信息",
				userContext.UserId, userContext.Username, param.ID, param.Code),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}
		return nil
	})
	xlogger.InfofCtx(ctx, "用户(%d)更新字典成功: old=%+v new=%+v", userContext.UserId, oldDict, param)
	return param.ID, nil
}

// DeleteById 删除字典
func (d *DictService) DeleteById(ctx context.Context, id int32) error {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return err
	}

	if id <= 0 {
		return ErrDictCodeNotFound
	}

	// 检查dict是否存在
	dict, err := d.dictRepo.GetDictById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDictCodeNotFound
		}
		xlogger.ErrorfCtx(ctx, "获取字典(%d)信息异常: %v", dict.ID, err)
		return errors.WithMessage(err, "字典信息查询失败")
	}

	// 删除字典
	err = d.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 删除字典对应的item
		err = d.dictRepo.TransactionDeleteItemByDictID(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "字典(%d)枚举删除失败:  err: %v", id, err)
			return errors.WithMessage(err, "字典枚举删除失败")
		}

		// 删除字典
		err = d.dictRepo.TransactionDeleteById(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "字典(%d)删除失败:  err: %v", id, err)
			return errors.WithMessage(err, "字典删除失败")
		}

		// 创建操作日志
		err = d.logService.CreateOperationLog(ctx, tx, &OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceDict,
			ResourceID:   id,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionDelete,
			BeforeData:   nil,
			AfterData:    nil,
			Description: fmt.Sprintf("用户(%d-%s)删除了字典(%d)信息",
				userContext.UserId, userContext.Username, id),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", id, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}
		return nil
	})
	xlogger.InfofCtx(ctx, "用户(%d)删除字典(%d)成功", userContext.UserId, id)

	// 清除code对应item缓存
	cacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, dict.Code)
	if _, err := d.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.ErrorfCtx(ctx, "清除code对应item列表数据缓存失败: %v", err)
	}
	return nil
}

// GetItemListByCode 获取item枚举列表
func (d *DictService) GetItemListByCode(ctx context.Context, code string) ([]*ItemInfo, error) {
	if len(code) == 0 {
		return nil, ErrDictCodeNotFound
	}

	// 尝试从缓存读取
	cacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, code)
	if data, err := d.cache.Get(ctx, cacheKey); err == nil && data != "" {
		var m []*ItemInfo
		if err := json.Unmarshal([]byte(data), &m); err == nil {
			return m, nil
		}
	}

	itemList, err := d.dictRepo.GetItemListByDictCode(ctx, code)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取系统字典枚举列表异常: %v", err)
		return nil, errors.WithMessage(err, "系统字典枚举列表查询失败")
	}
	itemInfoList := make([]*ItemInfo, 0, len(itemList))
	for _, item := range itemList {
		itemInfoList = append(itemInfoList, &ItemInfo{
			ID:          item.ID,
			ItemName:    item.ItemName,
			ItemCode:    item.ItemCode,
			Status:      item.Status,
			SortOrder:   item.SortOrder,
			Description: item.Description,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	// 缓存结果
	expirationTime := constant.SystemDictExpirationDay * 24 * time.Hour
	if len(itemInfoList) == 0 {
		// 如果结果为空，缓存1h，防止code错误
		expirationTime = 1 * time.Hour
	}
	if b, err := json.Marshal(itemInfoList); err == nil {
		_ = d.cache.Set(ctx, cacheKey, string(b), expirationTime)
	}
	return itemInfoList, nil
}

// CreateItem 创建字典item数据
func (d *DictService) CreateItem(ctx context.Context, param *DictItemParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	// 检查dict是否存在
	dict, err := d.dictRepo.GetDictById(ctx, param.DictID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrDictCodeNotFound
		}
		xlogger.ErrorfCtx(ctx, "获取字典(%d)信息异常: %v", param.DictID, err)
		return 0, errors.WithMessage(err, "字典信息查询失败")
	}

	// 检查code是否存在
	isDuplicate, err := d.dictRepo.IsCodeItemDuplicate(ctx, dict.ID, param.ItemCode, 0)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "查询item code是否存在异常: %v", err)
		return 0, errors.WithMessage(err, "查询字典item编码是否存在异常")
	}
	if isDuplicate {
		return 0, ErrDictItemCodeExist
	}

	// 创建字典item
	var item *model.SystemDictItem
	err = d.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 创建字典item
		item, err = d.dictRepo.TransactionCreateDictItem(ctx, tx, &model.SystemDictItem{
			DictID:      dict.ID,
			DictCode:    dict.Code,
			ItemCode:    param.ItemCode,
			ItemName:    param.ItemName,
			SortOrder:   param.SortOrder,
			Description: &param.Description,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "字典item创建失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "字典item创建失败")
		}

		// 创建操作日志
		err = d.logService.CreateOperationLog(ctx, tx, &OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceDictItem,
			ResourceID:   item.ID,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionCreate,
			BeforeData:   nil,
			AfterData:    item,
			Description: fmt.Sprintf("用户(%d-%s)在字典(%d)创建了item(%d-%s)",
				userContext.UserId, userContext.Username, dict.ID, item.ID, item.ItemCode),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}
		return nil

	})
	if err != nil {
		return 0, err
	}
	xlogger.InfofCtx(ctx, "用户(%d)创建字典item成功: %+v", userContext.UserId, item)

	// 清除code对应item缓存
	cacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, dict.Code)
	if _, err := d.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.ErrorfCtx(ctx, "清除code对应item列表数据缓存失败: %v", err)
	}
	return item.ID, nil
}

// UpdateItem 更新字典item数据
func (d *DictService) UpdateItem(ctx context.Context, param *DictItemParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	if param.ID <= 0 {
		return 0, ErrDictCodeItemNotFound
	}
	// 获取item信息
	oldItem, err := d.dictRepo.GetDictItemById(ctx, param.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrDictCodeItemNotFound
		}
		xlogger.ErrorfCtx(ctx, "获取字典item(%d)信息异常: %v", param.ID, err)
		return 0, errors.WithMessage(err, "字典item信息查询失败")
	}

	// 检查item code是否存在
	isDuplicate, err := d.dictRepo.IsCodeItemDuplicate(ctx, oldItem.DictID, param.ItemCode, oldItem.ID)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "查询code是否存在异常: %v", err)
		return 0, errors.WithMessage(err, "查询字典编码是否存在异常")
	}
	if isDuplicate {
		return 0, ErrDictItemCodeExist
	}

	// 更新字典
	err = d.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 更新字典
		item, err := d.dictRepo.TransactionUpdateDictItem(ctx, tx, &model.SystemDictItem{
			ID:          param.ID,
			DictID:      oldItem.DictID,
			DictCode:    oldItem.DictCode,
			ItemCode:    param.ItemCode,
			ItemName:    param.ItemName,
			Status:      &param.Status,
			Description: &param.Description,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "字典item更新失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "字典item更新失败")
		}

		// 创建操作日志
		err = d.logService.CreateOperationLog(ctx, tx, &OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceDictItem,
			ResourceID:   param.ID,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionUpdate,
			BeforeData:   oldItem,
			AfterData:    item,
			Description: fmt.Sprintf("用户(%d-%s)修改了字典item(%d-%s)信息",
				userContext.UserId, userContext.Username, param.ID, param.ItemCode),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}
		return nil
	})
	xlogger.InfofCtx(ctx, "用户(%d)更新字典item成功: old=%+v new=%+v", userContext.UserId, oldItem, param)

	// 清除code对应item缓存
	cacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, oldItem.DictCode)
	if _, err := d.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.ErrorfCtx(ctx, "清除code对应item列表数据缓存失败: %v", err)
	}
	return param.ID, nil
}

// DeleteItemById 删除字典item
func (d *DictService) DeleteItemById(ctx context.Context, id int32) error {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return err
	}

	if id <= 0 {
		return ErrDictCodeItemNotFound
	}
	// 获取item信息
	item, err := d.dictRepo.GetDictItemById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDictCodeItemNotFound
		}
		xlogger.ErrorfCtx(ctx, "获取字典item(%d)信息异常: %v", id, err)
		return errors.WithMessage(err, "字典item信息查询失败")
	}

	// 删除字典item
	err = d.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 删除字典item
		err = d.dictRepo.TransactionDeleteItemByID(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "字典(%d)枚举删除失败:  err: %v", id, err)
			return errors.WithMessage(err, "字典枚举删除失败")
		}

		// 创建操作日志
		err = d.logService.CreateOperationLog(ctx, tx, &OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceDictItem,
			ResourceID:   id,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionDelete,
			BeforeData:   nil,
			AfterData:    nil,
			Description: fmt.Sprintf("用户(%d-%s)删除了字典item(%d)信息",
				userContext.UserId, userContext.Username, id),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", id, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}
		return nil
	})
	xlogger.InfofCtx(ctx, "用户(%d)删除字典item(%d)成功", userContext.UserId, id)

	// 清除code对应item缓存
	cacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, item.DictCode)
	if _, err := d.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.ErrorfCtx(ctx, "清除code对应item列表数据缓存失败: %v", err)
	}
	return nil
}
