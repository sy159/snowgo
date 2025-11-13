package di

import (
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
	"snowgo/pkg/xlogger"
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
func BuildJwtManager(config config.JwtConfig) *jwt.Manager {
	if len(config.JwtSecret) == 0 ||
		len(config.Issuer) == 0 ||
		config.AccessExpirationTime == 0 ||
		config.RefreshExpirationTime == 0 {
		xlogger.Panic("Please initialize jwt config first, jwt config is empty")
	}
	jwtManager := jwt.NewJwtManager(&jwt.Config{
		JwtSecret:             config.JwtSecret,
		Issuer:                config.Issuer,
		AccessExpirationTime:  config.AccessExpirationTime,
		RefreshExpirationTime: config.RefreshExpirationTime,
	})
	return jwtManager
}

// BuildRepository 构建db操作
func BuildRepository(db *gorm.DB, dbMap map[string]*gorm.DB) *repo.Repository {
	if db == nil || db.Error != nil {
		xlogger.Panic("Please initialize mysql first")
	}
	return repo.NewRepository(db, dbMap)
}

// BuildRedisCache 构建缓存操作
func BuildRedisCache(rdb *redis.Client) xcache.Cache {
	if rdb == nil {
		xlogger.Panic("Please initialize redis first, redis cache is empty")
	}
	return xcache.NewRedisCache(rdb)
}

// BuildLock 构建锁
func BuildLock(rdb *redis.Client) xlock.Lock {
	if rdb == nil {
		xlogger.Panic("Please initialize redis first, redis cache is empty")
	}
	return xlock.NewRedisLock(rdb)
}

// NewContainer 构造所有依赖，注意参数传递的顺序
func NewContainer(jwtConfig config.JwtConfig, mysqlConfig config.MysqlConfig, otherDBConfig config.OtherDBConfig,
	redisConfig config.RedisConfig) (*Container, error) {
	jwtManager := BuildJwtManager(jwtConfig)
	myDB, err := mysql.NewMysql(mysqlConfig, otherDBConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "mysql init err")
	}
	rdb, err := xredis.NewRedis(redisConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "redis init err")
	}

	lock := BuildLock(rdb)

	// 构造db、redis操作
	repository := BuildRepository(myDB.DB, myDB.DbMap)
	redisCache := BuildRedisCache(rdb)

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

	return &Container{
		db: struct {
			MyDB *mysql.MyDB
			RDB  *redis.Client
		}{
			MyDB: myDB,
			RDB:  rdb,
		},
		Cache:      redisCache,
		JwtManager: jwtManager,
		Lock:       lock,
		AccountContainer: AccountContainer{
			UserService: userService,
			MenuService: menuService,
			RoleService: roleService,
		},
		SystemContainer: SystemContainer{
			OperationLogService: operationLogService,
		},
	}, nil
}

// GetContainer 获取注入的cache、service等
func GetContainer(c *gin.Context) *Container {
	val, exists := c.Get(constants.CONTAINER)
	if !exists {
		xlogger.Panic("Container not found in context")
	}
	container, ok := val.(*Container)
	if !ok {
		xlogger.Panic("Invalid container type")
	}
	return container
}

func (c *Container) Close() {
	if c.db.RDB != nil {
		_ = c.db.RDB.Close()
	}
	if c.db.MyDB != nil {
		c.db.MyDB.CloseAllMysql()
	}
	// 如果有其他需要关闭的资源，逐一关闭
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
