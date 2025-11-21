package xstr_tool

import (
	"snowgo/pkg"
	"strings"
)

const (
	lowercase       = "abcdefghijklmnopqrstuvwxyz"
	uppercase       = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits          = "0123456789"
	punctuation     = "\"!#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	LowerFlag       = 1 << iota // 小写字母
	UpperFlag                   // 大写字母
	DigitFlag                   // 数字
	PunctuationFlag             // 特殊字符
)

// ReverseStr 字符串反转
func ReverseStr(s string) string {
	sRune := []rune(s)
	for i, j := 0, len(sRune)-1; i < j; i, j = i+1, j-1 {
		sRune[i], sRune[j] = sRune[j], sRune[i]
	}
	return string(sRune)
}

// RandStr 生成随机字符串，调用方式：RandStr(10, LowerFlag|UpperFlag|DigitFlag)
func RandStr(n int, flag int) string {
	chars := ""
	if flag&LowerFlag != 0 {
		chars += lowercase
	}
	if flag&UpperFlag != 0 {
		chars += uppercase
	}
	if flag&DigitFlag != 0 {
		chars += digits
	}
	if flag&PunctuationFlag != 0 {
		chars += punctuation
	}

	// 检查是否有有效字符集
	if len(chars) == 0 {
		return ""
	}

	charsLen := len(chars)
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[common.WeakRandInt63n(int64(charsLen))]
	}
	return string(b)
}

// RandShuffleStr 生成随机不重复字符串，调用方式：RandShuffleStr(8, LowerFlag|DigitFlag)
func RandShuffleStr(n int, flag int) string {
	chars := ""
	if flag&LowerFlag != 0 {
		chars += lowercase
	}
	if flag&UpperFlag != 0 {
		chars += uppercase
	}
	if flag&DigitFlag != 0 {
		chars += digits
	}
	if flag&PunctuationFlag != 0 {
		chars += punctuation
	}

	// 检查是否有有效字符集
	if len(chars) == 0 {
		return ""
	}

	charByte := []byte(chars)
	for i := len(charByte) - 1; i > 0; i-- {
		// 随机交换位置，实现打乱效果
		num := common.WeakRandInt63n(int64(i + 1))
		charByte[i], charByte[num] = charByte[num], charByte[i]
	}
	return string(charByte[:n])
}

// IsUniqueStr 判断字符串是否不存在重复字符
func IsUniqueStr(s string) bool {
	for index, v := range s {
		if strings.LastIndex(s, string(v)) != index {
			return false
		}
	}
	return true
}
