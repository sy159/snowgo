package xcache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"time"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) Cache {
	return &RedisCache{client: client}
}

// Get 如果没有值会直接返回空
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", errors.WithStack(err)
	}
	return result, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) (int64, error) {
	result, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return result, nil
}

// IncrBy key不存在默认为0开始incr
func (r *RedisCache) IncrBy(ctx context.Context, key string, increment int64) (int64, error) {
	result, err := r.client.IncrBy(ctx, key, increment).Result()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return result, nil
}

// DecrBy key不存在默认为0开始decr
func (r *RedisCache) DecrBy(ctx context.Context, key string, decrement int64) (int64, error) {
	result, err := r.client.DecrBy(ctx, key, decrement).Result()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return result, nil
}

func (r *RedisCache) HSet(ctx context.Context, key string, field string, value string) error {
	err := r.client.HSet(ctx, key, field, value).Err()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *RedisCache) HGet(ctx context.Context, key string, field string) (string, error) {
	result, err := r.client.HGet(ctx, key, field).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", errors.WithStack(err)
	}
	return result, nil
}

func (r *RedisCache) HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, error) {
	result, err := r.client.HIncrBy(ctx, key, field, increment).Result()
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return result, nil
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.WithStack(err)
	}
	return exists > 0, err
}

func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	err := r.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
