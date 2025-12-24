package api

import (
	"database/sql"
	"log"
	"wim-service/internal/auth"
	"wim-service/internal/handler"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type Server struct {
	App               *fiber.App
	AuthService       *auth.AuthService
	AuthHandler       *AuthHandler
	AttachmentHandler *handler.AttachmentHandler
}

func NewServer(db *sql.DB, jwtSecret string, attachmentHandler *handler.AttachmentHandler) *Server {
	app := fiber.New(fiber.Config{
		AppName: "WIM Service API",
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	authService := auth.NewAuthService(db, jwtSecret)
	authHandler := NewAuthHandler(authService)

	server := &Server{
		App:               app,
		AuthService:       authService,
		AuthHandler:       authHandler,
		AttachmentHandler: attachmentHandler,
	}

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	s.App.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "wim-service",
		})
	})

	api := s.App.Group("/api")

	// Auth routes (public)
	authRoutes := api.Group("/auth")
	authRoutes.Post("/login", s.AuthHandler.Login)

	// Protected routes (requires JWT)
	protected := api.Group("/auth")
	protected.Use(JWTMiddleware(s.AuthService))
	protected.Get("/profile", s.AuthHandler.GetProfile)

	// Attachment upload routes
	attachment := api.Group("/attachment")

	// Public upload endpoint (no auth required)
	attachment.Post("/upload", s.AttachmentHandler.UploadImage)

	// Protected upload endpoint (requires JWT)
	attachmentProtected := attachment.Group("/secure")
	attachmentProtected.Use(JWTMiddleware(s.AuthService))
	attachmentProtected.Post("/upload", s.AttachmentHandler.UploadImage)
}

func (s *Server) Start(port string) error {
	log.Printf("[API] Starting server on port %s", port)
	return s.App.Listen(":" + port)
}

func (s *Server) Shutdown() error {
	log.Println("[API] Shutting down server...")
	return s.App.Shutdown()
}
