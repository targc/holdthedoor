package main

import (
	"flag"
	"log"
	"os"

	"holdthedoor/pkg/crypto"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	port := flag.String("port", "8080", "Server port")
	serverKeyPath := flag.String("server-key", "", "Server private key (Ed25519 PEM)")
	token := flag.String("token", "", "Agent authentication token")
	username := flag.String("username", "", "Web UI username (or USERNAME env)")
	password := flag.String("password", "", "Web UI password (or PASSWORD env)")
	jwtSecret := flag.String("jwt-secret", "", "JWT secret key (or JWT_SECRET env)")
	flag.Parse()

	if *serverKeyPath == "" || *token == "" {
		log.Fatal("--server-key and --token are required")
	}

	// Get auth from flags or env
	uiUsername := *username
	if uiUsername == "" {
		uiUsername = os.Getenv("USERNAME")
	}
	uiPassword := *password
	if uiPassword == "" {
		uiPassword = os.Getenv("PASSWORD")
	}
	uiJWTSecret := *jwtSecret
	if uiJWTSecret == "" {
		uiJWTSecret = os.Getenv("JWT_SECRET")
	}

	if uiUsername == "" || uiPassword == "" || uiJWTSecret == "" {
		log.Fatal("--username, --password, --jwt-secret (or env vars) are required")
	}

	serverKey, err := crypto.LoadPrivateKey(*serverKeyPath)
	if err != nil {
		log.Fatalf("Failed to load server key: %v", err)
	}

	server := NewServer(serverKey, *token, uiUsername, uiPassword, uiJWTSecret)

	app := fiber.New()
	app.Use(cors.New())

	server.SetupRoutes(app)

	// Serve static files
	app.Static("/", "./web")

	log.Printf("Server starting on :%s", *port)
	log.Fatal(app.Listen(":" + *port))
}
