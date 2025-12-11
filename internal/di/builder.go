package di

import (
	"context"
	"errors"
	"fmt"
	"snowgo/config"
	"snowgo/pkg/xmq/rabbitmq"
	"sync"
	"time"
)

type Option func(*containerOptions)

type containerOptions struct {
	jwtCfg       *config.JwtConfig
	mysqlCfg     *config.MysqlConfig
	otherDBCfg   *config.OtherDBConfig
	redisCfg     *config.RedisConfig
	producerCfg  *rabbitmq.ProducerConnConfig
	closeTimeout time.Duration
}

func defaultOpts() *containerOptions {
	return &containerOptions{
		closeTimeout: 10 * time.Second,
	}
}

func WithJWT(c config.JwtConfig) Option {
	return func(o *containerOptions) { o.jwtCfg = &c }
}

func WithMySQL(mysqlCfg config.MysqlConfig, otherCfg config.OtherDBConfig) Option {
	return func(o *containerOptions) {
		o.mysqlCfg = &mysqlCfg
		o.otherDBCfg = &otherCfg
	}
}

func WithRedis(redisCfg config.RedisConfig) Option {
	return func(o *containerOptions) { o.redisCfg = &redisCfg }
}

func WithProducer(cfg *rabbitmq.ProducerConnConfig) Option {
	return func(o *containerOptions) { o.producerCfg = cfg }
}

func WithCloseTimeout(d time.Duration) Option {
	return func(o *containerOptions) { o.closeTimeout = d }
}

// ContainerCloser 接口
type ContainerCloser interface {
	Close() error
}

// ContainerCloserCtx 接口
type ContainerCloserCtx interface {
	Close(ctx context.Context) error
}

// CloseManager 管理所有 closable 资源（兼容带 ctx / 不带 ctx）
type CloseManager struct {
	mu      sync.Mutex
	closers []func(ctx context.Context) error // 统一存储成接受 ctx 的函数
}

func NewCloseManager() *CloseManager {
	return &CloseManager{
		closers: make([]func(ctx context.Context) error, 0, 4),
	}
}

// RegisterFunc 注册退出函数
func (m *CloseManager) RegisterFunc(f func(ctx context.Context) error) {
	if f == nil {
		return
	}
	m.mu.Lock()
	m.closers = append(m.closers, f)
	m.mu.Unlock()
}

// RegisterCtx 注册实现 Close(ctx context.Context) error 的资源（推荐）
func (m *CloseManager) RegisterCtx(c ContainerCloserCtx) {
	if c == nil {
		return
	}
	m.mu.Lock()
	m.closers = append(m.closers, func(ctx context.Context) error {
		return c.Close(ctx)
	})
	m.mu.Unlock()
}

// Register 注册实现 Close() error 的资源
func (m *CloseManager) Register(c ContainerCloser) {
	if c == nil {
		return
	}
	m.RegisterFunc(func(ctx context.Context) error {
		done := make(chan error, 1)
		go func() {
			done <- c.Close()
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		}
	})
}

// CloseAll 按 LIFO 顺序关闭所有注册资源
func (m *CloseManager) CloseAll(ctx context.Context) error {
	m.mu.Lock()
	closers := make([]func(ctx context.Context) error, len(m.closers))
	copy(closers, m.closers)
	m.mu.Unlock()

	// 按 LIFO 顺序关闭（后注册先关闭）
	var errs []error
	for i := len(closers) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			errs = append(errs, ctx.Err())
			break
		}
		f := closers[i]
		if f == nil {
			continue
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("panic during close: %v", r)
					errs = append(errs, err)
				}
			}()
			if err := f(ctx); err != nil {
				errs = append(errs, err)
			}
		}()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
