package xstr_tool_test

import (
	"snowgo/pkg/xstr_tool"
	"strings"
	"testing"
)

var (
	lowercase   = "abcdefghijklmnopqrstuvwxyz"
	uppercase   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits      = "0123456789"
	punctuation = "\"!#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
)

// 测试字符串反转功能
func TestReverseStr(t *testing.T) {
	testCases := []struct {
		input  string
		output string
	}{
		{"hello", "olleh"},
		{"test", "tset"},
		{"", ""},
		{"a", "a"},
		{"ab", "ba"},
	}

	for _, tc := range testCases {
		t.Run("Reverse "+tc.input, func(t *testing.T) {
			result := xstr_tool.ReverseStr(tc.input)
			if result != tc.output {
				t.Errorf("ReverseStr(%s) = %s, want %s", tc.input, result, tc.output)
			}
		})
	}
}

// 测试随机字符串生成功能
func TestRandStr(t *testing.T) {
	// 测试生成指定长度的随机字符串
	result := xstr_tool.RandStr(10, xstr_tool.LowerFlag|xstr_tool.DigitFlag)
	if len(result) != 10 {
		t.Errorf("RandStr(10, LowerFlag|DigitFlag) length is not 10, got %d", len(result))
	}

	// 测试包含指定字符类型
	result = xstr_tool.RandStr(5, xstr_tool.UpperFlag|xstr_tool.PunctuationFlag)
	for _, c := range result {
		if !strings.ContainsRune(uppercase+punctuation, c) {
			t.Errorf("RandStr(5, UpperFlag|PunctuationFlag) contains invalid character: %c", c)
			break
		}
	}

	// 测试无效标记
	result = xstr_tool.RandStr(5, 0)
	if len(result) != 0 {
		t.Errorf("RandStr(5, 0) should return empty string")
	}
}

// 测试随机不重复字符串生成功能
func TestRandShuffleStr(t *testing.T) {
	// 测试生成指定长度的随机不重复字符串
	result := xstr_tool.RandShuffleStr(8, xstr_tool.LowerFlag|xstr_tool.DigitFlag)
	if len(result) != 8 {
		t.Errorf("RandShuffleStr(8, LowerFlag|DigitFlag) length is not 8, got %d", len(result))
	}

	// 测试字符唯一性
	if !xstr_tool.IsUniqueStr(result) {
		t.Errorf("RandShuffleStr(8, LowerFlag|DigitFlag) contains duplicate characters")
	}

	// 测试包含指定字符类型
	for _, c := range result {
		if !strings.ContainsRune(lowercase+digits, c) {
			t.Errorf("RandShuffleStr(8, LowerFlag|DigitFlag) contains invalid character: %c", c)
			break
		}
	}

	// 测试无效标记
	result = xstr_tool.RandShuffleStr(8, 0)
	if len(result) != 0 {
		t.Errorf("RandShuffleStr(8, 0) should return empty string")
	}
}

// 测试判断字符串是否无重复字符功能
func TestIsUniqueStr(t *testing.T) {
	testCases := []struct {
		input  string
		output bool
	}{
		{"abcde", true},
		{"aabcde", false},
		{"", true},
		{"a", true},
		{"aa", false},
	}

	for _, tc := range testCases {
		t.Run("IsUnique "+tc.input, func(t *testing.T) {
			result := xstr_tool.IsUniqueStr(tc.input)
			if result != tc.output {
				t.Errorf("IsUniqueStr(%s) = %t, want %t", tc.input, result, tc.output)
			}
		})
	}
}
