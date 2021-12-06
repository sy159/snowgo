package requests

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	defaultClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:   true,
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 4,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     60 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}
	defaultHeader      = map[string]string{"user-agent": "snowgo Request"}
	defaultMaxRetryNum = 0
)

type Option func(*request)

type request struct {
	Client      *http.Client
	Ctx         context.Context
	Body        string
	Header      map[string]string
	MaxRetryNum int // 最大重试次数
}

type response struct {
	Request  *http.Request
	Response *http.Response
	Json     string
	Text     string
}

func Request(method, url, body string, opts ...Option) (res *response, err error) {
	req := &request{
		Client:      defaultClient,
		Ctx:         context.Background(),
		Header:      defaultHeader,
		MaxRetryNum: defaultMaxRetryNum,
	}
	for _, opt := range opts {
		opt(req)
	}
	newRequest, err := http.NewRequestWithContext(req.Ctx, method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 处理header参数
	for key, value := range req.Header {
		newRequest.Header.Set(key, value)
	}

	for i := 0; i <= req.MaxRetryNum; i++ {
		resp, err := req.Client.Do(newRequest)
		if err == nil {
			return &response{newRequest, resp, "test", "test"}, nil
		}
		if i == req.MaxRetryNum {
			return nil, errors.New("Maximum number of retries reached ")
		}
	}
	return nil, err
}

// Get get请求
func Get(url string, opts ...Option) (res *response, err error) {
	return Request("GET", url, "", opts...)
}

// Post post请求
func Post(url, body string, opts ...Option) (res *response, err error) {
	return Request("POST", url, body, opts...)
}

// Delete delete请求
func Delete(url, body string, opts ...Option) (res *response, err error) {
	return Request("DELETE", url, body, opts...)
}

// Put put请求
func Put(url, body string, opts ...Option) (res *response, err error) {
	return Request("PUT", url, body, opts...)
}

// WithClient 设置连接池，默认keepalive is false
func WithClient(client *http.Client) Option {
	return func(r *request) {
		r.Client = client
	}
}

// WithCtx 设置ctx
func WithCtx(ctx context.Context) Option {
	return func(r *request) {
		r.Ctx = ctx
	}
}

// WithHeader 设置header
func WithHeader(header map[string]string) Option {
	return func(r *request) {
		r.Header = header
	}
}

// WithMaxRetryNum 设置自大重试次数，默认为0
func WithMaxRetryNum(maxRetryNum int) Option {
	return func(r *request) {
		r.MaxRetryNum = maxRetryNum
	}
}
