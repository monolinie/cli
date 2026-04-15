package template

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ScaffoldNextJS runs create-next-app in the given directory.
func ScaffoldNextJS(dir string) error {
	cmd := exec.Command("npx", "create-next-app@latest", ".",
		"--ts",
		"--tailwind",
		"--eslint",
		"--app",
		"--src-dir",
		"--import-alias", "@/*",
		"--use-npm",
	)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("create-next-app: %w", err)
	}

	if err := writeDockerfile(dir); err != nil {
		return err
	}

	return writeDockerignore(dir)
}

func writeDockerfile(dir string) error {
	dockerfile := `FROM node:20-alpine AS base

# Install dependencies
FROM base AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

# Build the application
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# Production image
FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
CMD ["node", "server.js"]
`
	return os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dockerfile), 0644)
}

func writeDockerignore(dir string) error {
	content := `node_modules
.next
.git
Dockerfile
.dockerignore
`
	return os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte(content), 0644)
}

// PatchNextConfig patches next.config to enable standalone output for Docker.
func PatchNextConfig(dir string) error {
	configPath := filepath.Join(dir, "next.config.ts")
	content := `import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
};

export default nextConfig;
`
	return os.WriteFile(configPath, []byte(content), 0644)
}
