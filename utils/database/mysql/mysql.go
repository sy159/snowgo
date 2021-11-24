package mysql

import (
	"fmt"
	"log"
	"os"
	"snowgo/config"
	. "snowgo/utils/logger"
	"time"

	"gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm/schema"

	"gorm.io/gorm"
)

var DB *gorm.DB

// InitMysql 初始化mysql连接,设置全局mysql db
func InitMysql() {
	db, err := connectMysql(config.MysqlConf)
	if err != nil {
		Panicf("redis init failed, err is %s", err.Error())
	}
	DB = db
}

// 连接mysql
func connectMysql(config config.MysqlConfig) (db *gorm.DB, err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&parseTime=%t&loc=%s",
		config.User, config.Password, config.Addr, config.Database, config.Charset, config.ParseTime, config.Loc,
	)

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
			SlowThreshold: time.Duration(config.SlowSqlTime) * time.Millisecond, //慢SQL阈值 默认200ms
			LogLevel:      logger.Info,                                          // info表示所有都打印，warn值打印慢sql
			Colorful:      true,                                                 // 彩色打印开启
		})
		gormConfig.Logger = loggerNew
	}

	// 建立连接
	db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}

	// 设置连接池信息
	sqlDB, err := db.DB()

	if err != nil {
		return nil, err
	}
	// 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	// 设置打开数据库连接的最大数量 默认值为0表示不限制，可以避免并发太高导致连接mysql出现too many connections的错误。
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	// 设置了连接可复用的最大时间。单位min
	sqlDB.SetConnMaxLifetime(time.Duration(config.MaxLifeTime) * time.Minute)

	// 使用插件
	//err := db.Use(&TracePlugin{})

	return db, nil
}
