package server

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/naseer2426/split-bot-whatsapp/internal/whatsapp"
)

// Server represents the HTTP server
type Server struct {
	handler *whatsapp.Handler
	port    string
	router  *gin.Engine
}

// NewServer creates a new HTTP server instance
func NewServer(handler *whatsapp.Handler) *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := gin.Default()

	// Allow CORS for all origins
	router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:   []string{"Content-Length"},
	}))

	return &Server{
		handler: handler,
		port:    port,
		router:  router,
	}
}

// Start starts the HTTP server in a goroutine
func (s *Server) Start() {
	s.setupRoutes()

	go func() {
		fmt.Printf("HTTP server starting on port %s\n", s.port)
		if err := s.router.Run(":" + s.port); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
}

// setupRoutes registers all HTTP routes
func (s *Server) setupRoutes() {
	s.router.POST("/send_message_to_group", s.sendMessageToGroupHandler)
	s.router.POST("/send_message_to_user", s.sendMessageToUserHandler)
	s.router.POST("/grafana_webhook", s.grafanaWebhookHandler)
}
