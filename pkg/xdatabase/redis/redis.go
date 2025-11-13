package redis

import (
	"context"
	"github.com/pkg/errors"
	"snowgo/config"
	"time"

	"github.com/go-redis/redis/v8"
)

// NewRedis 创建一个新的 redis 实例（不影响全局 RDB）
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
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
		return nil, errors.WithStack(err)
	}
	return rdb, nil
}
