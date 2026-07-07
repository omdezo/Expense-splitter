package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

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
