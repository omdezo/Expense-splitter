package cmd

import (
	"log"

	"expense-splitter/config"
	"expense-splitter/database"
	"expense-splitter/router"
)

// Execute wires up config, database, and router, then starts the HTTP server.
func Execute() {
	cfg := config.Load()

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	e := router.New(db)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
