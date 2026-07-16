package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// GetGroupAudit godoc
//
//	@Summary		Read a group's audit trail
//	@Description	Admins only. The full history: expense edits with **before/after** amounts, role transfers, membership decisions, and every payment-state transition. Returns a `{total, limit, offset, items}` envelope.
//	@Tags			ops
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string		true	"group id"	Format(uuid)
//	@Param			limit	query		int			false	"page size (1-200)"	default(50)
//	@Param			offset	query		int			false	"rows to skip"	default(0)
//	@Success		200		{object}	types.Page	"paginated audit entries"
//	@Failure		400		{object}	types.apiError	"invalid id or paging"
//	@Failure		401		{object}	types.apiError	"missing or invalid token"
//	@Failure		403		{object}	types.apiError	"not an admin of this group"
//	@Failure		404		{object}	types.apiError	"group not found"
//	@Router			/groups/{id}/audit [get]
func (h *Handler) GetGroupAudit(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[GetGroupAudit] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	limit, offset, apiErr := parsePage(c)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	page, apiErr := h.services.ListGroupAudit(c.Request().Context(), *identity, groupID, limit, offset)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, page)
}
