package router

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"expense-splitter/handler"
)

func New(db *pgxpool.Pool) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger(), middleware.Recover())

	e.GET("/health", handler.Health(db))

	return e
}
