package xcache_test

import (
	"context"
	"fmt"
	"snowgo/pkg/xcache"
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
	_ = client.Close()
}

func TestRedisCache(t *testing.T) {
	client := setupTestRedis()
	defer teardownTestRedis(client)

	redisCache := xcache.NewRedisCache(client)
	ctx := context.Background()
	key := "test-key"
	value := "test-value"
	hashKey := "test-hash"
	field := "test-field"
	zKey := "test-zset"

	// =========================
	// 1. Eval 测试
	// =========================
	t.Run("Eval INCR Script", func(t *testing.T) {
		_, _ = redisCache.Delete(ctx, key)

		script := `
local cnt = redis.call("INCR", KEYS[1])
return cnt
`
		res, err := redisCache.Eval(ctx, script, []string{key})
		if err != nil {
			t.Fatalf("Eval failed: %v", err)
		}
		fmt.Println("Eval INCR:", res)

		cnt, ok := res.(int64)
		if !ok || cnt != 1 {
			t.Fatalf("Eval returned wrong value: got %v want %v", res, 1)
		}
	})

	// =========================
	// 2. Set/Get
	// =========================
	t.Run("CacheSet and CacheGet", func(t *testing.T) {
		err := redisCache.Set(ctx, key, value, 5*time.Minute)
		if err != nil {
			t.Fatalf("CacheSet failed: %v", err)
		}
		got, err := redisCache.Get(ctx, key)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		if got != value {
			t.Fatalf("CacheGet returned wrong value: got %v want %v", got, value)
		}
	})

	// =========================
	// 3. Delete
	// =========================
	t.Run("CacheDelete", func(t *testing.T) {
		num, err := redisCache.Delete(ctx, key)
		if err != nil {
			t.Fatalf("CacheDelete failed: %v", err)
		}
		if num != 1 {
			t.Fatalf("CacheDelete returned wrong value: got %v want %v", num, 1)
		}

		got, err := redisCache.Get(ctx, key)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		if got != "" {
			t.Fatalf("CacheDelete failed, key still exists")
		}
	})

	// =========================
	// 4. IncrBy/DecrBy
	// =========================
	t.Run("CacheIncrBy and CacheDecrBy", func(t *testing.T) {
		incrKey := "test-incr-key"
		_, _ = redisCache.Delete(ctx, incrKey)

		cnt, err := redisCache.IncrBy(ctx, incrKey, 5)
		if err != nil || cnt != 5 {
			t.Fatalf("CacheIncrBy failed: %v", err)
		}
		cnt, err = redisCache.DecrBy(ctx, incrKey, 3)
		if err != nil || cnt != 2 {
			t.Fatalf("CacheDecrBy failed: %v", err)
		}
	})

	// =========================
	// 5. HSet/HGet/HGetAll/HDel/HLen
	// =========================
	t.Run("Hash Operations", func(t *testing.T) {
		_, _ = redisCache.HDel(ctx, hashKey, field)

		err := redisCache.HSet(ctx, hashKey, field, value)
		if err != nil {
			t.Fatalf("HSet failed: %v", err)
		}

		got, err := redisCache.HGet(ctx, hashKey, field)
		if err != nil {
			t.Fatalf("HGet failed: %v", err)
		}
		if got != value {
			t.Fatalf("HGet returned wrong value: got %v want %v", got, value)
		}

		all, err := redisCache.HGetAll(ctx, hashKey)
		if err != nil {
			t.Fatalf("HGetAll failed: %v", err)
		}
		if all[field] != value {
			t.Fatalf("HGetAll returned wrong value: got %v want %v", all[field], value)
		}

		length, err := redisCache.HLen(ctx, hashKey)
		if err != nil || length != 1 {
			t.Fatalf("HLen returned wrong value: got %v want %v", length, 1)
		}

		delNum, err := redisCache.HDel(ctx, hashKey, field)
		if err != nil || delNum != 1 {
			t.Fatalf("HDel failed: %v", err)
		}
	})

	// =========================
	// 6. Exists / Expire / TTL
	// =========================
	t.Run("Exists Expire TTL", func(t *testing.T) {
		_, _ = redisCache.Delete(ctx, key)
		exists, _ := redisCache.Exists(ctx, key)
		if exists {
			t.Fatalf("Exists before set should be false")
		}

		err := redisCache.Set(ctx, key, value, 2*time.Second)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		exists, _ = redisCache.Exists(ctx, key)
		if !exists {
			t.Fatalf("Exists after set should be true")
		}

		ttl, err := redisCache.TTL(ctx, key)
		if err != nil || ttl <= 0 {
			t.Fatalf("TTL failed: %v", err)
		}

		time.Sleep(3 * time.Second)

		exists, _ = redisCache.Exists(ctx, key)
		if exists {
			t.Fatalf("Key should expire")
		}
	})

	// =========================
	// 7. ZSet Operations
	// =========================
	t.Run("ZSet Operations", func(t *testing.T) {
		_, _ = redisCache.Delete(ctx, key)

		// ZAdd
		if err := redisCache.ZAdd(ctx, zKey, 10, "a"); err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}
		if err := redisCache.ZAdd(ctx, zKey, 20, "b"); err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}
		if err := redisCache.ZAdd(ctx, zKey, 15, "c"); err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}

		// ZCard
		card, err := redisCache.ZCard(ctx, zKey)
		if err != nil || card != 3 {
			t.Fatalf("ZCard returned wrong value: got %v want %v", card, 3)
		}

		// ZRange
		members, err := redisCache.ZRange(ctx, zKey, 0, -1)
		if err != nil {
			t.Fatalf("ZRange failed: %v", err)
		}
		expectedOrder := []string{"a", "c", "b"} // 按 score 升序
		for i, m := range expectedOrder {
			if members[i] != m {
				t.Fatalf("ZRange order incorrect at index %d: got %v want %v", i, members[i], m)
			}
		}

		// ZRem
		if err := redisCache.ZRem(ctx, zKey, "c"); err != nil {
			t.Fatalf("ZRem failed: %v", err)
		}

		// ZCard after remove
		card, err = redisCache.ZCard(ctx, zKey)
		if err != nil || card != 2 {
			t.Fatalf("ZCard after ZRem wrong: got %v want %v", card, 2)
		}

		// ZRange after remove
		members, err = redisCache.ZRange(ctx, zKey, 0, -1)
		if err != nil {
			t.Fatalf("ZRange failed: %v", err)
		}
		expectedAfterRemove := []string{"a", "b"}
		for i, m := range expectedAfterRemove {
			if members[i] != m {
				t.Fatalf("ZRange order incorrect after remove at index %d: got %v want %v", i, members[i], m)
			}
		}
	})

}
