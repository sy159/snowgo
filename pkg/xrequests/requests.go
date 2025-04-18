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
	defaultMaxRetries = 0
)

type Option func(*request)

type request struct {
	client     *http.Client
	ctx        context.Context
	body       string
	header     map[string]string
	maxRetries int
	request    *http.Request
	query      map[string]string
}

type Response struct {
	response   *http.Response
	Body       []byte
	StatusCode int
}

func Request(method, rawURL, body string, opts ...Option) (*Response, error) {
	req := &request{
		client:     defaultClient,
		ctx:        context.Background(),
		header:     cloneHeader(defaultHeader),
		maxRetries: defaultMaxRetries,
		body:       body,
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
	for i := 0; i <= req.maxRetries; i++ {
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

		resp, err := req.client.Do(httpReq)
		if err == nil {
			// 将原始的Body保存
			response := &Response{
				response:   resp,
				StatusCode: resp.StatusCode,
			}
			// 第一次读取响应体，复制一份到body字段
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			// 将响应体内容写入到body字段，缓存起来
			response.Body = body
			// 将原始的Body重新赋值，供外部使用
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(body))

			return response, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// cloneHeader 复制header，防止header被修改
func cloneHeader(header map[string]string) map[string]string {
	cpHeader := make(map[string]string, len(header))
	for k, v := range header {
		cpHeader[k] = v
	}
	return cpHeader
}

// Json 返回body json内容
func (res *Response) Json(v interface{}) error {
	return json.Unmarshal(res.Body, v)
}

// Map 返回body map内容
func (res *Response) Map() (map[string]interface{}, error) {
	var m map[string]interface{}
	err := json.Unmarshal(res.Body, &m)
	return m, err
}

// Text 返回body str内容
func (res *Response) Text() (string, error) {
	return string(res.Body), nil
}

func (res *Response) RawResponse() *http.Response {
	return res.response
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

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(maxRetries int) Option {
	return func(r *request) {
		r.maxRetries = maxRetries
	}
}

// WithRequest 设置自定义请求
func WithRequest(req *http.Request) Option {
	return func(r *request) {
		r.request = req
	}
}

// WithQuery 设置query
func WithQuery(params map[string]string) Option {
	return func(r *request) {
		r.query = params
	}
}
