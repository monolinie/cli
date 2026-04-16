package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <project-name>",
	Short: "Open project in browser",
	Long:  "Open the project URL in your default browser.",
	Args:  cobra.ExactArgs(1),
	RunE:  runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
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

	if len(domains) == 0 {
		return fmt.Errorf("no domains configured for project %q", project.Name)
	}

	d := domains[0]
	scheme := "http"
	if d.HTTPS {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s", scheme, d.Host)

	fmt.Printf("Opening %s...\n", url)
	return openBrowser(url)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform — open manually: %s", url)
	}
}
