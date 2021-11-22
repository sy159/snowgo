package user

import (
	"snowgo/utils/response"

	"github.com/gin-gonic/gin"
)

// GetUserInfo 用户信息
func GetUserInfo(c *gin.Context) {
	response.Success(c, gin.H{"username": "test", "age": 18})
}
