package server

import (
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type Agent struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Hostname string          `json:"hostname"`
	IP       string          `json:"ip"`
	OS       string          `json:"os"`
	Conn     *websocket.Conn `json:"-"`
	Output   chan []byte     `json:"-"` // Messages from agent to terminal
}

type Registry struct {
	mu     sync.RWMutex
	agents map[string]*Agent
}

func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*Agent),
	}
}

func (r *Registry) Add(conn *websocket.Conn, name, hostname, ip, os string) *Agent {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent := &Agent{
		ID:       uuid.Must(uuid.NewV7()).String(),
		Name:     name,
		Hostname: hostname,
		IP:       ip,
		OS:       os,
		Conn:     conn,
		Output:   make(chan []byte, 256),
	}
	r.agents[agent.ID] = agent
	return agent
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if agent, ok := r.agents[id]; ok {
		close(agent.Output)
		delete(r.agents, id)
	}
}

func (r *Registry) Get(id string) *Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[id]
}

func (r *Registry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		list = append(list, agent)
	}
	return list
}
