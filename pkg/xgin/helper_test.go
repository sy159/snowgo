package xgin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"snowgo/pkg/xgin"
)

func setupTestRouter(path string, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(path, handler)
	return r
}

// ParsePathID64 测试

func TestParsePathID64_ValidID(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 42 {
		t.Errorf("ParsePathID64() = %d, want 42", got)
	}
}

func TestParsePathID64_LargeInt64(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/9223372036854775807", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 9223372036854775807 {
		t.Errorf("ParsePathID64() = %d, want 9223372036854775807", got)
	}
}

func TestParsePathID64_Zero(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID64() = %d, want 0", got)
	}
}

func TestParsePathID64_Negative(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != -1 {
		t.Errorf("ParsePathID64() = %d, want -1", got)
	}
}

func TestParsePathID64_Empty(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID64() = %d, want 0", got)
	}
}

func TestParsePathID64_NonNumeric(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID64() = %d, want 0 for non-numeric", got)
	}
}

func TestParsePathID64_FloatString(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/3.14", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID64() = %d, want 0 for float string", got)
	}
}

func TestParsePathID64_MissingParam(t *testing.T) {
	// Route does NOT define :id — c.Param("id") returns "", ParseInt fails, returns 0
	var got int64
	r := setupTestRouter("/test/other", func(c *gin.Context) {
		got = xgin.ParsePathID64(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/other", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID64() with missing :id param = %d, want 0", got)
	}
}

// ParsePathID32 测试

func TestParsePathID32_ValidID(t *testing.T) {
	var got int32
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 42 {
		t.Errorf("ParsePathID32() = %d, want 42", got)
	}
}

func TestParsePathID32_MaxInt32(t *testing.T) {
	var got int32
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/2147483647", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 2147483647 {
		t.Errorf("ParsePathID32() = %d, want 2147483647", got)
	}
}

func TestParsePathID32_Zero(t *testing.T) {
	var got int32
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID32() = %d, want 0", got)
	}
}

func TestParsePathID32_Negative(t *testing.T) {
	var got int32
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != -1 {
		t.Errorf("ParsePathID32() = %d, want -1", got)
	}
}

func TestParsePathID32_OverflowInt64(t *testing.T) {
	var got int32
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	// 超出 int32 范围，应返回 0
	req := httptest.NewRequest(http.MethodGet, "/test/2147483648", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID32() = %d, want 0 for overflow", got)
	}
}

func TestParsePathID32_Empty(t *testing.T) {
	var got int32
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID32() = %d, want 0", got)
	}
}

func TestParsePathID32_NonNumeric(t *testing.T) {
	var got int32
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID32() = %d, want 0 for non-numeric", got)
	}
}

func TestParsePathID32_MissingParam(t *testing.T) {
	// Route does NOT define :id — c.Param("id") returns "", ParseInt fails, returns 0
	var got int32
	r := setupTestRouter("/test/other", func(c *gin.Context) {
		got = xgin.ParsePathID32(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/other", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID32() with missing :id param = %d, want 0", got)
	}
}
