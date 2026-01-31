package main

import (
	"flag"
	"log"

	"privateshell/pkg/crypto"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	port := flag.String("port", "8080", "Server port")
	serverKeyPath := flag.String("server-key", "", "Server private key (Ed25519 PEM)")
	token := flag.String("token", "", "Agent authentication token")
	flag.Parse()

	if *serverKeyPath == "" || *token == "" {
		log.Fatal("--server-key and --token are required")
	}

	serverKey, err := crypto.LoadPrivateKey(*serverKeyPath)
	if err != nil {
		log.Fatalf("Failed to load server key: %v", err)
	}

	server := NewServer(serverKey, *token)

	app := fiber.New()
	app.Use(cors.New())

	server.SetupRoutes(app)

	// Serve static files
	app.Static("/", "./web")

	log.Printf("Server starting on :%s", *port)
	log.Fatal(app.Listen(":" + *port))
}
