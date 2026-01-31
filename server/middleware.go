package main

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// jwtAuth middleware for REST API (Authorization header)
func (s *Server) jwtAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "missing authorization"})
		}

		tokenString := strings.TrimPrefix(auth, "Bearer ")
		if tokenString == auth {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "invalid authorization format"})
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "invalid token"})
		}

		return c.Next()
	}
}

// jwtAuthWS middleware for WebSocket (query param ?token=)
func (s *Server) jwtAuthWS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Query("token")
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "missing token"})
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "invalid token"})
		}

		return c.Next()
	}
}
