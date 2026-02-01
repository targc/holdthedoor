package server

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

func (s *Server) handleLogin(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request"})
	}

	if req.Username != s.username || req.Password != s.password {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "invalid credentials"})
	}

	// Generate JWT (1 hour expiry)
	expires := time.Now().Add(time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Username,
		"exp": expires.Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "failed to generate token"})
	}

	return c.JSON(LoginResponse{
		Token:   tokenString,
		Expires: expires.Unix(),
	})
}
