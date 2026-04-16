package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  "Set, get, and list configuration values stored in ~/.ml/config.yaml",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ReplaceAll(args[0], "-", "_")
		value := args[1]

		if err := config.Init(); err != nil {
			return err
		}

		if err := config.Set(key, value); err != nil {
			return err
		}

		color.Green("✓ Set %s", key)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ReplaceAll(args[0], "-", "_")

		if err := config.Init(); err != nil {
			return err
		}

		value := config.Get(key)
		if value == "" {
			color.Yellow("(not set)")
		} else {
			fmt.Println(value)
		}
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Init(); err != nil {
			return err
		}

		bold := color.New(color.Bold)
		for _, key := range config.ValidKeys() {
			value := config.Get(key)
			if value == "" {
				bold.Printf("%-20s", key)
				color.Yellow(" (not set)")
			} else {
				// Mask sensitive values
				display := value
				if strings.Contains(key, "key") || strings.Contains(key, "token") {
					if len(value) > 8 {
						display = value[:4] + "..." + value[len(value)-4:]
					}
				}
				bold.Printf("%-20s", key)
				fmt.Printf(" %s\n", display)
			}
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd, configGetCmd, configListCmd)
	rootCmd.AddCommand(configCmd)
}
