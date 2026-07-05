package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

func (h *Handler) SubmitProof(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[SubmitProof] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	paymentID := c.Param("id")
	if _, err := uuid.Parse(paymentID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid payment id"))
	}

	var req types.SubmitProofRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	p, apiErr := h.services.SubmitProof(c.Request().Context(), *identity, paymentID, req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, p)
}

func (h *Handler) ConfirmPayment(c echo.Context) error {
	return h.paymentAction(c, "ConfirmPayment", h.services.ConfirmPayment)
}

func (h *Handler) DisputePayment(c echo.Context) error {
	return h.paymentAction(c, "DisputePayment", h.services.DisputePayment)
}

func (h *Handler) FinalizePayment(c echo.Context) error {
	return h.paymentAction(c, "FinalizePayment", h.services.FinalizePayment)
}

func (h *Handler) RejectPayment(c echo.Context) error {
	return h.paymentAction(c, "RejectPayment", h.services.RejectPayment)
}

func (h *Handler) paymentAction(c echo.Context, name string, fn func(context.Context, types.Identity, string) (*types.PaymentView, types.APIError)) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Errorf("[%s] missing identity in context", name)
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}

	paymentID := c.Param("id")
	if _, err := uuid.Parse(paymentID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid payment id"))
	}

	p, apiErr := fn(c.Request().Context(), *identity, paymentID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, p)
}
