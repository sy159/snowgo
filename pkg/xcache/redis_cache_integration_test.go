//go:build integration

package xcache_test

import (
	"context"
	"snowgo/pkg/xcache"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       11, // separate DB to avoid polluting other integration tests
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// key prefixes each subtest to avoid cross-test state leakage
func testKey(prefix, name string) string {
	return prefix + ":" + name
}

func TestRedisCacheGetNonExistentKey(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	got, ok, err := cache.Get(ctx, "non-existent-key-12345")
	if err != nil {
		t.Fatalf("Get non-existent key error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for non-existent key")
	}
	if got != "" {
		t.Fatalf("expected empty string for non-existent key, got %q", got)
	}
}

func TestRedisCacheDeleteNonExistentKey(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	num, err := cache.Delete(ctx, "non-existent-key-12345")
	if err != nil {
		t.Fatalf("Delete non-existent key error: %v", err)
	}
	if num != 0 {
		t.Fatalf("expected 0 for non-existent key delete, got %d", num)
	}
}

func TestRedisCacheIncrByDecrByEdgeCases(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	key := testKey("test", "incr-decr-edge")
	_ = cache.Set(ctx, key, "not-a-number", 5*time.Minute)

	_, err := cache.IncrBy(ctx, key, 1)
	if err == nil {
		t.Fatal("expected error when incrementing non-numeric value")
	}
}

func TestRedisCacheEvalError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	_, err := cache.Eval(ctx, "invalid lua script {{{", []string{"test-key"})
	if err == nil {
		t.Fatal("expected error for invalid Lua script")
	}
}

func TestRedisCacheHGetNonExistentField(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	hashKey := testKey("test", "hget-nonexist")
	got, ok, err := cache.HGet(ctx, hashKey, "non-existent-field")
	if err != nil {
		t.Fatalf("HGet non-existent field error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for non-existent field")
	}
	if got != "" {
		t.Fatalf("expected empty string for non-existent field, got %q", got)
	}
}

func TestRedisCacheHDelNonExistentField(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	hashKey := testKey("test", "hdel-nonexist")
	delNum, err := cache.HDel(ctx, hashKey, "non-existent-field")
	if err != nil {
		t.Fatalf("HDel non-existent field error: %v", err)
	}
	if delNum != 0 {
		t.Fatalf("expected 0 for non-existent field delete, got %d", delNum)
	}
}

func TestRedisCacheZRangeError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	zKey := testKey("test", "zrange-error")
	_ = cache.Set(ctx, zKey, "not-a-zset", 5*time.Minute)

	_, err := cache.ZRange(ctx, zKey, 0, -1)
	if err == nil {
		t.Fatal("expected error when ranging over a non-zset key")
	}
}

func TestRedisCacheZCardError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	zKey := testKey("test", "zcard-error")
	_ = cache.Set(ctx, zKey, "not-a-zset", 5*time.Minute)

	_, err := cache.ZCard(ctx, zKey)
	if err == nil {
		t.Fatal("expected error when getting card of a non-zset key")
	}
}

func TestRedisCacheExistsError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	exists, err := cache.Exists(ctx, "some-key")
	if err != nil {
		t.Fatalf("Exists error: %v", err)
	}
	if exists {
		t.Fatal("key should not exist")
	}
}

func TestRedisCacheTTLError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	ttl, err := cache.TTL(ctx, "non-existent-ttl-key")
	if err != nil {
		t.Fatalf("TTL error: %v", err)
	}
	if ttl > 0 {
		t.Fatalf("expected TTL <= 0 for non-existent key, got %v", ttl)
	}
}

func TestRedisCacheHGetAllError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	hashKey := testKey("test", "hgetall-error")
	_ = cache.Set(ctx, hashKey, "not-a-hash", 5*time.Minute)

	_, err := cache.HGetAll(ctx, hashKey)
	if err == nil {
		t.Fatal("expected error when getting all fields of a non-hash key")
	}
}

func TestRedisCacheHLenError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	hashKey := testKey("test", "hlen-error")
	_ = cache.Set(ctx, hashKey, "not-a-hash", 5*time.Minute)

	_, err := cache.HLen(ctx, hashKey)
	if err == nil {
		t.Fatal("expected error when getting length of a non-hash key")
	}
}

func TestRedisCacheHIncrByError(t *testing.T) {
	client := setupTestRedis(t)
	cache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	hIncrKey := testKey("test", "hincr-error")
	_ = cache.Set(ctx, hIncrKey, "not-a-hash", 5*time.Minute)

	_, err := cache.HIncrBy(ctx, hIncrKey, "field", 1)
	if err == nil {
		t.Fatal("expected error when incrementing field of a non-hash key")
	}
}

