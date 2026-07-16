package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"expense-splitter/types"
)

// SignUp godoc
//
//	@Summary		Sign up a new user
//	@Description	Creates the Keycloak identity and the linked local user row. The new account starts at verification_status `registered` — it must be verified before it can join groups or record expenses.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		types.RegisterRequest	true	"email, password, name"
//	@Success		201		{object}	types.Principal			"the created account"
//	@Failure		400		{object}	types.apiError			"invalid body / weak password"
//	@Failure		409		{object}	types.apiError			"email already registered"
//	@Router			/auth/register [post]
func (h *Handler) SignUp(c echo.Context) error {
	var req types.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	p, apiErr := h.services.SignUp(c.Request().Context(), req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusCreated, p)
}

// Login godoc
//
//	@Summary		Log in
//	@Description	Exchanges email + password for an access token (~5 min) and a refresh token. Also returns your local `user` row, which is auto-provisioned/linked on first login. Copy the `access_token` into the **Authorize** button to use the rest of this page.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		types.LoginRequest	true	"email + password"
//	@Success		200		{object}	types.TokenResponse	"access_token, refresh_token, user"
//	@Failure		400		{object}	types.apiError		"invalid body"
//	@Failure		401		{object}	types.apiError		"bad credentials"
//	@Router			/auth/login [post]
func (h *Handler) Login(c echo.Context) error {
	var req types.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	tok, apiErr := h.services.Login(c.Request().Context(), req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, tok)
}

// Refresh godoc
//
//	@Summary		Refresh a session
//	@Description	Renews the access/refresh token pair without re-sending the password. Fails once the session has been revoked by logout.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		types.RefreshRequest	true	"refresh_token"
//	@Success		200		{object}	types.TokenResponse		"a fresh token pair"
//	@Failure		400		{object}	types.apiError			"invalid body"
//	@Failure		401		{object}	types.apiError			"refresh token is invalid or expired"
//	@Router			/auth/refresh [post]
func (h *Handler) Refresh(c echo.Context) error {
	var req types.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	tok, apiErr := h.services.Refresh(c.Request().Context(), req)
	if apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.JSON(http.StatusOK, tok)
}

// Logout godoc
//
//	@Summary		Log out
//	@Description	Revokes the session behind the given refresh token. Idempotent — logging out an already-revoked session still succeeds.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	types.LogoutRequest	true	"refresh_token"
//	@Success		204		"session revoked"
//	@Failure		400		{object}	types.apiError	"invalid body"
//	@Router			/auth/logout [post]
func (h *Handler) Logout(c echo.Context) error {
	var req types.LogoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.NewBadRequestError("invalid request body"))
	}
	if apiErr := req.Validate(); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}

	if apiErr := h.services.Logout(c.Request().Context(), req); apiErr != nil {
		return c.JSON(apiErr.Status, apiErr)
	}
	return c.NoContent(http.StatusNoContent)
}
