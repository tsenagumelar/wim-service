package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"wim-service/internal/api"
	"wim-service/internal/config"
	"wim-service/internal/handler"
)

func main() {
	log.Println("========================================")
	log.Println("  WIM API SERVER")
	log.Println("========================================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("[API] Failed to load config:", err)
	}
	defer cfg.DB.Close()

	// Initialize attachment handler
	attachmentHandler, err := handler.NewAttachmentHandler(
		cfg.AttachmentMinIOEndpoint,
		cfg.AttachmentMinIOAccess,
		cfg.AttachmentMinIOSecret,
		cfg.AttachmentMinIOBucket,
		cfg.AttachmentMinIOUseSSL,
	)
	if err != nil {
		log.Fatal("[API] Failed to create attachment handler:", err)
	}

	// Create API server
	apiServer := api.NewServer(cfg.DB, cfg.JWTSecret, attachmentHandler)

	log.Println("")
	log.Println("API Endpoints:")
	log.Printf("  → http://localhost:%s", cfg.APIPort)
	log.Println("")
	log.Println("Public Endpoints:")
	log.Printf("  - Health Check:  GET  /health")
	log.Printf("  - Login:         POST /api/auth/login")
	log.Println("")
	log.Println("Protected Endpoints (Require JWT Token):")
	log.Printf("  - Profile:       GET  /api/auth/profile")
	log.Printf("  - Upload Image:  POST /api/attachment/upload")
	log.Println("")
	log.Println("Press Ctrl+C to stop the API server")
	log.Println("========================================")
	log.Println("")

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("")
		log.Println("[API] Shutting down gracefully...")
		if err := apiServer.Shutdown(); err != nil {
			log.Printf("[API] Error during shutdown: %v", err)
		}
		log.Println("[API] Shutdown complete. Goodbye!")
		os.Exit(0)
	}()

	// Start server
	log.Printf("[API] Starting server on port %s...", cfg.APIPort)
	if err := apiServer.Start(cfg.APIPort); err != nil {
		log.Fatal("[API] Server error:", err)
	}
}
