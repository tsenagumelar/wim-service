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
	log.Println("[MAIN] WIM Service starting...")

	// Load configuration and initialize database
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("[MAIN] Failed to load config:", err)
	}
	defer cfg.DB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create ANPR processor
	anprProcessor, err := handler.NewFileProcessor(
		cfg.DB,
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

	// Create AXLE processor
	axleProcessor, err := handler.NewAxleProcessor(
		cfg.DB,
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

	log.Println("[MAIN] Both watchers started. Press Ctrl+C to stop.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("[MAIN] Shutting down gracefully...")
	cancel()
	log.Println("[MAIN] Shutdown complete.")
}
