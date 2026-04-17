package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dns"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/monolinie/cli/internal/github"
	"github.com/monolinie/cli/internal/home"
	"github.com/spf13/cobra"
)

var (
	flagForce    bool
	flagAll      bool
	flagDeleteEnv string
)

var deleteCmd = &cobra.Command{
	Use:   "delete <project-name> [project-name...]",
	Short: "Delete one or more projects (reverse provision)",
	Long: `Tear down projects: remove Dokploy services, DNS records, and GitHub repos.

Delete multiple projects at once:
  monolinie delete ww11 ww1135       # deletes both projects
  monolinie delete ww11 ww1135 -f    # skip confirmation

Use --all to delete all projects matching a prefix:
  monolinie delete test --all        # deletes all projects starting with "test"
  monolinie delete test --all -f     # skip confirmation`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().BoolVar(&flagAll, "all", false, "Delete all projects matching the prefix")
	deleteCmd.Flags().StringVar(&flagDeleteEnv, "env", "prod", "Home app environment (local or prod)")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))

	if flagAll {
		return deleteByPrefix(dk, args[0])
	}
	if len(args) == 1 {
		return deleteSingle(dk, args[0])
	}
	return deleteMultiple(dk, args)
}

func deleteByPrefix(dk *dokploy.Client, prefix string) error {
	bold := color.New(color.Bold)

	projects, err := dk.GetProjects()
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	var matched []dokploy.ProjectDetail
	for i := range projects {
		if strings.HasPrefix(strings.ToLower(projects[i].Name), strings.ToLower(prefix)) {
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
		if err := deleteProject(&p, dk); err != nil {
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

func deleteMultiple(dk *dokploy.Client, names []string) error {
	bold := color.New(color.Bold)

	bold.Printf("\nProjects to delete:\n")
	for _, n := range names {
		fmt.Printf("  - %s\n", n)
	}
	fmt.Println()

	if !flagForce {
		color.Red("  WARNING: This will permanently delete ALL %d projects above including:", len(names))
		fmt.Println("    - Dokploy projects and all services")
		fmt.Println("    - DNS records")
		fmt.Println("    - GitHub repositories")
		fmt.Println()
		fmt.Printf("  Type \"yes\" to confirm deletion: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Println("  Aborted.")
			return nil
		}
	}

	var failed []string
	for _, name := range names {
		project, err := findProjectByName(dk, name)
		if err != nil {
			color.Red("  ✗ %s: %v", name, err)
			failed = append(failed, name)
			continue
		}
		bold.Printf("\n→ Deleting %s...\n", name)
		if err := deleteProject(project, dk); err != nil {
			color.Red("  ✗ Failed to fully delete %s: %v", name, err)
			failed = append(failed, name)
		}
	}

	fmt.Println()
	if len(failed) > 0 {
		color.Yellow("Completed with errors. Failed projects: %s", strings.Join(failed, ", "))
	} else {
		color.New(color.FgGreen).Printf("All %d projects deleted successfully.\n\n", len(names))
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
	deleteProject(project, dk)

	fmt.Println()
	color.New(color.FgGreen).Printf("  Project %q has been deleted.\n\n", name)
	return nil
}

func deleteProject(project *dokploy.ProjectDetail, dk *dokploy.Client) error {
	name := project.Name
	projectID := project.ProjectID
	org := config.Get("github_org")
	domain := config.Get("domain")
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	// Fetch full project detail to get Docker Swarm service names
	// (the list endpoint may not populate nested AppName fields)
	var serviceNames []string
	if full, err := dk.GetProject(projectID); err == nil {
		for _, env := range full.Environments {
			for _, app := range env.Applications {
				if app.AppName != "" {
					serviceNames = append(serviceNames, app.AppName)
				}
			}
			for _, pg := range env.Postgres {
				if pg.AppName != "" {
					serviceNames = append(serviceNames, pg.AppName)
				}
			}
		}
	}

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

	// Step 3: Remove deploy keys and delete GitHub repo
	fmt.Printf("  Removing deploy keys...")
	github.RemoveAllDeployKeys(org, name)
	green.Println(" ✓")

	fmt.Printf("  Deleting GitHub repo...")
	if err := github.DeleteRepo(org, name); err != nil {
		yellow.Printf(" ⚠ %v (skipping)\n", err)
	} else {
		green.Println(" ✓")
	}

	// Deregister from home app (non-fatal)
	if hc, err := resolveHomeClient(flagDeleteEnv); err == nil {
		_, err := hc.DeregisterProject(home.DeregisterInput{
			DokployProjectID: projectID,
			Name:             name,
		})
		if err != nil {
			yellow.Printf("  ⚠ Failed to deregister from home app: %v\n", err)
		} else {
			green.Println("  ✓ Deregistered from home app")
		}
	}

	// Clean up orphaned Docker Swarm services and volumes
	if len(serviceNames) > 0 {
		serverIP := config.Get("dokploy_server_ip")
		if serverIP == "" {
			yellow.Println("  ⚠ dokploy_server_ip not configured, skipping Docker cleanup")
		} else {
			fmt.Printf("  Cleaning up Docker services...")
			// Build volume names (<appName>-data) for explicit removal
			var volumeNames []string
			for _, s := range serviceNames {
				volumeNames = append(volumeNames, s+"-data")
			}
			rmCmd := "docker service rm " + strings.Join(serviceNames, " ") + " 2>/dev/null; " +
				"for i in 1 2 3 4 5; do sleep 3; docker volume rm " + strings.Join(volumeNames, " ") + " 2>/dev/null; done; " +
				"docker system prune -af > /dev/null 2>&1"
			if out, err := exec.Command("ssh", "-o", "StrictHostKeyChecking=accept-new", "root@"+serverIP, rmCmd).CombinedOutput(); err != nil {
				yellow.Printf(" ⚠ %s (non-fatal)\n", strings.TrimSpace(string(out)))
			} else {
				green.Println(" ✓")
			}
		}
	}

	return nil
}
