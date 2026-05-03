package xcryption

import (
	"errors"
	"fmt"
	"math"
	common "snowgo/pkg"
	str "snowgo/pkg/xstr_tool"
	"strings"
)

const (
	chars   = "5dfucyar1j2txwgs8mvp4qhb3n6k7ez9" // 不重复的加密字符串，可用utils.RandShuffleStr生成
	charLen = int64(len(chars))
	divider = "i" // 分割标识符
)

// Id2Code id转code，可用于邀请码，短链接等生成
func Id2Code(id int64, minLength int) (string, error) {
	if id < 0 {
		return "", errors.New("id 不能为负数")
	}
	var sb strings.Builder
	for id/charLen > 0 {
		sb.WriteByte(chars[id%charLen])
		id /= charLen
	}
	sb.WriteByte(chars[id%charLen])
	code := str.ReverseStr(sb.String())
	fixLen := minLength - len(code)
	if fixLen > 0 {
		sb.Reset()
		sb.WriteString(code)
		sb.WriteByte(divider[0])
		for i := 0; i < fixLen-1; i++ {
			sb.WriteByte(chars[common.WeakRandInt63n(int64(len(chars)))])
		}
		code = sb.String()
	}
	return code, nil
}

// Code2Id code转id，用户解码
func Code2Id(code string) (id int64, err error) {
	if code == "" {
		return 0, errors.New("code is empty")
	}
	for i := 0; i < len(code); i++ {
		if code[i] == divider[0] {
			break
		}
		charIdx := strings.IndexByte(chars, code[i])
		if charIdx == -1 {
			return 0, errors.New("code decode failed")
		}
		charIndex := int64(charIdx)
		if i > 0 {
			// 溢出检查：id*charLen + charIndex 不能超过 int64 最大值
			if id > (math.MaxInt64-charIndex)/charLen {
				return 0, fmt.Errorf("code overflow at position %d", i)
			}
			id = id*charLen + charIndex
		} else {
			id = charIndex
		}
	}
	return id, nil
}
