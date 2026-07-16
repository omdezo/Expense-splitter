package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// JoinGroup godoc
//
//	@Summary		Request to join a group
//	@Description	Verified users only. Redeems a group's shareable invite token to create a `requested` membership. There is no open self-join — the group-admin must approve before you are a member.
//	@Tags			membership
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		types.JoinGroupRequest	true	"invite_token"
//	@Success		201		{object}	types.MembershipView	"membership in status requested"
//	@Failure		400		{object}	types.apiError			"invalid body or token"
//	@Failure		401		{object}	types.apiError			"missing or invalid token"
//	@Failure		403		{object}	types.apiError			"caller is not verified"
//	@Failure		404		{object}	types.apiError			"invite token does not match a group"
//	@Failure		409		{object}	types.apiError			"already a member or already requested"
//	@Router			/groups/join [post]
func (h *Handler) JoinGroup(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[JoinGroup] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	var req types.JoinGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	v, apiErr := h.services.RequestToJoin(c.Request().Context(), *identity, req.InviteToken)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusCreated, v)
}

// ListJoinRequests godoc
//
//	@Summary		List pending join requests
//	@Description	Group-admin only. The queue of users who redeemed the invite token and are waiting on a decision.
//	@Tags			membership
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string					true	"group id"	Format(uuid)
//	@Success		200	{array}		types.MembershipView	"pending requests"
//	@Failure		400	{object}	types.apiError			"invalid group id"
//	@Failure		401	{object}	types.apiError			"missing or invalid token"
//	@Failure		403	{object}	types.apiError			"not this group's admin"
//	@Failure		404	{object}	types.apiError			"group not found"
//	@Router			/groups/{id}/requests [get]
func (h *Handler) ListJoinRequests(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[ListJoinRequests] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	list, apiErr := h.services.ListJoinRequests(c.Request().Context(), *identity, groupID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, list)
}

// ApproveMember godoc
//
//	@Summary		Approve a join request
//	@Description	Group-admin only. Moves the membership `requested` -> `approved`. Only approved members count toward the fair-share split.
//	@Tags			membership
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"group id"	Format(uuid)
//	@Param			userId	path		string					true	"user id"	Format(uuid)
//	@Success		200		{object}	types.MembershipView	"the approved membership"
//	@Failure		400		{object}	types.apiError			"invalid group or user id"
//	@Failure		401		{object}	types.apiError			"missing or invalid token"
//	@Failure		403		{object}	types.apiError			"not this group's admin"
//	@Failure		404		{object}	types.apiError			"membership not found"
//	@Router			/groups/{id}/members/{userId}/approve [post]
func (h *Handler) ApproveMember(c echo.Context) error { return h.decideMember(c, true) }

// RejectMember godoc
//
//	@Summary		Reject a join request
//	@Description	Group-admin only. Moves the membership `requested` -> `rejected`.
//	@Tags			membership
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"group id"	Format(uuid)
//	@Param			userId	path		string					true	"user id"	Format(uuid)
//	@Success		200		{object}	types.MembershipView	"the rejected membership"
//	@Failure		400		{object}	types.apiError			"invalid group or user id"
//	@Failure		401		{object}	types.apiError			"missing or invalid token"
//	@Failure		403		{object}	types.apiError			"not this group's admin"
//	@Failure		404		{object}	types.apiError			"membership not found"
//	@Router			/groups/{id}/members/{userId}/reject [post]
func (h *Handler) RejectMember(c echo.Context) error { return h.decideMember(c, false) }

// PromoteToAdmin godoc
//
//	@Summary		Hand the group-admin role to another member
//	@Description	Group-admin only. Transfers the role to another approved member — there is always **exactly one** group-admin, so the caller is demoted to member in the same transaction. Logged to the audit trail.
//	@Tags			membership
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"group id"	Format(uuid)
//	@Param			userId	path		string					true	"the new admin's user id"	Format(uuid)
//	@Success		200		{object}	types.MembershipView	"the new admin's membership"
//	@Failure		400		{object}	types.apiError			"invalid group or user id"
//	@Failure		401		{object}	types.apiError			"missing or invalid token"
//	@Failure		403		{object}	types.apiError			"not this group's admin"
//	@Failure		404		{object}	types.apiError			"target is not an approved member"
//	@Router			/groups/{id}/members/{userId}/promote [post]
func (h *Handler) PromoteToAdmin(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[PromoteToAdmin] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	userID := c.Param("userId")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid user id"))
	}

	v, apiErr := h.services.TransferAdmin(c.Request().Context(), *identity, groupID, userID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, v)
}

// RemoveMember godoc
//
//	@Summary		Remove a member from a group
//	@Description	Group-admin only. A member with **any recorded expense cannot be removed** — doing so would corrupt the split.
//	@Tags			membership
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string	true	"group id"	Format(uuid)
//	@Param			userId	path	string	true	"user id"	Format(uuid)
//	@Success		204		"member removed"
//	@Failure		400		{object}	types.apiError	"invalid group or user id"
//	@Failure		401		{object}	types.apiError	"missing or invalid token"
//	@Failure		403		{object}	types.apiError	"not this group's admin"
//	@Failure		404		{object}	types.apiError	"membership not found"
//	@Failure		409		{object}	types.apiError	"member has recorded expenses"
//	@Router			/groups/{id}/members/{userId} [delete]
func (h *Handler) RemoveMember(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[RemoveMember] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	userID := c.Param("userId")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid user id"))
	}

	if apiErr := h.services.RemoveMember(c.Request().Context(), *identity, groupID, userID); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) decideMember(c echo.Context, approve bool) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[decideMember] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	userID := c.Param("userId")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid user id"))
	}

	var v *types.MembershipView
	var apiErr types.APIError
	if approve {
		v, apiErr = h.services.ApproveMember(c.Request().Context(), *identity, groupID, userID)
	} else {
		v, apiErr = h.services.RejectMember(c.Request().Context(), *identity, groupID, userID)
	}
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, v)
}
