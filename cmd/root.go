package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "monolinie",
	Short: "Monolinie studio CLI",
	Long:  "CLI tool for automating project setup in the Monolinie studio.",
}

func Execute() {
	// Allow invoking as "ml" (short alias) or "monolinie"
	if name := filepath.Base(os.Args[0]); name == "ml" {
		rootCmd.Use = "ml"
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
