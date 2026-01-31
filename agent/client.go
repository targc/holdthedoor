package main

import (
	"crypto/ed25519"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"privateshell/pkg/crypto"

	"github.com/gorilla/websocket"
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

type Client struct {
	serverURL    string
	token        string
	serverPubKey ed25519.PublicKey
	vmName       string
	conn         *websocket.Conn
	shell        *Shell
	done         chan struct{}
}

func NewClient(serverURL, token string, serverPubKey ed25519.PublicKey, vmName string) *Client {
	return &Client{
		serverURL:    serverURL,
		token:        token,
		serverPubKey: serverPubKey,
		vmName:       vmName,
		done:         make(chan struct{}),
	}
}

func (c *Client) Run() {
	for {
		select {
		case <-c.done:
			return
		default:
			if err := c.connect(); err != nil {
				log.Printf("Connection error: %v, retrying in 5s", err)
				time.Sleep(5 * time.Second)
				continue
			}
			c.handleMessages()
		}
	}
}

func (c *Client) connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.serverURL, nil)
	if err != nil {
		return err
	}
	c.conn = conn

	// Step 1: Generate challenge for server to sign
	challenge, err := crypto.GenerateChallenge()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Step 2: Send auth message with token + challenge
	if err := conn.WriteJSON(Message{
		Type:      "auth",
		Data:      c.token,
		Challenge: challenge,
	}); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// Step 3: Wait for server's auth_ok with signature
	var authResp Message
	if err := conn.ReadJSON(&authResp); err != nil {
		conn.Close()
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if authResp.Type == "auth_error" {
		conn.Close()
		return fmt.Errorf("auth failed: %s", authResp.Data)
	}

	if authResp.Type != "auth_ok" {
		conn.Close()
		return fmt.Errorf("unexpected response: %s", authResp.Type)
	}

	// Step 4: Verify server signed our challenge
	if !crypto.Verify(c.serverPubKey, challenge, authResp.Signature) {
		conn.Close()
		return fmt.Errorf("server signature verification failed")
	}

	log.Println("Server verified successfully")

	// Step 5: Send registration
	name := c.vmName
	if name == "" {
		name = getHostname()
	}
	if err := conn.WriteJSON(Message{
		Type:     "register",
		Name:     name,
		Hostname: getHostname(),
		IP:       getLocalIP(),
		OS:       runtime.GOOS,
	}); err != nil {
		conn.Close()
		return fmt.Errorf("failed to register: %w", err)
	}

	log.Println("Connected and registered")
	return nil
}

func (c *Client) handleMessages() {
	defer c.conn.Close()

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		switch msg.Type {
		case "shell_start":
			c.startShell()
		case "input":
			if c.shell != nil {
				c.shell.Write([]byte(msg.Data))
			}
		case "resize":
			if c.shell != nil {
				c.shell.Resize(msg.Cols, msg.Rows)
			}
		case "shell_stop":
			c.stopShell()
		}
	}
}

func (c *Client) startShell() {
	if c.shell != nil {
		c.shell.Close()
	}

	shell, err := NewShell()
	if err != nil {
		log.Printf("Failed to start shell: %v", err)
		c.conn.WriteJSON(Message{Type: "error", Data: err.Error()})
		return
	}
	c.shell = shell

	// Stream output to server
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := shell.Read(buf)
			if err != nil {
				return
			}
			c.conn.WriteJSON(Message{Type: "output", Data: string(buf[:n])})
		}
	}()

	log.Println("Shell started")
}

func (c *Client) stopShell() {
	if c.shell != nil {
		c.shell.Close()
		c.shell = nil
		log.Println("Shell stopped")
	}
}

func (c *Client) Close() {
	close(c.done)
	c.stopShell()
	if c.conn != nil {
		c.conn.Close()
	}
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
