package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dns"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/monolinie/cli/internal/github"
	"github.com/monolinie/cli/internal/template"
	"github.com/spf13/cobra"
)

var (
	flagNoDB   bool
	flagPublic bool
)

var newCmd = &cobra.Command{
	Use:   "new <project-name>",
	Short: "Create a new project",
	Long:  "Create a new project: GitHub repo, Next.js scaffold, Dokploy deployment, DNS, and database.",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

func init() {
	newCmd.Flags().BoolVar(&flagNoDB, "no-db", false, "Skip database creation")
	newCmd.Flags().BoolVar(&flagPublic, "public", false, "Create a public GitHub repo")
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load and validate config
	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	org := config.Get("github_org")
	domain := config.Get("domain")
	serverIP := config.Get("dokploy_server_ip")
	previewHost := fmt.Sprintf("%s.preview.%s", name, domain)

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	// Step 0: Get Dokploy SSH key for deploy key
	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))
	sshKeys, err := dk.GetSSHKeys()
	if err != nil {
		return fmt.Errorf("get ssh keys: %w", err)
	}
	if len(sshKeys) == 0 {
		return fmt.Errorf("no SSH keys configured in Dokploy — add one under Settings → SSH Keys")
	}

	// Step 1: Create GitHub repo
	bold.Printf("\n→ Creating GitHub repo %s/%s...\n", org, name)
	if err := github.CreateRepo(org, name, !flagPublic); err != nil {
		return fmt.Errorf("create repo: %w", err)
	}

	// Add Dokploy SSH key as deploy key
	if err := github.AddDeployKey(org, name, "Dokploy", sshKeys[0].PublicKey); err != nil {
		return fmt.Errorf("add deploy key: %w", err)
	}
	green.Println("  ✓ Repository created")

	// Step 2: Clone and scaffold
	bold.Println("→ Scaffolding Next.js project...")
	tmpDir := filepath.Join(os.TempDir(), "monolinie-"+name)
	os.RemoveAll(tmpDir)

	if err := github.CloneRepo(org, name, tmpDir); err != nil {
		return fmt.Errorf("clone repo: %w", err)
	}

	if err := template.ScaffoldNextJS(tmpDir); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	if err := template.PatchNextConfig(tmpDir); err != nil {
		return fmt.Errorf("patch next config: %w", err)
	}
	green.Println("  ✓ Next.js project scaffolded")

	// Step 3: Commit and push
	bold.Println("→ Pushing to GitHub...")
	if err := github.CommitAndPush(tmpDir, "Initial commit from monolinie CLI"); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	green.Println("  ✓ Code pushed")

	// Step 4: Create Dokploy project
	bold.Println("→ Setting up Dokploy project...")

	result, err := dk.CreateProject(name, "Created by monolinie CLI")
	if err != nil {
		return fmt.Errorf("dokploy project: %w", err)
	}
	green.Printf("  ✓ Project created (ID: %s)\n", result.Project.ProjectID)
	envID := result.Environment.EnvironmentID

	// Step 5: Create application
	app, err := dk.CreateApplication(envID, name, "Next.js app")
	if err != nil {
		return fmt.Errorf("dokploy application: %w", err)
	}
	green.Printf("  ✓ Application created (ID: %s)\n", app.ApplicationID)

	// Step 5b: Set build type to Dockerfile
	if err := dk.SaveBuildType(app.ApplicationID, "dockerfile", "./Dockerfile"); err != nil {
		return fmt.Errorf("set build type: %w", err)
	}

	// Step 5c: Connect to GitHub via SSH
	gitURL := fmt.Sprintf("git@github.com:%s/%s.git", org, name)
	if err := dk.SaveGitProvider(app.ApplicationID, gitURL, "main", "/", sshKeys[0].SSHKeyID); err != nil {
		return fmt.Errorf("connect github: %w", err)
	}
	green.Println("  ✓ Connected to GitHub")

	// Step 6: Create PostgreSQL database (unless --no-db)
	var dbConnStr string
	if !flagNoDB {
		bold.Println("→ Creating PostgreSQL database...")
		dbPass := generatePassword(16)
		dbUser := sanitizeDBName(name)
		dbName := sanitizeDBName(name)

		pg, err := dk.CreatePostgres(envID, name+"-db", dbName, dbUser, dbPass)
		if err != nil {
			return fmt.Errorf("create postgres: %w", err)
		}

		if err := dk.DeployPostgres(pg.PostgresID); err != nil {
			return fmt.Errorf("deploy postgres: %w", err)
		}

		dbConnStr = fmt.Sprintf("postgresql://%s:%s@%s-db:5432/%s", dbUser, dbPass, name, dbName)
		green.Printf("  ✓ PostgreSQL created (ID: %s)\n", pg.PostgresID)
	}

	// Step 7: Set environment variables
	bold.Println("→ Setting environment variables...")
	envVars := "NODE_ENV=production\nHOSTNAME=0.0.0.0\nPORT=3000"
	if dbConnStr != "" {
		envVars += fmt.Sprintf("\nDATABASE_URL=%s", dbConnStr)
	}
	if err := dk.SaveEnvironment(app.ApplicationID, envVars); err != nil {
		return fmt.Errorf("set env vars: %w", err)
	}
	green.Println("  ✓ Environment variables set")

	// Step 8: Create DNS record
	bold.Println("→ Setting up DNS...")
	dnsClient := dns.NewClient(config.Get("hetzner_dns_token"))

	zone, err := dnsClient.GetZoneByName(domain)
	if err != nil {
		return fmt.Errorf("get DNS zone: %w", err)
	}

	recordName := name + ".preview"
	if err := dnsClient.CreateRecord(zone.ID, "A", recordName, serverIP, 300); err != nil {
		return fmt.Errorf("create DNS record: %w", err)
	}
	green.Printf("  ✓ DNS A record: %s → %s\n", previewHost, serverIP)

	// Step 9: Create domain in Dokploy with Let's Encrypt
	if _, err := dk.CreateDomain(app.ApplicationID, previewHost, 3000, true, "letsencrypt"); err != nil {
		return fmt.Errorf("create domain: %w", err)
	}
	green.Println("  ✓ Domain configured with Let's Encrypt")

	// Step 10: Deploy
	bold.Println("→ Deploying application...")
	if err := dk.DeployApplication(app.ApplicationID); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	green.Println("  ✓ Deployment started")

	// Summary
	fmt.Println()
	bold.Println("═══════════════════════════════════════════")
	bold.Printf("  Project: %s\n", name)
	bold.Println("═══════════════════════════════════════════")
	fmt.Printf("  Repo:    https://github.com/%s/%s\n", org, name)
	fmt.Printf("  URL:     https://%s\n", previewHost)
	if dbConnStr != "" {
		fmt.Printf("  DB:      %s\n", dbConnStr)
	}
	fmt.Printf("  Dokploy: %s\n", config.Get("dokploy_url"))
	bold.Println("═══════════════════════════════════════════")
	fmt.Println()

	// Cleanup temp dir
	os.RemoveAll(tmpDir)

	return nil
}

func generatePassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
}

func sanitizeDBName(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, byte(c))
		} else if c == '-' {
			result = append(result, '_')
		}
	}
	return string(result)
}
