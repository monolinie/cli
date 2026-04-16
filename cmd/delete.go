package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dns"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/monolinie/cli/internal/github"
	"github.com/spf13/cobra"
)

var flagForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete <project-name>",
	Short: "Delete a project",
	Long:  "Tear down a project: remove Dokploy services, DNS record, and GitHub repo.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	// Confirmation
	if !flagForce {
		color.Red("\n  WARNING: This will permanently delete project %q including:", name)
		fmt.Println("    - Dokploy project and all services")
		fmt.Println("    - DNS record")
		fmt.Println("    - GitHub repository")
		fmt.Println()
		fmt.Printf("  Type the project name to confirm: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.TrimSpace(scanner.Text()) != name {
			fmt.Println("  Aborted.")
			return nil
		}
	}

	org := config.Get("github_org")
	domain := config.Get("domain")
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))

	// Step 1: Remove Dokploy project
	bold.Println("\n→ Removing Dokploy project...")
	project, err := findProjectByName(dk, name)
	if err != nil {
		yellow.Printf("  ⚠ %s (skipping)\n", err)
	} else {
		if err := dk.RemoveProject(project.ProjectID); err != nil {
			yellow.Printf("  ⚠ remove project: %s (skipping)\n", err)
		} else {
			green.Println("  ✓ Dokploy project removed")
		}
	}

	// Step 2: Remove DNS record
	bold.Println("→ Removing DNS record...")
	dnsClient := dns.NewClient(config.Get("hetzner_dns_token"))
	zone, err := dnsClient.GetZoneByName(domain)
	if err != nil {
		yellow.Printf("  ⚠ get zone: %s (skipping)\n", err)
	} else {
		recordName := name + ".preview"
		if err := dnsClient.DeleteRecord(zone.ID, "A", recordName); err != nil {
			yellow.Printf("  ⚠ delete record: %s (skipping)\n", err)
		} else {
			green.Println("  ✓ DNS record removed")
		}
	}

	// Step 3: Delete GitHub repo
	bold.Println("→ Deleting GitHub repository...")
	if err := github.DeleteRepo(org, name); err != nil {
		yellow.Printf("  ⚠ %s (skipping)\n", err)
	} else {
		green.Println("  ✓ GitHub repository deleted")
	}

	fmt.Println()
	green.Printf("  Project %q has been deleted.\n\n", name)

	return nil
}
