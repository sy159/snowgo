package xresponse_test

import (
	"encoding/json"
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
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "success", resp["msg"])
	assert.Equal(t, map[string]interface{}{"key": "value"}, resp["data"])
	assert.NotNil(t, resp["timestamp"])
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
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "success", resp["msg"])
	assert.Equal(t, map[string]interface{}{"key": "value"}, resp["data"])
	assert.NotNil(t, resp["timestamp"])
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
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "success", resp["msg"])
	assert.Equal(t, map[string]interface{}{"key": "value"}, resp["data"])
	assert.NotNil(t, resp["timestamp"])
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
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(400), resp["code"])
	assert.Equal(t, "Bad Request", resp["msg"])
	// Fail passes nil data which becomes empty struct {} in JSON
	assert.Equal(t, map[string]interface{}{}, resp["data"])
	assert.NotNil(t, resp["timestamp"])
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
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(500), resp["code"])
	assert.Equal(t, "Internal Server Error", resp["msg"])
	// FailByError passes nil data which becomes empty struct {} in JSON
	assert.Equal(t, map[string]interface{}{}, resp["data"])
	assert.NotNil(t, resp["timestamp"])
}
