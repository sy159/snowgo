package config

import (
	"fmt"
	"github.com/spf13/viper"
	"snowgo/pkg/xenv"
)

var (
	configPath   = "./config"
	ServerConf   ServerConfig  // ServerConf 全局server配置
	LogConf      LogConfig     // LogConf 全局日志配置
	RedisConf    RedisConfig   // RedisConf 全局redis配置
	MysqlConf    MysqlConfig   // MysqlConfig 全局mysql配置
	OtherMapConf OtherDBConfig //OtherDBConfig 多数据库配置
	JwtConf      JwtConfig     // JwtConf 全局jwt配置
)

// ServerConfig server启动配置
type ServerConfig struct {
	//IsDebug      bool   `json:"isDebug" toml:"isDebug" yaml:"isDebug"`
	EnableAccessLog bool   `json:"enable_access_log" toml:"enableAccessLog" yaml:"enableAccessLog"`
	EnablePprof     bool   `json:"enable_pprof" toml:"enablePprof" yaml:"enablePprof"`
	Name            string `json:"name" toml:"name" yaml:"name"`
	Version         string `json:"version" toml:"version" yaml:"version"`
	Addr            string `json:"addr" toml:"addr" yaml:"addr"`
	Port            uint32 `json:"port" toml:"port" yaml:"port"`
	ReadTimeout     uint   `json:"read_timeout" toml:"readTimeout" yaml:"readTimeout" `
	WriteTimeout    uint   `json:"write_timeout" toml:"writeTimeout" yaml:"writeTimeout"`
	MaxHeaderMB     int    `json:"max_header_mb" toml:"maxHeaderMB" yaml:"maxHeaderMB"`
}

// LogConfig log配置
type LogConfig struct {
	Writer               string `json:"writer" toml:"writer" yaml:"writer"`                                               // 日志输出的位置：console控制台输出，file输出到文件，multi控制台跟日志文件同时输出
	AccountEncoder       string `json:"account_encoder" toml:"accountEncoder" yaml:"accountEncoder"`                      // 访问文件解析格式：normal正常格式输出；json输出为json
	LogEncoder           string `json:"log_encoder" toml:"logEncoder" yaml:"logEncoder"`                                  // log文件解析格式：normal正常格式输出；json输出为json
	AccountFileMaxAgeDay uint   `json:"account_file_max_age_day" toml:"accountFileMaxAgeDay" yaml:"accountFileMaxAgeDay"` // 访问文件最多保留多少天
	LogFileMaxAgeDay     uint   `json:"log_file_max_age_day" toml:"logFileMaxAgeDay" yaml:"logFileMaxAgeDay"`             // 日志文件最多保留多少天
}

// RedisConfig redis连接配置
type RedisConfig struct {
	Addr         string `validate:"required" json:"addr" toml:"addr" yaml:"addr"`                        // 地址
	Password     string `json:"password" toml:"password" yaml:"password"`                                // 密码
	DB           int    `validate:"min=0"  json:"db" toml:"db" yaml:"db"`                                // 数据库
	DialTimeout  int    `validate:"gt=0"  json:"dial_timeout" toml:"dialTimeout" yaml:"dialTimeout"`     // 拨号超时(秒)
	ReadTimeout  int    `validate:"gt=0"  json:"read_timeout" toml:"readTimeout" yaml:"readTimeout"`     // 读取超时(秒)
	WriteTimeout int    `validate:"gt=0"  json:"write_timeout" toml:"writeTimeout" yaml:"writeTimeout"`  // 写入超时(秒)
	MinIdleConns int    `validate:"gt=0"  json:"min_idle_conns" toml:"minIdleConns" yaml:"minIdleConns"` // 最小空闲连接数
	IdleTimeout  int    `validate:"gt=0"  json:"idle_timeout" toml:"idleTimeout" yaml:"idleTimeout"`     // 空闲超时(秒)
	PoolSize     int    `validate:"gt=0"  json:"pool_size" toml:"poolSize" yaml:"poolSize"`              // 连接池最大链接数
}

