package di

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"snowgo/internal/constants"
	"snowgo/internal/dal/repo"
	accountDao "snowgo/internal/dao/account"
	accountService "snowgo/internal/service/account"
	"snowgo/pkg/xcache"
	xmysql "snowgo/pkg/xdatabase/mysql"
	xredis "snowgo/pkg/xdatabase/redis"
	"snowgo/pkg/xlogger"
)

// Container 统一管理依赖
type Container struct {
	// 通用
	Cache xcache.Cache

	// 这里只提供对api使用的service，不提供dao操作
	UserService *accountService.UserService
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

// NewContainer 构造所有依赖，注意参数传递的顺序
func NewContainer() *Container {
	// 构造db、redis操作
	repository := BuildRepository(xmysql.DB, xmysql.DbMap)
	redisCache := BuildRedisCache(xredis.RDB)

	// 构造Dao
	userDao := accountDao.NewUserDao(repository)

	// 构造Service依赖
	userService := accountService.NewUserService(repository, userDao, redisCache)

	return &Container{
		Cache:       redisCache,
		UserService: userService,
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
