package server

import (
	"log"

	"github.com/gofiber/contrib/websocket"
)

func (s *Server) handleTerminalWS(c *websocket.Conn) {
	vmID := c.Params("id")

	agent := s.registry.Get(vmID)
	if agent == nil {
		log.Printf("Terminal: VM not found: %s", vmID)
		c.WriteJSON(Message{Type: "error", Data: "VM not found"})
		c.Close()
		return
	}

	log.Printf("Terminal session started for: %s (%s)", agent.Name, vmID)

	// Tell agent to start shell
	if err := agent.Conn.WriteJSON(Message{Type: "shell_start"}); err != nil {
		log.Printf("Failed to start shell on agent: %v", err)
		c.Close()
		return
	}

	done := make(chan struct{})

	// Forward agent output to browser (read from Output channel)
	go func() {
		defer close(done)
		for data := range agent.Output {
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}()

	// Forward browser input to agent
	go func() {
		for {
			var msg Message
			if err := c.ReadJSON(&msg); err != nil {
				agent.Conn.WriteJSON(Message{Type: "shell_stop"})
				return
			}
			if msg.Type == "input" || msg.Type == "resize" {
				if err := agent.Conn.WriteJSON(msg); err != nil {
					return
				}
			}
		}
	}()

	<-done
	log.Printf("Terminal session ended for: %s", agent.Name)
}
