package cmd

import (
	"log"

	"expense-splitter/config"
	"expense-splitter/database"
	"expense-splitter/database/migration"
	"expense-splitter/router"
)

func Execute() {
	cfg := config.Load()

	db, err := database.New(cfg.Postgres.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migration.Run(db.Pool); err != nil {
		log.Fatal(err)
	}

	e := router.New(db.Pool)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
