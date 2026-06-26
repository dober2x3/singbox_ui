package clashapi

import (
	"github.com/gin-gonic/gin"
)

func RegisterLogsRoutes(rg *gin.RouterGroup, pm *PortManager) {
	rg.GET("/logs", func(c *gin.Context) {
		name := c.Param("name")
		port, ok := pm.Get(name)
		if !ok {
			c.JSON(404, gin.H{"error": "instance not found: " + name})
			return
		}
		ProxyWS(c, port, "/logs")
	})

	rg.GET("/logs/level", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		level, err := client.GetLogLevel()
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"level": level})
	})

	rg.PUT("/logs/level", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		var body struct {
			Level string `json:"level"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "missing 'level' field"})
			return
		}
		if err := client.SetLogLevel(body.Level); err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	})
}
