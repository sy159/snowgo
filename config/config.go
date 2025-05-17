package config

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/spf13/viper"
	"snowgo/pkg/xenv"
)

// 配置原子存储和文件路径映射
var (
	configAtomic      atomic.Value
	configFilePathMap = map[string]string{
		xenv.ProdConstant: "config.prod",
		xenv.UatConstant:  "config.uat",
		xenv.DevConstant:  "config.dev",
		"container":       "config.container",
	}
	initFlag          uint32
	defaultConfigPath = "./config"
)

// Config 全局配置结构体
type Config struct {
	Application ApplicationConfig `mapstructure:"application"`
	Log         LogConfig         `mapstructure:"log"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Mysql       MysqlConfig       `mapstructure:"mysql"`
	Jwt         JwtConfig         `mapstructure:"jwt"`
	OtherDB     OtherDBConfig     `mapstructure:"dbMap"`
}

// ApplicationConfig 应用基础配置
type ApplicationConfig struct {
	EnableAccessLog bool         `mapstructure:"enableAccessLog"`
	EnablePprof     bool         `mapstructure:"enablePprof"`
	Server          ServerConfig `mapstructure:"server"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Name         string `mapstructure:"name"`
	Version      string `mapstructure:"version"`
	Addr         string `mapstructure:"addr"`
	Port         uint32 `mapstructure:"port"`
	ReadTimeout  uint   `mapstructure:"readTimeout"`
	WriteTimeout uint   `mapstructure:"writeTimeout"`
	MaxHeaderMB  int    `mapstructure:"maxHeaderMB"`
}

// LogConfig 日志配置
type LogConfig struct {
	Writer               string `mapstructure:"writer"`
	AccountEncoder       string `mapstructure:"accountEncoder"`
	LogEncoder           string `mapstructure:"logEncoder"`
	AccountFileMaxAgeDay uint   `mapstructure:"accountFileMaxAgeDay"`
	LogFileMaxAgeDay     uint   `mapstructure:"logFileMaxAgeDay"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	DialTimeout  int    `mapstructure:"dialTimeout"`
	ReadTimeout  int    `mapstructure:"readTimeout"`
	WriteTimeout int    `mapstructure:"writeTimeout"`
	MinIdleConns int    `mapstructure:"minIdleConns"`
	IdleTimeout  int    `mapstructure:"idleTimeout"`
	PoolSize     int    `mapstructure:"poolSize"`
}

// MysqlConfig MySQL配置
type MysqlConfig struct {
	SeparationRW      bool     `mapstructure:"separationRW"`
	DSN               string   `mapstructure:"dsn"`
	TablePre          string   `mapstructure:"table_pre"`
	MaxIdleConns      int      `mapstructure:"maxIdleConns"`
	MaxOpenConns      int      `mapstructure:"maxOpenConns"`
	ConnMaxIdleTime   int      `mapstructure:"connMaxIdleTime"`
	ConnMaxLifeTime   int      `mapstructure:"connMaxLifeTime"`
	PrintSqlLog       bool     `mapstructure:"printSqlLog"`
	SlowThresholdTime int      `mapstructure:"slowThresholdTime"`
	MainsDSN          []string `mapstructure:"mainsDSN"`
	SlavesDSN         []string `mapstructure:"slavesDSN"`
}

// JwtConfig JWT配置
type JwtConfig struct {
	Issuer                string `mapstructure:"issuer"`
	JwtSecret             string `mapstructure:"jwtSecret"`
	AccessExpirationTime  int    `mapstructure:"accessExpirationTime"`
	RefreshExpirationTime int    `mapstructure:"refreshExpirationTime"`
}

// OtherDBConfig 其他数据库配置
type OtherDBConfig struct {
	DBMap map[string]MysqlConfig `mapstructure:",remain"`
}

// Get 获取当前配置（线程安全）
func Get() Config {
	if cfg := configAtomic.Load(); cfg != nil {
		return cfg.(Config)
	}
	return Config{}
}

// Init 初始化配置（自动根据环境加载）
func Init(configPath string) {
	if !atomic.CompareAndSwapUint32(&initFlag, 0, 1) {
		panic("config: already initialized")
	}

	env := xenv.Env()
	configName, ok := configFilePathMap[env]
	if !ok {
		panic(fmt.Sprintf("config: unknown environment %q", env))
	}

	v := initViper(configName, configPath)

	// 初始加载配置
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("config: failed to read config: %v", err))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Sprintf("config: failed to unmarshal config: %v", err))
	}
	configAtomic.Store(cfg)

	// 非生产环境启用热加载
	//if env != xenv.ProdConstant {
	//	enableHotReload(v)
	//}
}

// initViper 初始化Viper实例
func initViper(configName, configPath string) *viper.Viper {
	if len(configPath) == 0 {
		configPath = defaultConfigPath
	}
	v := viper.New()
	v.SetConfigName(configName)
	v.AddConfigPath(configPath)
	v.SetConfigType("yaml") // 根据实际配置文件类型设置
	return v
}

//enableHotReload 启用配置热加载
//func enableHotReload(v *viper.Viper) {
//	v.WatchConfig()
//	v.OnConfigChange(func(e fsnotify.Event) {
//		fmt.Printf("config: detected config change: %s\n", e.Name)
//		var newCfg Config
//		if err := v.Unmarshal(&newCfg); err != nil {
//			fmt.Printf("config: failed to reload config: %v\n", err)
//			return
//		}
//		configAtomic.Store(newCfg)
//		fmt.Println("config: configuration reloaded successfully")
//	})
//}

// GetMaxOpenConn 最大打开连接数
func (m MysqlConfig) GetMaxOpenConn() int {
	if m.MaxOpenConns <= 0 {
		return 100
	}
	return m.MaxOpenConns
}

// GetMaxIdleConn 最大空闲连接数
func (m MysqlConfig) GetMaxIdleConn() int {
	if m.MaxIdleConns <= 0 {
		return runtime.NumCPU()*2 + 1
	}
	return m.MaxIdleConns
}

// GetConnMaxIdleTime  链接最大等待时间
func (m MysqlConfig) GetConnMaxIdleTime() int {
	if m.ConnMaxIdleTime <= 0 {
		return 30
	}
	return m.ConnMaxIdleTime
}

// GetConnMaxLifeTime 连接最多使用时间
func (m MysqlConfig) GetConnMaxLifeTime() int {
	if m.ConnMaxLifeTime <= 0 {
		return 180
	}
	return m.ConnMaxLifeTime
}
