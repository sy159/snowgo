package requests

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
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
	}
	req.request, err = http.NewRequestWithContext(req.ctx, method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(req)
	}

	// 处理header参数
	for key, value := range req.header {
		req.request.Header.Set(key, value)
	}

	// 进行重试
	for i := 0; i <= req.maxRetryNum; i++ {
		resp, err := req.client.Do(req.request)
		if err == nil {
			return &response{req.request, resp}, nil
		}
		if i == req.maxRetryNum {
			return nil, errors.New("Maximum number of retries reached ")
		}
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
	body, err := ioutil.ReadAll(res.Response.Body)
	if err != nil {
		return resMap, err
	}
	err = json.Unmarshal(body, &resMap)
	return resMap, err
}

// Text 返回body str内容，只能读取一次
func (res *response) Text() (string, error) {
	defer res.Response.Body.Close()
	body, err := ioutil.ReadAll(res.Response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
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
		r.client = client
	}
}

// WithCtx 设置ctx
func WithCtx(ctx context.Context) Option {
	return func(r *request) {
		r.ctx = ctx
	}
}

// WithHeader 设置header
func WithHeader(header map[string]string) Option {
	return func(r *request) {
		r.header = header
	}
}

// WithMaxRetryNum 设置自大重试次数，默认为0
func WithMaxRetryNum(maxRetryNum int) Option {
	return func(r *request) {
		r.maxRetryNum = maxRetryNum
	}
}

// WithRequest 设置请求参数，默认为NewRequestWithContext
func WithRequest(req *http.Request) Option {
	return func(r *request) {
		r.request = req
	}
}
