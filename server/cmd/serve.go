package cmd

import (
	"github.com/spf13/cobra"

	"expense-splitter/handler"
	"expense-splitter/keycloak"
	"expense-splitter/middleware"
	"expense-splitter/router"
	"expense-splitter/services"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := NewApp()
		if err != nil {
			return err
		}
		defer app.Close()

		kc := keycloak.New(app.Cfg.Keycloak)
		svc := services.New(app.DB.Pool, app.Logger, kc)
		h := handler.New(svc, app.Logger)
		auth := middleware.NewAuth(app.Cfg.Keycloak)

		e := router.New(h, auth)
		return e.Start(":" + app.Cfg.Port)
	},
}
