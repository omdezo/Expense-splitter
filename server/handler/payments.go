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
//
//	@Summary		Upload proof of payment (debtor)
//	@Description	**Key 1 of 2.** The debtor evidences that they paid their creditor in real life, moving the payment to `proof_submitted`. Also the way out of `disputed` — re-submit and both keys run again.
//	@Description
//	@Description	Accepts either shape:
//	@Description	- **JSON** `{"note": "paid 40 cash on day 3"}`
//	@Description	- **multipart/form-data** with an `image` file field (receipt or transfer screenshot)
//	@Description
//	@Description	Images are capped at 5 MiB and validated by **magic bytes** (jpeg/png/gif/webp) — never by file extension. The stored sha256 makes a later swap detectable.
//	@Tags			payments
//	@Accept			json
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"payment id"	Format(uuid)
//	@Param			body	body		types.SubmitProofRequest	false	"text note (JSON variant)"
//	@Param			image	formData	file						false	"receipt image, max 5 MiB (multipart variant)"
//	@Success		200		{object}	types.PaymentView			"payment now proof_submitted"
//	@Failure		400		{object}	types.apiError				"invalid id/body, oversized file, or not a real image"
//	@Failure		401		{object}	types.apiError				"missing or invalid token"
//	@Failure		403		{object}	types.apiError				"you are not this payment's debtor"
//	@Failure		404		{object}	types.apiError				"payment not found"
//	@Failure		409		{object}	types.apiError				"payment is not in a state that accepts proof"
//	@Router			/payments/{id}/proof [post]
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

// GetProof godoc
//
//	@Summary		Get a payment's proof metadata
//	@Description	Visible to the debtor, the creditor, and admins. Returns the note and/or the image's metadata — content type, size, and sha256. Use `/proof/image` for the bytes.
//	@Tags			payments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string			true	"payment id"	Format(uuid)
//	@Success		200	{object}	types.ProofView	"proof note and/or image metadata"
//	@Failure		400	{object}	types.apiError	"invalid payment id"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"not a party to this payment"
//	@Failure		404	{object}	types.apiError	"payment or proof not found"
//	@Router			/payments/{id}/proof [get]
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

// GetProofImage godoc
//
//	@Summary		Download the proof image bytes
//	@Description	Visible to the debtor, the creditor, and admins. Streams the stored image from object storage with its real content type.
//	@Tags			payments
//	@Produce		image/jpeg
//	@Produce		image/png
//	@Security		BearerAuth
//	@Param			id	path		string			true	"payment id"	Format(uuid)
//	@Success		200	{file}		binary			"the proof image"
//	@Failure		400	{object}	types.apiError	"invalid payment id"
//	@Failure		401	{object}	types.apiError	"missing or invalid token"
//	@Failure		403	{object}	types.apiError	"not a party to this payment"
//	@Failure		404	{object}	types.apiError	"no image proof on this payment"
//	@Router			/payments/{id}/proof/image [get]
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

// ConfirmPayment godoc
//
//	@Summary		Attest that you received the money (creditor)
//	@Description	**Key 2 of 2.** The creditor — the person owed — confirms the money actually arrived, moving `proof_submitted` -> `creditor_confirmed`. This is the *only* transition that produces `creditor_confirmed`, and only the creditor may call it.
//	@Tags			payments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"payment id"	Format(uuid)
//	@Success		200	{object}	types.PaymentView	"payment now creditor_confirmed"
//	@Failure		400	{object}	types.apiError		"invalid payment id"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Failure		403	{object}	types.apiError		"you are not this payment's creditor"
//	@Failure		404	{object}	types.apiError		"payment not found"
//	@Failure		409	{object}	types.apiError		"payment is not awaiting your confirmation"
//	@Router			/payments/{id}/confirm [post]
func (h *Handler) ConfirmPayment(c echo.Context) error {
	return h.paymentAction(c, "ConfirmPayment", h.services.ConfirmPayment)
}

// DisputePayment godoc
//
//	@Summary		Deny receiving the money (creditor)
//	@Description	The creditor says the money never arrived, moving the payment to `disputed`. A disputed payment can **never** be reported settled — the debtor must re-submit proof and both keys must run again.
//	@Tags			payments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"payment id"	Format(uuid)
//	@Success		200	{object}	types.PaymentView	"payment now disputed"
//	@Failure		400	{object}	types.apiError		"invalid payment id"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Failure		403	{object}	types.apiError		"you are not this payment's creditor"
//	@Failure		404	{object}	types.apiError		"payment not found"
//	@Failure		409	{object}	types.apiError		"payment is not in a disputable state"
//	@Router			/payments/{id}/dispute [post]
func (h *Handler) DisputePayment(c echo.Context) error {
	return h.paymentAction(c, "DisputePayment", h.services.DisputePayment)
}

// FinalizePayment godoc
//
//	@Summary		Finalize a payment as settled (admin)
//	@Description	The group-admin (own group) or global admin (any group) marks a `creditor_confirmed` payment as **settled** — the only transition that produces `settled`, and it is terminal.
//	@Description
//	@Description	A debtor can never settle their own payment, **even if they are also the group-admin**. The global admin's override may finalize any non-settled payment and can do nothing else.
//	@Tags			payments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"payment id"	Format(uuid)
//	@Success		200	{object}	types.PaymentView	"payment now settled"
//	@Failure		400	{object}	types.apiError		"invalid payment id"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Failure		403	{object}	types.apiError		"not an admin for this payment's group, or you are the debtor"
//	@Failure		404	{object}	types.apiError		"payment not found"
//	@Failure		409	{object}	types.apiError		"payment has not been confirmed by the creditor"
//	@Router			/payments/{id}/finalize [post]
func (h *Handler) FinalizePayment(c echo.Context) error {
	return h.paymentAction(c, "FinalizePayment", h.services.FinalizePayment)
}

// RejectPayment godoc
//
//	@Summary		Reject a payment back to disputed (admin)
//	@Description	The group-admin (own group) or global admin (any group) sends a payment to `disputed`, returning it to the debtor to re-submit proof. `settled` stays terminal — even for the global admin.
//	@Tags			payments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string				true	"payment id"	Format(uuid)
//	@Success		200	{object}	types.PaymentView	"payment now disputed"
//	@Failure		400	{object}	types.apiError		"invalid payment id"
//	@Failure		401	{object}	types.apiError		"missing or invalid token"
//	@Failure		403	{object}	types.apiError		"not an admin for this payment's group"
//	@Failure		404	{object}	types.apiError		"payment not found"
//	@Failure		409	{object}	types.apiError		"payment is already settled"
//	@Router			/payments/{id}/reject [post]
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
