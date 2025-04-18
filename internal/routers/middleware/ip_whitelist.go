package middleware

import (
	"net"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xresponse"

	"github.com/gin-gonic/gin"
)

// IPWhiteList 返回一个中间件，仅允许白名单内 IP 访问
func IPWhiteList(whiteList []string) gin.HandlerFunc {
	var cidrs []*net.IPNet
	for _, cidr := range whiteList {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			cidrs = append(cidrs, network)
		}
	}
	return func(c *gin.Context) {
		ip := net.ParseIP(c.ClientIP())
		for _, network := range cidrs {
			if network.Contains(ip) {
				c.Next()
				return
			}
		}
		xresponse.FailByError(c, e.HttpForbidden)
		c.Abort()
	}
}
