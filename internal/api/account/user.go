package account

import (
	"snowgo/utils/response"

	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	// todo 登录处理，返回token等操作
}

// GetUserInfo 用户信息
func GetUserInfo(c *gin.Context) {
	response.Success(c, gin.H{"username": "test", "age": 18})
}
