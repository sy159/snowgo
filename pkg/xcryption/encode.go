package xcryption

import (
	"errors"
	"math/rand"
	str "snowgo/pkg/xstr_tool"
	"strings"
	"time"
)

const (
	chars   = "5dfucyar1j2txwgs8mvp4qhb3n6k7ez9" // 不重复的加密字符串，可用utils.RandShuffleStr生成
	charLen = uint(len(chars))
	divider = "i" // 分割标识符
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) // nolint:gosec

// Id2Code id转code，可用于邀请码，短链接等生成
func Id2Code(id uint, minLength int) (code string) {
	for id/charLen > 0 {
		code += string(chars[id%charLen])
		id /= charLen
	}
	code += string(chars[id%charLen])
	code = str.ReverseStr(code)
	fixLen := minLength - len(code)
	if fixLen > 0 {
		code += divider
		for i := 0; i < fixLen-1; i++ {
			//code += string(chars[i])
			code += string(chars[seededRand.Intn(len(chars))])
		}
	}
	return code
}

// Code2Id code转id，用户解码
func Code2Id(code string) (id uint, err error) {
	for i := 0; i < len(code); i++ {
		if string(code[i]) == divider {
			break
		}
		charIdx := strings.Index(chars, string(code[i]))
		if charIdx == -1 {
			return 0, errors.New("code decode failed")
		}
		charIndex := uint(charIdx) // nolint:gosec
		if i > 0 {
			id = id*charLen + charIndex
		} else {
			id = charIndex
		}
	}
	return id, nil
}
