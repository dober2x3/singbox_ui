package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
	"singbox-config-service/internal/certificate"
	"singbox-config-service/internal/prober"
	"singbox-config-service/internal/scheduler"
	"singbox-config-service/internal/singbox"
	"singbox-config-service/internal/speedtest"
	"singbox-config-service/internal/subscription"
	"singbox-config-service/internal/warp"
	"singbox-config-service/internal/wireguard"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

func main() {
	// Initialize config
	cfg, err := config.Init()
	if err != nil {
		log.Printf("Warning: Failed to initialize config: %v", err)
		log.Println("Some features may not work properly")
	}

	// Initialize Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Printf("Warning: Failed to create Docker client: %v", err)
		log.Println("Docker-dependent features will not be available")
	}
	if dockerClient != nil {
		defer dockerClient.Close()
	}

	// Create domain services
	singboxSvc := singbox.NewService(dockerClient, cfg)
	wgSvc := wireguard.NewService(cfg.GetDataDir())
	warpSvc := warp.NewService(cfg.GetDataDir())
	certSvc := certificate.NewService(cfg.GetSingboxDir())
	subStore := subscription.NewFileStore(cfg.GetDataDir())
	subSvc := subscription.NewService(subStore)
	proberSvc := prober.NewService(cfg.GetDataDir(), subSvc)
	speedtestSvc := speedtest.NewService(dockerClient, cfg)

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

	// Pull sing-box image in background
	if dockerClient != nil {
		go func() {
			log.Println("Pulling sing-box image in background...")
			if err := dockerClient.EnsureImage(context.Background(), "ghcr.io/sagernet/sing-box:latest", ""); err != nil {
				log.Printf("Warning: Failed to pull sing-box image: %v", err)
			} else {
				log.Println("sing-box image is ready")
			}
		}()
	}

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

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Static file server
	setupStaticFiles(r)

	// Start server
	listenAddr := cfg.GetListenAddr()
	log.Printf("Sing-box Config Service is starting on %s", listenAddr)
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
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
