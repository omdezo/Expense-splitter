package cmd

import (
	"log"

	"go.uber.org/zap"

	"expense-splitter/config"
	"expense-splitter/database"
	"expense-splitter/database/migration"
	"expense-splitter/handler"
	"expense-splitter/middleware"
	"expense-splitter/router"
	"expense-splitter/services"
)

func Execute() {
	cfg := config.Load()

	zl, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = zl.Sync() }()
	logger := zl.Sugar()

	db, err := database.New(cfg.Postgres.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migration.Run(db.Pool); err != nil {
		log.Fatal(err)
	}

	svc := services.New(db.Pool, logger)
	h := handler.New(svc, logger)
	auth := middleware.NewAuth(cfg.Keycloak)

	e := router.New(h, auth)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
