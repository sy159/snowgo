package system

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gen"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"time"
)

// DictDao 系统字典
type DictDao struct {
	repo *repo.Repository
}

func NewDictDao(repo *repo.Repository) *DictDao {
	return &DictDao{
		repo: repo,
	}
}

type DictListCondition struct {
	Name      string     `json:"name" form:"name"`
	Code      string     `json:"code" form:"code"`
	StartTime *time.Time `json:"start_time" form:"start_time"`
	EndTime   *time.Time `json:"end_time" form:"end_time"`
	Offset    int32      `json:"offset" form:"offset"`
	Limit     int32      `json:"limit" form:"limit"`
}

// GetDictById 查询字典by id
func (d *DictDao) GetDictById(ctx context.Context, dictId int32) (*model.SystemDict, error) {
	if dictId <= 0 {
		return nil, errors.New("字典id不存在")
	}
	m := d.repo.Query().SystemDict
	dict, err := m.WithContext(ctx).Where(m.ID.Eq(dictId)).First()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return dict, nil
}

// GetDictList 字典列表
func (d *DictDao) GetDictList(ctx context.Context, condition *DictListCondition) ([]*model.SystemDict, int64, error) {
	m := d.repo.Query().SystemDict
	dictList, total, err := m.WithContext(ctx).
		Scopes(
			d.NameScope(condition.Name),
			d.CodeScope(condition.Code),
			d.StartTimeScope(condition.StartTime),
			d.EndTimeScope(condition.EndTime),
		).
		FindByPage(int(condition.Offset), int(condition.Limit))
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return dictList, total, nil
}

func (d *DictDao) NameScope(name string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(name) == 0 {
			return tx
		}
		m := d.repo.Query().SystemDict
		return tx.Where(m.Name.Like("%" + name + "%"))
	}
}

func (d *DictDao) CodeScope(code string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(code) == 0 {
			return tx
		}
		m := d.repo.Query().SystemDict
		return tx.Where(m.Code.Like("%" + code + "%"))
	}
}

func (d *DictDao) StartTimeScope(starTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if starTime == nil {
			return tx
		}
		m := d.repo.Query().SystemDict
		tx = tx.Where(m.CreatedAt.Gte(*starTime))
		return tx
	}
}

func (d *DictDao) EndTimeScope(endTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if endTime == nil {
			return tx
		}
		m := d.repo.Query().SystemDict
		tx = tx.Where(m.CreatedAt.Lte(*endTime))
		return tx
	}
}

func (d *DictDao) IsCodeDuplicate(ctx context.Context, code string, dictId int32) (bool, error) {
	m := d.repo.Query().SystemDict
	dictQuery := m.WithContext(ctx).
		Select(m.ID).
		Where(m.Code.Eq(code))
	if dictId > 0 {
		dictQuery = dictQuery.Where(m.ID.Neq(dictId))
	}
	_, err := dictQuery.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, errors.WithStack(err)
	}
	return true, nil
}

// TransactionCreateDict 创建字典
func (d *DictDao) TransactionCreateDict(ctx context.Context, tx *query.Query, dict *model.SystemDict) (*model.SystemDict, error) {
	err := tx.WithContext(ctx).SystemDict.Create(dict)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return dict, nil
}

// TransactionUpdateDict 更新字典
func (d *DictDao) TransactionUpdateDict(ctx context.Context, tx *query.Query, dict *model.SystemDict) (*model.SystemDict, error) {
	if dict.ID <= 0 {
		return nil, errors.New("字典id不存在")
	}
	err := tx.WithContext(ctx).SystemDict.Where(tx.SystemDict.ID.Eq(dict.ID)).Save(dict)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return dict, nil
}

// TransactionDeleteById 删除字典
func (d *DictDao) TransactionDeleteById(ctx context.Context, tx *query.Query, id int32) error {
	if id <= 0 {
		return errors.New("字典id不存在")
	}
	_, err := tx.WithContext(ctx).SystemDict.Where(tx.SystemDict.ID.Eq(id)).Delete()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// GetItemListByDictCode 查询字典枚举by code
func (d *DictDao) GetItemListByDictCode(ctx context.Context, dictCode string) ([]*model.SystemDictItem, error) {
	if len(dictCode) == 0 {
		return nil, errors.New("字典code不存在")
	}
	m := d.repo.Query().SystemDictItem
	itemList, err := m.WithContext(ctx).
		Where(m.DictCode.Eq(dictCode)).
		Order(m.SortOrder.Asc(), m.ID.Asc()).
		Find()

	if err != nil {
		return nil, errors.WithStack(err)
	}
	return itemList, nil
}
