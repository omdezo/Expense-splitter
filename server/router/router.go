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

	// Public auth endpoints that wrap Keycloak (no bearer token required).
	e.POST("/auth/register", h.SignUp)
	e.POST("/auth/login", h.Login)

	// Public group status by shareable token — the ONLY unauthenticated group view.
	e.GET("/public/groups/:token", h.PublicGroupStatus)

	e.GET("/me", h.Me, auth.Require())
	e.POST("/register", h.Register, auth.Require())
	e.POST("/verification", h.SubmitVerification, auth.Require())
	e.GET("/groups", h.ListMyGroups, auth.Require())
	e.POST("/groups", h.CreateGroup, auth.Require())
	e.POST("/groups/join", h.JoinGroup, auth.Require())
	e.GET("/groups/:id", h.GetGroup, auth.Require())
	e.PATCH("/groups/:id", h.UpdateGroup, auth.Require())
	e.POST("/groups/:id/close", h.CloseGroup, auth.Require())
	e.GET("/groups/:id/settlement", h.GetSettlementPlan, auth.Require())
	e.GET("/groups/:id/summary", h.GetGroupSummary, auth.Require())
	e.POST("/groups/:id/expenses", h.RecordExpense, auth.Require())
	e.GET("/groups/:id/expenses", h.ListExpenses, auth.Require())
	e.PATCH("/groups/:id/expenses/:expenseId", h.UpdateExpense, auth.Require())
	e.DELETE("/groups/:id/expenses/:expenseId", h.DeleteExpense, auth.Require())
	e.GET("/groups/:id/audit", h.GetGroupAudit, auth.Require())
	e.GET("/groups/:id/requests", h.ListJoinRequests, auth.Require())
	e.POST("/groups/:id/members/:userId/approve", h.ApproveMember, auth.Require())
	e.POST("/groups/:id/members/:userId/reject", h.RejectMember, auth.Require())
	e.POST("/groups/:id/members/:userId/promote", h.PromoteToAdmin, auth.Require())
	e.DELETE("/groups/:id/members/:userId", h.RemoveMember, auth.Require())

	e.POST("/payments/:id/proof", h.SubmitProof, auth.Require())
	e.POST("/payments/:id/confirm", h.ConfirmPayment, auth.Require())
	e.POST("/payments/:id/dispute", h.DisputePayment, auth.Require())
	e.POST("/payments/:id/finalize", h.FinalizePayment, auth.Require())
	e.POST("/payments/:id/reject", h.RejectPayment, auth.Require())

	e.POST("/admin/users/:id/approve", h.ApproveUser, auth.Require())
	e.POST("/admin/users/:id/reject", h.RejectUser, auth.Require())

	return e
}
