package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"wim-service/internal/config"
	"wim-service/internal/ftpwatcher"
	"wim-service/internal/handler"
)

func main() {
	log.Println("========================================")
	log.Println("  WIM ANPR FTP WATCHER")
	log.Println("========================================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("[ANPR] Failed to load config:", err)
	}
	defer cfg.DB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize dimension handler if enabled
	var dimensionHandler *handler.DimensionHandler
	if cfg.DimensionEnabled {
		log.Println("[ANPR] Vehicle Dimension Detection: ENABLED")
		dimensionHandler, err = handler.NewDimensionHandler(
			cfg.DB,
			cfg.SiteUUID,
			cfg.DimensionModelPath,
			cfg.DimensionThreshold,
		)
		if err != nil {
			log.Fatal("[ANPR] Failed to create dimension handler:", err)
		}

		calibration := cfg.GetCameraCalibration()
		if err := dimensionHandler.SetCalibration(calibration); err != nil {
			log.Fatal("[ANPR] Failed to set camera calibration:", err)
		}

		log.Printf("[ANPR] Camera: %dx%d, Height: %.2fm, Tilt: %.2f°",
			cfg.CameraImageWidth, cfg.CameraImageHeight,
			cfg.CameraHeight, cfg.CameraTiltAngle)
	} else {
		log.Println("[ANPR] Vehicle Dimension Detection: DISABLED")
	}

	// Create ANPR processor
	anprProcessor, err := handler.NewFileProcessor(
		cfg.DB,
		cfg.SiteUUID,
		cfg.ANPRFTPDir,
		cfg.ANPRMinIOEndpoint,
		cfg.ANPRMinIOAccess,
		cfg.ANPRMinIOSecret,
		cfg.ANPRMinIOBucket,
		cfg.ANPRMinIOUseSSL,
	)
	if err != nil {
		log.Fatal("[ANPR] Failed to create ANPR processor:", err)
	}

	// Link dimension handler
	if dimensionHandler != nil {
		anprProcessor.SetDimensionHandler(dimensionHandler)
	}

	// Create FTP watcher
	anprWatcher := ftpwatcher.New(
		cfg.ANPRFTPHost,
		cfg.ANPRFTPUser,
		cfg.ANPRFTPPass,
		cfg.ANPRFTPDir,
		cfg.ANPRFTPInterval,
		anprProcessor.HandleNewFile,
	)

	log.Println("")
	log.Println("Configuration:")
	log.Printf("  FTP Host:     %s", cfg.ANPRFTPHost)
	log.Printf("  FTP Dir:      %s", cfg.ANPRFTPDir)
	log.Printf("  Interval:     %v", cfg.ANPRFTPInterval)
	log.Printf("  MinIO:        %s", cfg.ANPRMinIOEndpoint)
	log.Printf("  Bucket:       %s", cfg.ANPRMinIOBucket)
	log.Println("")
	log.Println("Press Ctrl+C to stop the watcher")
	log.Println("========================================")
	log.Println("")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("")
		log.Println("[ANPR] Shutting down gracefully...")
		cancel()
	}()

	// Start watcher
	log.Println("[ANPR] Starting FTP watcher...")
	if err := anprWatcher.Start(ctx); err != nil {
		log.Printf("[ANPR] Watcher stopped: %v", err)
	}

	log.Println("[ANPR] Shutdown complete. Goodbye!")
}
