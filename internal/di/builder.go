package di

import (
	"snowgo/config"
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
