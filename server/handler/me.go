package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// Me returns the authenticated principal: the token claims plus the resolved
// local user when one exists.
func (h *Handler) Me(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		// Require middleware guarantees an identity; nil here is a wiring bug.
		h.logger.Error("[Me] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	resp, apiErr := h.services.Me(c.Request().Context(), *identity)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, resp)
}
