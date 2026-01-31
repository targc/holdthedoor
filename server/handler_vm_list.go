package main

import (
	"github.com/gofiber/fiber/v2"
)

type VMListResponse struct {
	VMs []*Agent `json:"vms"`
}

func (s *Server) listVMs(c *fiber.Ctx) error {
	vms := s.registry.List()
	return c.JSON(VMListResponse{VMs: vms})
}
