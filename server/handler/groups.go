package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// CreateGroup godoc
//
//	@Summary		Create a trip group
//	@Description	Any verified user can create a group and becomes its group-admin. The global admin may instead create one *on behalf of others* by setting `group_admin_id` — they then assign that member as group-admin and are not part of the trip or the split.
//	@Description
//	@Description	`expected_member_count` is a planning hint only and has **zero** effect on any calculation — fair shares always divide by the count of actual members at settlement.
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		types.CreateGroupRequest	true	"name, trip dates, optional group_admin_id"
//	@Success		201		{object}	types.Group					"the created group, including its invite token"
//	@Failure		400		{object}	types.apiError				"invalid body / bad date range"
//	@Failure		401		{object}	types.apiError				"missing or invalid token"
//	@Failure		403		{object}	types.apiError				"caller is not verified"
//	@Router			/groups [post]
func (h *Handler) CreateGroup(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[CreateGroup] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	var req types.CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	g, apiErr := h.services.CreateGroup(c.Request().Context(), *identity, req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusCreated, g)
}

// UpdateGroup godoc
//
//	@Summary		Update group metadata
//	@Description	Group-admin only, and only while the group is **open**. Changes name, trip dates, or the planning hint.
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"group id"	Format(uuid)
//	@Param			body	body		types.UpdateGroupRequest	true	"fields to change"
//	@Success		200		{object}	types.Group					"the updated group"
//	@Failure		400		{object}	types.apiError				"invalid id or body"
//	@Failure		401		{object}	types.apiError				"missing or invalid token"
//	@Failure		403		{object}	types.apiError				"not this group's admin"
//	@Failure		404		{object}	types.apiError				"group not found"
//	@Failure		409		{object}	types.apiError				"group is closed"
//	@Router			/groups/{id} [patch]
func (h *Handler) UpdateGroup(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[UpdateGroup] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	var req types.UpdateGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	g, apiErr := h.services.UpdateGroup(c.Request().Context(), *identity, groupID, req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, g)
}

// ListMyGroups godoc
//
//	@Summary		List the groups you belong to
//	@Description	Every group where the caller has an approved membership, with their role in each.
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		types.GroupListItem	"your groups"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Router			/groups [get]
func (h *Handler) ListMyGroups(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[ListMyGroups] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	list, apiErr := h.services.ListMyGroups(c.Request().Context(), *identity)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, list)
}

// GetGroup godoc
//
//	@Summary		Get a group's metadata and members
//	@Description	Members and admins only. Returns trip metadata plus the member list — this is the non-financial view; use `/groups/{id}/summary` for the money.
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"group id"	Format(uuid)
//	@Success		200	{object}	types.GroupDetail	"metadata + members"
//	@Failure		400	{object}	types.apiError		"invalid group id"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Failure		403	{object}	types.apiError		"not a member of this group"
//	@Failure		404	{object}	types.apiError		"group not found"
//	@Router			/groups/{id} [get]
func (h *Handler) GetGroup(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[GetGroup] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	detail, apiErr := h.services.GetGroup(c.Request().Context(), *identity, groupID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, detail)
}

// GetSettlementReport godoc
//
//	@Summary		Download the settlement report PDF
//	@Description	Available only once **every** computed payment has reached `settled`. Returns the final report as a PDF attachment.
//	@Tags			settlement
//	@Produce		application/pdf
//	@Security		BearerAuth
//	@Param			id	path		string			true	"group id"	Format(uuid)
//	@Success		200	{file}		binary			"the settlement report PDF"
//	@Failure		400	{object}	types.apiError	"invalid group id"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"not a member of this group"
//	@Failure		404	{object}	types.apiError	"group not found"
//	@Failure		409	{object}	types.apiError	"group is not fully settled yet"
//	@Router			/groups/{id}/report.pdf [get]
func (h *Handler) GetSettlementReport(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[GetSettlementReport] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	pdf, apiErr := h.services.SettlementReport(c.Request().Context(), *identity, groupID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	c.Response().Header().Set("Content-Disposition", `attachment; filename="settlement-report.pdf"`)
	return c.Blob(http.StatusOK, "application/pdf", pdf)
}

// CloseGroup godoc
//
//	@Summary		Close the group and compute settlement
//	@Description	Group-admin only. **This is the pivotal action:** it freezes expenses, computes each member's fair share, and generates the payment plan — the minimum set of transfers that zeroes every balance.
//	@Description
//	@Description	Runs exactly once, under a row lock, over a consistent snapshot: a concurrent second close gets 409, and expenses in flight during the close are blocked and then rejected.
//	@Tags			groups
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"group id"	Format(uuid)
//	@Success		200	{object}	types.CloseResult	"the computed plan"
//	@Failure		400	{object}	types.apiError		"invalid group id"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Failure		403	{object}	types.apiError		"not this group's admin"
//	@Failure		404	{object}	types.apiError		"group not found"
//	@Failure		409	{object}	types.apiError		"group is already closed"
//	@Router			/groups/{id}/close [post]
func (h *Handler) CloseGroup(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[CloseGroup] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	res, apiErr := h.services.CloseGroup(c.Request().Context(), *identity, groupID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, res)
}
