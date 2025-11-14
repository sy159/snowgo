package di

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"snowgo/config"
	"snowgo/internal/constants"
	"snowgo/internal/dal/repo"
	accountDao "snowgo/internal/dao/account"
	"snowgo/internal/dao/log"
	accountService "snowgo/internal/service/account"
	logService "snowgo/internal/service/log"
	"snowgo/pkg/xauth/jwt"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xdatabase/mysql"
	xredis "snowgo/pkg/xdatabase/redis"
	"snowgo/pkg/xlock"
	"sync"
	"time"
)

// Container 统一管理依赖
type Container struct {
	// 通用
	db struct {
		MyDB *mysql.MyDB
		RDB  *redis.Client
	}
	Cache      xcache.Cache
	JwtManager *jwt.Manager
	Lock       xlock.Lock

	// 这里只提供对api使用的service，不提供dao操作
	AccountContainer
	SystemContainer

	// 关闭控制
	closeMgr     *CloseManager
	once         sync.Once
	closeTimeout time.Duration
	closed       bool
}

type AccountContainer struct {
	UserService *accountService.UserService
	MenuService *accountService.MenuService
	RoleService *accountService.RoleService
}

type SystemContainer struct {
	OperationLogService *logService.OperationLogService
}

// BuildJwtManager 构建jwt操作
func BuildJwtManager(config config.JwtConfig) (*jwt.Manager, error) {
	if len(config.JwtSecret) == 0 ||
		len(config.Issuer) == 0 ||
		config.AccessExpirationTime == 0 ||
		config.RefreshExpirationTime == 0 {
		return nil, errors.New("Please initialize jwt config first, jwt config is empty")
	}
	jwtManager := jwt.NewJwtManager(&jwt.Config{
		JwtSecret:             config.JwtSecret,
		Issuer:                config.Issuer,
		AccessExpirationTime:  config.AccessExpirationTime,
		RefreshExpirationTime: config.RefreshExpirationTime,
	})
	return jwtManager, nil
}

// BuildRepository 构建db操作
func BuildRepository(db *gorm.DB, dbMap map[string]*gorm.DB) (*repo.Repository, error) {
	if db == nil {
		return nil, errors.New("Please initialize mysql first")
	}
	return repo.NewRepository(db, dbMap), nil
}

// BuildRedisCache 构建缓存操作
func BuildRedisCache(rdb *redis.Client) (xcache.Cache, error) {
	if rdb == nil {
		return nil, errors.New("Please initialize redis first")
	}
	return xcache.NewRedisCache(rdb), nil
}

// BuildLock 构建锁
func BuildLock(rdb *redis.Client) (xlock.Lock, error) {
	if rdb == nil {
		return nil, errors.New("Please initialize redis first")
	}
	return xlock.NewRedisLock(rdb), nil
}

