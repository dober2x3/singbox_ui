package clashapi

import (
	"github.com/gin-gonic/gin"
)

func RegisterTrafficRoutes(rg *gin.RouterGroup, pm *PortManager) {
	rg.GET("/traffic", func(c *gin.Context) {
		name := c.Param("name")
		port, ok := pm.Get(name)
		if !ok {
			c.JSON(404, gin.H{"error": "instance not found: " + name})
			return
		}
		ProxyWSBinary(c, port, "/traffic")
	})

	rg.GET("/memory", func(c *gin.Context) {
		name := c.Param("name")
		port, ok := pm.Get(name)
		if !ok {
			c.JSON(404, gin.H{"error": "instance not found: " + name})
			return
		}
		ProxyWS(c, port, "/memory")
	})
}
