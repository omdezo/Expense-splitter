package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"expense-splitter/services"
	"expense-splitter/types"
)

type Handler struct {
	services *services.Services
	logger   *types.Logger
}

func New(svc *services.Services, logger *types.Logger) *Handler {
	return &Handler{services: svc, logger: logger}
}

// parsePage reads ?limit= and ?offset= with sane defaults and caps.
func parsePage(c echo.Context) (limit, offset int, apiErr types.APIError) {
	limit = types.PageDefaultLimit
	if raw := c.QueryParam("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > types.PageMaxLimit {
			return 0, 0, types.NewBadRequestError("limit must be an integer between 1 and 200")
		}
		limit = v
	}
	if raw := c.QueryParam("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			return 0, 0, types.NewBadRequestError("offset must be a non-negative integer")
		}
		offset = v
	}
	return limit, offset, nil
}
