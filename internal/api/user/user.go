package user

import "github.com/gin-gonic/gin"

// GetUserInfo 用户信息
func GetUserInfo(c *gin.Context) {
	c.JSON(200, gin.H{"msg": "用户信息获取成功", "status": 200, "username": "test"})
}
