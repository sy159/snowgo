package xlock

import (
	"testing"
)

func TestNewRedisLock_NilClient(t *testing.T) {
	_, err := NewRedisLock(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil redis client")
	}
}
