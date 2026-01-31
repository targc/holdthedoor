package main

import (
	"crypto/ed25519"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type Server struct {
	serverKey ed25519.PrivateKey
	token     string
	registry  *Registry
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewServer(serverKey ed25519.PrivateKey, token string) *Server {
	return &Server{
		serverKey: serverKey,
		token:     token,
		registry:  NewRegistry(),
	}
}

func (s *Server) SetupRoutes(app *fiber.App) {
	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Agent WebSocket endpoint
	app.Get("/ws/agent", websocket.New(s.handleAgentWS))

	// Terminal WebSocket endpoint
	app.Get("/ws/terminal/:id", websocket.New(s.handleTerminalWS))

	// REST API
	api := app.Group("/api")
	api.Get("/vms", s.listVMs)
}
