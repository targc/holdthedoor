package main

import (
	"log"

	"privateshell/pkg/crypto"

	"github.com/gofiber/contrib/websocket"
)

type Message struct {
	Type      string `json:"type"`
	Data      string `json:"data,omitempty"`
	Challenge string `json:"challenge,omitempty"`
	Signature string `json:"signature,omitempty"`
	Name      string `json:"name,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	IP        string `json:"ip,omitempty"`
	OS        string `json:"os,omitempty"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
}

func (s *Server) handleAgentWS(c *websocket.Conn) {
	// Step 1: Wait for auth message from agent
	var authMsg Message
	if err := c.ReadJSON(&authMsg); err != nil || authMsg.Type != "auth" {
		log.Printf("Agent auth failed: expected auth message")
		c.Close()
		return
	}

	// Step 2: Verify agent's token
	if authMsg.Data != s.token {
		log.Printf("Agent auth failed: invalid token")
		c.WriteJSON(Message{Type: "auth_error", Data: "invalid token"})
		c.Close()
		return
	}

	// Step 3: Sign agent's challenge to prove server identity
	serverSig := crypto.Sign(s.serverKey, authMsg.Challenge)

	// Step 4: Send auth_ok with server's proof
	if err := c.WriteJSON(Message{
		Type:      "auth_ok",
		Signature: serverSig,
	}); err != nil {
		log.Printf("Failed to send auth_ok: %v", err)
		c.Close()
		return
	}

	// Step 5: Wait for registration message
	var regMsg Message
	if err := c.ReadJSON(&regMsg); err != nil || regMsg.Type != "register" {
		log.Printf("Agent registration failed: %v", err)
		c.Close()
		return
	}

	agent := s.registry.Add(c, regMsg.Name, regMsg.Hostname, regMsg.IP, regMsg.OS)
	log.Printf("Agent registered: %s (%s)", agent.Name, agent.ID)

	defer func() {
		s.registry.Remove(agent.ID)
		log.Printf("Agent disconnected: %s (%s)", agent.Name, agent.ID)
	}()

	// Read messages from agent and forward to Output channel
	for {
		_, data, err := c.ReadMessage()
		if err != nil {
			return
		}
		select {
		case agent.Output <- data:
		default:
			// Drop if channel full (no terminal connected)
		}
	}
}
