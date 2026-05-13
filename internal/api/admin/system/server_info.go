package system

import (
	"github.com/gin-gonic/gin"
	"snowgo/pkg/xresponse"

	"snowgo/internal/service/admin/system"
)

// GetServerInfo 服务信息
func GetServerInfo(c *gin.Context) {
	overview := system.GetServerOverview()
	xresponse.Success(c, overview)
}
