package xcolor_test

import (
	"net/http"
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
	// === Happy path ===
	t.Run("happy: 2xx green", func(t *testing.T) {
		got := xcolor.StatusCodeColor(http.StatusOK)
		if !strings.Contains(got, "200") {
			t.Fatalf("StatusCodeColor(200) doesn't contain code: %q", got)
		}
		if !strings.Contains(got, "\x1b[42m") {
			t.Fatalf("StatusCodeColor(200) should be green background: %q", got)
		}
	})

	t.Run("happy: 3xx purple", func(t *testing.T) {
		got := xcolor.StatusCodeColor(http.StatusFound)
		if !strings.Contains(got, "302") {
			t.Fatalf("StatusCodeColor(302) doesn't contain code: %q", got)
		}
		if !strings.Contains(got, "\x1b[45m") {
			t.Fatalf("StatusCodeColor(302) should be purple background: %q", got)
		}
	})

	t.Run("happy: 4xx yellow", func(t *testing.T) {
		got := xcolor.StatusCodeColor(http.StatusBadRequest)
		if !strings.Contains(got, "400") {
			t.Fatalf("StatusCodeColor(400) doesn't contain code: %q", got)
		}
		if !strings.Contains(got, "\x1b[43m") {
			t.Fatalf("StatusCodeColor(400) should be yellow background: %q", got)
		}
	})

	t.Run("happy: 5xx red", func(t *testing.T) {
		got := xcolor.StatusCodeColor(http.StatusInternalServerError)
		if !strings.Contains(got, "500") {
			t.Fatalf("StatusCodeColor(500) doesn't contain code: %q", got)
		}
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("StatusCodeColor(500) should be red background: %q", got)
		}
	})

	// === Boundary values ===
	t.Run("boundary: 0 falls to default (red)", func(t *testing.T) {
		got := xcolor.StatusCodeColor(0)
		if !strings.Contains(got, "0") {
			t.Fatalf("StatusCodeColor(0) doesn't contain code: %q", got)
		}
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("StatusCodeColor(0) should be red background (default): %q", got)
		}
	})

	t.Run("boundary: 100 (below 200) falls to default (red)", func(t *testing.T) {
		got := xcolor.StatusCodeColor(100)
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("StatusCodeColor(100) should be red background: %q", got)
		}
	})

	t.Run("boundary: 418 (I'm a teapot) → yellow", func(t *testing.T) {
		got := xcolor.StatusCodeColor(418)
		if !strings.Contains(got, "\x1b[43m") {
			t.Fatalf("StatusCodeColor(418) should be yellow background: %q", got)
		}
	})

	t.Run("boundary: 599 (max server error) → red", func(t *testing.T) {
		got := xcolor.StatusCodeColor(599)
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("StatusCodeColor(599) should be red background: %q", got)
		}
	})
}

func TestBizCodeColor(t *testing.T) {
	// === Happy path ===
	t.Run("happy: 0 (OK) green", func(t *testing.T) {
		got := xcolor.BizCodeColor(0)
		if !strings.Contains(got, "\x1b[42m") {
			t.Fatalf("BizCodeColor(0) should be green background: %q", got)
		}
	})

	t.Run("happy: 200 green", func(t *testing.T) {
		got := xcolor.BizCodeColor(200)
		if !strings.Contains(got, "\x1b[42m") {
			t.Fatalf("BizCodeColor(200) should be green background: %q", got)
		}
	})

	t.Run("happy: 301 purple", func(t *testing.T) {
		got := xcolor.BizCodeColor(301)
		if !strings.Contains(got, "\x1b[45m") {
			t.Fatalf("BizCodeColor(301) should be purple background: %q", got)
		}
	})

	t.Run("happy: 400 yellow", func(t *testing.T) {
		got := xcolor.BizCodeColor(400)
		if !strings.Contains(got, "\x1b[43m") {
			t.Fatalf("BizCodeColor(400) should be yellow background: %q", got)
		}
	})

	t.Run("happy: 500 red", func(t *testing.T) {
		got := xcolor.BizCodeColor(500)
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("BizCodeColor(500) should be red background: %q", got)
		}
	})

	// === Boundary values ===
	t.Run("boundary: negative code → red", func(t *testing.T) {
		got := xcolor.BizCodeColor(-1)
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("BizCodeColor(-1) should be red background: %q", got)
		}
	})

	t.Run("boundary: very large code → red", func(t *testing.T) {
		got := xcolor.BizCodeColor(999999)
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("BizCodeColor(999999) should be red background: %q", got)
		}
	})

	t.Run("boundary: 1xx code → red (below 200)", func(t *testing.T) {
		got := xcolor.BizCodeColor(100)
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("BizCodeColor(100) should be red background: %q", got)
		}
	})
}

func TestMethodColor(t *testing.T) {
	// === Happy path ===
	t.Run("happy: GET green", func(t *testing.T) {
		got := xcolor.MethodColor(http.MethodGet)
		if !strings.Contains(got, "\x1b[42m") {
			t.Fatalf("MethodColor(GET) should be green background: %q", got)
		}
	})

	t.Run("happy: POST blue", func(t *testing.T) {
		got := xcolor.MethodColor(http.MethodPost)
		if !strings.Contains(got, "\x1b[44m") {
			t.Fatalf("MethodColor(POST) should be blue background: %q", got)
		}
	})

	t.Run("happy: PUT yellow", func(t *testing.T) {
		got := xcolor.MethodColor(http.MethodPut)
		if !strings.Contains(got, "\x1b[43m") {
			t.Fatalf("MethodColor(PUT) should be yellow background: %q", got)
		}
	})

	t.Run("happy: PATCH yellow", func(t *testing.T) {
		got := xcolor.MethodColor(http.MethodPatch)
		if !strings.Contains(got, "\x1b[43m") {
			t.Fatalf("MethodColor(PATCH) should be yellow background: %q", got)
		}
	})

	t.Run("happy: DELETE red", func(t *testing.T) {
		got := xcolor.MethodColor(http.MethodDelete)
		if !strings.Contains(got, "\x1b[41m") {
			t.Fatalf("MethodColor(DELETE) should be red background: %q", got)
		}
	})

	t.Run("happy: OPTIONS → white (default)", func(t *testing.T) {
		got := xcolor.MethodColor("OPTIONS")
		if !strings.Contains(got, "\x1b[47m") {
			t.Fatalf("MethodColor(OPTIONS) should be white background: %q", got)
		}
	})

	// === Boundary values ===
	t.Run("boundary: empty string → white (default)", func(t *testing.T) {
		got := xcolor.MethodColor("")
		if !strings.Contains(got, "\x1b[47m") {
			t.Fatalf("MethodColor(\"\") should be white background: %q", got)
		}
	})

	t.Run("boundary: unknown method → white (default)", func(t *testing.T) {
		got := xcolor.MethodColor("CONNECT")
		if !strings.Contains(got, "\x1b[47m") {
			t.Fatalf("MethodColor(CONNECT) should be white background: %q", got)
		}
	})
}
