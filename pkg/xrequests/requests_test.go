package xrequests_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"snowgo/pkg/xrequests"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "hello world"}`))
	}))
	defer server.Close()

	resp, err := xrequests.Get(server.URL)
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, resp.Response.StatusCode)
	}

	var result map[string]string
	err = resp.Json(&result)
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}
	fmt.Println(result)
	if result["message"] != "hello world" {
		t.Errorf("Expected message 'hello world', got %v", result["message"])
	}
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	body := `{"name": "test"}`
	resp, err := xrequests.Post(server.URL, body)
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, resp.Response.StatusCode)
	}

	var result map[string]string
	err = resp.Json(&result)
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}

	fmt.Println(result)
	if result["name"] != "test" {
		t.Errorf("Expected name 'test', got %v", result["name"])
	}
}

func TestWithHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Custom-Header") != "value" {
			t.Errorf("Expected header 'Custom-Header' to be 'value', got %v", r.Header.Get("Custom-Header"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := xrequests.Get(server.URL, xrequests.WithHeader(map[string]string{"Custom-Header": "value"}))
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}
}

func TestWithCtx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "hello world"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := xrequests.Get(server.URL, xrequests.WithCtx(ctx))
	if err == nil {
		t.Fatalf("Expected xerror due to context timeout, got no xerror")
	}
}

func TestWithMaxRetryNum(t *testing.T) {
	retryCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		fmt.Println(retryCount)
		if retryCount < 3 {
			time.Sleep(1 * time.Second)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"xerror": "server xerror"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	defaultClient := &http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := xrequests.Get(server.URL, xrequests.WithMaxRetryNum(2), xrequests.WithClient(defaultClient))
	if err != nil {
		t.Fatalf("预期无错误，但得到的是 %v", err)
	}

	if retryCount != 3 { // 初次尝试 + 2次重试
		t.Errorf("预期3次请求，得到的是 %v", retryCount)
	}

	if resp.Response.StatusCode != http.StatusOK {
		t.Errorf("预期状态码 %v，得到的是 %v", http.StatusOK, resp.Response.StatusCode)
	}

	var result map[string]string
	err = resp.Json(&result)
	if err != nil {
		t.Fatalf("预期无错误，但得到的是 %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("预期消息 'success'，但得到的是 %v", result["message"])
	}
}

func TestWithClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "hello world"}`))
	}))
	defer server.Close()
	customClient := &http.Client{Timeout: 1 * time.Millisecond}
	res, err := xrequests.Get(server.URL, xrequests.WithClient(customClient))
	fmt.Println(res, err)
	if err == nil {
		t.Fatalf("Expected xerror due to client timeout, got no xerror")
	}
}

func TestWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name": "test"}` {
			t.Errorf("Expected body '{\"name\": \"test\"}', got %v", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := xrequests.Post(server.URL, "", xrequests.WithBody(`{"name": "test"}`))
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := xrequests.Delete(server.URL, "")
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, resp.Response.StatusCode)
	}
}

func TestPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := xrequests.Put(server.URL, "")
	if err != nil {
		t.Fatalf("Expected no xerror, got %v", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, resp.Response.StatusCode)
	}
}
