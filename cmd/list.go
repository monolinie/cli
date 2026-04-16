package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long:  "List all projects deployed in Dokploy.",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))
	projects, err := dk.GetProjects()
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)

	bold.Printf("\n  %-20s %-12s %-12s %s\n", "NAME", "APPS", "DATABASES", "URL")
	dim.Println("  " + "────────────────────────────────────────────────────────────────")

	for _, p := range projects {
		apps := 0
		dbs := 0
		var firstAppID string
		for _, env := range p.Environments {
			if firstAppID == "" && len(env.Applications) > 0 {
				firstAppID = env.Applications[0].ApplicationID
			}
			apps += len(env.Applications)
			dbs += len(env.Postgres)
		}
		url := ""
		if firstAppID != "" {
			if domains, err := dk.GetDomainsByApplication(firstAppID); err == nil && len(domains) > 0 {
				scheme := "http"
				if domains[0].HTTPS {
					scheme = "https"
				}
				url = fmt.Sprintf("%s://%s", scheme, domains[0].Host)
			}
		}
		fmt.Printf("  %-20s %-12d %-12d %s\n",
			p.Name,
			apps,
			dbs,
			url,
		)
	}
	fmt.Println()

	return nil
}
