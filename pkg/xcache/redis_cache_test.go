package xcache_test

import (
	"testing"

	"snowgo/pkg/xcache"
)

func TestNewRedisCache_NilClient(t *testing.T) {
	_, err := xcache.NewRedisCache(nil)
	if err == nil {
		t.Fatal("expected error for nil redis client")
	}
}
