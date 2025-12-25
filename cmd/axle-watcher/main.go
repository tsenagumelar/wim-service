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
	log.Println("  WIM AXLE FTP WATCHER")
	log.Println("========================================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("[AXLE] Failed to load config:", err)
	}
	defer cfg.DB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create AXLE processor
	axleProcessor, err := handler.NewAxleProcessor(
		cfg.DB,
		cfg.SiteUUID,
		cfg.AxleFTPDir,
		cfg.AxleMinIOEndpoint,
		cfg.AxleMinIOAccess,
		cfg.AxleMinIOSecret,
		cfg.AxleMinIOBucket,
		cfg.AxleMinIOUseSSL,
	)
	if err != nil {
		log.Fatal("[AXLE] Failed to create AXLE processor:", err)
	}

	// Create FTP watcher
	axleWatcher := ftpwatcher.New(
		cfg.AxleFTPHost,
		cfg.AxleFTPUser,
		cfg.AxleFTPPass,
		cfg.AxleFTPDir,
		cfg.AxleFTPInterval,
		axleProcessor.HandleNewFileAXLE,
	)

	log.Println("")
	log.Println("Configuration:")
	log.Printf("  FTP Host:     %s", cfg.AxleFTPHost)
	log.Printf("  FTP Dir:      %s", cfg.AxleFTPDir)
	log.Printf("  Interval:     %v", cfg.AxleFTPInterval)
	log.Printf("  MinIO:        %s", cfg.AxleMinIOEndpoint)
	log.Printf("  Bucket:       %s", cfg.AxleMinIOBucket)
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
		log.Println("[AXLE] Shutting down gracefully...")
		cancel()
	}()

	// Start watcher
	log.Println("[AXLE] Starting FTP watcher...")
	if err := axleWatcher.Start(ctx); err != nil {
		log.Printf("[AXLE] Watcher stopped: %v", err)
	}

	log.Println("[AXLE] Shutdown complete. Goodbye!")
}
