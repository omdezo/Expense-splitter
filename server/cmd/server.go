package cmd

import (
	"log"

	"expense-splitter/config"
	"expense-splitter/database"
	"expense-splitter/router"
)

func Execute() {
	cfg := config.Load()

	db, err := database.New(cfg.Postgres.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	e := router.New(db)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