// NewContainer 构造所有依赖，注意参数传递的顺序
func NewContainer(opts ...Option) (container *Container, err error) {
	opt := defaultOpts()
	for _, fn := range opts {
		fn(opt)
	}

	// require mysql/redis per your requirement
	if opt.mysqlCfg == nil {
		return nil, errors.New("mysql config required")
	}
	if opt.redisCfg == nil {
		return nil, errors.New("redis config required")
	}

	container = &Container{
		closeMgr:     NewCloseManager(),
		closeTimeout: opt.closeTimeout,
	}
	defer func() {
		// 当init 失败，释放资源
		if err != nil && container != nil && container.closeMgr != nil {
			_ = container.closeMgr.CloseAll()
		}
	}()

	// mysql db
	myDB, err := mysql.NewMysql(*opt.mysqlCfg, *opt.otherDBCfg)
	if err != nil {
		return nil, errors.WithMessage(err, "mysql init err")
	}
	container.db.MyDB = myDB
	container.closeMgr.Register(myDB) // 自动注册关闭 清理资源

	// redis db
	rdb, err := xredis.NewRedis(*opt.redisCfg)
	if err != nil {
		return nil, errors.WithMessage(err, "redis init err")
	}
	container.db.RDB = rdb
	container.closeMgr.Register(rdb) // 自动注册关闭 清理资源

	if opt.jwtCfg != nil {
		jwtManager, err := BuildJwtManager(*opt.jwtCfg)
		if err != nil {
			return nil, errors.WithMessage(err, "jwt init err")
		}
		container.JwtManager = jwtManager
	}

	// 构造db、redis操作
	repository, err := BuildRepository(myDB.DB, myDB.DbMap)
	if err != nil {
		return nil, errors.WithMessage(err, "repo init err")
	}

	redisCache, err := BuildRedisCache(rdb)
	if err != nil {
		return nil, errors.WithMessage(err, "redis cache init err")
	}
	container.Cache = redisCache

	lock, err := BuildLock(rdb)
	if err != nil {
		return nil, errors.WithMessage(err, "lock init err")
	}
	container.Lock = lock

	// 构造Dao
	userDao := accountDao.NewUserDao(repository)
	menuDao := accountDao.NewMenuDao(repository)
	roleDao := accountDao.NewRoleDao(repository)
	operationLogDao := log.NewOperationLogDao(repository)

	// 构造Service依赖
	operationLogService := logService.NewOperationLogService(repository, operationLogDao)
	menuService := accountService.NewMenuService(repository, redisCache, menuDao, operationLogService)
	roleService := accountService.NewRoleService(repository, roleDao, redisCache, operationLogService)
	userService := accountService.NewUserService(repository, userDao, redisCache, roleService, operationLogService)

	// account
	container.AccountContainer = AccountContainer{
		UserService: userService,
		MenuService: menuService,
		RoleService: roleService,
	}
	// system
	container.SystemContainer = SystemContainer{
		OperationLogService: operationLogService,
	}
	return container, nil
}

func (c *Container) CloseWithContext(ctx context.Context) error {
	var retErr error
	c.once.Do(func() {
		// 使用 c.closeTimeout优先（若 ctx 没超时）
		timeout := c.closeTimeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// 在一个 goroutine 中运行 CloseAll，这样可以 select 超时
		done := make(chan struct{})
		var closeErr error
		go func() {
			if c.closeMgr != nil {
				closeErr = c.closeMgr.CloseAll()
			}
			close(done)
		}()

		select {
		case <-timeoutCtx.Done():
			retErr = timeoutCtx.Err()
		case <-done:
			retErr = closeErr
		}
		c.closed = true
	})
	return retErr
}

// Close 关闭
func (c *Container) Close() error {
	return c.CloseWithContext(context.Background())
}

// GetContainer 获取注入的cache、service等
func GetContainer(c *gin.Context) *Container {
	val, exists := c.Get(constants.CONTAINER)
	if !exists {
		panic("Container not found in context")
	}
	container, ok := val.(*Container)
	if !ok {
		panic("Invalid container type")
	}
	return container
}

// GetContainerSafe 未获取到不会报错，用于work使用
func GetContainerSafe(c *gin.Context) (*Container, bool) {
	val, exists := c.Get(constants.CONTAINER)
	if !exists {
		return nil, false
	}
	container, ok := val.(*Container)
	return container, ok
}

// GetMyDB 获取注入的mysql client
func (c *Container) GetMyDB() *mysql.MyDB {
	return c.db.MyDB
}

// GetRDB 获取注入的redis client
func (c *Container) GetRDB() *redis.Client {
	return c.db.RDB
}

// GetAccountContainer 获取注入的cache、service等
func GetAccountContainer(c *gin.Context) *AccountContainer {
	container := GetContainer(c)
	return &container.AccountContainer
}

// GetSystemContainer 获取注入的cache、service等
func GetSystemContainer(c *gin.Context) *SystemContainer {
	container := GetContainer(c)
	return &container.SystemContainer
}
