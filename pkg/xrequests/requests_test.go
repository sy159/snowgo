package xrequests_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"snowgo/pkg/xrequests"
)

// TestBasicResponse 验证基本 GET 请求和 Response 方法
func TestBasicResponse(t *testing.T) {
	payload := `{"foo":"bar"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	res, err := xrequests.Get(srv.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// StatusCode
	if res.StatusCode != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, res.StatusCode)
	}

	// Text
	text, err := res.Text()
	if err != nil {
		t.Fatalf("Text() error: %v", err)
	}
	if text != payload {
		t.Errorf("Expected Text() '%s', got '%s'", payload, text)
	}

	// Json into struct
	var obj struct {
		Foo string `json:"foo"`
	}
	if err := res.Json(&obj); err != nil {
		t.Fatalf("Json() error: %v", err)
	}
	if obj.Foo != "bar" {
		t.Errorf("Expected Json.Foo 'bar', got '%s'", obj.Foo)
	}

	// Map
	m, err := res.Map()
	if err != nil {
		t.Fatalf("Map() error: %v", err)
	}
	if m["foo"] != "bar" {
		t.Errorf("Expected Map[foo]='bar', got '%v'", m["foo"])
	}
}

// TestOptions 分别测试各种 Option 行为
func TestOptions(t *testing.T) {
	// WithHeader
	srvHdr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Hdr") != "val" {
			t.Errorf("WithHeader failed, got '%s'", r.Header.Get("X-Test-Hdr"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srvHdr.Close()
	_, err := xrequests.Get(srvHdr.URL, xrequests.WithHeader(map[string]string{"X-Test-Hdr": "val"}))
	if err != nil {
		t.Fatalf("WithHeader error: %v", err)
	}

	// WithBody + WithQuery + POST wrapper
	srvBody := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "1" {
			t.Errorf("WithQuery failed, got '%s'", r.URL.RawQuery)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != "hello" {
			t.Errorf("WithBody failed, got '%s'", string(b))
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srvBody.Close()
	resp, err := xrequests.Post(srvBody.URL, "", xrequests.WithBody("hello"), xrequests.WithQuery(map[string]string{"q": "1"}))
	if err != nil {
		t.Fatalf("WithBody/WithQuery error: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d", resp.StatusCode)
	}

	// WithCtx 超时
	srvCtx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srvCtx.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	_, err = xrequests.Get(srvCtx.URL, xrequests.WithCtx(ctx))
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// WithMaxRetries 网络错误重试
	count := 0
	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		count++
		if count <= 2 {
			return nil, errors.New("neterr")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})
	client := &http.Client{Transport: rt}
	res, err := xrequests.Get("http://test", xrequests.WithClient(client), xrequests.WithMaxRetries(2))
	if err != nil {
		t.Fatalf("WithMaxRetries error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 attempts, got %d", count)
	}
	var m map[string]interface{}
	if err := res.Json(&m); err != nil {
		t.Fatalf("Json after retry error: %v", err)
	}
}

// TestWrappers 测试 Post/Delete/Put 封装
func TestWrappers(t *testing.T) {
	handler := func(code int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }
	}
	// POST
	srvP := httptest.NewServer(handler(http.StatusAccepted))
	defer srvP.Close()
	if res, err := xrequests.Post(srvP.URL, `{"x":1}`); err != nil || res.StatusCode != http.StatusAccepted {
		t.Errorf("Post wrapper failed: %v, %d", err, res.StatusCode)
	}
	// DELETE
	srvD := httptest.NewServer(handler(http.StatusNoContent))
	defer srvD.Close()
	if res, err := xrequests.Delete(srvD.URL, ``); err != nil || res.StatusCode != http.StatusNoContent {
		t.Errorf("Delete wrapper failed: %v, %d", err, res.StatusCode)
	}
	// PUT
	srvU := httptest.NewServer(handler(http.StatusResetContent))
	defer srvU.Close()
	if res, err := xrequests.Put(srvU.URL, ``); err != nil || res.StatusCode != http.StatusResetContent {
		t.Errorf("Put wrapper failed: %v, %d", err, res.StatusCode)
	}
}

// roundTripperFunc 用于模拟 Transport
type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
