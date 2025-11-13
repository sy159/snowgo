package mysql

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
	"log"
	"os"
	"runtime"
	"snowgo/config"
	"snowgo/pkg/xcolor"
	"strings"
	"time"
)

type MyDB struct {
	DB    *gorm.DB
	DbMap map[string]*gorm.DB
}

// NewMysql 创建mysql db
func NewMysql(cfg config.MysqlConfig, otherCfg config.OtherDBConfig) (*MyDB, error) {
	if len(cfg.DSN) == 0 && len(cfg.MainsDSN) == 0 && len(cfg.SlavesDSN) == 0 {
		return nil, errors.New("Please initialize mysql configuration first")
	}
	db, err := connectMysql(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	myDB := &MyDB{
		DB:    db,
		DbMap: map[string]*gorm.DB{"default": db},
	}
	for k, v := range otherCfg.DBMap {
		otherDb, err := connectMysql(v)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		myDB.DbMap[k] = otherDb
	}
	return myDB, nil
}

// 设置数据库连接池相关参数的默认值
func processConfig(cfg config.MysqlConfig) config.MysqlConfig {
	processCfg := cfg
	if processCfg.MaxOpenConns <= 0 {
		processCfg.MaxOpenConns = 100
	}
	if processCfg.MaxIdleConns <= 0 {
		processCfg.MaxIdleConns = runtime.NumCPU()*2 + 1
	}
	// 保证 idle <= open
	if processCfg.MaxIdleConns > processCfg.MaxOpenConns {
		processCfg.MaxIdleConns = processCfg.MaxOpenConns
	}
	if processCfg.ConnMaxLifeTime <= 0 {
		processCfg.ConnMaxLifeTime = 180 // 单位分钟
	}
	if processCfg.ConnMaxIdleTime <= 0 {
		processCfg.ConnMaxIdleTime = 30 // 单位分钟
	}

	if processCfg.SlowThresholdTime <= 0 {
		processCfg.SlowThresholdTime = 2000 // 单位毫秒
	}
	return processCfg
}

func ensureTimeout(dsn string) string {
	if strings.Contains(dsn, "timeout=") {
		return dsn // 已设置timeout跳过
	}
	// 自动追加参数
	if strings.Contains(dsn, "?") {
		return dsn + "&timeout=5s"
	}
	return dsn + "?timeout=5s"
}

// 连接mysql
func connectMysql(mysqlConfig config.MysqlConfig) (db *gorm.DB, err error) {
	if mysqlConfig.DSN == "" {
		return nil, errors.New("mysql init failed, dsn is empty")
	}

	mysqlConfig = processConfig(mysqlConfig) // 处理配置

	// 连接额外配置信息
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   mysqlConfig.TablePre, // 表前缀
			SingularTable: true,                 // 使用单数表名，启用该选项时，`User` 的表名应该是 `user`而不是users
		},
		SkipDefaultTransaction: true,
	}

	// 打印SQL设置
	if mysqlConfig.PrintSqlLog {
		cfg := config.Get()
		loggerNew := logger.New(
			log.New(
				os.Stdout,
				fmt.Sprintf("\r\n%s %s",
					xcolor.GreenFont(fmt.Sprintf("[%s:%s]", cfg.Application.Server.Name, cfg.Application.Server.Version)),
					xcolor.YellowFont("[mysql] | "),
				),
				log.LstdFlags,
			),
			logger.Config{
				SlowThreshold:             time.Duration(mysqlConfig.SlowThresholdTime) * time.Millisecond, //慢SQL阈值
				LogLevel:                  logger.Info,                                                     // info表示所有都打印，warn值打印慢sql
				Colorful:                  true,                                                            // 彩色打印开启
				IgnoreRecordNotFoundError: true,
			})
		gormConfig.Logger = loggerNew
	}

	// 建立连接
	dsn := ensureTimeout(mysqlConfig.DSN) // 处理链接超时
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
	sqlDB.SetMaxIdleConns(mysqlConfig.MaxIdleConns)
	// 设置打开数据库连接的最大数量 默认值为0表示不限制，可以避免并发太高导致连接mysql出现too many connections的错误。
	sqlDB.SetMaxOpenConns(mysqlConfig.MaxOpenConns)
	// 设置了连接可复用的最大时间。单位min
	sqlDB.SetConnMaxLifetime(time.Duration(mysqlConfig.ConnMaxLifeTime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(mysqlConfig.ConnMaxIdleTime) * time.Minute)

	// 未开启读写分离配置
	if !mysqlConfig.SeparationRW {
		return db, nil
	} else {
		if len(mysqlConfig.MainsDSN) == 0 {
			return nil, errors.New("读写分离需要配置主库地址")
		}
		if len(mysqlConfig.SlavesDSN) == 0 {
			mysqlConfig.SlavesDSN = mysqlConfig.MainsDSN
		}
	}

	// 读写分离配置
	var sources []gorm.Dialector
	var replicas []gorm.Dialector
	if len(mysqlConfig.MainsDSN) > 0 {
		for _, uri := range mysqlConfig.MainsDSN {
			sources = append(sources, mysql.Open(ensureTimeout(uri)))
		}
	}
	if len(mysqlConfig.SlavesDSN) > 0 {
		for _, uri := range mysqlConfig.SlavesDSN {
			replicas = append(replicas, mysql.Open(ensureTimeout(uri)))
		}
	}
	// 使用插件
	//err := db.Use(&TracePlugin{})
	err = db.Use(dbresolver.Register(dbresolver.Config{
		Sources:  sources,
		Replicas: replicas,
		Policy:   dbresolver.RandomPolicy{}, // RoundRobinPolicy 轮询； RandomPolicy 随机； WeightedPolicy 权重
	}).SetMaxOpenConns(mysqlConfig.MaxOpenConns).
		SetMaxIdleConns(mysqlConfig.MaxIdleConns).
		SetConnMaxIdleTime(time.Duration(mysqlConfig.ConnMaxIdleTime) * time.Minute).
		SetConnMaxLifetime(time.Duration(mysqlConfig.ConnMaxLifeTime) * time.Minute))
	if err != nil {
		return nil, err
	}

	return db, nil
}

// CloseMysql 关闭数据库连接
func closeMysql(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

// CloseAllMysql 关闭所有数据库连接
func (m *MyDB) CloseAllMysql() {
	visited := make(map[*gorm.DB]bool, 4)

	if _, ok := visited[m.DB]; !ok {
		closeMysql(m.DB)
		visited[m.DB] = true
	}

	for _, v := range m.DbMap {
		if _, ok := visited[v]; !ok {
			closeMysql(v)
			visited[v] = true
		}
	}
}

// CheckDBAlive 检查数据库连接是否存活
func (m *MyDB) CheckDBAlive(ctx context.Context) (bool, error) {
	for _, db := range m.DbMap {
		if db == nil {
			return false, errors.New("db is nil")
		}
		sqlDB, err := db.DB()
		if err != nil {
			return false, errors.WithStack(err)
		}
		// 尝试执行简单的查询来检查连接是否存活
		err = sqlDB.PingContext(ctx)
		if err != nil {
			return false, errors.WithStack(err)
		}
	}
	return true, nil
}

// GetDB 返回指定名字的 gorm.DB（name = "default" 若不传）
func (m *MyDB) GetDB(name string) (*gorm.DB, error) {
	if name == "" {
		name = "default"
	}
	db, ok := m.DbMap[name]
	if !ok || db == nil {
		return nil, fmt.Errorf("db %s not found", name)
	}
	return db, nil
}
