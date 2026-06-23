package main

import (
	"log"

	"expense-splitter/config"
	"expense-splitter/database"
	"expense-splitter/router"
)

func main() {
	cfg := config.Load()

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	e := router.New(db)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
