package repo

import (
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
	"snowgo/internal/dal/query"
	"snowgo/pkg/database"
	"snowgo/pkg/database/mysql"
	"snowgo/pkg/logger"
)

type Repository struct {
	query      *query.Query
	writeQuery *query.Query
	readQuery  *query.Query
	db         *database.BaseRepository
}

func NewRepository() *Repository {
	if mysql.DB.Error != nil {
		logger.Panic("Please initialize mysql first")
	}
	baseRepo := database.NewBaseRepository(mysql.DB, mysql.DbMap)
	return &Repository{
		query:      query.Use(baseRepo.DB()),
		writeQuery: query.Use(baseRepo.Use(dbresolver.Write).DB()),
		readQuery:  query.Use(baseRepo.Use(dbresolver.Read).DB()),
		db:         baseRepo,
	}
}

// Query 根据读写情况选择读写db
func (db *Repository) Query() *query.Query {
	return db.query
}

// WriteQuery 主库db
func (db *Repository) WriteQuery() *query.Query {
	return db.writeQuery
}

func (db *Repository) ReadQuery() *query.Query {
	return db.readQuery
}

// DB 获取DB
func (db *Repository) DB() *gorm.DB {
	return db.db.DB()
}

// ChangeDB 切换其他的db连接
func (db *Repository) ChangeDB(dbName string) *Repository {
	repository := db.db.GetBaseRepository(dbName)
	return &Repository{
		query:      query.Use(repository.DB()),
		writeQuery: query.Use(repository.Use(dbresolver.Write).DB()),
		readQuery:  query.Use(repository.Use(dbresolver.Read).DB()),
		db:         repository,
	}
}
