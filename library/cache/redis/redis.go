package redisConf

// RedisConfig redis配置
type RedisConfig struct {
	Addr         string `validate:"required" json:"addr" toml:"addr" yaml:"addr"`                      // 地址
	Port         uint32 `validate:"required" json:"port" toml:"port" yaml:"port"`                      // 端口号
	Password     string `json:"password" toml:"password" yaml:"password"`                              // 密码
	DB           int    `validate:"min=0"  json:"db" toml:"db" yaml:"db"`                              // 数据库
	DialTimeout  int    `validate:"gt=0"  json:"dialTimeout" toml:"dialTimeout" yaml:"dialTimeout"`    // 拨号超时(秒)
	ReadTimeout  int    `validate:"gt=0"  json:"readTimeout" toml:"readTimeout" yaml:"readTimeout"`    // 读取超时(秒)
	WriteTimeout int    `validate:"gt=0"  json:"writeTimeout" toml:"writeTimeout" yaml:"writeTimeout"` // 写入超时(秒)
	MinIdleConns int    `validate:"gt=0"  json:"minIdleConns" toml:"minIdleConns" yaml:"minIdleConns"` // 最小空闲连接数
	IdleTimeout  int    `validate:"gt=0"  json:"idleTimeout" toml:"idleTimeout" yaml:"idleTimeout"`    // 空闲超时(秒)
	PoolSize     int    `validate:"gt=0"  json:"poolSize" toml:"poolSize" yaml:"poolSize"`             // 连接池最大链接数
}