// MysqlConfig mysql连接配置
type MysqlConfig struct {
	SeparationRW      bool     `json:"separation_rw" toml:"separationRW" yaml:"separationRW"`                 // 表前缀
	DSN               string   `json:"dsn" toml:"dsn" yaml:"dsn"`                                             // db dsn
	TablePre          string   `json:"table_pre" toml:"table_pre" yaml:"table_pre"`                           // 表前缀
	MaxIdleConns      int      `json:"max_idle_conns" toml:"maxIdleConns" yaml:"maxIdleConns"`                // 设置闲置的连接数，默认值为2；
	MaxOpenConns      int      `json:"max_open_conns" toml:"maxOpenConns" yaml:"maxOpenConns"`                // 设置最大打开的连接数，默认值为0，表示不限制。
	ConnMaxIdleTime   int      `json:"conn_max_idle_time" toml:"connMaxIdleTime" yaml:"connMaxIdleTime"`      // 连接空闲最大等待时间。单位min
	ConnMaxLifeTime   int      `json:"conn_max_life_time" toml:"connMaxLifeTime" yaml:"connMaxLifeTime"`      // 设置了连接可复用的最大时间。单位min
	PrintSqlLog       bool     `json:"print_sql_log" toml:"printSqlLog" yaml:"printSqlLog"`                   // 是否打印SQL
	SlowThresholdTime int      `json:"slow_threshold_time" toml:"slowThresholdTime" yaml:"slowThresholdTime"` // 慢sql阈值 单位ms(在设置printSqlLog=true有用)
	MainsDSN          []string `json:"mains_dsn" toml:"mainsDSN" yaml:"mainsDSN"`                             // 主库dsn separation_rw为t生效
	SlavesDSN         []string `json:"slaves_dsn" toml:"slavesDSN" yaml:"slavesDSN"`                          // 从库dsn separation_rw为t生效
}

type OtherDBConfig struct {
	DbMap map[string]MysqlConfig `json:"db_map" toml:"db_map" yaml:"db_map"`
}

// JwtConfig jwt配置
type JwtConfig struct {
	Issuer                string `json:"issuer" toml:"issuer" yaml:"issuer"`                                                // 发布人
	JwtSecret             string `json:"jwt_secret" toml:"jwtSecret" yaml:"jwtSecret"`                                      // jwt加密秘钥
	AccessExpirationTime  int    `json:"access_expiration_time" toml:"accessExpirationTime" yaml:"accessExpirationTime"`    // 访问token到期时间，单位min
	RefreshExpirationTime int    `json:"refresh_expiration_time" toml:"refreshExpirationTime" yaml:"refreshExpirationTime"` // 刷新token到期时间，单位min

}

type Option func(option)

type option struct {
	configName string
}

var configFilePathMap = map[string]string{
	xenv.ProdConstant: "config.prod",
	xenv.UatConstant:  "config.uat",
	xenv.DevConstant:  "config.dev",
}

// GetMaxOpenConn 最大打开连接数
func (m *MysqlConfig) GetMaxOpenConn() int {
	if m.MaxOpenConns <= 0 {
		m.MaxOpenConns = 5
	}
	return m.MaxOpenConns
}

// GetMaxIdleConn 最大空闲连接数
func (m *MysqlConfig) GetMaxIdleConn() int {
	if m.MaxIdleConns <= 0 {
		m.MaxIdleConns = 2
	}
	return m.MaxIdleConns
}

// GetConnMaxIdleTime  链接最大等待时间
func (m *MysqlConfig) GetConnMaxIdleTime() int {
	if m.ConnMaxIdleTime <= 0 {
		m.ConnMaxIdleTime = 30
	}
	return m.ConnMaxIdleTime
}

// GetConnMaxLifeTime 连接最多使用时间
func (m *MysqlConfig) GetConnMaxLifeTime() int {
	if m.ConnMaxLifeTime <= 0 {
		m.ConnMaxLifeTime = 180
	}
	return m.ConnMaxLifeTime
}

