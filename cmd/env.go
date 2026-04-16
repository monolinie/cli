package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env <project-name>",
	Short: "Manage environment variables",
	Long:  "List, get, or set environment variables for a project.",
}

var envListCmd = &cobra.Command{
	Use:   "list <project-name>",
	Short: "List all environment variables",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvList,
}

var envGetCmd = &cobra.Command{
	Use:   "get <project-name> <key>",
	Short: "Get an environment variable",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvGet,
}

var envSetCmd = &cobra.Command{
	Use:   "set <project-name> <KEY=VALUE> [KEY=VALUE...]",
	Short: "Set environment variables",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runEnvSet,
}

func init() {
	envCmd.AddCommand(envListCmd, envGetCmd, envSetCmd)
	rootCmd.AddCommand(envCmd)
}

func getAppForProject(name string) (*dokploy.Client, *dokploy.ApplicationDetail, error) {
	if err := config.Init(); err != nil {
		return nil, nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, nil, err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))

	project, err := findProjectByName(dk, name)
	if err != nil {
		return nil, nil, err
	}

	appRef, err := findAppInProject(project)
	if err != nil {
		return nil, nil, err
	}

	app, err := dk.GetApplication(appRef.ApplicationID)
	if err != nil {
		return nil, nil, err
	}

	return dk, app, nil
}

func parseEnvVars(env string) map[string]string {
	vars := make(map[string]string)
	for _, line := range strings.Split(env, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}
	return vars
}

func envMapToString(vars map[string]string) string {
	var lines []string
	for k, v := range vars {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(lines, "\n")
}

func runEnvList(cmd *cobra.Command, args []string) error {
	_, app, err := getAppForProject(args[0])
	if err != nil {
		return err
	}

	vars := parseEnvVars(app.Env)
	if len(vars) == 0 {
		fmt.Println("No environment variables set.")
		return nil
	}

	bold := color.New(color.Bold)
	for k, v := range vars {
		bold.Printf("%-24s", k)
		// Mask long values that look like secrets
		if (strings.Contains(strings.ToLower(k), "key") ||
			strings.Contains(strings.ToLower(k), "secret") ||
			strings.Contains(strings.ToLower(k), "token") ||
			strings.Contains(strings.ToLower(k), "password")) && len(v) > 8 {
			fmt.Printf(" %s...%s\n", v[:4], v[len(v)-4:])
		} else {
			fmt.Printf(" %s\n", v)
		}
	}

	return nil
}

func runEnvGet(cmd *cobra.Command, args []string) error {
	_, app, err := getAppForProject(args[0])
	if err != nil {
		return err
	}

	vars := parseEnvVars(app.Env)
	value, ok := vars[args[1]]
	if !ok {
		color.Yellow("(not set)")
		return nil
	}
	fmt.Println(value)
	return nil
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	name := args[0]
	dk, app, err := getAppForProject(name)
	if err != nil {
		return err
	}

	vars := parseEnvVars(app.Env)

	// Parse KEY=VALUE pairs from args
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format %q — use KEY=VALUE", arg)
		}
		vars[parts[0]] = parts[1]
	}

	if err := dk.UpdateEnvironment(app.ApplicationID, envMapToString(vars)); err != nil {
		return fmt.Errorf("save environment: %w", err)
	}

	green := color.New(color.FgGreen)
	green.Printf("✓ Updated environment variables for %s\n", name)
	color.Yellow("  Run `monolinie redeploy %s` to apply changes.\n", name)

	return nil
}
