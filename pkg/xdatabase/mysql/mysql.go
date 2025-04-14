package mysql

import (
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
	"log"
	"os"
	"snowgo/config"
	. "snowgo/pkg/xlogger"
	"time"
)

var DB *gorm.DB
var DbMap = map[string]*gorm.DB{}

// InitMysql 初始化mysql连接,设置全局mysql db
func InitMysql() {
	if len(config.MysqlConf.DSN) == 0 && len(config.MysqlConf.MainsDSN) == 0 && len(config.MysqlConf.SlavesDSN) == 0 {
		Panic("Please initialize mysql configuration first")
	}
	db, err := connectMysql(config.MysqlConf)
	if err != nil {
		Panicf("mysql init failed, err is %s", err.Error())
	}
	DB = db

	DbMap["default"] = db
	for k, v := range config.OtherMapConf.DbMap {
		otherDb, err := connectMysql(v)
		if err != nil {
			Panicf("mysql %s init failed, err is %s", k, err.Error())
		}
		DbMap[k] = otherDb
	}
}

// 连接mysql
func connectMysql(config config.MysqlConfig) (db *gorm.DB, err error) {
	if config.DSN == "" {
		return nil, errors.New("mysql init failed, dsn is empty")
	}

	// 连接额外配置信息
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   config.TablePre, // 表前缀
			SingularTable: true,            // 使用单数表名，启用该选项时，`User` 的表名应该是 `user`而不是users
		},
		SkipDefaultTransaction: true,
	}

	// 打印SQL设置
	if config.PrintSqlLog {
		loggerNew := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             time.Duration(config.SlowThresholdTime) * time.Millisecond, //慢SQL阈值
			LogLevel:                  logger.Info,                                                // info表示所有都打印，warn值打印慢sql
			Colorful:                  true,                                                       // 彩色打印开启
			IgnoreRecordNotFoundError: true,
		})
		gormConfig.Logger = loggerNew
	}

	// 建立连接
	db, err = gorm.Open(mysql.Open(config.DSN), gormConfig)
	if err != nil {
		return nil, err
	}

	// 设置连接池信息
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(config.GetMaxIdleConn())
	// 设置打开数据库连接的最大数量 默认值为0表示不限制，可以避免并发太高导致连接mysql出现too many connections的错误。
	sqlDB.SetMaxOpenConns(config.GetMaxOpenConn())
	// 设置了连接可复用的最大时间。单位min
	sqlDB.SetConnMaxLifetime(time.Duration(config.GetConnMaxLifeTime()) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(config.GetConnMaxIdleTime()) * time.Minute)

	// 未开启读写分离配置
	if !config.SeparationRW {
		return db, nil
	}

	// 读写分离配置
	var sources []gorm.Dialector
	var replicas []gorm.Dialector
	if len(config.MainsDSN) > 0 {
		for _, uri := range config.MainsDSN {
			sources = append(sources, mysql.Open(uri))
		}
	}
	if len(config.SlavesDSN) > 0 {
		for _, uri := range config.SlavesDSN {
			replicas = append(replicas, mysql.Open(uri))
		}
	}
	// 使用插件
	//err := db.Use(&TracePlugin{})
	err = db.Use(dbresolver.Register(dbresolver.Config{
		Sources:  sources,
		Replicas: replicas,
		Policy:   dbresolver.RandomPolicy{},
	}).SetMaxOpenConns(config.GetMaxOpenConn()).
		SetMaxIdleConns(config.GetMaxIdleConn()).
		SetConnMaxIdleTime(time.Duration(config.GetConnMaxIdleTime()) * time.Minute).
		SetConnMaxLifetime(time.Duration(config.GetConnMaxLifeTime()) * time.Minute))
	if err != nil {
		return nil, err
	}

	return db, nil
}

// CloseMysql 关闭数据库连接
func CloseMysql(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

// CloseAllMysql 关闭所有数据库连接
func CloseAllMysql(db *gorm.DB, dbMap map[string]*gorm.DB) {
	visited := make(map[*gorm.DB]bool, 4)

	if _, ok := visited[db]; !ok {
		CloseMysql(db)
		visited[db] = true
	}

	for _, v := range dbMap {
		if _, ok := visited[v]; !ok {
			CloseMysql(v)
			visited[v] = true
		}
	}
}
