package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h *Handler) Health(c echo.Context) error {
	if err := h.services.Health(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": "down", "database": "down"})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "database": "up"})
}
