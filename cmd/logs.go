package cmd

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

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

	serverIP := config.Get("dokploy_server_ip")
	if serverIP == "" {
		return fmt.Errorf("dokploy_server_ip not configured — run: ml config set dokploy_server_ip <ip>")
	}

	safeLogPath := regexp.MustCompile(`^[a-zA-Z0-9/_.\-:]+$`)
	if !safeLogPath.MatchString(latest.LogPath) || strings.Contains(latest.LogPath, "..") {
		return fmt.Errorf("refusing to read log: path contains unexpected characters: %s", latest.LogPath)
	}
	if !strings.HasPrefix(latest.LogPath, "/etc/dokploy/logs/") {
		return fmt.Errorf("refusing to read log: unexpected path prefix: %s", latest.LogPath)
	}

	readCmd := fmt.Sprintf("cat %s", latest.LogPath)
	if flagLogLines > 0 {
		readCmd = fmt.Sprintf("tail -n %d %s", flagLogLines, latest.LogPath)
	}

	out, err := exec.Command("ssh", "-o", "StrictHostKeyChecking=accept-new", "root@"+serverIP, readCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("read log via SSH: %s", string(out))
	}

	fmt.Print(string(out))

	return nil
}
