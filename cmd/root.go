package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "monolinie",
	Aliases: []string{"ml"},
	Short:   "Monolinie studio CLI",
	Long:    "CLI tool for automating project setup in the Monolinie studio.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
