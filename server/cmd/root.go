package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "expense-splitter",
	Short: "Expense Splitter backend",
	Long:  "Expense Splitter — a backend API for splitting shared trip expenses.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd, seedCmd)
}
