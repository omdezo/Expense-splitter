package handler

import (
	"expense-splitter/services"
	"expense-splitter/types"
)

// Handler holds the dependencies shared by all HTTP handlers — the service layer
// and a logger. Handlers are methods on it so dependencies are explicit and the
// set is easy to grow.
type Handler struct {
	services *services.Services
	logger   *types.Logger
}

func New(svc *services.Services, logger *types.Logger) *Handler {
	return &Handler{services: svc, logger: logger}
}
