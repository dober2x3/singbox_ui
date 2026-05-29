package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"singbox-config-service/handlers"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

func main() {
	// initialize sing-box environment
	if err := Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize sing-box environment: %v", err)
		log.Println("Some features may not work properly")
	}

	// set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// create Gin router
	r := gin.Default()

	// add CORS middleware
	r.Use(corsMiddleware())

	// API route group - provides tool APIs only
	api := r.Group("/api")
	{
		// WireGuard key generation tool and multi-client management
		wg := api.Group("/wireguard")
		{
			// key generation
			wg.POST("/keygen", handlers.GenerateWireGuardKeys)
			wg.POST("/pubkey", handlers.GetPublicKeyFromPrivate)
			wg.GET("/keys-cache", handlers.GetKeysCache)
			wg.GET("/public-ip", handlers.GetPublicIP)

			// client config
			wg.GET("/client-config", handlers.GetClientConfig)
			wg.POST("/client-config", handlers.SaveClientConfig)
			wg.POST("/save-client-file", handlers.SaveClientConfigFile)
			wg.GET("/client-files", handlers.ListClientConfigFiles)
		}

		// sing-box management tool (Docker container mode)
		singbox := api.Group("/singbox")
		{
			singbox.GET("/version", handlers.GetSingboxVersion)
			singbox.GET("/config", handlers.GetConfig)
			singbox.POST("/config", handlers.SaveConfig)
			singbox.POST("/run", handlers.RunSingbox)
			singbox.POST("/stop", handlers.StopSingbox)
			singbox.GET("/logs", handlers.GetSingboxLogs)
			singbox.GET("/status", handlers.CheckSingboxStatus)
			singbox.POST("/ensure-image", handlers.EnsureImage)

			// certificate management
			singbox.POST("/certificate", handlers.GenerateSelfSignedCert)
			singbox.GET("/certificate", handlers.GetCertificateInfo)
			singbox.POST("/certificate/upload", handlers.UploadCertificate)
			singbox.POST("/reality/keypair", handlers.GenerateRealityKeypair)
			singbox.POST("/reality/public-key", handlers.DeriveRealityPublicKey)
			singbox.POST("/reality/check-tls", handlers.CheckTLS13Support)

			// multi-config multi-container management
			singbox.GET("/instances", handlers.ListNamedConfigs)
			singbox.POST("/instances/:name/config", handlers.SaveNamedConfigWithContainer)
			singbox.GET("/instances/:name/config", handlers.LoadNamedConfigFromContainer)
			singbox.POST("/instances/:name/check", handlers.CheckNamedConfig)
			singbox.DELETE("/instances/:name", handlers.DeleteNamedConfigWithContainer)
			singbox.POST("/instances/:name/run", handlers.RunNamedContainer)
			singbox.POST("/instances/:name/stop", handlers.StopNamedContainer)
			singbox.GET("/instances/:name/status", handlers.GetNamedContainerStatus)
			singbox.GET("/instances/:name/logs", handlers.GetNamedContainerLogs)
			singbox.GET("/containers", handlers.ListAllContainers)
		}

		// subscription management
		sub := api.Group("/subscription")
		{
		sub.GET("", handlers.GetSubscriptions)          // get all subscriptions
		sub.POST("", handlers.AddSubscription)          // add subscription
		sub.POST("/:id/refresh", handlers.RefreshSubscription)         // refresh single subscription
		sub.PATCH("/:id/settings", handlers.UpdateSubscriptionSettings) // update auto-update settings
		sub.DELETE("/:id", handlers.DeleteSubscription)                 // delete subscription
		sub.POST("/refresh-all", handlers.RefreshAllSubscriptions) // refresh all subscriptions
		sub.GET("/nodes", handlers.GetAllNodes)         // get all nodes
		sub.GET("/user-agents", handlers.GetUserAgents) // get predefined UA list
		}

		// node prober
		prober := api.Group("/prober")
		{
			prober.GET("/status", handlers.GetProberStatus)
			prober.GET("/results", handlers.GetProbeResults)
			prober.GET("/results/:tag", handlers.GetProbeResult)
			prober.GET("/best", handlers.GetBestNode)
			prober.GET("/online", handlers.GetOnlineNodes)
			prober.POST("/nodes", handlers.AddProberNode)
			prober.PUT("/nodes", handlers.UpdateProberNodes)
			prober.DELETE("/nodes/:tag", handlers.RemoveProberNode)
			prober.DELETE("/nodes", handlers.ClearProberNodes)
			prober.POST("/start", handlers.StartProber)
			prober.POST("/stop", handlers.StopProber)
			prober.POST("/sync", handlers.SyncNodesFromSubscription)
			prober.POST("/save", handlers.SaveProbeResultsToSubscription)
		}

		// proxy speed test: start temporary sing-box instance to test nodes via SOCKS/HTTP proxy
		speedtest := api.Group("/speedtest")
		{
			speedtest.POST("/start", handlers.StartSpeedTest)
			speedtest.GET("/status", handlers.GetSpeedTestStatus)
			speedtest.POST("/stop", handlers.StopSpeedTest)
		}

		// Cloudflare WARP: auto-register, WARP+ license binding, endpoint scan
		warp := api.Group("/warp")
		{
			warp.GET("/account", handlers.GetWarpAccount)
			warp.DELETE("/account", handlers.DeleteWarpAccount)
			warp.POST("/register", handlers.RegisterWarp)
			warp.POST("/license", handlers.BindWarpLicense)
			warp.POST("/scan", handlers.ScanWarpEndpoints)
		}

	}

	// health check
	r.GET("/health", handlers.HealthCheck)

	// static file server - serve frontend pages
	setupStaticFiles(r)

	// start server (configurable via LISTEN_ADDR env var)
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "127.0.0.1:7000"
	}
	log.Printf("Sing-box Config Service is starting on %s", listenAddr)
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupStaticFiles configure static file serving
func setupStaticFiles(r *gin.Engine) {
	// get embedded file system
	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Printf("Warning: Failed to load embedded frontend files: %v", err)
		return
	}

	// create HTTP file server
	fileServer := http.FileServer(http.FS(staticFS))

	// serve static files
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// if path starts with /api, it's an API request but the route doesn't exist
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

		// try to serve static file
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	log.Println("Static frontend files loaded from embedded dist directory")
}

// corsMiddleware CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
