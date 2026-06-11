package clashapi

import (
	"github.com/gin-gonic/gin"
)

func RegisterConnectionsRoutes(rg *gin.RouterGroup, pm *PortManager) {
	rg.GET("/connections", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		resp, err := client.GetConnections()
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, resp)
	})

	rg.DELETE("/connections", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		if err := client.CloseAllConnections(); err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	})

	rg.DELETE("/connections/:id", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		if err := client.CloseConnection(c.Param("id")); err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	})
}
