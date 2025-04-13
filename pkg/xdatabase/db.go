package xdatabase

import (
	"context"
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
func (d *BaseRepository) GetDB(dbName string) *gorm.DB {
	return d.dbMap[dbName]
}

// GetBaseRepository 其他库的repo
func (d *BaseRepository) GetBaseRepository(dbName string) *BaseRepository {
	return &BaseRepository{
		db: d.GetDB(dbName),
	}
}

// Use 使用read or w db
func (d *BaseRepository) Use(op dbresolver.Operation) *BaseRepository {
	return &BaseRepository{
		db: d.db,
		op: op,
	}
}

func (d *BaseRepository) DB() *gorm.DB {
	// 读写分离
	if len(d.op) != 0 {
		return d.db.Clauses(d.op)
	}
	return d.db
}

func (d *BaseRepository) Save(ctx context.Context, value interface{}) *gorm.DB {
	return d.DB().WithContext(ctx).Save(value)
}
