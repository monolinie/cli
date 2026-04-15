# monolinie

CLI tool for automating project setup in the Monolinie studio.

One command to go from zero to a fully deployed Next.js app with a GitHub repo, PostgreSQL database, DNS, and HTTPS.

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

### Check project status

```bash
monolinie status my-app
```

Checks DNS resolution and HTTPS availability for the project.

## Requirements

- [GitHub CLI](https://cli.github.com/) (`gh`) — authenticated
- [Dokploy](https://dokploy.com/) instance with API access
- [Hetzner DNS](https://dns.hetzner.com/) account for DNS management
