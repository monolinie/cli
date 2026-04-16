package cmd

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/monolinie/cli/internal/config"
	"github.com/monolinie/cli/internal/dns"
	"github.com/monolinie/cli/internal/dokploy"
	"github.com/spf13/cobra"
)

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage project domains",
	Long:  "Add or remove custom domains for a project.",
}

var domainListCmd = &cobra.Command{
	Use:   "list <project-name>",
	Short: "List domains for a project",
	Args:  cobra.ExactArgs(1),
	RunE:  runDomainList,
}

var domainAddCmd = &cobra.Command{
	Use:   "add <project-name> <domain>",
	Short: "Add a domain to a project",
	Args:  cobra.ExactArgs(2),
	RunE:  runDomainAdd,
}

var domainRemoveCmd = &cobra.Command{
	Use:   "remove <project-name> <domain>",
	Short: "Remove a domain from a project",
	Args:  cobra.ExactArgs(2),
	RunE:  runDomainRemove,
}

var flagDomainApp string

func init() {
	domainCmd.PersistentFlags().StringVar(&flagDomainApp, "app", "", "Target a specific app by name")
	domainCmd.AddCommand(domainListCmd, domainAddCmd, domainRemoveCmd)
	rootCmd.AddCommand(domainCmd)
}

func runDomainList(cmd *cobra.Command, args []string) error {
	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))

	project, err := findProjectByName(dk, args[0])
	if err != nil {
		return err
	}

	app, err := findAppInProject(project, flagDomainApp)
	if err != nil {
		return err
	}

	domains, err := dk.GetDomainsByApplication(app.ApplicationID)
	if err != nil {
		return fmt.Errorf("get domains: %w", err)
	}

	if len(domains) == 0 {
		fmt.Println("No domains configured.")
		return nil
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)

	bold.Printf("\n  %-40s %-8s %-8s %s\n", "HOST", "PORT", "HTTPS", "CERT")
	dim.Println("  " + "────────────────────────────────────────────────────────────────")

	for _, d := range domains {
		https := "no"
		if d.HTTPS {
			https = "yes"
		}
		fmt.Printf("  %-40s %-8d %-8s %s\n", d.Host, d.Port, https, d.CertificateType)
	}
	fmt.Println()

	return nil
}

func runDomainAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	host := args[1]

	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))
	serverIP := config.Get("dokploy_server_ip")
	baseDomain := config.Get("domain")

	project, err := findProjectByName(dk, name)
	if err != nil {
		return err
	}
	app, err := findAppInProject(project, flagDomainApp)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	// Create DNS record if the host is under our managed domain
	dnsClient := dns.NewClient(config.Get("hetzner_dns_token"))
	if strings.HasSuffix(host, "."+baseDomain) {
		bold.Println("\n→ Creating DNS record...")
		zone, err := dnsClient.GetZoneByName(baseDomain)
		if err != nil {
			return fmt.Errorf("get zone: %w", err)
		}
		recordName := host[:len(host)-len(baseDomain)-1] // strip ".domain.com"
		if err := dnsClient.CreateRecord(zone.ID, "A", recordName, serverIP, 300); err != nil {
			if strings.Contains(err.Error(), "already exist") {
				green.Printf("  ✓ DNS A record already exists: %s\n", host)
			} else {
				return fmt.Errorf("create DNS record: %w", err)
			}
		} else {
			green.Printf("  ✓ DNS A record: %s → %s\n", host, serverIP)
		}

		// Wait for DNS
		fmt.Printf("  Waiting for DNS propagation...")
		if err := waitForDomainDNS(host, serverIP, 3*time.Minute); err != nil {
			color.Yellow(" timed out (domain may work after DNS propagates)")
		} else {
			green.Println(" ✓")
		}
	}

	// Add domain in Dokploy
	bold.Println("→ Configuring domain...")
	if _, err := dk.CreateDomain(app.ApplicationID, host, 3000, true, "letsencrypt"); err != nil {
		return fmt.Errorf("create domain: %w", err)
	}
	green.Printf("  ✓ Domain %s configured with HTTPS\n\n", host)

	return nil
}

func runDomainRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	host := args[1]

	if err := config.Init(); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	dk := dokploy.NewClient(config.Get("dokploy_url"), config.Get("dokploy_api_key"))
	baseDomain := config.Get("domain")

	project, err := findProjectByName(dk, name)
	if err != nil {
		return err
	}
	app, err := findAppInProject(project, flagDomainApp)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	// Find the domain
	domains, err := dk.GetDomainsByApplication(app.ApplicationID)
	if err != nil {
		return fmt.Errorf("get domains: %w", err)
	}

	var domainID string
	for _, d := range domains {
		if d.Host == host {
			domainID = d.DomainID
			break
		}
	}
	if domainID == "" {
		return fmt.Errorf("domain %q not found on project %q", host, name)
	}

	// Remove domain from Dokploy
	bold.Println("\n→ Removing domain...")
	if err := dk.DeleteDomain(domainID); err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	green.Println("  ✓ Domain removed from Dokploy")

	// Remove DNS record if under our managed domain
	if strings.HasSuffix(host, "."+baseDomain) {
		bold.Println("→ Removing DNS record...")
		dnsClient := dns.NewClient(config.Get("hetzner_dns_token"))
		zone, err := dnsClient.GetZoneByName(baseDomain)
		if err != nil {
			yellow.Printf("  ⚠ get zone: %s (skipping)\n", err)
		} else {
			recordName := host[:len(host)-len(baseDomain)-1]
			if err := dnsClient.DeleteRecord(zone.ID, "A", recordName); err != nil {
				yellow.Printf("  ⚠ delete record: %s (skipping)\n", err)
			} else {
				green.Println("  ✓ DNS record removed")
			}
		}
	}

	fmt.Println()
	return nil
}

func waitForDomainDNS(host, expectedIP string, timeout time.Duration) error {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ips, err := resolver.LookupHost(ctx, host)
		cancel()
		if err == nil {
			for _, ip := range ips {
				if ip == expectedIP {
					return nil
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timed out")
}
