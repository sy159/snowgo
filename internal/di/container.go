package di

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/constants"
	"snowgo/internal/dal/repo"
	accountDao "snowgo/internal/dao/account"
	accountService "snowgo/internal/service/account"
	"snowgo/utils/cache"
	"snowgo/utils/database/redis"
	"snowgo/utils/logger"
)

// Container 统一管理依赖
type Container struct {
	// 通用
	Cache cache.Cache

	// 这里只提供对api使用的service，不提供dao操作
	UserService *accountService.UserService
}

// BuildRepository 构建db操作
func BuildRepository() *repo.Repository {
	return repo.NewRepository()
}

// BuildRedisCache 构建缓存操作
func BuildRedisCache() cache.Cache {
	if redis.RDB == nil {
		logger.Panic("Please initialize redis first, redis cache is empty")
	}
	return cache.NewRedisCache(redis.RDB)
}

// NewContainer 构造所有依赖，注意参数传递的顺序
func NewContainer() *Container {
	// 构造db、redis操作
	repository := BuildRepository()
	redisCache := BuildRedisCache()

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
		logger.Panic("Container not found in context")
	}
	container, ok := val.(*Container)
	if !ok {
		logger.Panic("Invalid container type")
	}
	return container
}
