package handler

import (
	"expense-splitter/services"
	"expense-splitter/types"
)

type Handler struct {
	services *services.Services
	logger   *types.Logger
}

func New(svc *services.Services, logger *types.Logger) *Handler {
	return &Handler{services: svc, logger: logger}
}
