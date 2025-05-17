package redis

import (
	"context"
	"snowgo/config"
	"snowgo/pkg/xlogger"
	"time"

	"github.com/go-redis/redis/v8"
)

var RDB *redis.Client

// InitRedis 初始化redis连接,设置全局Redis
func InitRedis() {
	cfg := config.Get()
	if cfg.Redis == (config.RedisConfig{}) {
		xlogger.Panic("Please initialize redis configuration first")
	}
	RDB = NewRedis(cfg.Redis)
}

// NewRedis 创建一个新的 redis 实例（不影响全局 RDB）
func NewRedis(cfg config.RedisConfig) *redis.Client {
	dialTimeout := time.Duration(cfg.DialTimeout) * time.Second
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  dialTimeout,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
	})

	// 使用超时上下文验证连接
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		xlogger.Panicf("NewRedis instance init failed, err: %s", err.Error())
	}
	return rdb
}

// CloseRedis 关闭redis连接
func CloseRedis(rdb *redis.Client) {
	_ = rdb.Close()
}
