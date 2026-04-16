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

var (
	flagForce bool
	flagAll   bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <project-name>",
	Short: "Delete a project (reverse provision)",
	Long: `Tear down a project: remove Dokploy services, DNS record, and GitHub repo.

Use --all to delete all projects matching a prefix:
  monolinie delete test --all        # deletes all projects starting with "test"
  monolinie delete test --all -f     # skip confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().BoolVar(&flagAll, "all", false, "Delete all projects matching the prefix")
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

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))

	if flagAll {
		return deleteByPrefix(dk, name)
	}
	return deleteSingle(dk, name)
}

func deleteByPrefix(dk *dokploy.Client, prefix string) error {
	bold := color.New(color.Bold)

	projects, err := dk.GetProjects()
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	var matched []dokploy.ProjectDetail
	for i := range projects {
		if strings.HasPrefix(projects[i].Name, prefix) {
			matched = append(matched, projects[i])
		}
	}

	if len(matched) == 0 {
		fmt.Printf("No projects found with prefix %q\n", prefix)
		return nil
	}

	bold.Printf("\nFound %d project(s) matching prefix %q:\n", len(matched), prefix)
	for _, p := range matched {
		fmt.Printf("  - %s\n", p.Name)
	}
	fmt.Println()

	if !flagForce {
		color.Red("  WARNING: This will permanently delete ALL %d projects above including:", len(matched))
		fmt.Println("    - Dokploy projects and all services")
		fmt.Println("    - DNS records")
		fmt.Println("    - GitHub repositories")
		fmt.Println()
		fmt.Printf("  Type %q to confirm deletion: ", prefix)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.TrimSpace(scanner.Text()) != prefix {
			fmt.Println("  Aborted.")
			return nil
		}
	}

	var failed []string
	for _, p := range matched {
		bold.Printf("\n→ Deleting %s...\n", p.Name)
		if err := deleteProject(p.Name, p.ProjectID, dk); err != nil {
			color.Red("  ✗ Failed to fully delete %s: %v", p.Name, err)
			failed = append(failed, p.Name)
		}
	}

	fmt.Println()
	if len(failed) > 0 {
		color.Yellow("Completed with errors. Failed projects: %s", strings.Join(failed, ", "))
	} else {
		color.New(color.FgGreen).Printf("All %d projects deleted successfully.\n\n", len(matched))
	}
	return nil
}

func deleteSingle(dk *dokploy.Client, name string) error {
	bold := color.New(color.Bold)

	project, err := findProjectByName(dk, name)
	if err != nil {
		return err
	}

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

	bold.Printf("\n→ Deleting %s...\n", name)
	deleteProject(name, project.ProjectID, dk)

	fmt.Println()
	color.New(color.FgGreen).Printf("  Project %q has been deleted.\n\n", name)
	return nil
}

func deleteProject(name, projectID string, dk *dokploy.Client) error {
	org := config.Get("github_org")
	domain := config.Get("domain")
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	// Step 1: Remove Dokploy project
	fmt.Printf("  Removing Dokploy project...")
	if err := dk.RemoveProject(projectID); err != nil {
		yellow.Printf(" ⚠ %v (skipping)\n", err)
	} else {
		green.Println(" ✓")
	}

	// Step 2: Remove DNS record
	fmt.Printf("  Removing DNS record...")
	dnsClient := dns.NewClient(config.Get("hetzner_dns_token"))
	zone, err := dnsClient.GetZoneByName(domain)
	if err != nil {
		yellow.Printf(" ⚠ %v (skipping)\n", err)
	} else {
		recordName := name + ".preview"
		if err := dnsClient.DeleteRecord(zone.ID, "A", recordName); err != nil {
			yellow.Printf(" ⚠ %v (skipping)\n", err)
		} else {
			green.Println(" ✓")
		}
	}

	// Step 3: Delete GitHub repo
	fmt.Printf("  Deleting GitHub repo...")
	if err := github.DeleteRepo(org, name); err != nil {
		yellow.Printf(" ⚠ %v (skipping)\n", err)
	} else {
		green.Println(" ✓")
	}

	return nil
}
