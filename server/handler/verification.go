package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

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

func (h *Handler) ApproveUser(c echo.Context) error {
	return h.setVerification(c, types.VerificationVerified)
}

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
