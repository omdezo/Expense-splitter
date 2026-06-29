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

	e.GET("/health", h.Health)

	e.GET("/me", h.Me, auth.Require())
	e.POST("/register", h.Register, auth.Require())
	e.POST("/verification", h.SubmitVerification, auth.Require())
	e.POST("/groups", h.CreateGroup, auth.Require())
	e.POST("/groups/join", h.JoinGroup, auth.Require())
	e.GET("/groups/:id/requests", h.ListJoinRequests, auth.Require())
	e.POST("/groups/:id/members/:userId/approve", h.ApproveMember, auth.Require())
	e.POST("/groups/:id/members/:userId/reject", h.RejectMember, auth.Require())

	e.POST("/admin/users/:id/approve", h.ApproveUser, auth.Require())
	e.POST("/admin/users/:id/reject", h.RejectUser, auth.Require())

	return e
}
