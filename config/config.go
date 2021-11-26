package config

import (
	"snowgo/utils/logger"

	"github.com/spf13/viper"
)

var (
	configPath = "./config"
	ServerConf ServerConfig // ServerConf 全局server配置
	RedisConf  RedisConfig  // RedisConf 全局redis配置
	MysqlConf  MysqlConfig  // MysqlConfig 全局mysql配置
	JwtConf    JwtConfig    // JwtConf 全局jwt配置
)

// ServerConfig server启动配置
type ServerConfig struct {
	IsDebug      bool   `json:"isDebug" toml:"isDebug" yaml:"isDebug"`
	AccessLog    bool   `json:"accessLog" toml:"accessLog" yaml:"accessLog"`
	Name         string `json:"name" toml:"name" yaml:"name"`
	Version      string `json:"version" toml:"version" yaml:"version"`
	Addr         string `json:"addr" toml:"addr" yaml:"addr"`
	Port         uint32 `json:"port" toml:"port" yaml:"port"`
	ReadTimeout  uint   `json:"readTimeout" toml:"readTimeout" yaml:"readTimeout" `
	WriteTimeout uint   `json:"writeTimeout" toml:"writeTimeout" yaml:"writeTimeout"`
	MaxHeaderMB  int    `json:"maxHeaderMB" toml:"maxHeaderMB" yaml:"maxHeaderMB"`
}

// RedisConfig redis连接配置
type RedisConfig struct {
	Addr         string `validate:"required" json:"addr" toml:"addr" yaml:"addr"`                      // 地址
	Password     string `json:"password" toml:"password" yaml:"password"`                              // 密码
	DB           int    `validate:"min=0"  json:"db" toml:"db" yaml:"db"`                              // 数据库
	DialTimeout  int    `validate:"gt=0"  json:"dialTimeout" toml:"dialTimeout" yaml:"dialTimeout"`    // 拨号超时(秒)
	ReadTimeout  int    `validate:"gt=0"  json:"readTimeout" toml:"readTimeout" yaml:"readTimeout"`    // 读取超时(秒)
	WriteTimeout int    `validate:"gt=0"  json:"writeTimeout" toml:"writeTimeout" yaml:"writeTimeout"` // 写入超时(秒)
	MinIdleConns int    `validate:"gt=0"  json:"minIdleConns" toml:"minIdleConns" yaml:"minIdleConns"` // 最小空闲连接数
	IdleTimeout  int    `validate:"gt=0"  json:"idleTimeout" toml:"idleTimeout" yaml:"idleTimeout"`    // 空闲超时(秒)
	PoolSize     int    `validate:"gt=0"  json:"poolSize" toml:"poolSize" yaml:"poolSize"`             // 连接池最大链接数
}

// MysqlConfig mysql连接配置
type MysqlConfig struct {
	Addr         string `json:"addr" toml:"addr" yaml:"addr"`                         // 数据库地址
	User         string `json:"user" toml:"user" yaml:"user"`                         // 用户名
	Password     string `json:"password" toml:"password" yaml:"password"`             // 用户密码
	Database     string `json:"database" toml:"database" yaml:"database"`             // 数据库名
	Charset      string `json:"charset" toml:"charset" yaml:"charset"`                // 编码方式
	ParseTime    bool   `json:"parseTime" toml:"parseTime" yaml:"parseTime"`          // 是否支持把数据库datetime和date类型转换为golang的time.Time类型
	Loc          string `json:"loc" toml:"loc" yaml:"loc"`                            // 使用时区
	TablePre     string `json:"table_pre" toml:"table_pre" yaml:"table_pre"`          // 表前缀
	MaxIdleConns int    `json:"maxIdleConns" toml:"maxIdleConns" yaml:"maxIdleConns"` // 设置闲置的连接数，默认值为2；
	MaxOpenConns int    `json:"maxOpenConns" toml:"maxOpenConns" yaml:"maxOpenConns"` // 设置最大打开的连接数，默认值为0，表示不限制。
	MaxLifeTime  int    `json:"maxLifeTime" toml:"maxLifeTime" yaml:"maxLifeTime"`    // 设置了连接可复用的最大时间。单位min
	PrintSqlLog  bool   `json:"printSqlLog" toml:"printSqlLog" yaml:"printSqlLog"`    // 是否打印SQL
	SlowSqlTime  int    `json:"slowSqlTime" toml:"slowSqlTime" yaml:"slowSqlTime"`    // 慢sql阈值 单位ms(在设置printSqlLog=true有用)
}

