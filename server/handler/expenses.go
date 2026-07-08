package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

func (h *Handler) RecordExpense(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[RecordExpense] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	var req types.RecordExpenseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	e, apiErr := h.services.RecordExpense(c.Request().Context(), *identity, groupID, req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusCreated, e)
}

func (h *Handler) ListExpenses(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[ListExpenses] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}

	var filter types.ExpenseFilter
	if cat := c.QueryParam("category"); cat != "" {
		filter.Category = types.ExpenseCategory(cat)
		if !filter.Category.Valid() {
			return c.JSON(http.StatusBadRequest, types.NewBadRequestError("category must be one of: lodging, fuel, food, transport, other"))
		}
	}
	if payer := c.QueryParam("paid_by"); payer != "" {
		if _, err := uuid.Parse(payer); err != nil {
			return c.JSON(http.StatusBadRequest, types.NewBadRequestError("paid_by must be a valid user id"))
		}
		filter.PaidBy = payer
	}
	filter.Search = c.QueryParam("q")

	limit, offset, apiErr := parsePage(c)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	page, apiErr := h.services.ListExpenses(c.Request().Context(), *identity, groupID, filter, limit, offset)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, page)
}

func (h *Handler) DeleteExpense(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[DeleteExpense] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	expenseID := c.Param("expenseId")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}
	if _, err := uuid.Parse(expenseID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid expense id"))
	}

	if apiErr := h.services.DeleteExpense(c.Request().Context(), *identity, groupID, expenseID); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UpdateExpense(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[UpdateExpense] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	groupID := c.Param("id")
	expenseID := c.Param("expenseId")
	if _, err := uuid.Parse(groupID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid group id"))
	}
	if _, err := uuid.Parse(expenseID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid expense id"))
	}

	var req types.UpdateExpenseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	e, apiErr := h.services.UpdateExpense(c.Request().Context(), *identity, groupID, expenseID, req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, e)
}
