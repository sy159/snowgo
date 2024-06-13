package cache_test

import (
	"context"
	"fmt"
	"snowgo/utils/cache"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func setupTestRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	return client
}

func teardownTestRedis(client *redis.Client) {
	//client.FlushDB(context.Background())
	client.Close()
}

func TestRedisCache(t *testing.T) {
	client := setupTestRedis()
	defer teardownTestRedis(client)

	redisCache := cache.NewRedisCache(client)

	ctx := context.Background()
	key := "test-key"
	value := "test-value"
	field := "test-field"
	hashKey := "test-hash"

	t.Run("CacheSet and CacheGet", func(t *testing.T) {
		got, err := redisCache.Get(ctx, key)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		fmt.Println(got, err)
		err = redisCache.Set(ctx, key, value, 5*time.Minute)
		fmt.Println(err)
		if err != nil {
			t.Fatalf("CacheSet failed: %v", err)
		}
		got, err = redisCache.Get(ctx, key)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		fmt.Println(got, err)
	})

	t.Run("CacheDelete", func(t *testing.T) {
		got, err := redisCache.Get(ctx, key)
		fmt.Println(got, err)
		num, err := redisCache.Delete(ctx, key, key)
		if err != nil {
			t.Fatalf("CacheDelete failed: %v", err)
		}
		fmt.Println(num, err)

		got, err = redisCache.Get(ctx, key)
		fmt.Println(got, err)
	})

	t.Run("CacheIncrBy and CacheDecrBy", func(t *testing.T) {
		incrKey := "test-incr-key"
		_, _ = redisCache.Delete(ctx, incrKey)
		_, err := redisCache.IncrBy(ctx, incrKey, 5)
		if err != nil {
			t.Fatalf("CacheIncrBy failed: %v", err)
		}

		value, err := redisCache.Get(ctx, incrKey)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		if value != "5" {
			t.Fatalf("CacheIncrBy returned wrong value: got %v want %v", value, "5")
		}

		_, err = redisCache.DecrBy(ctx, incrKey, 3)
		if err != nil {
			t.Fatalf("CacheDecrBy failed: %v", err)
		}

		value, err = redisCache.Get(ctx, incrKey)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		if value != "2" {
			t.Fatalf("CacheDecrBy returned wrong value: got %v want %v", value, "2")
		}
	})

	t.Run("CacheHSet and CacheHGet", func(t *testing.T) {
		got, err := redisCache.HGet(ctx, hashKey, field)
		if err != nil {
			t.Fatalf("CacheHGet failed: %v", err)
		}
		fmt.Println(got, err)
		err = redisCache.HSet(ctx, hashKey, field, value)
		if err != nil {
			t.Fatalf("CacheHSet failed: %v", err)
		}

		got, err = redisCache.HGet(ctx, hashKey, field)
		fmt.Println(got, err)
		if err != nil {
			t.Fatalf("CacheHGet failed: %v", err)
		}
		if got != value {
			t.Fatalf("CacheHGet returned wrong value: got %v want %v", got, value)
		}
	})

	t.Run("CacheExists", func(t *testing.T) {
		exists, err := redisCache.Exists(ctx, key)
		fmt.Println(exists, err)
		if err != nil {
			t.Fatalf("CacheExists failed: %v", err)
		}
		if exists {
			t.Fatalf("CacheExists returned wrong value: got %v want %v", exists, false)
		}

		err = redisCache.Set(ctx, key, value, time.Minute)
		if err != nil {
			t.Fatalf("CacheSet failed: %v", err)
		}

		exists, err = redisCache.Exists(ctx, key)
		fmt.Println(exists, err)
		if err != nil {
			t.Fatalf("CacheExists failed: %v", err)
		}
		if !exists {
			t.Fatalf("CacheExists returned wrong value: got %v want %v", exists, true)
		}
	})

	t.Run("CacheExpire", func(t *testing.T) {
		err := redisCache.Expire(ctx, key, time.Second*2)
		if err != nil {
			t.Fatalf("CacheExpire failed: %v", err)
		}

		time.Sleep(time.Second * 3)
		exists, err := redisCache.Exists(ctx, key)
		if err != nil {
			t.Fatalf("CacheExists failed: %v", err)
		}
		if exists {
			t.Fatalf("CacheExists returned wrong value: got %v want %v", exists, false)
		}
	})
}
