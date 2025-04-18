package xresponse_test

import (
	"net/http"
	"net/http/httptest"
	"snowgo/pkg/xerror"
	"snowgo/pkg/xresponse"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setUp() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	return r
}

func TestString(t *testing.T) {
	r := setUp()
	r.GET("/test-string", func(c *gin.Context) {
		xresponse.String(c, "Hello, World!")
	})

	req, _ := http.NewRequest("GET", "/test-string", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello, World!", w.Body.String())
}

func TestJson(t *testing.T) {
	r := setUp()
	r.GET("/test-json", func(c *gin.Context) {
		xresponse.Json(c, 0, "success", map[string]string{"key": "value"})
	})

	req, _ := http.NewRequest("GET", "/test-json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expected := `{"code":0,"msg":"success","data":{"key":"value"}}`
	assert.JSONEq(t, expected, w.Body.String())
}

func TestJsonByError(t *testing.T) {
	r := setUp()
	r.GET("/test-json-by-error", func(c *gin.Context) {
		xresponse.JsonByError(c, xerror.OK, map[string]string{"key": "value"})
	})

	req, _ := http.NewRequest("GET", "/test-json-by-error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expected := `{"code":0,"msg":"success","data":{"key":"value"}}`
	assert.JSONEq(t, expected, w.Body.String())
}

func TestSuccess(t *testing.T) {
	r := setUp()
	r.GET("/test-success", func(c *gin.Context) {
		xresponse.Success(c, map[string]string{"key": "value"})
	})

	req, _ := http.NewRequest("GET", "/test-success", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expected := `{"code":0,"msg":"success","data":{"key":"value"}}`
	assert.JSONEq(t, expected, w.Body.String())
}

func TestFail(t *testing.T) {
	r := setUp()
	r.GET("/test-fail", func(c *gin.Context) {
		xresponse.Fail(c, 400, "Bad Request")
	})

	req, _ := http.NewRequest("GET", "/test-fail", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expected := `{"code":400,"msg":"Bad Request","data":{}}`
	assert.JSONEq(t, expected, w.Body.String())
}

func TestFailByError(t *testing.T) {
	r := setUp()
	r.GET("/test-fail-by-error", func(c *gin.Context) {
		xresponse.FailByError(c, xerror.HttpInternalServerError)
	})

	req, _ := http.NewRequest("GET", "/test-fail-by-error", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expected := `{"code":500,"msg":"Internal Server Error","data":{}}`
	assert.JSONEq(t, expected, w.Body.String())
}
