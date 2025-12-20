package common

import (
	"crypto/rand"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"k8s.io/utils/env"
	"math/big"
	mrand "math/rand"
	"reflect"
	"strconv"
	"sync"
	"time"
)

var (
	weakOnce sync.Once
	weakRng  *mrand.Rand
	sfNode   *snowflake.Node
	fbMu     sync.Mutex
	fbLast   int64
	fbSeq    uint64
)

func init() {
	// 单机固定 NodeID=1（如需多节点，可从 env 配置）
	nodeID, _ := env.GetInt("SNOWFLAKE_NODE", 1)
	n, err := snowflake.NewNode(int64(nodeID))
	if err != nil {
		panic("Failed to initialize snowflake node: " + err.Error())
	}
	sfNode = n
}

// 初始化高性能 RNG（非安全）
func initWeakRng() {
	// #nosec G404 -- 非安全随机，仅用于生成测试数据/混淆用途，不用于安全用途
	weakRng = mrand.New(mrand.NewSource(time.Now().UnixNano()))
}

// WeakRandInt63n 返回高性能随机数（非安全）
func WeakRandInt63n(max int64) int64 {
	if max <= 0 {
		return 0
	}
	weakOnce.Do(initWeakRng)
	return weakRng.Int63n(max)
}

// SecureRandInt63n 返回 [0, max) 的安全随机整数
func SecureRandInt63n(max int64) (int64, error) {
	if max <= 0 {
		return 0, nil
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return nBig.Int64(), nil
}

// GenerateID 生成全局唯一消息 ID（数字字符串）
func GenerateID() string {
	if sfNode != nil {
		return sfNode.Generate().String()
	}
	now := time.Now().UnixMilli()
	fbMu.Lock()
	if now == fbLast {
		fbSeq++
	} else {
		fbLast = now
		fbSeq = 0
	}
	seq := fbSeq & ((1 << 22) - 1)
	fbMu.Unlock()

	// #nosec G115 -- Unix millisecond timestamp is always non-negative
	id := (uint64(now) << 22) | seq
	return strconv.FormatUint(id, 10)
}

// ErrorToString 错误转字符串（可安全处理 panic）
func ErrorToString(err interface{}) string {
	switch v := err.(type) {
	case nil:
		return ""
	case error:
		return v.Error()
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// StructToMap 结构体转 map[string]interface{}，仅提取 tagName 的标签字段  示例：tagName="json" → 取 json:"xxx" 的字段
func StructToMap(in interface{}, tagName string) (map[string]interface{}, error) {
	out := make(map[string]interface{})

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("StructToMap only accepts struct or struct pointer; got %T", v)
	}

	t := v.Type()
	// 指定tagName值为map中key;字段值为map中value
	for i := 0; i < v.NumField(); i++ {
		fi := t.Field(i)
		tagValue := fi.Tag.Get(tagName)
		if tagValue == "" {
			continue
		}
		out[tagValue] = v.Field(i).Interface()
	}
	return out, nil
}