// ============================================================
// Comprehensive RedisCache integration test with key isolation
// ============================================================

func TestRedisCache(t *testing.T) {
	client := setupTestRedis(t)

	redisCache, _ := xcache.NewRedisCache(client)
	ctx := context.Background()

	// Unique key prefixes per subtest to prevent state leakage
	evalKey := testKey("tc", "eval-incr")
	setKey := testKey("tc", "set-get")
	delKey := testKey("tc", "delete")
	incrKey := testKey("tc", "incr-decr")
	hashKey := testKey("tc", "hash")
	expireKey := testKey("tc", "expire-ttl")
	zKey := testKey("tc", "zset")
	hIncrKey := testKey("tc", "hincr")
	expKey := testKey("tc", "expire-manual")

	t.Cleanup(func() {
		// Clean up all keys used in this test
		keys := []string{evalKey, setKey, delKey, incrKey, hashKey, expireKey, zKey, hIncrKey, expKey}
		for _, k := range keys {
			_ = client.Del(ctx, k).Err()
		}
	})

	// =========================
	// 1. Eval
	// =========================
	t.Run("Eval INCR Script", func(t *testing.T) {
		script := `
		local cnt = redis.call("INCR", KEYS[1])
		return cnt
		`
		res, err := redisCache.Eval(ctx, script, []string{evalKey})
		if err != nil {
			t.Fatalf("Eval failed: %v", err)
		}

		cnt, ok := res.(int64)
		if !ok || cnt != 1 {
			t.Fatalf("Eval returned wrong value: got %v want 1", res)
		}
	})

	// =========================
	// 2. Set/Get
	// =========================
	t.Run("CacheSet and CacheGet", func(t *testing.T) {
		err := redisCache.Set(ctx, setKey, "test-value", 5*time.Minute)
		if err != nil {
			t.Fatalf("CacheSet failed: %v", err)
		}
		got, ok, err := redisCache.Get(ctx, setKey)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		if !ok {
			t.Fatal("expected ok=true for existing key")
		}
		if got != "test-value" {
			t.Fatalf("CacheGet returned wrong value: got %q want %q", got, "test-value")
		}
	})

	// =========================
	// 3. Delete
	// =========================
	t.Run("CacheDelete", func(t *testing.T) {
		// First set a value to delete
		err := redisCache.Set(ctx, delKey, "to-delete", 5*time.Minute)
		if err != nil {
			t.Fatalf("setup Set failed: %v", err)
		}

		num, err := redisCache.Delete(ctx, delKey)
		if err != nil {
			t.Fatalf("CacheDelete failed: %v", err)
		}
		if num != 1 {
			t.Fatalf("CacheDelete returned wrong value: got %v want 1", num)
		}

		got, ok, err := redisCache.Get(ctx, delKey)
		if err != nil {
			t.Fatalf("CacheGet failed: %v", err)
		}
		if ok {
			t.Fatal("expected ok=false after delete")
		}
		if got != "" {
			t.Fatal("key still exists after delete")
		}
	})

	// =========================
	// 4. IncrBy/DecrBy
	// =========================
	t.Run("CacheIncrBy and CacheDecrBy", func(t *testing.T) {
		cnt, err := redisCache.IncrBy(ctx, incrKey, 5)
		if err != nil || cnt != 5 {
			t.Fatalf("CacheIncrBy failed: err=%v cnt=%d", err, cnt)
		}
		cnt, err = redisCache.DecrBy(ctx, incrKey, 3)
		if err != nil || cnt != 2 {
			t.Fatalf("CacheDecrBy failed: err=%v cnt=%d", err, cnt)
		}
	})

	// =========================
	// 5. Hash Operations
	// =========================
	t.Run("Hash Operations", func(t *testing.T) {
		field := "test-field"
		value := "test-value"

		err := redisCache.HSet(ctx, hashKey, field, value)
		if err != nil {
			t.Fatalf("HSet failed: %v", err)
		}

		got, ok, err := redisCache.HGet(ctx, hashKey, field)
		if err != nil {
			t.Fatalf("HGet failed: %v", err)
		}
		if !ok {
			t.Fatal("expected ok=true for existing field")
		}
		if got != value {
			t.Fatalf("HGet returned wrong value: got %q want %q", got, value)
		}

		all, err := redisCache.HGetAll(ctx, hashKey)
		if err != nil {
			t.Fatalf("HGetAll failed: %v", err)
		}
		if all[field] != value {
			t.Fatalf("HGetAll returned wrong value: got %q want %q", all[field], value)
		}

		length, err := redisCache.HLen(ctx, hashKey)
		if err != nil || length != 1 {
			t.Fatalf("HLen wrong: got %d want 1", length)
		}

		delNum, err := redisCache.HDel(ctx, hashKey, field)
		if err != nil || delNum != 1 {
			t.Fatalf("HDel failed: err=%v delNum=%d", err, delNum)
		}
	})

	// =========================
	// 6. Exists / Expire / TTL
	// =========================
	t.Run("Exists Expire TTL", func(t *testing.T) {
		exists, err := redisCache.Exists(ctx, expireKey)
		if err != nil {
			t.Fatalf("Exists error: %v", err)
		}
		if exists {
			t.Fatal("key should not exist before set")
		}

		err = redisCache.Set(ctx, expireKey, "value", 2*time.Second)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		exists, err = redisCache.Exists(ctx, expireKey)
		if err != nil || !exists {
			t.Fatalf("expected key to exist after set")
		}

		ttl, err := redisCache.TTL(ctx, expireKey)
		if err != nil || ttl <= 0 {
			t.Fatalf("TTL wrong after set: err=%v ttl=%v", err, ttl)
		}

		time.Sleep(3 * time.Second)

		exists, err = redisCache.Exists(ctx, expireKey)
		if err != nil || exists {
			t.Fatal("key should have expired")
		}
	})

	// =========================
	// 7. ZSet Operations
	// =========================
	t.Run("ZSet Operations", func(t *testing.T) {
		if err := redisCache.ZAdd(ctx, zKey, 10, "a"); err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}
		if err := redisCache.ZAdd(ctx, zKey, 20, "b"); err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}
		if err := redisCache.ZAdd(ctx, zKey, 15, "c"); err != nil {
			t.Fatalf("ZAdd failed: %v", err)
		}

		card, err := redisCache.ZCard(ctx, zKey)
		if err != nil || card != 3 {
			t.Fatalf("ZCard wrong: got %d want 3", card)
		}

		members, err := redisCache.ZRange(ctx, zKey, 0, -1)
		if err != nil {
			t.Fatalf("ZRange failed: %v", err)
		}
		expectedOrder := []string{"a", "c", "b"}
		for i, m := range expectedOrder {
			if members[i] != m {
				t.Fatalf("ZRange index %d: got %q want %q", i, members[i], m)
			}
		}

		if err := redisCache.ZRem(ctx, zKey, "c"); err != nil {
			t.Fatalf("ZRem failed: %v", err)
		}

		card, err = redisCache.ZCard(ctx, zKey)
		if err != nil || card != 2 {
			t.Fatalf("ZCard after ZRem: got %d want 2", card)
		}

		members, err = redisCache.ZRange(ctx, zKey, 0, -1)
		if err != nil {
			t.Fatalf("ZRange failed: %v", err)
		}
		expectedAfterRemove := []string{"a", "b"}
		for i, m := range expectedAfterRemove {
			if members[i] != m {
				t.Fatalf("ZRange after remove index %d: got %q want %q", i, members[i], m)
			}
		}
	})

	// =========================
	// 8. HIncrBy
	// =========================
	t.Run("HIncrBy", func(t *testing.T) {
		val, err := redisCache.HIncrBy(ctx, hIncrKey, "counter", 5)
		if err != nil || val != 5 {
			t.Fatalf("HIncrBy failed: err=%v val=%d", err, val)
		}
		val, err = redisCache.HIncrBy(ctx, hIncrKey, "counter", -3)
		if err != nil || val != 2 {
			t.Fatalf("HIncrBy decrement failed: err=%v val=%d", err, val)
		}
	})

	// =========================
	// 9. Expire
	// =========================
	t.Run("Expire", func(t *testing.T) {
		_ = redisCache.Set(ctx, expKey, "value", 0)

		err := redisCache.Expire(ctx, expKey, 2*time.Second)
		if err != nil {
			t.Fatalf("Expire failed: %v", err)
		}

		ttl, err := redisCache.TTL(ctx, expKey)
		if err != nil || ttl <= 0 {
			t.Fatalf("TTL after Expire: err=%v ttl=%v", err, ttl)
		}

		time.Sleep(3 * time.Second)
		exists, err := redisCache.Exists(ctx, expKey)
		if err != nil || exists {
			t.Fatal("key should have expired after Expire")
		}
	})
}
