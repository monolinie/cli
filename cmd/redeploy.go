package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/spf13/cobra"
)

var redeployCmd = &cobra.Command{
	Use:   "redeploy <project-name> [app-name]",
	Short: "Redeploy a project",
	Long:  "Trigger a fresh deployment for a project. Optionally specify which app.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRedeploy,
}

func init() {
	rootCmd.AddCommand(redeployCmd)
}

func runRedeploy(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))

	project, err := findProjectByName(dk, name)
	if err != nil {
		return err
	}

	appName := ""
	if len(args) > 1 {
		appName = args[1]
	}
	app, err := findAppInProject(project, appName)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	bold.Printf("\n→ Redeploying %s...\n", name)
	if err := dk.DeployApplication(app.ApplicationID); err != nil {
		return fmt.Errorf("redeploy: %w", err)
	}
	green.Println("  ✓ Deployment triggered")

	fmt.Printf("\n  Track progress: %s\n\n", config.Get("dokploy_url"))

	return nil
}
