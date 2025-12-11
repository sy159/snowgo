package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"snowgo/config"
)

// NewRedis 创建一个新的 redis 实例（不影响全局 RDB）
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	dialTimeout := cfg.DialTimeout
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  dialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		IdleTimeout:  cfg.IdleTimeout,
	})

	// 使用超时上下文验证连接
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, errors.WithStack(err)
	}
	return rdb, nil
}
