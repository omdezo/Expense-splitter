package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Health godoc
//
//	@Summary		Health check
//	@Description	Liveness probe. Reports `ok` only when the database is reachable; returns 503 otherwise.
//	@Tags			public
//	@Produce		json
//	@Success		200	{object}	map[string]string	"{\"status\":\"ok\",\"database\":\"up\"}"
//	@Failure		503	{object}	map[string]string	"{\"status\":\"down\",\"database\":\"down\"}"
//	@Router			/health [get]
func (h *Handler) Health(c echo.Context) error {
	if err := h.services.Health(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": "down", "database": "down"})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "database": "up"})
}
