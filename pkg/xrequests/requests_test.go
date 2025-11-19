package xrequests_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"snowgo/pkg/xrequests"
	"testing"
	"time"
)

// -------------------- Mock Servers --------------------

// 用于单元测试的 Mock Server
func mockServerForTest() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		resp := map[string]interface{}{
			"method": r.Method,
			"query":  r.URL.RawQuery,
			"body":   string(body),
			"header": r.Header,
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
}

// 用于 Benchmark 的简单 Mock Server
func mockServerForBench() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`ok`))
	}))
}

func TestPostJSONWithAllOptions(t *testing.T) {
	// -------------------- Mock Server --------------------
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟处理慢一点，方便验证 Timeout
		time.Sleep(time.Millisecond * 50)

		// 返回请求信息，用于校验
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		respData := map[string]interface{}{
			"method": r.Method,
			"query":  r.URL.RawQuery,
			"body":   string(body),
			"header": r.Header,
		}
		respBytes, _ := json.Marshal(respData)

		// 设置响应头
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Resp-Test", "resp-value")
		w.WriteHeader(201)
		_, _ = w.Write(respBytes)
	}))
	defer server.Close()

	// -------------------- 请求参数 --------------------
	jsonData := map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
	}
	customHeader := map[string]string{
		"X-Test": "abc",
	}
	queryParams := map[string]string{
		"q": "golang",
	}

	// 设置超时时间足够大，不触发
	timeout := time.Second * 1

	// -------------------- 发起请求 --------------------
	resp, err := xrequests.Post(
		server.URL,
		xrequests.WithJSON(jsonData),
		xrequests.WithHeader(customHeader),
		xrequests.WithQuery(queryParams),
		xrequests.WithTimeout(timeout),
	)
	if err != nil {
		t.Fatal(err)
	}

	// -------------------- 校验 Response --------------------
	// 1. StatusCode
	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	// 2. Header
	if resp.GetHeader("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", resp.GetHeader("Content-Type"))
	}
	if resp.GetHeader("X-Resp-Test") != "resp-value" {
		t.Fatalf("expected X-Resp-Test resp-value, got %s", resp.GetHeader("X-Resp-Test"))
	}

	// 3. Body 解析
	var bodyMap map[string]interface{}
	if err := resp.Json(&bodyMap); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	// 4. 校验请求方法
	if bodyMap["method"] != "POST" {
		t.Fatalf("expected method POST, got %v", bodyMap["method"])
	}

	// 5. 校验 Query 参数
	if bodyMap["query"] != "q=golang" {
		t.Fatalf("expected query q=golang, got %v", bodyMap["query"])
	}

	// 6. 校验请求 Header
	headerMap := bodyMap["header"].(map[string]interface{})
	vals := headerMap["X-Test"].([]interface{})
	if len(vals) == 0 || vals[0].(string) != "abc" {
		t.Fatalf("expected X-Test=abc, got %v", vals)
	}

	// 7. 校验请求 Body (JSON)
	if bodyMap["body"] == nil || bodyMap["body"] == "" {
		t.Fatalf("expected body not empty")
	}
	var bodySent map[string]interface{}
	if err := json.Unmarshal([]byte(bodyMap["body"].(string)), &bodySent); err != nil {
		t.Fatalf("failed to unmarshal body JSON: %v", err)
	}
	if bodySent["name"] != "Alice" || bodySent["email"] != "alice@example.com" {
		t.Fatalf("body content mismatch, got %v", bodySent)
	}

	// 8. 校验 Response 原始对象
	raw := resp.RawResponse()
	if raw == nil {
		t.Fatal("RawResponse is nil")
	}
	if raw.StatusCode != 201 {
		t.Fatalf("RawResponse.StatusCode expected 201, got %d", raw.StatusCode)
	}
}

