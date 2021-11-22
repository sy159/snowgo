package config

import (
	redisConf "snowgo/library/cache/redis"
	serverConf "snowgo/library/server"
	"snowgo/utils/logger"

	"github.com/spf13/viper"
)

var (
	ServerConf serverConf.ServerConfig // ServerConf 全局server配置
	RedisConf  redisConf.RedisConfig   // RedisConf 全局redis配置
)

// InitConf 加载所有需要配置文件
func InitConf() {
	// 加载服务配置文件
	if err := loadServerConf(); err != nil {
		logger.Panicf("server config failed to load, err is %s", err)
	}

	// 加载redis配置文件
	if err := loadRedisConf(); err != nil {
		logger.Panicf("redis config failed to load, err is %s", err)
	}

}

// 初始化服务配置
func loadServerConf() (err error) {
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

// 初始化redis配置
func loadRedisConf() (err error) {
	v := viper.New()
	v.SetConfigName("redis") // 设置文件名称
	//v.SetConfigType("toml")
	v.AddConfigPath("./config") // 设置文件所在路径

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

//监控配置和重新获取配置
//v.WatchConfig()
//
//	v.OnConfigChange(func(e fsnotify.Event) {
//		// 处理
//		fmt.Println("Config file changed:", e.Name)
//	})
