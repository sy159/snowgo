package system

import (
	"context"
	"errors"
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
func (d *DictDao) GetDictById(ctx context.Context, dictId int32) (*model.SysDict, error) {
	if dictId <= 0 {
		return nil, errors.New("字典id不存在")
	}
	m := d.repo.Query().SysDict
	dict, err := m.WithContext(ctx).Where(m.ID.Eq(dictId)).First()
	if err != nil {
		return nil, err
	}
	return dict, nil
}

// GetDictList 字典列表
func (d *DictDao) GetDictList(ctx context.Context, condition *DictListCondition) ([]*model.SysDict, int64, error) {
	m := d.repo.Query().SysDict
	dictList, total, err := m.WithContext(ctx).
		Scopes(
			d.NameScope(condition.Name),
			d.CodeScope(condition.Code),
			d.StartTimeScope(condition.StartTime),
			d.EndTimeScope(condition.EndTime),
		).
		FindByPage(int(condition.Offset), int(condition.Limit))
	if err != nil {
		return nil, 0, err
	}
	return dictList, total, nil
}

func (d *DictDao) NameScope(name string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(name) == 0 {
			return tx
		}
		m := d.repo.Query().SysDict
		return tx.Where(m.Name.Like("%" + name + "%"))
	}
}

func (d *DictDao) CodeScope(code string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(code) == 0 {
			return tx
		}
		m := d.repo.Query().SysDict
		return tx.Where(m.Code.Like("%" + code + "%"))
	}
}

func (d *DictDao) StartTimeScope(starTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if starTime == nil {
			return tx
		}
		m := d.repo.Query().SysDict
		tx = tx.Where(m.CreatedAt.Gte(*starTime))
		return tx
	}
}

func (d *DictDao) EndTimeScope(endTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if endTime == nil {
			return tx
		}
		m := d.repo.Query().SysDict
		tx = tx.Where(m.CreatedAt.Lte(*endTime))
		return tx
	}
}

func (d *DictDao) IsCodeDuplicate(ctx context.Context, code string, dictId int32) (bool, error) {
	m := d.repo.Query().SysDict
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
		return true, err
	}
	return true, nil
}

// CreateDict 创建字典
func (d *DictDao) CreateDict(ctx context.Context, q *query.Query, dict *model.SysDict) (*model.SysDict, error) {
	err := q.WithContext(ctx).SysDict.Create(dict)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

// UpdateDict 更新字典
func (d *DictDao) UpdateDict(ctx context.Context, q *query.Query, dict *model.SysDict) (*model.SysDict, error) {
	if dict.ID <= 0 {
		return nil, errors.New("字典id不存在")
	}
	err := q.WithContext(ctx).SysDict.Where(q.SysDict.ID.Eq(dict.ID)).Save(dict)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

// DeleteById 删除字典
func (d *DictDao) DeleteById(ctx context.Context, q *query.Query, id int32) error {
	if id <= 0 {
		return errors.New("字典id不存在")
	}
	_, err := q.WithContext(ctx).SysDict.Where(q.SysDict.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return nil
}

// DeleteItemByDictID 删除字典枚举 by id
func (d *DictDao) DeleteItemByDictID(ctx context.Context, q *query.Query, dictId int32) error {
	if dictId <= 0 {
		return errors.New("字典id不存在")
	}
	_, err := q.WithContext(ctx).SysDictItem.Where(q.SysDictItem.DictID.Eq(dictId)).Delete()
	if err != nil {
		return err
	}
	return nil
}

// UpdateItemByDictID 更新字典枚举dict code by id
func (d *DictDao) UpdateItemByDictID(ctx context.Context, q *query.Query, dictId int32, dictCode string) error {
	if dictId <= 0 {
		return errors.New("字典id不存在")
	}
	_, err := q.WithContext(ctx).SysDictItem.
		Where(q.SysDictItem.DictID.Eq(dictId)).
		UpdateSimple(q.SysDictItem.DictCode.Value(dictCode))
	if err != nil {
		return err
	}
	return nil
}

// GetItemListByDictCode 查询字典枚举by code
func (d *DictDao) GetItemListByDictCode(ctx context.Context, dictCode string) ([]*model.SysDictItem, error) {
	if len(dictCode) == 0 {
		return nil, errors.New("字典code不存在")
	}
	m := d.repo.Query().SysDictItem
	itemList, err := m.WithContext(ctx).
		Where(m.DictCode.Eq(dictCode)).
		Order(m.SortOrder.Asc(), m.ID.Asc()).
		Find()

	if err != nil {
		return nil, err
	}
	return itemList, nil
}

// IsCodeItemDuplicate 判断同一个dict下item的code是否有存在的
func (d *DictDao) IsCodeItemDuplicate(ctx context.Context, dictId int32, itemCode string, dictItemId int32) (bool, error) {
	m := d.repo.Query().SysDictItem
	dictItemQuery := m.WithContext(ctx).
		Select(m.ID).
		Where(m.DictID.Eq(dictId), m.ItemCode.Eq(itemCode))
	if dictItemId > 0 {
		dictItemQuery = dictItemQuery.Where(m.ID.Neq(dictItemId))
	}
	_, err := dictItemQuery.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// CreateDictItem 创建字典枚举
func (d *DictDao) CreateDictItem(ctx context.Context, q *query.Query, item *model.SysDictItem) (*model.SysDictItem, error) {
	err := q.WithContext(ctx).SysDictItem.Create(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// GetDictItemById 查询字典item by id
func (d *DictDao) GetDictItemById(ctx context.Context, itemId int32) (*model.SysDictItem, error) {
	if itemId <= 0 {
		return nil, errors.New("字典item id不存在")
	}
	m := d.repo.Query().SysDictItem
	item, err := m.WithContext(ctx).Where(m.ID.Eq(itemId)).First()
	if err != nil {
		return nil, err
	}
	return item, nil
}

// UpdateDictItem 更新字典item
func (d *DictDao) UpdateDictItem(ctx context.Context, q *query.Query, item *model.SysDictItem) (*model.SysDictItem, error) {
	if item.ID <= 0 {
		return nil, errors.New("字典item id不存在")
	}
	err := q.WithContext(ctx).SysDictItem.Where(q.SysDictItem.ID.Eq(item.ID)).Save(item)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// DeleteItemByID 删除字典枚举 by id
func (d *DictDao) DeleteItemByID(ctx context.Context, q *query.Query, id int32) error {
	if id <= 0 {
		return errors.New("字典item id不存在")
	}
	_, err := q.WithContext(ctx).SysDictItem.Where(q.SysDictItem.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return nil
}
