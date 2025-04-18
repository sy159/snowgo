package xrequests

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

var (
	defaultClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConns:        40,
			MaxIdleConnsPerHost: 5,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     120 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // 校验https
			},
		},
		Timeout: 5 * time.Second,
	}
	defaultHeader = map[string]string{
		"user-agent":   "snowgo Request",
		"Content-Type": "application/json; charset=UTF-8",
	}
	defaultMaxRetryNum = 1
)

type Option func(*request)

type request struct {
	client      *http.Client
	ctx         context.Context
	body        string
	header      map[string]string
	maxRetryNum int
	hook        func(*http.Request)
	request     *http.Request
	query       map[string]string
}

type Response struct {
	Request  *http.Request
	Response *http.Response
	body     []byte
}

func Request(method, rawURL, body string, opts ...Option) (*Response, error) {
	req := &request{
		client:      defaultClient,
		ctx:         context.Background(),
		header:      cloneHeader(defaultHeader),
		maxRetryNum: defaultMaxRetryNum,
		body:        body,
	}

	// 加载配置
	for _, opt := range opts {
		opt(req)
	}

	// 构建 URL（包含 query 参数）
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if len(req.query) > 0 {
		q := parsedURL.Query()
		for k, v := range req.query {
			q.Set(k, v)
		}
		parsedURL.RawQuery = q.Encode()
	}

	var lastErr error
	for i := 0; i <= req.maxRetryNum; i++ {
		var bodyReader io.Reader
		if req.body != "" {
			bodyReader = bytes.NewBufferString(req.body)
		}

		httpReq := req.request
		if httpReq == nil {
			httpReq, err = http.NewRequestWithContext(req.ctx, method, parsedURL.String(), bodyReader)
			if err != nil {
				return nil, err
			}
			for k, v := range req.header {
				httpReq.Header.Set(k, v)
			}
		}

		if req.hook != nil {
			req.hook(httpReq)
		}

		resp, err := req.client.Do(httpReq)
		if err == nil {
			return &Response{Request: httpReq, Response: resp}, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func cloneHeader(src map[string]string) map[string]string {
	cp := make(map[string]string, len(src))
	for k, v := range src {
		cp[k] = v
	}
	return cp
}

func (res *Response) readBody() ([]byte, error) {
	if res.body != nil {
		return res.body, nil
	}
	body, err := io.ReadAll(res.Response.Body)
	res.Response.Body.Close()
	if err != nil {
		return nil, err
	}
	res.body = body
	return body, nil
}

// Json 返回body json内容
func (res *Response) Json(v interface{}) error {
	body, err := res.readBody()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// Map 返回body map内容
func (res *Response) Map() (map[string]interface{}, error) {
	body, err := res.readBody()
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(body, &m)
	return m, err
}

// Text 返回body str内容
func (res *Response) Text() (string, error) {
	body, err := res.readBody()
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (res *Response) StatusCode() int {
	if res.Response != nil {
		return res.Response.StatusCode
	}
	return 0
}

// Get 发起GET请求
func Get(url string, opts ...Option) (*Response, error) {
	return Request("GET", url, "", opts...)
}

// Post 发起POST请求
func Post(url, body string, opts ...Option) (*Response, error) {
	return Request("POST", url, body, opts...)
}

// Delete 发起DELETE请求
func Delete(url, body string, opts ...Option) (*Response, error) {
	return Request("DELETE", url, body, opts...)
}

// Put 发起PUT请求
func Put(url, body string, opts ...Option) (*Response, error) {
	return Request("PUT", url, body, opts...)
}

// WithClient 设置HTTP客户端
func WithClient(client *http.Client) Option {
	return func(r *request) {
		r.client = client
	}
}

// WithCtx 设置请求上下文
func WithCtx(ctx context.Context) Option {
	return func(r *request) {
		r.ctx = ctx
	}
}

// WithHeader 设置请求头
func WithHeader(header map[string]string) Option {
	return func(r *request) {
		r.header = cloneHeader(header)
	}
}

// WithBody 设置请求体内容
func WithBody(body string) Option {
	return func(r *request) {
		r.body = body
	}
}

// WithMaxRetryNum 设置最大重试次数
func WithMaxRetryNum(maxRetryNum int) Option {
	return func(r *request) {
		r.maxRetryNum = maxRetryNum
	}
}

// WithRequest 设置自定义请求
func WithRequest(req *http.Request) Option {
	return func(r *request) {
		r.request = req
	}
}

// WithHook 设置自定义操作
func WithHook(hook func(*http.Request)) Option {
	return func(r *request) {
		r.hook = hook
	}
}

// WithQuery 设置query
func WithQuery(params map[string]string) Option {
	return func(r *request) {
		r.query = params
	}
}
