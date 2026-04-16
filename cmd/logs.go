package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/spf13/cobra"
)

var flagLogLines int

var logsCmd = &cobra.Command{
	Use:   "logs <project-name> [app-name]",
	Short: "View deployment logs",
	Long:  "Show the latest deployment log for a project. Optionally specify which app.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().IntVarP(&flagLogLines, "lines", "n", 0, "Number of lines to show (0 = all)")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
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

	deployments, err := dk.GetDeployments(app.ApplicationID)
	if err != nil {
		return fmt.Errorf("get deployments: %w", err)
	}

	if len(deployments) == 0 {
		fmt.Println("No deployments found.")
		return nil
	}

	latest := deployments[0]
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)

	bold.Printf("Deployment: %s\n", latest.Title)
	dim.Printf("Status: %s | Created: %s\n\n", latest.Status, latest.CreatedAt)

	if latest.LogPath == "" {
		fmt.Println("No log available for this deployment.")
		return nil
	}

	logContent, err := dk.GetDeploymentLog(latest.LogPath)
	if err != nil {
		return fmt.Errorf("read log: %w", err)
	}

	if flagLogLines > 0 {
		lines := splitLines(logContent)
		if len(lines) > flagLogLines {
			lines = lines[len(lines)-flagLogLines:]
		}
		for _, line := range lines {
			fmt.Println(line)
		}
	} else {
		fmt.Print(logContent)
	}

	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
