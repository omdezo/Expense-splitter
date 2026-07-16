package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// GetSettlementPlan godoc
//
//	@Summary		Get the computed payment plan
//	@Description	Members and admins, **closed groups only**. The full `{from, to, amount, status}` list produced at close, plus an "N of M settled" progress count. All payments start `pending`.
//	@Description
//	@Description	The plan is the minimum-ish set of transfers from a greedy largest-debtor/largest-creditor match — at most `n-1` transfers, and it always reconciles to zero.
//	@Tags			settlement
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string							true	"group id"	Format(uuid)
//	@Success		200	{object}	types.SettlementPlanResponse	"the payment plan + progress"
//	@Failure		400	{object}	types.apiError					"invalid group id"
//	@Failure		401	{object}	types.apiError					"missing or invalid token"
//	@Failure		403	{object}	types.apiError					"not a member of this group"
//	@Failure		404	{object}	types.apiError					"group not found"
//	@Failure		409	{object}	types.apiError					"group is still open — close it first"
//	@Router			/groups/{id}/settlement [get]
func (h *Handler) GetSettlementPlan(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[GetSettlementPlan] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	plan, apiErr := h.services.GetSettlementPlan(c.Request().Context(), *identity, groupID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, plan)
}

// GetGroupSummary godoc
//
//	@Summary		Get a group's financial summary
//	@Description	Members and admins. Trip name and dates, member count, total spent, spend per category, and per member: total **paid**, their **fair share**, and their **net balance** (paid - fair share). Works on open groups too — it is the live picture of who is up and who is down.
//	@Tags			settlement
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"group id"	Format(uuid)
//	@Success		200	{object}	types.GroupSummary	"totals, per-category spend, per-member balances"
//	@Failure		400	{object}	types.apiError		"invalid group id"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Failure		403	{object}	types.apiError		"not a member of this group"
//	@Failure		404	{object}	types.apiError		"group not found"
//	@Router			/groups/{id}/summary [get]
func (h *Handler) GetGroupSummary(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[GetGroupSummary] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	summary, apiErr := h.services.GroupSummary(c.Request().Context(), *identity, groupID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, summary)
}

// PublicGroupStatus godoc
//
//	@Summary		Check a group's status by share token
//	@Description	**The only unauthenticated group view** — no bearer token required. Given a group's shareable token, returns a deliberately minimal status (no amounts, no member financials) so a link can be shared safely.
//	@Tags			public
//	@Produce		json
//	@Param			token	path		string					true	"group share token"	Format(uuid)
//	@Success		200		{object}	types.PublicGroupStatus	"minimal public status"
//	@Failure		400		{object}	types.apiError			"invalid token"
//	@Failure		404		{object}	types.apiError			"no group for this token"
//	@Router			/public/groups/{token} [get]
func (h *Handler) PublicGroupStatus(c echo.Context) error {
	token := c.Param("token")
	if _, err := uuid.Parse(token); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group token"))
	}

	status, apiErr := h.services.PublicGroupStatus(c.Request().Context(), token)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, status)
}
