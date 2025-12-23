package system

import (
	"context"
	"github.com/pkg/errors"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
	daoSystem "snowgo/internal/dao/system"
	"snowgo/pkg/xlogger"
	"time"
)

// DictRepo 定义opt dict相关db操作接口
type DictRepo interface {
	GetDictList(ctx context.Context, condition *daoSystem.DictListCondition) ([]*model.SystemDict, int64, error)
}

type DictService struct {
	db       *repo.Repository
	dictRepo DictRepo
}

func NewDictService(db *repo.Repository, dictRepo DictRepo) *DictService {
	return &DictService{
		db:       db,
		dictRepo: dictRepo,
	}
}

type DictListCondition struct {
	Name      string `json:"name" form:"name"`
	Code      string `json:"code" form:"code"`
	Status    string `json:"status" form:"status"`
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`
	Offset    int32  `json:"offset" form:"offset"`
	Limit     int32  `json:"limit" form:"limit"`
}

type DictInfo struct {
	ID          int32
	Code        string
	Name        string
	Status      *string
	Description *string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

type DictList struct {
	List  []*DictInfo
	Total int64
}

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
		Status:    condition.Status,
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
	for _, dictInfo := range dictList {
		dictInfoList = append(dictInfoList, &DictInfo{
			ID:          dictInfo.ID,
			Name:        dictInfo.Name,
			Code:        dictInfo.Code,
			Status:      dictInfo.Status,
			Description: dictInfo.Description,
			CreatedAt:   dictInfo.CreatedAt,
			UpdatedAt:   dictInfo.UpdatedAt,
		})
	}
	return &DictList{List: dictInfoList, Total: total}, nil
}
