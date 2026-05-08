package xgin

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParsePathID64 从 URL path 参数中解析 int64 类型的 ID，非法值返回 0。
func ParsePathID64(c *gin.Context) int64 {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// ParsePathID32 从 URL path 参数中解析 int32 类型的 ID，非法值或超出范围返回 0。
func ParsePathID32(c *gin.Context) int32 {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		return 0
	}
	return int32(id)
}
