package xcache

import (
	"context"
	"time"
)

type Cache interface {
	// Eval 直接执行 script：script KEYS ARGV...
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)

	// Get retrieves the value for a given key.
	Get(ctx context.Context, key string) (string, error)

	// Set sets a key-value pair with an expiration time.
	Set(ctx context.Context, key string, value string, expiration time.Duration) error

	// Delete deletes one or more keys.
	Delete(ctx context.Context, keys ...string) (int64, error)

	// IncrBy increments the value of a key by the given amount.
	IncrBy(ctx context.Context, key string, increment int64) (int64, error)

	// DecrBy decrements the value of a key by the given amount.
	DecrBy(ctx context.Context, key string, decrement int64) (int64, error)

	// HSet sets the value of a field in a hash.
	HSet(ctx context.Context, key string, field string, value string) error

	// HGet retrieves the value of a field in a hash.
	HGet(ctx context.Context, key string, field string) (string, error)

	// HGetAll retrieves all fields and values in a hash.
	HGetAll(ctx context.Context, key string) (map[string]string, error)

	// HDel deletes one or more fields from a hash.
	HDel(ctx context.Context, key string, fields ...string) (int64, error)

	// HLen returns the number of fields contained in a hash.
	HLen(ctx context.Context, key string) (int64, error)

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// Expire sets an expiration time for a key.
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// TTL returns the remaining time-to-live of a key.
	// Possible return values:
	//   - TTL == -2: the key does not exist.
	//   - TTL == -1: the key exists but has no expiration.
	//   - TTL >= 0: the key's remaining time-to-live.
	TTL(ctx context.Context, key string) (time.Duration, error)
}
