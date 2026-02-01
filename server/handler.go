package server

import (
	"crypto/ed25519"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type Server struct {
	serverKey ed25519.PrivateKey
	token     string
	username  string
	password  string
	jwtSecret string
	registry  *Registry
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewServer(serverKey ed25519.PrivateKey, token, username, password, jwtSecret string) *Server {
	return &Server{
		serverKey: serverKey,
		token:     token,
		username:  username,
		password:  password,
		jwtSecret: jwtSecret,
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

	// Agent WebSocket endpoint (token auth - no JWT)
	app.Get("/ws/agent", websocket.New(s.handleAgentWS))

	// Terminal WebSocket endpoint (JWT auth via query param)
	app.Get("/ws/terminal/:id", s.jwtAuthWS(), websocket.New(s.handleTerminalWS))

	// Public API
	app.Post("/api/login", s.handleLogin)

	// Protected API
	api := app.Group("/api", s.jwtAuth())
	api.Get("/vms", s.listVMs)
}
