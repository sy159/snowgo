package di

import (
	"errors"
	"snowgo/config"
	"sync"
	"time"
)

type Option func(*containerOptions)

type containerOptions struct {
	jwtCfg       *config.JwtConfig
	mysqlCfg     *config.MysqlConfig
	otherDBCfg   *config.OtherDBConfig
	redisCfg     *config.RedisConfig
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

func WithCloseTimeout(d time.Duration) Option {
	return func(o *containerOptions) { o.closeTimeout = d }
}

// ContainerCloser 接口
type ContainerCloser interface {
	Close() error
}

// CloseManager 管理所有closable资源
type CloseManager struct {
	mu      sync.Mutex
	closers []ContainerCloser
}

func NewCloseManager() *CloseManager {
	return &CloseManager{}
}

func (m *CloseManager) Register(c ContainerCloser) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closers = append(m.closers, c)
}

func (m *CloseManager) CloseAll() error {
	m.mu.Lock()
	closers := make([]ContainerCloser, len(m.closers))
	copy(closers, m.closers)
	m.mu.Unlock()

	// 按 LIFO 顺序关闭（后注册先关闭）
	var errs []error
	for i := len(closers) - 1; i >= 0; i-- {
		c := closers[i]
		if c == nil {
			continue
		}
		if err := c.Close(); err != nil {
			// 记录日志并收集错误
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
