package config

import (
	"snowgo/utils/logger"

	"github.com/spf13/viper"
)

var (
	ServerConf ServerConfig // ServerConf ServerConfig实例
)

// ServerConfig server配置
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

type RedisConf struct {
	Addr     string `json:"addr" toml:"addr" yaml:"addr" xml:"addr"`
	Port     uint32 `json:"port" toml:"port" yaml:"port" xml:"port"`
	Password string `json:"password" toml:"password" yaml:"password" xml:"password"`
}

// InitConf 加载所有需要配置文件
func InitConf() {
	// 加载服务配置文件
	if err := LoadServerConf(); err != nil {
		logger.Panicf("server config failed to load, err is %s", err)
	}

}

// LoadServerConf 初始化服务配置
func LoadServerConf() (err error) {
	v := viper.New()
	v.SetConfigName("app") // 设置文件名称
	//v.SetConfigType("toml")
	v.AddConfigPath("./config") // 设置文件所在路径

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

// 监控配置和重新获取配置
//	v.WatchConfig()
//
//	v.OnConfigChange(func(e fsnotify.Event) {
//		// 处理
//		fmt.Println("Config file changed:", e.Name)
//	})
