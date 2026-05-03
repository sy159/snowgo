package xcolor

import (
	"net/http"
	e "snowgo/pkg/xerror"
	"strconv"
)

// RedFont ...
func RedFont(msg string) string {
	return "\x1b[31m" + msg + "\x1b[0m"
}

// RedBackground ...
func RedBackground(msg string) string {
	return "\x1b[41m" + msg + "\x1b[0m"
}

// GreenFont ...
func GreenFont(msg string) string {
	return "\x1b[32m" + msg + "\x1b[0m"
}

// GreenBackground ...
func GreenBackground(msg string) string {
	return "\x1b[42m" + msg + "\x1b[0m"
}

// YellowFont 黄色字体
func YellowFont(msg string) string {
	return "\x1b[33m" + msg + "\x1b[0m"
}

// YellowBackground 黄色背景
func YellowBackground(msg string) string {
	return "\x1b[43m" + msg + "\x1b[0m"
}

// BlueFont 蓝色字体
func BlueFont(msg string) string {
	return "\x1b[34m" + msg + "\x1b[0m"
}

// BlueBackground 蓝色背景
func BlueBackground(msg string) string {
	return "\x1b[44m" + msg + "\x1b[0m"
}

// PurpleFont 紫色字体
func PurpleFont(msg string) string {
	return "\x1b[35m" + msg + "\x1b[0m"
}

// PurpleBackground 紫色背景
func PurpleBackground(msg string) string {
	return "\x1b[45m" + msg + "\x1b[0m"
}

// WhiteFont 白色字体
func WhiteFont(msg string) string {
	return "\x1b[37m" + msg + "\x1b[0m"
}

// WhiteBackground 白色背景
func WhiteBackground(msg string) string {
	return "\x1b[47m" + msg + "\x1b[0m"
}

// StatusCodeColor 根据状态码返回对应颜色
func StatusCodeColor(statusCode int) string {
	msg := strconv.Itoa(statusCode)
	switch {
	case statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices:
		return GreenBackground(msg)
	case statusCode >= http.StatusMultipleChoices && statusCode < http.StatusBadRequest:
		return PurpleBackground(msg)
	case statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError:
		return YellowBackground(msg)
	default:
		return RedBackground(msg)
	}
}

// BizCodeColor 根据业务code返回对应颜色
func BizCodeColor(bizCode int) string {
	msg := strconv.Itoa(bizCode)
	switch {
	case bizCode == e.OK.GetErrCode() || (bizCode >= http.StatusOK && bizCode < http.StatusMultipleChoices):
		return GreenBackground(msg)
	case bizCode >= http.StatusMultipleChoices && bizCode < http.StatusBadRequest:
		return PurpleBackground(msg)
	case bizCode >= http.StatusBadRequest && bizCode < http.StatusInternalServerError:
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
