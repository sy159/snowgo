package xrequests

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}
	defaultHeader = map[string]string{
		"user-agent":   "snowgo Request",
		"Content-Type": "application/json; charset=UTF-8",
	}
	defaultMaxRetryNum = 0
)

type Option func(*request)

type request struct {
	client      *http.Client
	ctx         context.Context
	body        string
	header      map[string]string
	maxRetryNum int // 最大重试次数
	request     *http.Request
}

type response struct {
	Request  *http.Request
	Response *http.Response
}

func Request(method, url, body string, opts ...Option) (res *response, err error) {
	req := &request{
		client:      defaultClient,
		ctx:         context.Background(),
		header:      defaultHeader,
		maxRetryNum: defaultMaxRetryNum,
		body:        body,
	}

	// 加载配置
	for _, opt := range opts {
		opt(req)
	}

	req.request, err = http.NewRequestWithContext(req.ctx, method, url, bytes.NewBufferString(req.body))
	if err != nil {
		return nil, err
	}

	// 处理header参数
	for key, value := range req.header {
		req.request.Header.Set(key, value)
	}

	// 进行重试
	for i := 0; i <= req.maxRetryNum; i++ {
		resp, doErr := req.client.Do(req.request)
		fmt.Println(doErr)
		if doErr == nil {
			return &response{req.request, resp}, nil
		}
		if i == req.maxRetryNum && req.maxRetryNum != 0 {
			return nil, errors.New("maximum number of retries reached")
		}
		err = doErr
	}
	return nil, err
}

// Json 返回body json内容，只能读取一次
func (res *response) Json(v interface{}) error {
	defer res.Response.Body.Close()
	return json.NewDecoder(res.Response.Body).Decode(v)
}

// Map 返回body map内容，只能读取一次
func (res *response) Map() (map[string]interface{}, error) {
	defer res.Response.Body.Close()
	var resMap map[string]interface{}
	body, err := io.ReadAll(res.Response.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &resMap)
	return resMap, err
}

// Text 返回body str内容，只能读取一次
func (res *response) Text() (string, error) {
	defer res.Response.Body.Close()
	body, err := io.ReadAll(res.Response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Get 发起GET请求
func Get(url string, opts ...Option) (*response, error) {
	return Request("GET", url, "", opts...)
}

// Post 发起POST请求
func Post(url, body string, opts ...Option) (*response, error) {
	return Request("POST", url, body, opts...)
}

// Delete 发起DELETE请求
func Delete(url, body string, opts ...Option) (*response, error) {
	return Request("DELETE", url, body, opts...)
}

// Put 发起PUT请求
func Put(url, body string, opts ...Option) (*response, error) {
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
		r.header = header
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
