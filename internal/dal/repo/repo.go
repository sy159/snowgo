package repo

import (
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
	"snowgo/internal/dal/query"
	"snowgo/utils/database"
	"snowgo/utils/database/mysql"
	"snowgo/utils/logger"
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
