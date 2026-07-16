package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// RunNudges godoc
//
//	@Summary		Send reminders for stalled payments
//	@Description	Admins only. Nudges whoever currently holds the ball on each payment that has sat idle longer than `hours`. **Idempotent** — re-running inside the same window will not double-notify.
//	@Tags			ops
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"group id"	Format(uuid)
//	@Param			hours	query		int						false	"idle threshold in hours (0-720)"	default(24)
//	@Success		200		{object}	types.NudgeRunResult	"who was nudged, and who was skipped"
//	@Failure		400		{object}	types.apiError			"invalid group id or hours"
//	@Failure		401		{object}	types.apiError			"missing or invalid token"
//	@Failure		403		{object}	types.apiError			"not an admin of this group"
//	@Failure		404		{object}	types.apiError			"group not found"
//	@Router			/groups/{id}/nudges [post]
func (h *Handler) RunNudges(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[RunNudges] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	hours := 24
	if raw := c.QueryParam("hours"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 || v > 720 {
			return c.JSON(http.StatusBadRequest, types.NewBadRequestError("hours must be an integer between 0 and 720"))
		}
		hours = v
	}

	result, apiErr := h.services.RunNudges(c.Request().Context(), *identity, groupID, hours)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, result)
}
