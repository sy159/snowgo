package xrequests

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	common "snowgo/pkg"
	"time"
)

var (
	defaultClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     90 * time.Second,
			TLSClientConfig: &tls.Config{
				//InsecureSkipVerify: false, // 校验https
				MinVersion: tls.VersionTLS12, // 强制 TLS 1.2 以上
			},
		},
		Timeout: 10 * time.Second,
	}
	defaultHeader = map[string]string{
		"User-Agent":   "snowgo Request",
		"Content-Type": "application/json; charset=UTF-8",
	}
	defaultMaxRetries = 0
)

type Option func(*requestOptions)

type requestOptions struct {
	client     *http.Client
	ctx        context.Context
	body       []byte
	header     map[string]string
	maxRetries int
	request    *http.Request
	query      map[string]string
	formData   map[string]string // 表单数据
	json       interface{}       // 支持直接传入JSON对象
	timeout    time.Duration     // 支持单独设置超时
}

// Response 响应结构体
type Response struct {
	response   *http.Response
	Body       []byte
	StatusCode int
	Header     http.Header
}

// prepareRequestBody 准备请求体（JSON > FormData > body）
func prepareRequestBody(opts *requestOptions) error {
	switch {
	case opts.json != nil:
		b, err := json.Marshal(opts.json)
		if err != nil {
			return err
		}
		opts.body = b
		if opts.header == nil {
			opts.header = make(map[string]string)
		}
		// 只在未设置时写入
		if opts.header["Content-Type"] == "" {
			opts.header["Content-Type"] = "application/json"
		}

	case len(opts.formData) > 0:
		form := url.Values{}
		for k, v := range opts.formData {
			form.Set(k, v)
		}
		opts.body = []byte(form.Encode())
		if opts.header == nil {
			opts.header = make(map[string]string)
		}
		if opts.header["Content-Type"] == "" {
			opts.header["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}
	return nil
}

// createHTTPRequest 创建HTTP请求（支持自定义 request，且要求可重试时 body 可重置）
func createHTTPRequest(method, urlStr string, opts *requestOptions) (*http.Request, error) {
	if opts.request != nil {
		req := opts.request.Clone(opts.ctx)
		req.URL, _ = url.Parse(urlStr) // 统一用外部拼好的 URL

		// 把 opts.bodyBytes 写进去
		if len(opts.body) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(opts.body))
			req.ContentLength = int64(len(opts.body))
		}

		// 合并 header（不覆盖用户已设）
		for k, v := range opts.header {
			if req.Header.Get(k) == "" {
				req.Header.Set(k, v)
			}
		}
		return req, nil
	}

	var bodyReader io.Reader
	if len(opts.body) > 0 {
		bodyReader = bytes.NewReader(opts.body)
	}

	req, err := http.NewRequestWithContext(opts.ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, err
	}

	// 设置 header（覆盖默认或先前设置）
	for k, v := range opts.header {
		req.Header.Set(k, v)
	}
	return req, nil
}

// copyClient 复制HTTP客户端（浅拷贝 transport 等）
func copyClient(client *http.Client) *http.Client {
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Transport:     transport,
		CheckRedirect: client.CheckRedirect,
		Jar:           client.Jar,
		Timeout:       client.Timeout,
	}
}

// cloneHeader 复制header，防止header被修改
func cloneHeader(header map[string]string) map[string]string {
	cpHeader := make(map[string]string, len(header))
	for k, v := range header {
		cpHeader[k] = v
	}
	return cpHeader
}

// handleResponse 读取并封装响应
func handleResponse(resp *http.Response) (*Response, error) {
	// 读取 body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// 读取失败，确保关闭原始 body
		_ = resp.Body.Close()
		return nil, fmt.Errorf("read response body failed: %w", err)
	}
	// 关闭原始响应体，释放底层连接
	_ = resp.Body.Close()

	// 把 resp.Body 重置为可读的 reader，这样 RawResponse() 也能被读取（读取的是缓存的 body）
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return &Response{
		response:   resp,
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
	}, nil
}

func mergeHeader(defaults, custom map[string]string) map[string]string {
	if defaults == nil {
		defaults = make(map[string]string)
	}
	for k, v := range custom {
		defaults[k] = v
	}
	return defaults
}

func sleepBackoff(ctx context.Context, i int, base time.Duration) {
	backoff := base * (1 << i)
	if backoff <= 0 {
		backoff = 100 * time.Millisecond
	}
	maxBackoff := time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	jitter := time.Duration(common.WeakRandInt63n(int64(backoff)))

	timer := time.NewTimer(jitter)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}

