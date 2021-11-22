package redis

import (
	"context"
	"fmt"
	"snowgo/config"
	"snowgo/utils/logger"
	"time"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client

// InitRedis 初始化redis连接,设置全局Redis
func InitRedis() {
	dialTimeout := time.Duration(config.RedisConf.DialTimeout) * time.Second
	RDB = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.RedisConf.Addr, config.RedisConf.Port),
		Password:     config.RedisConf.Password,
		DB:           config.RedisConf.DB,
		DialTimeout:  dialTimeout,
		ReadTimeout:  time.Duration(config.RedisConf.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.RedisConf.WriteTimeout) * time.Second,
		PoolSize:     config.RedisConf.PoolSize,
		MinIdleConns: config.RedisConf.MinIdleConns,
		IdleTimeout:  time.Duration(config.RedisConf.IdleTimeout) * time.Second,
	})
	// 使用超时上下文，验证redis
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	_, err := RDB.Ping(ctx).Result()
	if err != nil {
		logger.Panicf("redis init failed, err is %s", err.Error())
	}
}
