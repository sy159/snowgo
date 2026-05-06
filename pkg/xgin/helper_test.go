package xgin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(path string, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET(path, handler)
	return r
}

func TestParsePathID_ValidID(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = ParsePathID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 42 {
		t.Errorf("ParsePathID() = %d, want 42", got)
	}
}

func TestParsePathID_LargeInt64(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = ParsePathID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/9223372036854775807", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 9223372036854775807 {
		t.Errorf("ParsePathID() = %d, want 9223372036854775807", got)
	}
}

func TestParsePathID_Zero(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = ParsePathID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID() = %d, want 0", got)
	}
}

func TestParsePathID_Negative(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = ParsePathID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != -1 {
		t.Errorf("ParsePathID() = %d, want -1", got)
	}
}

func TestParsePathID_Empty(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = ParsePathID(c)
	})

	// Empty param — gin still routes to this handler
	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID() = %d, want 0", got)
	}
}

func TestParsePathID_NonNumeric(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = ParsePathID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID() = %d, want 0 for non-numeric", got)
	}
}

func TestParsePathID_FloatString(t *testing.T) {
	var got int64
	r := setupTestRouter("/test/:id", func(c *gin.Context) {
		got = ParsePathID(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/3.14", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got != 0 {
		t.Errorf("ParsePathID() = %d, want 0 for float string", got)
	}
}
