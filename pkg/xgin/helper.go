package xgin

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePathID 从 URL path 参数中解析 int64 类型的 ID，非法值返回 0。
func ParsePathID(c *gin.Context) int64 {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return id
}
