package xdatabase

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

type BaseRepository struct {
	op    dbresolver.Operation // 读写指定
	db    *gorm.DB             // 默认db
	dbMap map[string]*gorm.DB  // 多个数据库连接使用
}

func NewBaseRepository(db *gorm.DB, dbMap map[string]*gorm.DB) *BaseRepository {
	return &BaseRepository{
		op:    "",
		db:    db,
		dbMap: dbMap,
	}
}

// GetDB 根据库名获取其他的数据库连接
func (d *BaseRepository) GetDB(dbName string) (*gorm.DB, error) {
	if d.dbMap == nil {
		return nil, fmt.Errorf("database %q not found: dbMap is nil", dbName)
	}
	db, ok := d.dbMap[dbName]
	if !ok || db == nil {
		return nil, fmt.Errorf("database %q not found", dbName)
	}
	return db, nil
}

// GetBaseRepository 其他库的repo
func (d *BaseRepository) GetBaseRepository(dbName string) (*BaseRepository, error) {
	db, err := d.GetDB(dbName)
	if err != nil {
		return nil, err
	}
	return &BaseRepository{
		db:    db,
		dbMap: d.dbMap,
	}, nil
}

// Use 使用read or w db
func (d *BaseRepository) Use(op dbresolver.Operation) *BaseRepository {
	return &BaseRepository{
		db:    d.db,
		dbMap: d.dbMap,
		op:    op,
	}
}

func (d *BaseRepository) DB() *gorm.DB {
	// 读写分离
	if len(d.op) != 0 {
		return d.db.Clauses(d.op)
	}
	return d.db
}

func (d *BaseRepository) Save(ctx context.Context, value any) *gorm.DB {
	return d.DB().WithContext(ctx).Save(value)
}
