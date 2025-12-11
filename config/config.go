package config

import (
	"fmt"
	"sync/atomic"
	"time"

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
	EnableAccessLog bool         `mapstructure:"enable_access_log"`
	EnablePprof     bool         `mapstructure:"enable_pprof"`
	EnableTrace     bool         `mapstructure:"enable_trace"`
	TempoEndpoint   string       `mapstructure:"tempo_endpoint"`
	Server          ServerConfig `mapstructure:"server"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	Name         string        `mapstructure:"name"`
	Version      string        `mapstructure:"version"`
	Addr         string        `mapstructure:"addr"`
	Port         uint32        `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	MaxHeaderMB  int           `mapstructure:"max_header_mb"`
}

// LogConfig 日志配置
type LogConfig struct {
	Output               string `mapstructure:"output"`
	AccessEncoder        string `mapstructure:"access_encoder"`
	LogEncoder           string `mapstructure:"log_encoder"`
	AccessFileMaxAgeDays uint32 `mapstructure:"access_file_max_age_days"`
	LogFileMaxAgeDays    uint32 `mapstructure:"log_file_max_age_days"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr         string        `mapstructure:"addr"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	PoolSize     int           `mapstructure:"pool_size"`
}

// MysqlConfig MySQL配置
type MysqlConfig struct {
	EnableReadWriteSeparation bool          `mapstructure:"enable_read_write_separation"`
	DSN                       string        `mapstructure:"dsn"`
	TablePrefix               string        `mapstructure:"table_prefix"`
	MaxIdleConns              int           `mapstructure:"max_idle_conns"`
	MaxOpenConns              int           `mapstructure:"max_open_conns"`
	ConnMaxIdleTime           time.Duration `mapstructure:"conn_max_idle_time"`
	ConnMaxLifeTime           time.Duration `mapstructure:"conn_max_life_time"`
	EnableSqlLog              bool          `mapstructure:"enable_sql_log"`
	SlowSqlThresholdTime      time.Duration `mapstructure:"slow_sql_threshold_time"`
	MainsDSN                  []string      `mapstructure:"mains_dsn"`
	SlavesDSN                 []string      `mapstructure:"slaves_dsn"`
}

// JwtConfig JWT配置
type JwtConfig struct {
	Issuer                string        `mapstructure:"issuer"`
	JwtSecret             string        `mapstructure:"jwt_secret"`
	AccessExpirationTime  time.Duration `mapstructure:"access_expiration_time"`
	RefreshExpirationTime time.Duration `mapstructure:"refresh_expiration_time"`
}

// RabbitMQProducerConfig rabbitmq配置
type RabbitMQProducerConfig struct {
	URL                            string        `mapstructure:"url"`
	ChannelPoolSize                int           `mapstructure:"channel_pool_size"`
	ChannelAcquireTimeout          time.Duration `mapstructure:"channel_acquire_timeout"`
	ChannelConfirmTimeoutThreshold int           `mapstructure:"channel_confirm_timeout_threshold"`
	MessageConfirmTimeout          time.Duration `mapstructure:"message_confirm_timeout"`
	ReconnectInitialDelayTime      time.Duration `mapstructure:"reconnect_initial_delay_time"`
	ReconnectMaxDelayTime          time.Duration `mapstructure:"reconnect_max_delay_time"`
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
