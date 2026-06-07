// Package main is the entry point for the sing-box config service HTTP server.
// It initializes all domain services, sets up Gin routes, and starts the HTTP listener.
//
// @title           Singbox UI API
// @version         1.0
// @description     API for managing sing-box, WireGuard, subscriptions, probing, and WARP
// @host            localhost:8080
// @BasePath        /api
// @schemes         http
//
// @tag.name        subscription
// @tag.description Subscription management
// @tag.name        singbox
// @tag.description sing-box container lifecycle and config management
// @tag.name        wireguard
// @tag.description WireGuard key generation and client configuration
// @tag.name        certificate
// @tag.description Self-signed certificate management
// @tag.name        reality
// @tag.description REALITY keypair generation and TLS checks
// @tag.name        prober
// @tag.description Node health probing and latency measurement
// @tag.name        speedtest
// @tag.description Proxy speed testing
// @tag.name        warp
// @tag.description WARP registration, licensing, and endpoint scanning
// @tag.name        system
// @tag.description System health and status endpoints
package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/tunnelrunner"
	"singbox-config-service/internal/certificate"
	"singbox-config-service/internal/prober"
	"singbox-config-service/internal/scheduler"
	"singbox-config-service/internal/singbox"
	"singbox-config-service/internal/speedtest"
	"singbox-config-service/internal/subscription"
	"singbox-config-service/internal/warp"
	"singbox-config-service/internal/wireguard"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "singbox-config-service/internal/docs"
)

// distFS embeds the compiled frontend dist directory for static file serving.
//
//go:embed dist/*
var distFS embed.FS

// main is the application entry point. It initializes configuration, domain services,
// HTTP handlers, the background task scheduler, and finally starts the Gin HTTP server.
func main() {
	// Parse CLI flags
	serveDashboard := flag.Bool("dashboard", false, "Serve embedded frontend dashboard")
	singboxBin := flag.String("singbox-bin", "", "Path to sing-box binary (native/OpenWrt mode)")
	flag.Parse()

	// Initialize config
	cfg, err := config.Init()
	cfg.SetSingboxBinPath(*singboxBin)
	if err != nil {
		log.Printf("Warning: Failed to initialize config: %v", err)
		log.Println("Some features may not work properly")
	}

	// Create domain services
	rt, err := singbox.NewRuntime(cfg)
	if err != nil {
		log.Printf("Warning: Failed to create sing-box runtime: %v", err)
		log.Println("sing-box features will not be available")
		rt = &singbox.NoopRuntime{}
	}
	singboxSvc := singbox.NewService(rt, cfg)
	wgSvc := wireguard.NewService(cfg.GetDataDir())
	warpSvc := warp.NewService(cfg.GetDataDir())
	certSvc := certificate.NewService(cfg.GetSingboxDir())
	subStore := subscription.NewFileStore(cfg.GetDataDir())
	subSvc := subscription.NewService(subStore)
	proberSvc := prober.NewService(cfg.GetDataDir(), subSvc)
	speedtestSvc := speedtest.NewService(tunnelrunner.NewRunner(cfg), cfg)

	// Create auto-update scheduler
	sched := scheduler.New(subSvc, nil)

	// Create HTTP handlers
	wgHandler := wireguard.NewHandler(wgSvc)
	warpHandler := warp.NewHandler(warpSvc)
	certHandler := certificate.NewHandler(certSvc)
	singboxHandler := singbox.NewHandler(singboxSvc)
	subHandler := subscription.NewHandler(subSvc)
	proberHandler := prober.NewHandler(proberSvc)
	speedtestHandler := speedtest.NewHandler(speedtestSvc)

	// Initialize prober
	if err := proberSvc.Init(); err != nil {
		log.Printf("Warning: Failed to initialize prober: %v", err)
	} else {
		// Prober nodes are loaded during Init(); auto-start if there are nodes
		if results := proberSvc.GetAllResults(); len(results) > 0 {
			proberSvc.Start()
			log.Println("Prober started with saved nodes")
		}
	}

	// Start subscription auto-update scheduler
	sched.Start()

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(corsMiddleware())

	// API route group
	api := r.Group("/api")
	{
		// WireGuard routes
		wgGroup := api.Group("/wireguard")
		wgHandler.RegisterRoutes(wgGroup)

		// sing-box management routes (includes certificate and reality)
		singboxGroup := api.Group("/singbox")
		singboxHandler.RegisterRoutes(singboxGroup)
		certHandler.RegisterRoutes(singboxGroup)

		// Reality key management (under singbox group)
		singboxGroup.POST("/reality/keypair", certificate.GenerateRealityKeypair)
		singboxGroup.POST("/reality/public-key", certificate.DeriveRealityPublicKey)
		singboxGroup.POST("/reality/check-tls", certificate.CheckTLS13Support)

		// Subscription routes
		subGroup := api.Group("/subscription")
		subHandler.RegisterRoutes(subGroup)

		// Prober routes
		proberGroup := api.Group("/prober")
		proberHandler.RegisterRoutes(proberGroup)

		// Speed test routes
		speedtestGroup := api.Group("/speedtest")
		speedtestHandler.RegisterRoutes(speedtestGroup)

		// WARP routes
		warpGroup := api.Group("/warp")
		warpHandler.RegisterRoutes(warpGroup)
	}

	// Swagger API documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", healthCheck)

	// Static file server (optional, controlled by --dashboard flag)
	if *serveDashboard {
		setupStaticFiles(r)
		log.Println("Dashboard static files are being served")
	} else {
		log.Println("Dashboard serving disabled (use --dashboard to enable)")
	}

	// Start server
	listenAddr := cfg.GetListenAddr()
	log.Printf("Sing-box Config Service is starting on %s", listenAddr)
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// healthCheck handles GET /health requests
// @Summary      Health check
// @Description  Returns the server health status
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// setupStaticFiles configure static file serving
func setupStaticFiles(r *gin.Engine) {
	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Printf("Warning: Failed to load embedded frontend files: %v", err)
		return
	}

	fileServer := http.FileServer(http.FS(staticFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

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
