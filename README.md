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
go install github.com/monolinie/cli/cmd/monolinie@latest
# or
go install github.com/monolinie/cli/cmd/ml@latest
```

Or build from source:

```bash
git clone https://github.com/monolinie/cli.git
cd cli
go install ./cmd/monolinie
# or
go install ./cmd/ml
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

Sensitive values (keys and tokens) are masked in the output.

Get a single config value:

```bash
monolinie config get github-org
```

## Commands

| Command                                   | Description                                       |
| ----------------------------------------- | ------------------------------------------------- |
| `monolinie new <name>`                    | Create a new project (repo, app, DB, DNS, deploy) |
| `monolinie list`                          | List all projects with app/DB counts and URLs     |
| `monolinie status <name>`                 | Check DNS resolution and HTTPS availability       |
| `monolinie logs <name>`                   | Show latest deployment log                        |
| `monolinie redeploy <name>`               | Trigger a fresh deployment                        |
| `monolinie env list <name>`               | List environment variables                        |
| `monolinie env get <name> <KEY>`          | Get a single env var                              |
| `monolinie env set <name> K=V ...`        | Set one or more env vars                          |
| `monolinie domain list <name>`            | List domains for a project                        |
| `monolinie domain add <name> <domain>`    | Add domain (DNS + HTTPS)                          |
| `monolinie domain remove <name> <domain>` | Remove domain and DNS record                      |
| `monolinie open <name>`                   | Open project URL in browser                       |
| `monolinie sync <local\|prod>`            | Sync Dokploy projects with the home app database  |
| `monolinie delete <name> [name...]`       | Delete one or more projects and all resources     |

### Home app environments

The `sync` command requires an environment argument, while `new` and `delete` accept an optional `--env` flag (defaults to `prod`):

```bash
# Sync with local dev server
ml sync local    # → http://localhost:3000

# Sync with production
ml sync prod     # → https://preview.monolinie.com

# Create project and register with local home app
ml new my-app --env local

# Delete project and deregister from local home app
ml delete my-app --env local
```

URLs are configurable:

```bash
monolinie config set home_local_url http://localhost:3000
monolinie config set home_prod_url https://preview.monolinie.com
monolinie config set home_api_key your-api-key
```

### Flags

- `new`: `--public` (public repo), `--no-db` (skip database), `--env <local|prod>` (home app environment, default: prod)
- `sync`: `--prune` (remove orphaned projects from the home app database)
- `logs`: `-n, --lines <N>` (last N lines)
- `delete`: `-f, --force` (skip confirmation), `--all` (bulk delete by prefix), `--env <local|prod>` (home app environment, default: prod). Accepts multiple names: `ml delete ww11 ww1135`

## Requirements

- [GitHub CLI](https://cli.github.com/) (`gh`) — authenticated
- [Dokploy](https://dokploy.com/) instance with API access
- [Hetzner DNS](https://dns.hetzner.com/) account for DNS management
