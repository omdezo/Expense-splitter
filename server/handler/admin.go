package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

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
