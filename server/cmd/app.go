package cmd

import (
	"go.uber.org/zap"

	"expense-splitter/config"
	"expense-splitter/database"
	"expense-splitter/database/migration"
	"expense-splitter/types"
)

type App struct {
	Cfg    *types.Config
	Logger *types.Logger
	DB     *database.DB
}

func NewApp() (*App, error) {
	cfg := config.Load()

	zl, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	logger := zl.Sugar()

	db, err := database.New(cfg.Postgres.DSN())
	if err != nil {
		return nil, err
	}

	if err := migration.Run(db.Pool); err != nil {
		db.Close()
		return nil, err
	}

	return &App{Cfg: cfg, Logger: logger, DB: db}, nil
}

func (a *App) Close() {
	a.DB.Close()
	_ = a.Logger.Sync()
}
