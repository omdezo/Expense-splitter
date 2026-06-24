package router

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"expense-splitter/handler"
	"expense-splitter/types"
)

func New(db *types.DBPool) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger(), middleware.Recover())

	e.GET("/health", handler.Health(db))

	return e
}