// JwtConfig jwt配置
type JwtConfig struct {
	Issuer                string `json:"issuer" toml:"issuer" yaml:"issuer"`                                              // 发布人
	JwtSecret             string `json:"jwtSecret" toml:"jwtSecret" yaml:"jwtSecret"`                                     // jwt加密秘钥
	AccessExpirationTime  int    `json:"accessExpirationTime" toml:"accessExpirationTime" yaml:"accessExpirationTime"`    // 访问token到期时间，单位min
	RefreshExpirationTime int    `json:"refreshExpirationTime" toml:"refreshExpirationTime" yaml:"refreshExpirationTime"` // 刷新token到期时间，单位min

}

type Option func(option)

type option struct{}

// InitConf 加载所有需要配置文件
func InitConf(options ...Option) {
	// 加载服务配置文件
	if err := loadServerConf("application"); err != nil {
		logger.Panicf("server config failed to load, err is %s", err)
	}

	// 加载需要注册的配置项目
	for _, f := range options {
		f(option{})
	}

}

// 初始化服务配置
func loadServerConf(configName string) (err error) {
	v := viper.New()
	v.SetConfigName(configName) // 设置文件名称
	//v.SetConfigType("toml")
	v.AddConfigPath(configPath) // 设置文件所在路径

	if err = v.ReadInConfig(); err != nil {
		return
	}

	// 绑定配置文件
	isDebug := v.GetBool("isDebug")

	ServerConf.IsDebug = isDebug
	ServerConf.AccessLog = v.GetBool("accessLog")
	// 判断是正式环境还是测试环境,根据不同环境获取配置
	subKey := "debug-server"
	if !isDebug {
		subKey = "production-server"
	}
	serverSub := v.Sub(subKey)

	if err = serverSub.Unmarshal(&ServerConf); err != nil {
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

	// 判断是正式环境还是测试环境,根据不同环境获取配置
	subKey := "debug-mysql"
	if !ServerConf.IsDebug {
		subKey = "production-mysql"
	}
	// 读取的是基础数据库配置
	serverSub := v.Sub(subKey)

	if err = serverSub.Unmarshal(&MysqlConf); err != nil {
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

	// 判断是正式环境还是测试环境,根据不同环境获取配置
	subKey := "debug-redis"
	if !ServerConf.IsDebug {
		subKey = "production-redis"
	}
	serverSub := v.Sub(subKey)

	if err = serverSub.Unmarshal(&RedisConf); err != nil {
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

	// 判断是正式环境还是测试环境,根据不同环境获取配置
	subKey := "debug-jwt"
	if !ServerConf.IsDebug {
		subKey = "production-jwt"
	}
	serverSub := v.Sub(subKey)

	if err = serverSub.Unmarshal(&JwtConf); err != nil {
		return
	}
	return
}

// WithMysqlConf 加载mysql配置文件
func WithMysqlConf(configName string) Option {
	return func(o option) {
		// 加载mysql配置文件
		if err := loadMysqlConf(configName); err != nil {
			logger.Panicf("mysql config failed to load, err is %s", err)
		}
	}
}

// WithRedisConf 加载redis配置文件
func WithRedisConf(configName string) Option {
	return func(o option) {
		// 加载redis配置文件
		if err := loadRedisConf(configName); err != nil {
			logger.Panicf("redis config failed to load, err is %s", err)
		}
	}
}

// WithJwtConf 加载redis配置文件
func WithJwtConf(configName string) Option {
	return func(o option) {
		// 加载redis配置文件
		if err := loadJwtConf(configName); err != nil {
			logger.Panicf("jwt config failed to load, err is %s", err)
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