// -------------------- Option Tests --------------------
func TestWithHeader(t *testing.T) {
	server := mockServerForTest()
	defer server.Close()

	custom := map[string]string{"X-Test": "abc"}
	resp, err := xrequests.Get(server.URL, xrequests.WithHeader(custom))
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	if err := resp.Json(&m); err != nil {
		t.Fatal(err)
	}

	headerMap := m["header"].(map[string]interface{})
	vals := headerMap["X-Test"].([]interface{})
	if len(vals) == 0 || vals[0].(string) != "abc" {
		t.Fatalf("expected X-Test=abc, got %v", vals)
	}
}

func TestWithJSON(t *testing.T) {
	server := mockServerForTest()
	defer server.Close()

	data := map[string]string{"hello": "world"}
	resp, err := xrequests.Post(server.URL, xrequests.WithJSON(data))
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	if err := resp.Json(&m); err != nil {
		t.Fatal(err)
	}
	if m["body"] == nil || m["body"] == "" {
		t.Fatal("body should contain JSON")
	}
}

func TestWithBodyString(t *testing.T) {
	server := mockServerForTest()
	defer server.Close()

	resp, err := xrequests.Post(server.URL, xrequests.WithBodyString("testbody"))
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	if err := resp.Json(&m); err != nil {
		t.Fatal(err)
	}
	if m["body"] != "testbody" {
		t.Fatalf("expected body=testbody, got %v", m["body"])
	}
}

func TestWithQuery(t *testing.T) {
	server := mockServerForTest()
	defer server.Close()

	resp, err := xrequests.Get(server.URL, xrequests.WithQuery(map[string]string{"q": "123"}))
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	if err := resp.Json(&m); err != nil {
		t.Fatal(err)
	}
	if m["query"] != "q=123" {
		t.Fatalf("expected query=q=123, got %v", m["query"])
	}
}

func TestWithTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond * 200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	start := time.Now()
	_, err := xrequests.Get(server.URL, xrequests.WithTimeout(time.Millisecond*50))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > time.Millisecond*100 {
		t.Fatal("timeout did not trigger in time")
	}
}

func TestWithMaxRetries(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("Server does not support hijacking")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatal(err)
			}
			_ = conn.Close()
			return
		}
		_, _ = w.Write([]byte(`{"ok":1}`))
	}))
	defer server.Close()

	resp, err := xrequests.Get(server.URL, xrequests.WithMaxRetries(3))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if count < 2 {
		t.Fatal("retry did not happen")
	}
}

func TestWithClientAndCtx(t *testing.T) {
	server := mockServerForTest()
	defer server.Close()

	client := &http.Client{Timeout: time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	resp, err := xrequests.Get(server.URL,
		xrequests.WithClient(client),
		xrequests.WithCtx(ctx),
	)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestWithRequest(t *testing.T) {
	server := mockServerForTest()
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := xrequests.Get(server.URL, xrequests.WithRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// -------------------- HTTP Methods --------------------

func TestHTTPMethods(t *testing.T) {
	server := mockServerForTest()
	defer server.Close()

	methods := []struct {
		name   string
		method func(string, ...xrequests.Option) (*xrequests.Response, error)
	}{
		{"GET", xrequests.Get},
		{"POST", xrequests.Post},
		{"PUT", xrequests.Put},
		{"DELETE", xrequests.Delete},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			resp, err := m.method(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 {
				t.Fatalf("%s expected 200, got %d", m.name, resp.StatusCode)
			}
		})
	}
}

// -------------------- Benchmarks --------------------

func BenchmarkGet(b *testing.B) {
	server := mockServerForBench()
	defer server.Close()

	for i := 0; i < b.N; i++ {
		_, err := xrequests.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetParallel(b *testing.B) {
	server := mockServerForBench()
	defer server.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := xrequests.Get(server.URL)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkPostJSON(b *testing.B) {
	server := mockServerForBench()
	defer server.Close()

	data := map[string]string{"hello": "world"}

	for i := 0; i < b.N; i++ {
		_, err := xrequests.Post(server.URL, xrequests.WithJSON(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostJSONParallel(b *testing.B) {
	server := mockServerForBench()
	defer server.Close()

	data := map[string]string{"hello": "world"}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := xrequests.Post(server.URL, xrequests.WithJSON(data))
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
