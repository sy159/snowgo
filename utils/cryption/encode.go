package main

import (
	"errors"
	"math/rand"
	"snowgo/utils"
	"strings"
)

var (
	chars   = "5dfucyar1j2txwgs8mvp4qhb3n6k7ez9" // 不重复的加密字符串
	charLen = uint(len(chars))
	divider = "i" // 分割标识符
)

func id2Code(id uint, minLength int) (code string) {
	for id/charLen > 0 {
		code += string(chars[id%charLen])
		id /= charLen
	}
	code += string(chars[id%charLen])
	code = utils.ReverseStr(code)
	fixLen := minLength - len(code)
	if fixLen > 0 {
		code += divider
		for i := 0; i < fixLen-1; i++ {
			//code += string(chars[i])
			code += string(chars[rand.Intn(len(chars)-1)])
		}
	}
	return
}

func code2Id(code string) (id uint, err error) {
	for i := 0; i < len(code); i++ {
		if string(code[i]) == divider {
			break
		}
		charIdx := strings.Index(chars, string(code[i]))
		if charIdx == -1 {
			err = errors.New("code decode failed")
			return
		}
		charIndex := uint(charIdx)
		if i > 0 {
			id = id*charLen + charIndex
		} else {
			id = charIndex
		}
	}
	return
}
