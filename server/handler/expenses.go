package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// RecordExpense godoc
//
//	@Summary		Record an expense you paid
//	@Description	Approved members only, while the group is **open**. `paid_by` must equal the caller — you can only record what *you* paid.
//	@Description
//	@Description	`amount` is integer **baisa** (1.000 OMR = 1000) and must be positive. `occurred_on` must fall inside the trip's date range. `category` is one of: lodging, fuel, food, transport, other.
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"group id"	Format(uuid)
//	@Param			body	body		types.RecordExpenseRequest	true	"amount (baisa), category, description, occurred_on, paid_by"
//	@Success		201		{object}	types.Expense				"the recorded expense"
//	@Failure		400		{object}	types.apiError				"bad amount/date/category, or paid_by is not the caller"
//	@Failure		401		{object}	types.apiError				"missing or invalid token"
//	@Failure		403		{object}	types.apiError				"not an approved member"
//	@Failure		404		{object}	types.apiError				"group not found"
//	@Failure		409		{object}	types.apiError				"group is closed"
//	@Router			/groups/{id}/expenses [post]
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

// ListExpenses godoc
//
//	@Summary		List a group's expenses
//	@Description	Members and admins only. Filter by category or payer, and search descriptions with `q`. Returns a `{total, limit, offset, items}` envelope where `items` is an array of expenses.
//	@Description
//	@Description	Descriptions are truncated to 80 characters, cut on a word boundary so the last word stays whole (e.g. `"dinner at the ..."`).
//	@Tags			expenses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string		true	"group id"	Format(uuid)
//	@Param			category	query		string		false	"filter by category"	Enums(lodging, fuel, food, transport, other)
//	@Param			paid_by		query		string		false	"filter by payer's user id"	Format(uuid)
//	@Param			q			query		string		false	"search in description"
//	@Param			limit		query		int			false	"page size (1-200)"	default(50)
//	@Param			offset		query		int			false	"rows to skip"	default(0)
//	@Success		200			{object}	types.Page	"paginated expenses"
//	@Failure		400			{object}	types.apiError	"invalid id, filter, or paging"
//	@Failure		401			{object}	types.apiError	"missing or invalid token"
//	@Failure		403			{object}	types.apiError	"not a member of this group"
//	@Failure		404			{object}	types.apiError	"group not found"
//	@Router			/groups/{id}/expenses [get]
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

// DeleteExpense godoc
//
//	@Summary		Delete an expense
//	@Description	You may delete **only your own** expenses, and only while the group is open.
//	@Tags			expenses
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path	string	true	"group id"		Format(uuid)
//	@Param			expenseId	path	string	true	"expense id"	Format(uuid)
//	@Success		204			"expense deleted"
//	@Failure		400			{object}	types.apiError	"invalid group or expense id"
//	@Failure		401			{object}	types.apiError	"missing or invalid token"
//	@Failure		403			{object}	types.apiError	"not your expense"
//	@Failure		404			{object}	types.apiError	"expense not found"
//	@Failure		409			{object}	types.apiError	"group is closed"
//	@Router			/groups/{id}/expenses/{expenseId} [delete]
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

// UpdateExpense godoc
//
//	@Summary		Update an expense
//	@Description	You may edit **only your own** expenses, and only while the group is open. Any change to the amount is written to the audit trail with **before/after** values.
//	@Tags			expenses
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string						true	"group id"		Format(uuid)
//	@Param			expenseId	path		string						true	"expense id"	Format(uuid)
//	@Param			body		body		types.UpdateExpenseRequest	true	"fields to change"
//	@Success		200			{object}	types.Expense				"the updated expense"
//	@Failure		400			{object}	types.apiError				"invalid ids or body"
//	@Failure		401			{object}	types.apiError				"missing or invalid token"
//	@Failure		403			{object}	types.apiError				"not your expense"
//	@Failure		404			{object}	types.apiError				"expense not found"
//	@Failure		409			{object}	types.apiError				"group is closed"
//	@Router			/groups/{id}/expenses/{expenseId} [patch]
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
