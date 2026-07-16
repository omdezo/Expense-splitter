package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// AdminListUsers godoc
//
//	@Summary		List all users
//	@Description	Global admin only. Filter by verification status to work the approval queue. Returns a `{total, limit, offset, items}` envelope.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status	query		string		false	"filter by verification status"	Enums(registered, pending_verification, verified, rejected)
//	@Param			limit	query		int			false	"page size (1-200)"	default(50)
//	@Param			offset	query		int			false	"rows to skip"	default(0)
//	@Success		200		{object}	types.Page	"paginated users"
//	@Failure		400		{object}	types.apiError	"invalid status or paging"
//	@Failure		401		{object}	types.apiError	"missing or invalid token"
//	@Failure		403		{object}	types.apiError	"global admin only"
//	@Router			/admin/users [get]
func (h *Handler) AdminListUsers(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[AdminListUsers] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	limit, offset, apiErr := parsePage(c)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	page, apiErr := h.services.AdminListUsers(c.Request().Context(), *identity, c.QueryParam("status"), limit, offset)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, page)
}

// AdminGetUser godoc
//
//	@Summary		Get one user's full detail
//	@Description	Global admin only. The account plus its group memberships and roles.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string					true	"user id"	Format(uuid)
//	@Success		200	{object}	types.AdminUserDetail	"the user and their memberships"
//	@Failure		400	{object}	types.apiError			"invalid user id"
//	@Failure		401	{object}	types.apiError			"missing or invalid token"
//	@Failure		403	{object}	types.apiError			"global admin only"
//	@Failure		404	{object}	types.apiError			"user not found"
//	@Router			/admin/users/{id} [get]
func (h *Handler) AdminGetUser(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[AdminGetUser] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	userID := c.Param("id")
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid user id"))
	}

	detail, apiErr := h.services.AdminGetUser(c.Request().Context(), *identity, userID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, detail)
}

// AdminDeleteUser godoc
//
//	@Summary		Delete a user
//	@Description	Global admin only. Refused when the user has recorded expenses or is party to a payment — removing them would corrupt a split.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"user id"	Format(uuid)
//	@Success		204	"user deleted"
//	@Failure		400	{object}	types.apiError	"invalid user id"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"global admin only"
//	@Failure		404	{object}	types.apiError	"user not found"
//	@Failure		409	{object}	types.apiError	"user has expenses or payments"
//	@Router			/admin/users/{id} [delete]
func (h *Handler) AdminDeleteUser(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[AdminDeleteUser] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	userID := c.Param("id")
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid user id"))
	}

	if apiErr := h.services.AdminDeleteUser(c.Request().Context(), *identity, userID); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.NoContent(http.StatusNoContent)
}

// AdminListGroups godoc
//
//	@Summary		List all groups
//	@Description	Global admin only. Every group in the system, regardless of membership. Returns a `{total, limit, offset, items}` envelope.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int			false	"page size (1-200)"	default(50)
//	@Param			offset	query		int			false	"rows to skip"	default(0)
//	@Success		200		{object}	types.Page	"paginated groups"
//	@Failure		400		{object}	types.apiError	"invalid paging"
//	@Failure		401		{object}	types.apiError	"missing or invalid token"
//	@Failure		403		{object}	types.apiError	"global admin only"
//	@Router			/admin/groups [get]
func (h *Handler) AdminListGroups(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[AdminListGroups] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	limit, offset, apiErr := parsePage(c)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	page, apiErr := h.services.AdminListGroups(c.Request().Context(), *identity, limit, offset)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, page)
}

// AdminDeleteGroup godoc
//
//	@Summary		Delete a group
//	@Description	Global admin only, and **pristine groups only** — a group with any recorded expense or computed payment cannot be deleted.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"group id"	Format(uuid)
//	@Success		204	"group deleted"
//	@Failure		400	{object}	types.apiError	"invalid group id"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"global admin only"
//	@Failure		404	{object}	types.apiError	"group not found"
//	@Failure		409	{object}	types.apiError	"group is not pristine"
//	@Router			/admin/groups/{id} [delete]
func (h *Handler) AdminDeleteGroup(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[AdminDeleteGroup] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	if apiErr := h.services.AdminDeleteGroup(c.Request().Context(), *identity, groupID); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.NoContent(http.StatusNoContent)
}
