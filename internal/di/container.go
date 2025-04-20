package di

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"snowgo/config"
	"snowgo/internal/constants"
	"snowgo/internal/dal/repo"
	accountDao "snowgo/internal/dao/account"
	accountService "snowgo/internal/service/account"
	"snowgo/pkg/xauth/jwt"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xlock"
	"snowgo/pkg/xlogger"
)

// Container 统一管理依赖
type Container struct {
	// 通用
	Cache      xcache.Cache
	JwtManager *jwt.Manager
	Lock       xlock.Lock

	// 这里只提供对api使用的service，不提供dao操作
	AccountContainer
}

type AccountContainer struct {
	UserService *accountService.UserService
	MenuService *accountService.MenuService
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
	if db.Error != nil {
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
func NewContainer(jwtConfig config.JwtConfig, rdb *redis.Client, db *gorm.DB, dbMap map[string]*gorm.DB) *Container {
	jwtManager := BuildJwtManager(jwtConfig)
	lock := BuildLock(rdb)

	// 构造db、redis操作
	repository := BuildRepository(db, dbMap)
	redisCache := BuildRedisCache(rdb)

	// 构造Dao
	userDao := accountDao.NewUserDao(repository)
	menuDao := accountDao.NewMenuDao(repository)

	// 构造Service依赖
	userService := accountService.NewUserService(repository, userDao, redisCache)
	menuService := accountService.NewMenuService(repository, redisCache, menuDao)

	return &Container{
		Cache:      redisCache,
		JwtManager: jwtManager,
		Lock:       lock,
		AccountContainer: AccountContainer{
			UserService: userService,
			MenuService: menuService,
		},
	}
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

// GetAccountContainer 获取注入的cache、service等
func GetAccountContainer(c *gin.Context) *AccountContainer {
	container := GetContainer(c)
	return &container.AccountContainer
}
