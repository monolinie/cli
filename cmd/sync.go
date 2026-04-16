package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/home"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Dokploy projects with the home app database",
	Long:  "Reconcile Dokploy projects with the home app database. Creates missing records and reports orphaned ones.",
	Args:  cobra.NoArgs,
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := config.Init(); err != nil {
		return err
	}

	homeURL := config.Get("home_url")
	homeKey := config.Get("home_api_key")
	if homeURL == "" || homeKey == "" {
		return fmt.Errorf("home_url and home_api_key must be configured\nRun:\n  monolinie config set home_url <url>\n  monolinie config set home_api_key <key>")
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	bold.Println("\n→ Syncing with home app...")

	hc := home.NewClient(homeURL, homeKey)
	result, err := hc.Sync()
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Println()

	if len(result.Created) > 0 {
		green.Printf("  Created %d project(s):\n", len(result.Created))
		for _, p := range result.Created {
			fmt.Printf("    + %s (ID: %s)\n", p.Name, p.ID)
		}
	}

	if len(result.Updated) > 0 {
		green.Printf("  Updated %d project(s):\n", len(result.Updated))
		for _, p := range result.Updated {
			fmt.Printf("    ~ %s: %s\n", p.Name, p.Field)
		}
	}

	if len(result.Orphaned) > 0 {
		yellow.Printf("  Orphaned %d project(s) (Dokploy project missing):\n", len(result.Orphaned))
		for _, p := range result.Orphaned {
			fmt.Printf("    ? %s (DB ID: %s, Dokploy ID: %s)\n", p.Name, p.ID, p.DokployProjectID)
		}
	}

	if len(result.Errors) > 0 {
		color.Red("  Errors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("    ✗ %s: %s\n", e.Name, e.Error)
		}
	}

	fmt.Printf("\n  Unchanged: %d\n\n", result.Unchanged)

	return nil
}
