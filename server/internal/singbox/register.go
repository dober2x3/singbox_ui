package singbox

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/version", h.GetVersion)
	rg.GET("/config", h.GetConfig)
	rg.POST("/config", h.SaveConfig)
	rg.POST("/run", h.RunSingbox)
	rg.POST("/stop", h.StopSingbox)
	rg.GET("/logs", h.GetSingboxLogs)
	rg.GET("/status", h.CheckSingboxStatus)
	rg.POST("/ensure-image", h.EnsureImage)
	rg.GET("/instances", h.ListNamedConfigs)
	rg.POST("/instances/:name/config", h.SaveNamedConfigWithContainer)
	rg.GET("/instances/:name/config", h.LoadNamedConfigFromContainer)
	rg.POST("/instances/:name/check", h.CheckNamedConfig)
	rg.DELETE("/instances/:name", h.DeleteNamedConfigWithContainer)
	rg.POST("/instances/:name/run", h.RunNamedContainer)
	rg.POST("/instances/:name/stop", h.StopNamedContainer)
	rg.GET("/instances/:name/status", h.GetNamedContainerStatus)
	rg.GET("/instances/:name/logs", h.GetNamedContainerLogs)
	rg.GET("/containers", h.ListAllContainers)
}