// InitConf 加载所有需要配置文件
func InitConf(options ...Option) {
	configName, ok := configFilePathMap[xenv.Env()]
	if !ok {
		panic(fmt.Sprintf("env config file not found, xenv is %s", xenv.Env()))
	}

	// 加载服务配置文件
	if err := loadServerConf(configName); err != nil {
		panic(fmt.Sprintf("server config failed to load, err is %s", err))
	}

	// 加载需要注册的配置项目
	for _, f := range options {
		f(option{configName: configName})
	}

}

// 初始化服务配置
func loadServerConf(configName string) (err error) {
	v := viper.New()
	v.SetConfigName(configName) // 设置文件名称
	//v.SetConfigType("yaml")
	v.AddConfigPath(configPath) // 设置文件所在路径

	if err = v.ReadInConfig(); err != nil {
		fmt.Println(err)
		return
	}

	//ServerConf.IsDebug = isDebug
	ServerConf.EnableAccessLog = v.GetBool("application.enableAccessLog")
	ServerConf.EnablePprof = v.GetBool("application.enablePprof")
	ServerConf.Name = v.GetString("application.server.name")
	ServerConf.Version = v.GetString("application.server.version")
	ServerConf.Addr = v.GetString("application.server.addr")
	ServerConf.Port = v.GetUint32("application.server.port")
	ServerConf.ReadTimeout = uint(v.GetInt("application.server.readTimeout"))
	ServerConf.WriteTimeout = uint(v.GetInt("application.server.writeTimeout"))
	ServerConf.MaxHeaderMB = v.GetInt("application.server.maxHeaderMB")

	// log
	if err = v.UnmarshalKey("log", &LogConf); err != nil {
		return
	}

	return
}

// 初始化mysql配置
func loadMysqlConf(configName string) (err error) {
	v := viper.New()
	v.SetConfigName(configName) // 设置文件名称
	//v.SetConfigType("toml")
	v.AddConfigPath(configPath) // 设置文件所在路径

	if err = v.ReadInConfig(); err != nil {
		return
	}

	if err = v.UnmarshalKey("mysql", &MysqlConf); err != nil {
		return
	}

	if err = v.Unmarshal(&OtherMapConf); err != nil {
		return
	}
	return
}

// 初始化redis配置
func loadRedisConf(configName string) (err error) {
	v := viper.New()
	v.SetConfigName(configName) // 设置文件名称
	//v.SetConfigType("toml")
	v.AddConfigPath(configPath) // 设置文件所在路径

	if err = v.ReadInConfig(); err != nil {
		return
	}

	if err = v.UnmarshalKey("redis", &RedisConf); err != nil {
		return
	}
	return
}

// 初始化jwt配置
func loadJwtConf(configName string) (err error) {
	v := viper.New()
	v.SetConfigName(configName) // 设置文件名称
	//v.SetConfigType("toml")
	v.AddConfigPath(configPath) // 设置文件所在路径

	if err = v.ReadInConfig(); err != nil {
		return
	}

	if err = v.UnmarshalKey("jwt", &JwtConf); err != nil {
		return
	}
	return
}

// WithMysqlConf 加载mysql配置文件
func WithMysqlConf() Option {
	return func(o option) {
		// 加载mysql配置文件
		if err := loadMysqlConf(o.configName); err != nil {
			panic(fmt.Sprintf("mysql config failed to load, err is %s", err))
		}
	}
}

// WithRedisConf 加载redis配置文件
func WithRedisConf() Option {
	return func(o option) {
		// 加载redis配置文件
		if err := loadRedisConf(o.configName); err != nil {
			panic(fmt.Sprintf("redis config failed to load, err is %s", err))
		}
	}
}

// WithJwtConf 加载redis配置文件
func WithJwtConf() Option {
	return func(o option) {
		// 加载redis配置文件
		if err := loadJwtConf(o.configName); err != nil {
			panic(fmt.Sprintf("jwt config failed to load, err is %s", err))
		}
	}
}

//监控配置和重新获取配置
//v.WatchConfig()
//
//	v.OnConfigChange(func(e fsnotify.Event) {
//		// 处理
//		fmt.Println("Config file changed:", e.Name)
//	})
