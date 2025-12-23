package system

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gen"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
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
	Status    string     `json:"status" form:"status"`
	StartTime *time.Time `json:"start_time" form:"start_time"`
	EndTime   *time.Time `json:"end_time" form:"end_time"`
	Offset    int32      `json:"offset" form:"offset"`
	Limit     int32      `json:"limit" form:"limit"`
}

// GetDictList 字典列表
func (d *DictDao) GetDictList(ctx context.Context, condition *DictListCondition) ([]*model.SystemDict, int64, error) {
	m := d.repo.Query().SystemDict
	dictList, total, err := m.WithContext(ctx).
		Scopes(
			d.NameScope(condition.Name),
			d.CodeScope(condition.Code),
			d.StatusScope(condition.Status),
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

func (d *DictDao) StatusScope(status string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(status) == 0 {
			return tx
		}
		m := d.repo.Query().SystemDict
		tx = tx.Where(m.Status.Eq(status))
		return tx
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

func (d *DictDao) CreateDict(ctx context.Context, dict *model.SystemDict) (*model.SystemDict, error) {
	m := d.repo.Query().SystemDict
	err := m.WithContext(ctx).Create(dict)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return dict, nil
}
