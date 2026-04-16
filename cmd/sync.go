package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/home"
	"github.com/spf13/cobra"
)

var flagPrune bool

var syncCmd = &cobra.Command{
	Use:   "sync <local|prod>",
	Short: "Sync Dokploy projects with the home app database",
	Long:  "Reconcile Dokploy projects with the home app database. Creates missing records and reports orphaned ones.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&flagPrune, "prune", false, "Remove orphaned projects from the home app")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := config.Init(); err != nil {
		return err
	}

	hc, err := resolveHomeClient(args[0])
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	bold.Printf("\n→ Syncing with home app (%s)...\n", args[0])
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
		if flagPrune {
			yellow.Printf("  Pruning %d orphaned project(s):\n", len(result.Orphaned))
			for _, p := range result.Orphaned {
				_, err := hc.DeregisterProject(home.DeregisterInput{
					DokployProjectID: p.DokployProjectID,
					Name:             p.Name,
				})
				if err != nil {
					fmt.Printf("    ✗ %s: %v\n", p.Name, err)
				} else {
					green.Printf("    - %s removed\n", p.Name)
				}
			}
		} else {
			yellow.Printf("  Orphaned %d project(s) (Dokploy project missing):\n", len(result.Orphaned))
			for _, p := range result.Orphaned {
				fmt.Printf("    ? %s (DB ID: %s, Dokploy ID: %s)\n", p.Name, p.ID, p.DokployProjectID)
			}
			fmt.Println("  Run with --prune to remove them")
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
