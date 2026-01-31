package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"holdthedoor/pkg/crypto"
)

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/ws/agent", "Platform server WebSocket URL")
	token := flag.String("token", "", "Authentication token")
	serverPubKeyPath := flag.String("server-pubkey", "", "Server public key (Ed25519 PEM)")
	name := flag.String("name", "", "VM display name (defaults to hostname)")
	flag.Parse()

	if *token == "" || *serverPubKeyPath == "" {
		log.Fatal("--token and --server-pubkey are required")
	}

	serverPubKey, err := crypto.LoadPublicKey(*serverPubKeyPath)
	if err != nil {
		log.Fatalf("Failed to load server public key: %v", err)
	}

	client := NewClient(*serverURL, *token, serverPubKey, *name)

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		client.Close()
		os.Exit(0)
	}()

	log.Printf("Connecting to %s", *serverURL)
	client.Run()
}
