package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/monolinie/cli/internal/config"
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

	if err := config.Init(); err != nil {
		return err
	}

	domain := config.Get("domain")
	url := fmt.Sprintf("https://%s.preview.%s", name, domain)

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
