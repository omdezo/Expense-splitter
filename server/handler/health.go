package handler

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

func Health(db *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := db.Ping(context.Background()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": "down", "database": "down"})
		}
		return c.JSON(http.StatusOK, echo.Map{"status": "ok", "database": "up"})
	}
}
