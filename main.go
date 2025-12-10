package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"wim-service/internal/api"
	"wim-service/internal/auth"
	"wim-service/internal/config"
	"wim-service/internal/ftpwatcher"
	"wim-service/internal/handler"
)

func main() {
	log.Println("[MAIN] WIM Service starting...")

	// Load configuration and initialize database
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("[MAIN] Failed to load config:", err)
	}
	defer cfg.DB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create default admin user if not exists
	authService := auth.NewAuthService(cfg.DB, cfg.JWTSecret)
	if err := authService.CreateUser("admin", "admin@wim.local", "admin123", "admin"); err != nil {
		log.Printf("[MAIN] Note: %v (user might already exist)", err)
	}

	// Start API Server in goroutine
	apiServer := api.NewServer(cfg.DB, cfg.JWTSecret)
	go func() {
		log.Println("[MAIN] Starting API Server...")
		log.Printf("[MAIN] API Server: http://localhost:%s", cfg.APIPort)
		log.Printf("[MAIN] Health check: http://localhost:%s/health", cfg.APIPort)
		log.Printf("[MAIN] Login: POST http://localhost:%s/api/auth/login", cfg.APIPort)
		log.Printf("[MAIN] Profile: GET http://localhost:%s/api/auth/profile", cfg.APIPort)
		log.Println("[MAIN] Default credentials - username: admin, password: admin123")
		if err := apiServer.Start(cfg.APIPort); err != nil {
			log.Printf("[MAIN] API Server error: %v", err)
		}
	}()

	// Create dimension handler if enabled
	var dimensionHandler *handler.DimensionHandler
	if cfg.DimensionEnabled {
		log.Println("[MAIN] Vehicle dimension detection is ENABLED")
		dimensionHandler, err = handler.NewDimensionHandler(
			cfg.DB,
			cfg.SiteUUID, // Site UUID from master_site
			cfg.DimensionModelPath,
			cfg.DimensionThreshold,
		)
		if err != nil {
			log.Fatal("[MAIN] Failed to create dimension handler:", err)
		}

		// Set camera calibration
		calibration := cfg.GetCameraCalibration()
		if err := dimensionHandler.SetCalibration(calibration); err != nil {
			log.Fatal("[MAIN] Failed to set camera calibration:", err)
		}
		log.Println("[MAIN] Camera calibration:")
		log.Printf("  Resolution: %dx%d", cfg.CameraImageWidth, cfg.CameraImageHeight)
		log.Printf("  Height: %.2fm, Tilt: %.2f°", cfg.CameraHeight, cfg.CameraTiltAngle)
		log.Printf("  Reference: %dpx = %.2fm at %.2fm", cfg.CameraRefPixelLength, cfg.CameraRefRealLength, cfg.CameraRefDistance)
	} else {
		log.Println("[MAIN] Vehicle dimension detection is DISABLED")
	}

	// Create ANPR file processor
	anprProcessor, err := handler.NewFileProcessor(
		cfg.DB,
		cfg.SiteUUID, // Site UUID from master_site
		cfg.ANPRFTPDir,
		cfg.ANPRMinIOEndpoint,
		cfg.ANPRMinIOAccess,
		cfg.ANPRMinIOSecret,
		cfg.ANPRMinIOBucket,
		cfg.ANPRMinIOUseSSL,
	)
	if err != nil {
		log.Fatal("[MAIN] Failed to create ANPR processor:", err)
	}

	// Set dimension handler for ANPR processor if enabled
	if dimensionHandler != nil {
		anprProcessor.SetDimensionHandler(dimensionHandler)
	}

	// Create AXLE processor
	axleProcessor, err := handler.NewAxleProcessor(
		cfg.DB,
		cfg.SiteUUID, // Site UUID from master_site
		cfg.AxleFTPDir,
		cfg.AxleMinIOEndpoint,
		cfg.AxleMinIOAccess,
		cfg.AxleMinIOSecret,
		cfg.AxleMinIOBucket,
		cfg.AxleMinIOUseSSL,
	)
	if err != nil {
		log.Fatal("[MAIN] Failed to create AXLE processor:", err)
	}

	// Create ANPR watcher
	anprWatcher := ftpwatcher.New(
		cfg.ANPRFTPHost,
		cfg.ANPRFTPUser,
		cfg.ANPRFTPPass,
		cfg.ANPRFTPDir,
		cfg.ANPRFTPInterval,
		anprProcessor.HandleNewFile,
	)

	// Create AXLE watcher
	axleWatcher := ftpwatcher.New(
		cfg.AxleFTPHost,
		cfg.AxleFTPUser,
		cfg.AxleFTPPass,
		cfg.AxleFTPDir,
		cfg.AxleFTPInterval,
		axleProcessor.HandleNewFileAXLE,
	)

	// Start ANPR watcher in goroutine
	go func() {
		log.Println("[MAIN] Starting ANPR watcher...")
		log.Printf("[MAIN] ANPR FTP: %s, Dir: %s, Interval: %v", cfg.ANPRFTPHost, cfg.ANPRFTPDir, cfg.ANPRFTPInterval)
		log.Printf("[MAIN] ANPR MinIO: %s, Bucket: %s", cfg.ANPRMinIOEndpoint, cfg.ANPRMinIOBucket)
		if err := anprWatcher.Start(ctx); err != nil {
			log.Printf("[MAIN] ANPR watcher error: %v", err)
		}
	}()

	// Start AXLE watcher in goroutine
	go func() {
		log.Println("[MAIN] Starting AXLE watcher...")
		log.Printf("[MAIN] AXLE FTP: %s, Dir: %s, Interval: %v", cfg.AxleFTPHost, cfg.AxleFTPDir, cfg.AxleFTPInterval)
		log.Printf("[MAIN] AXLE MinIO: %s, Bucket: %s", cfg.AxleMinIOEndpoint, cfg.AxleMinIOBucket)
		if err := axleWatcher.Start(ctx); err != nil {
			log.Printf("[MAIN] AXLE watcher error: %v", err)
		}
	}()

	log.Println("[MAIN] All services started successfully!")
	log.Println("[MAIN] Services running:")
	log.Printf("[MAIN]   - API Server: http://localhost:%s", cfg.APIPort)
	log.Println("[MAIN]   - ANPR Watcher (background)")
	log.Println("[MAIN]   - AXLE Watcher (background)")
	if cfg.DimensionEnabled {
		log.Println("[MAIN]   - Vehicle Dimension Detection (enabled)")
	}
	log.Println("[MAIN] Press Ctrl+C to stop all services.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("[MAIN] Shutting down gracefully...")
	cancel()

	// Shutdown API server
	if err := apiServer.Shutdown(); err != nil {
		log.Printf("[MAIN] Error shutting down API server: %v", err)
	}

	log.Println("[MAIN] Shutdown complete.")
}
