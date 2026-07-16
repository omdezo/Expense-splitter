package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// Me godoc
//
//	@Summary		Who am I
//	@Description	Returns the claims from your Keycloak token plus the linked `local_user` row (id, verification status, global-admin flag).
//	@Tags			account
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	services.MeResponse	"token claims + local user"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Router			/me [get]
func (h *Handler) Me(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {

		h.logger.Error("[Me] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	resp, apiErr := h.services.Me(c.Request().Context(), *identity)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, resp)
}
