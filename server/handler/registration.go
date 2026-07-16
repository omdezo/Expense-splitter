package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// Register godoc
//
//	@Summary		Link the caller's token to a local user row
//	@Description	Idempotent. Creates the local row for the bearer token's subject, or links/returns the existing one. Login does this automatically — this endpoint exists for tokens minted outside the login flow.
//	@Tags			account
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	types.Principal	"the linked local account"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		409	{object}	types.apiError	"email already registered to another subject"
//	@Router			/register [post]
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
