package xcache

import (
	"context"
	"time"
)

type Cache interface {
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

	// HIncrBy increments the value of a field in a hash by the given amount.
	//HIncrBy(ctx context.Context, key string, field string, increment int64) (int64, xerror)

	// Exists checks if a key exists.
	Exists(ctx context.Context, key string) (bool, error)

	// Expire sets an expiration time for a key.
	Expire(ctx context.Context, key string, expiration time.Duration) error
}
