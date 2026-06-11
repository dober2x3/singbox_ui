package clashapi

import (
	"github.com/gin-gonic/gin"
)

func RegisterRulesRoutes(rg *gin.RouterGroup, pm *PortManager) {
	rg.GET("/rules", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		rules, err := client.GetRules()
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, rules)
	})
}

func RegisterModeRoutes(rg *gin.RouterGroup, pm *PortManager) {
	rg.GET("/mode", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		mode, err := client.GetMode()
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"mode": mode})
	})

	rg.PUT("/mode", func(c *gin.Context) {
		_, client := getClient(c, pm)
		if client == nil {
			return
		}
		var body struct {
			Mode string `json:"mode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "missing 'mode' field"})
			return
		}
		if err := client.SetMode(body.Mode); err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	})
}
