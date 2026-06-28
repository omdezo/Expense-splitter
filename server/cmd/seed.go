package cmd

import (
	"github.com/spf13/cobra"

	"expense-splitter/database/seeding"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed baseline data (the default global admin)",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := NewApp()
		if err != nil {
			return err
		}
		defer app.Close()

		return seeding.New(app.DB.Pool, app.Cfg, app.Logger).Run(cmd.Context())
	},
}
