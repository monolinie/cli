package cmd

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dokploy"
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
	if err := config.Validate(); err != nil {
		return err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))

	project, err := findProjectByName(dk, name)
	if err != nil {
		return err
	}

	app, err := findAppInProject(project, "")
	if err != nil {
		return err
	}

	domains, err := dk.GetDomainsByApplication(app.ApplicationID)
	if err != nil {
		return fmt.Errorf("get domains: %w", err)
	}

	bold := color.New(color.Bold)
	bold.Printf("Status for %s\n\n", project.Name)

	if len(domains) == 0 {
		fmt.Println("  No domains configured.")
	}

	for _, d := range domains {
		host := d.Host
		scheme := "http"
		if d.HTTPS {
			scheme = "https"
		}

		// Check DNS
		fmt.Printf("  DNS (%s): ", host)
		ips, err := net.LookupHost(host)
		if err != nil {
			color.Red("✗ not resolving")
		} else {
			color.Green("✓ %s", ips[0])
		}

		// Check HTTP
		url := fmt.Sprintf("%s://%s", scheme, host)
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
		fmt.Println()
	}

	// Links
	fmt.Printf("  Repo:    https://github.com/%s/%s\n", config.Get("github_org"), project.Name)
	fmt.Printf("  Dokploy: %s\n", config.Get("dokploy_url"))
	fmt.Println()

	return nil
}
