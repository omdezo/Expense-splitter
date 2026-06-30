package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

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

func (h *Handler) ApproveMember(c echo.Context) error { return h.decideMember(c, true) }

func (h *Handler) RejectMember(c echo.Context) error { return h.decideMember(c, false) }

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
