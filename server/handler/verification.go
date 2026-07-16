package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

// SubmitVerification godoc
//
//	@Summary		Submit your account for verification
//	@Description	Moves your account `registered` -> `pending_verification`, putting it in the global admin's queue. You must reach `verified` before you can join a group, record expenses, or be settled against.
//	@Tags			account
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	types.Principal	"account now pending_verification"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"caller is not provisioned"
//	@Failure		409	{object}	types.apiError	"account is not in a submittable state"
//	@Router			/verification [post]
func (h *Handler) SubmitVerification(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[SubmitVerification] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	p, apiErr := h.services.SubmitVerification(c.Request().Context(), *identity)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, p)
}

// ApproveUser godoc
//
//	@Summary		Approve a user's verification
//	@Description	Global admin only. Marks the target account `verified`, unlocking group participation.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string			true	"user id"	Format(uuid)
//	@Success		200	{object}	types.Principal	"the verified account"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"global admin only"
//	@Failure		404	{object}	types.apiError	"user not found"
//	@Router			/admin/users/{id}/approve [post]
func (h *Handler) ApproveUser(c echo.Context) error {
	return h.setVerification(c, types.VerificationVerified)
}

// RejectUser godoc
//
//	@Summary		Reject a user's verification
//	@Description	Global admin only. Marks the target account `rejected` — it stays blocked from joining groups or recording expenses.
//	@Tags			admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string			true	"user id"	Format(uuid)
//	@Success		200	{object}	types.Principal	"the rejected account"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"global admin only"
//	@Failure		404	{object}	types.apiError	"user not found"
//	@Router			/admin/users/{id}/reject [post]
func (h *Handler) RejectUser(c echo.Context) error {
	return h.setVerification(c, types.VerificationRejected)
}

func (h *Handler) setVerification(c echo.Context, decision types.VerificationStatus) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[setVerification] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	targetID := c.Param("id")
	if targetID == "" {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("missing user id"))
	}

	p, apiErr := h.services.SetVerification(c.Request().Context(), *identity, targetID, decision)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, p)
}
