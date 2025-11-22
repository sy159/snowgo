package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand"
	"reflect"
	"sync"
	"time"
)

var (
	weakOnce sync.Once
	weakRng  *mrand.Rand
)

func initWeakRng() {
	// nosec G404非安全场景，仅用于生成混淆/测试数据
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

// ErrorToString 错误转为字符串
func ErrorToString(err interface{}) string {
	switch v := err.(type) {
	case error:
		return v.Error()
	default:
		return err.(string)
	}
}

// StructToMap 结构体转为Map[string]interface{}, 直接通过序列化，反序列化为map会存在数字类型（整型、浮点型等）都会序列化成float64类型。
func StructToMap(in interface{}, tagName string) (map[string]interface{}, error) {
	out := make(map[string]interface{})

	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct { // 非结构体返回错误提示
		return nil, fmt.Errorf("Struct to map only accepts struct or struct pointer; got %T\n", v)
	}

	t := v.Type()
	// 指定tagName值为map中key;字段值为map中value
	for i := 0; i < v.NumField(); i++ {
		fi := t.Field(i)
		if tagValue := fi.Tag.Get(tagName); tagValue != "" {
			out[tagValue] = v.Field(i).Interface()
		}
	}
	return out, nil
}
