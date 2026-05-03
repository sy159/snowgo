package xcolor_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"snowgo/pkg/xcolor"
)

func TestFontColors(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"RedFont", xcolor.RedFont},
		{"GreenFont", xcolor.GreenFont},
		{"YellowFont", xcolor.YellowFont},
		{"BlueFont", xcolor.BlueFont},
		{"PurpleFont", xcolor.PurpleFont},
		{"WhiteFont", xcolor.WhiteFont},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("hello")
			if !strings.Contains(got, "hello") {
				t.Errorf("%s output doesn't contain message: %q", tt.name, got)
			}
			if !strings.HasPrefix(got, "\x1b[") {
				t.Errorf("%s should start with ANSI escape: %q", tt.name, got)
			}
		})
	}
}

func TestBackgroundColors(t *testing.T) {
	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"RedBackground", xcolor.RedBackground},
		{"GreenBackground", xcolor.GreenBackground},
		{"YellowBackground", xcolor.YellowBackground},
		{"BlueBackground", xcolor.BlueBackground},
		{"PurpleBackground", xcolor.PurpleBackground},
		{"WhiteBackground", xcolor.WhiteBackground},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("hello")
			if !strings.Contains(got, "hello") {
				t.Errorf("%s output doesn't contain message: %q", tt.name, got)
			}
			if !strings.HasPrefix(got, "\x1b[") {
				t.Errorf("%s should start with ANSI escape: %q", tt.name, got)
			}
		})
	}
}

func TestStatusCodeColor(t *testing.T) {
	tests := []struct {
		code int
	}{
		{http.StatusOK},
		{http.StatusFound},
		{http.StatusBadRequest},
		{http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := xcolor.StatusCodeColor(tt.code)
			if !strings.Contains(got, strconv.Itoa(tt.code)) {
				t.Errorf("StatusCodeColor(%d) doesn't contain code: %q", tt.code, got)
			}
		})
	}
}

func TestBizCodeColor(t *testing.T) {
	tests := []struct {
		code int
	}{
		{0},   // OK
		{200}, // 2xx success
		{301}, // 3xx redirect
		{400}, // 4xx client error
		{500}, // 5xx server error
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.code), func(t *testing.T) {
			got := xcolor.BizCodeColor(tt.code)
			if !strings.Contains(got, strconv.Itoa(tt.code)) {
				t.Errorf("BizCodeColor(%d) doesn't contain code: %q", tt.code, got)
			}
		})
	}
}

func TestMethodColor(t *testing.T) {
	tests := []struct {
		method string
	}{
		{http.MethodGet},
		{http.MethodPost},
		{http.MethodPut},
		{http.MethodPatch}, // same as PUT
		{http.MethodDelete},
		{"OPTIONS"}, // unknown method → default
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := xcolor.MethodColor(tt.method)
			if !strings.Contains(got, tt.method) {
				t.Errorf("MethodColor(%s) doesn't contain method: %q", tt.method, got)
			}
		})
	}
}
