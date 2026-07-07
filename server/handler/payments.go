package handler

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"expense-splitter/middleware"
	"expense-splitter/types"
)

const maxProofImageBytes = 5 << 20 // 5 MiB

// SubmitProof accepts either a JSON text note or a multipart image upload
// (form field "image") — the debtor's proof in the two-key workflow.
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

	if strings.HasPrefix(c.Request().Header.Get(echo.HeaderContentType), echo.MIMEMultipartForm) {
		fh, err := c.FormFile("image")
		if err != nil {
			return c.JSON(http.StatusBadRequest, types.NewBadRequestError(`multipart proof must include an "image" file field`))
		}
		if fh.Size > maxProofImageBytes {
			return c.JSON(http.StatusBadRequest, types.NewBadRequestError("image must be at most 5 MiB"))
		}
		f, err := fh.Open()
		if err != nil {
			return c.JSON(http.StatusBadRequest, types.NewBadRequestError("could not read uploaded file"))
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, maxProofImageBytes+1))
		if err != nil || int64(len(data)) > maxProofImageBytes {
			return c.JSON(http.StatusBadRequest, types.NewBadRequestError("image must be at most 5 MiB"))
		}

		p, apiErr := h.services.SubmitImageProof(c.Request().Context(), *identity, paymentID, data)
		if apiErr != nil {
			return c.JSON(apiErr.Status, apiErr)
		}
		return c.JSON(http.StatusOK, p)
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

func (h *Handler) GetProof(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[GetProof] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}
	paymentID := c.Param("id")
	if _, err := uuid.Parse(paymentID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid payment id"))
	}

	view, apiErr := h.services.GetProof(c.Request().Context(), *identity, paymentID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, view)
}

func (h *Handler) GetProofImage(c echo.Context) error {
	identity := middleware.GetIdentity(c)
	if identity == nil {
		h.logger.Error("[GetProofImage] missing identity in context")
		return c.JSON(http.StatusInternalServerError, types.NewServerError())
	}
	paymentID := c.Param("id")
	if _, err := uuid.Parse(paymentID); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid payment id"))
	}

	data, contentType, apiErr := h.services.GetProofImage(c.Request().Context(), *identity, paymentID)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.Blob(http.StatusOK, contentType, data)
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
