package clashapi

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterProxiesRoutes(rg *gin.RouterGroup, pm *PortManager) {
	rg.GET("/proxies", func(c *gin.Context) {
		port, client := getClient(c, pm)
		if port == 0 {
			return
		}
		resp, err := client.GetProxies()
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, resp)
	})

	rg.GET("/proxies/:group", func(c *gin.Context) {
		port, client := getClient(c, pm)
		if port == 0 {
			return
		}
		resp, err := client.GetProxy(c.Param("group"))
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, resp)
	})

	rg.PUT("/proxies/:group", func(c *gin.Context) {
		port, client := getClient(c, pm)
		if port == 0 {
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "missing 'name' field"})
			return
		}
		if err := client.SwitchProxy(c.Param("group"), body.Name); err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	})

	rg.GET("/proxies/:group/delay", func(c *gin.Context) {
		port, client := getClient(c, pm)
		if port == 0 {
			return
		}
		url := c.DefaultQuery("url", "https://www.gstatic.com/generate_204")
		timeout, _ := strconv.Atoi(c.DefaultQuery("timeout", "5000"))
		delay, err := client.GetProxyDelay(c.Param("group"), url, timeout)
		if err != nil {
			c.JSON(502, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"delay": delay})
	})
}

func getClient(c *gin.Context, pm *PortManager) (int, *Client) {
	name := c.Param("name")
	port, ok := pm.Get(name)
	if !ok {
		c.JSON(404, gin.H{"error": "instance not found: " + name})
		return 0, nil
	}
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	return port, NewClient(baseURL, "")
}
