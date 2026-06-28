package router

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"expense-splitter/handler"
	appmw "expense-splitter/middleware"
)

func New(h *handler.Handler, auth *appmw.Auth) *echo.Echo {
	e := echo.New()
	e.Use(echomw.Logger(), echomw.Recover())

	// Public — no authentication.
	e.GET("/health", h.Health)

	// Authenticated — requires a valid Keycloak bearer token.
	e.GET("/me", h.Me, auth.Require())

	return e
}
