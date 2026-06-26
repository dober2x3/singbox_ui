package clashapi

import "github.com/gin-gonic/gin"

func RegisterAllRoutes(rg *gin.RouterGroup, pm *PortManager) {
	RegisterProxiesRoutes(rg, pm)
	RegisterTrafficRoutes(rg, pm)
	RegisterConnectionsRoutes(rg, pm)
	RegisterLogsRoutes(rg, pm)
	RegisterRulesRoutes(rg, pm)
	RegisterModeRoutes(rg, pm)
}
