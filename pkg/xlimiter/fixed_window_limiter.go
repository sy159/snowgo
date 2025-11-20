package xlimiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"snowgo/pkg/xcache"
)

type FixedWindowLimiter struct {
	cache     xcache.Cache
	windowSec int64  // 窗口长度，秒
	maxFails  int64  // 窗口内最大允许次数（当达到或超过即视为超限）
	key       string // 完整的 redis key（例如 "login:fail:alice"）
}

// NewFixedWindowLimiter 固定窗口限流，适用于登录失败计数、短信请求次数限制、接口短期频率限制等
func NewFixedWindowLimiter(cache xcache.Cache, key string, windowSecond int64, maxFails int64) *FixedWindowLimiter {
	if windowSecond <= 0 {
		windowSecond = 60 // 默认 60s，避免 0 导致问题
	}
	if maxFails <= 0 {
		maxFails = 1
	}
	return &FixedWindowLimiter{
		cache:     cache,
		windowSec: windowSecond,
		maxFails:  maxFails,
		key:       key,
	}
}

// Add 尝试增加一次计数（原子）
// 返回：allowed 是否允许继续操作，count 当前计数（在做 incr 后的值），ttl 剩余窗口时间（time.Duration），err 错误
func (f *FixedWindowLimiter) Add(ctx context.Context) (allowed bool, count int64, ttl time.Duration, err error) {
	// Lua 脚本：INCR -> 如果首次 INCR 设置 EXPIRE -> 获取 TTL -> 根据 count 与 maxFails 决定返回值
	script := `
local key = KEYS[1]
local maxFails = tonumber(ARGV[1])
local window = tonumber(ARGV[2])

local cnt = redis.call("INCR", key)
local t = redis.call("TTL", key)
if cnt == 1 or t < 0 then
    redis.call("EXPIRE", key, window)
    t = window
end

-- 超限判断：cnt >= maxFails 返回 blocked
local allowed = 1
if cnt >= maxFails then
    allowed = 0
end

return {allowed, cnt, t}
`

	res, err := f.cache.Eval(ctx, script, []string{f.key}, strconv.FormatInt(f.maxFails, 10), strconv.FormatInt(f.windowSec, 10))
	if err != nil {
		return false, 0, 0, err
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 3 {
		return false, 0, 0, fmt.Errorf("unexpected redis eval result type: %T", res)
	}

	allowedFlag, err := parseRedisInt(arr[0])
	if err != nil {
		return false, 0, 0, err
	}
	cnt, err := parseRedisInt(arr[1])
	if err != nil {
		return false, 0, 0, err
	}
	ttlSec, err := parseRedisInt(arr[2])
	if err != nil {
		return false, 0, 0, err
	}

	// 规范化 TTL：有可能为 -2/-1，统一返回 0 而不是负值
	if ttlSec < 0 {
		ttlSec = 0
	}

	allowed = allowedFlag == 1
	count = cnt
	ttl = time.Duration(ttlSec) * time.Second

	return allowed, count, ttl, nil
}

// Reset 删除计数，重置
func (f *FixedWindowLimiter) Reset(ctx context.Context) error {
	_, err := f.cache.Delete(ctx, f.key)
	return err
}

// parseRedisInt 从 Redis 返回的 interface{} 解析为 int64（兼容 int64 / string / []byte）
func parseRedisInt(v interface{}) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	case []byte:
		return strconv.ParseInt(string(t), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}
