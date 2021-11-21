// +build windows

package color

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
)

var _ = RandomColor()

// RandomColor generates a random color.
func RandomColor() string {
	return fmt.Sprintf("#%s", strconv.FormatInt(int64(rand.Intn(16777216)), 16))
}

// RedFont ...
func RedFont(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// RedBackground ...
func RedBackground(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// GreenFont ...
func GreenFont(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// GreenBackground ...
func GreenBackground(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// YellowFont 黄色字体
func YellowFont(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// YellowBackground 黄色背景
func YellowBackground(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// BlueFont 蓝色字体
func BlueFont(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// BlueBackground 蓝色背景
func BlueBackground(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// WhiteFont 白色字体
func WhiteFont(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// WhiteBackground 白色背景
func WhiteBackground(msg string) string {
	return fmt.Sprintf("%s", msg)
}

// StatusCodeColor 根据状态码返回对应颜色
func StatusCodeColor(statusCode int) string {
	msg := strconv.Itoa(statusCode)
	switch {
	case statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices:
		return GreenBackground(msg)
	case statusCode >= http.StatusMultipleChoices && statusCode < http.StatusBadRequest:
		return WhiteBackground(msg)
	case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
		return YellowBackground(msg)
	default:
		return RedBackground(msg)
	}
}

// MethodColor 根据method返回对应颜色
func MethodColor(method string) string {
	switch method {
	case http.MethodGet:
		return GreenBackground(method)
	case http.MethodPost:
		return BlueBackground(method)
	case http.MethodPut, http.MethodPatch:
		return YellowBackground(method)
	case http.MethodDelete:
		return RedBackground(method)
	default:
		return WhiteBackground(method)
	}
}