func Request(method, rawURL string, opts ...Option) (*Response, error) {
	reqOpts := &requestOptions{
		client:     defaultClient,
		ctx:        context.Background(),
		header:     cloneHeader(defaultHeader),
		maxRetries: defaultMaxRetries,
	}

	// 加载配置
	for _, opt := range opts {
		opt(reqOpts)
	}

	if reqOpts.maxRetries < 0 {
		reqOpts.maxRetries = 0
	}

	// 准备 body: JSON > formData > body
	if err := prepareRequestBody(reqOpts); err != nil {
		return nil, err
	}

	// 构建 URL（包含 query 参数）
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse url failed: %w", err)
	}

	// 合并 query
	if len(reqOpts.query) > 0 {
		q := parsedURL.Query()
		for k, v := range reqOpts.query {
			q.Set(k, v)
		}
		parsedURL.RawQuery = q.Encode()
	}

	// 可能覆盖超时
	client := reqOpts.client
	if reqOpts.timeout > 0 {
		client = copyClient(reqOpts.client)
		client.Timeout = reqOpts.timeout
	}

	var lastErr error
	for i := 0; i <= reqOpts.maxRetries; i++ {
		select {
		case <-reqOpts.ctx.Done():
			return nil, reqOpts.ctx.Err()
		default:
		}
		httpReq, err := createHTTPRequest(method, parsedURL.String(), reqOpts)
		if err != nil {
			return nil, fmt.Errorf("create http request failed: %w", err)
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = err
			if i < reqOpts.maxRetries {
				sleepBackoff(reqOpts.ctx, i, 100*time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("http request failed: %w", lastErr)
		}
		response, err := handleResponse(resp)
		if err != nil {
			lastErr = err
			// 如果还有重试机会，继续重试；否则返回错误
			if i < reqOpts.maxRetries {
				sleepBackoff(reqOpts.ctx, i, 100*time.Millisecond)
				continue
			}
			return nil, err
		}
		return response, nil
	}
	return nil, lastErr
}

func (res *Response) Close() error {
	if res.response != nil && res.response.Body != nil {
		return res.response.Body.Close()
	}
	return nil
}

func (res *Response) GetHeader(key string) string {
	if res.response == nil {
		return ""
	}
	return res.response.Header.Get(key)
}

func (res *Response) Headers() http.Header {
	if res.response == nil {
		return nil
	}
	return res.response.Header.Clone()
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
func (res *Response) Text() string {
	return string(res.Body)
}

func (res *Response) RawResponse() *http.Response {
	return res.response
}

// Get 发起GET请求
func Get(url string, opts ...Option) (*Response, error) {
	return Request("GET", url, opts...)
}

// Post 发起POST请求
func Post(url string, opts ...Option) (*Response, error) {
	return Request("POST", url, opts...)
}

// Delete 发起DELETE请求
func Delete(url string, opts ...Option) (*Response, error) {
	return Request("DELETE", url, opts...)
}

// Put 发起PUT请求
func Put(url string, opts ...Option) (*Response, error) {
	return Request("PUT", url, opts...)
}

// WithClient 设置HTTP客户端
func WithClient(client *http.Client) Option {
	return func(r *requestOptions) {
		r.client = client
	}
}

// WithCtx 设置请求上下文
func WithCtx(ctx context.Context) Option {
	return func(r *requestOptions) {
		r.ctx = ctx
	}
}

// WithHeader 设置请求头
func WithHeader(header map[string]string) Option {
	return func(r *requestOptions) {
		r.header = mergeHeader(r.header, header)
	}
}

// WithBody 设置请求体内容
func WithBody(body []byte) Option {
	return func(r *requestOptions) {
		r.body = body
	}
}

// WithBodyString 封装WithBody
func WithBodyString(body string) Option {
	return func(o *requestOptions) { o.body = []byte(body) }
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(maxRetries int) Option {
	return func(r *requestOptions) {
		r.maxRetries = maxRetries
	}
}

// WithRequest 设置自定义请求
func WithRequest(req *http.Request) Option {
	return func(r *requestOptions) {
		r.request = req
	}
}

// WithQuery 设置query
func WithQuery(params map[string]string) Option {
	return func(r *requestOptions) {
		r.query = params
	}
}

// WithFormData form data请求类型
func WithFormData(formData map[string]string) Option {
	return func(o *requestOptions) { o.formData = formData }
}

// WithJSON json请求类型
func WithJSON(jsonBody interface{}) Option {
	return func(o *requestOptions) { o.json = jsonBody }
}

// WithTimeout 设置单接口超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(o *requestOptions) { o.timeout = timeout }
}
