# monolinie

CLI tool for automating project setup and management in the Monolinie studio.

> **Shorthand:** You can use `ml` as an abbreviation for `monolinie` in all commands, e.g. `ml new my-app` instead of `monolinie new my-app`.

One command to go from zero to a fully deployed Next.js app with a GitHub repo, PostgreSQL database, DNS, and HTTPS — plus commands to manage, monitor, and tear down projects.

## What it does

`monolinie new <project-name>` runs through these steps automatically:

1. Creates a GitHub repository (private by default)
2. Scaffolds a Next.js project with Dockerfile
3. Pushes the initial commit
4. Creates a Dokploy project and application
5. Provisions a PostgreSQL database
6. Sets environment variables
7. Creates a DNS A record (Hetzner)
8. Configures the domain with Let's Encrypt
9. Triggers the first deployment

## Install

```bash
go install github.com/monolinie/cli@latest
```

Or build from source:

```bash
git clone https://github.com/monolinie/cli.git
cd cli
go build -o monolinie .
# Or build with the short name:
go build -o ml .
```

## Configuration

Store your credentials and settings with `monolinie config set`:

```bash
monolinie config set github-org your-org
monolinie config set domain your-domain.com
monolinie config set dokploy-url https://dokploy.your-domain.com
monolinie config set dokploy-api-key your-api-key
monolinie config set dokploy-server-ip 1.2.3.4
monolinie config set hetzner-dns-token your-token
```

Config is stored in `~/.monolinie/config.yaml`.

View current config:

```bash
monolinie config list
```

## Usage

### Create a new project

```bash
monolinie new my-app
```

Options:
- `--public` — create a public GitHub repo (default: private)
- `--no-db` — skip PostgreSQL database creation

### List all projects

```bash
monolinie list
```

Shows all projects deployed in Dokploy with app count, database count, and URLs.

### Check project status

```bash
monolinie status my-app
```

Checks DNS resolution and HTTPS availability for the project.

### View deployment logs

```bash
monolinie logs my-app
```

Shows the latest deployment log. Options:
- `-n, --lines <N>` — show only the last N lines

### Redeploy a project

```bash
monolinie redeploy my-app
```

Triggers a fresh deployment without changing anything else.

### Manage environment variables

```bash
# List all env vars
monolinie env list my-app

# Get a single env var
monolinie env get my-app DATABASE_URL

# Set env vars (one or more KEY=VALUE pairs)
monolinie env set my-app NODE_ENV=production DEBUG=true
```

Sensitive values (keys, tokens, passwords) are masked in the output.

After setting variables, run `monolinie redeploy my-app` to apply changes.

### Manage domains

```bash
# List domains for a project
monolinie domain list my-app

# Add a domain (creates DNS record + configures HTTPS)
monolinie domain add my-app custom.your-domain.com

# Remove a domain (removes from Dokploy + deletes DNS record)
monolinie domain remove my-app custom.your-domain.com
```

When adding a domain under your configured base domain, the CLI automatically creates the DNS A record and waits for propagation before configuring HTTPS with Let's Encrypt.

### Open project in browser

```bash
monolinie open my-app
```

Opens the project URL in your default browser.

### Delete a project

```bash
monolinie delete my-app
```

Tears down everything:
- Removes the Dokploy project and all services
- Deletes the DNS record
- Deletes the GitHub repository

Options:
- `-f, --force` — skip confirmation prompt

You will be asked to type the project name to confirm unless `--force` is used.

## Requirements

- [GitHub CLI](https://cli.github.com/) (`gh`) — authenticated
- [Dokploy](https://dokploy.com/) instance with API access
- [Hetzner DNS](https://dns.hetzner.com/) account for DNS management
