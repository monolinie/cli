package cmd

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <project-name>",
	Short: "Check project status",
	Long:  "Check if a project's domain resolves and the app responds.",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := validateProjectName(name); err != nil {
		return err
	}

	if err := config.Init(); err != nil {
		return err
	}

	domain := config.Get("domain")
	host := fmt.Sprintf("%s.preview.%s", name, domain)
	bold := color.New(color.Bold)

	bold.Printf("Status for %s\n\n", name)

	// Check DNS
	fmt.Printf("  DNS (%s): ", host)
	ips, err := net.LookupHost(host)
	if err != nil {
		color.Red("✗ not resolving")
	} else {
		color.Green("✓ %s", ips[0])
	}

	// Check HTTP
	url := "https://" + host
	fmt.Printf("  HTTPS (%s): ", url)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		color.Red("✗ %s", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode < 400 {
			color.Green("✓ %d", resp.StatusCode)
		} else {
			color.Yellow("⚠ %d", resp.StatusCode)
		}
	}

	// Links
	fmt.Println()
	fmt.Printf("  Repo:    https://github.com/%s/%s\n", config.Get("github_org"), name)
	fmt.Printf("  URL:     %s\n", url)
	fmt.Printf("  Dokploy: %s\n", config.Get("dokploy_url"))
	fmt.Println()

	return nil
}
