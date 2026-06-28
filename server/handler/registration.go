package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

func (h *Handler) Register(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[Register] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	p, apiErr := h.services.Register(c.Request().Context(), *identity)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusCreated, p)
}
