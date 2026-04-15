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

	if err := writeDockerignore(dir); err != nil {
		return err
	}

	return applyTemplate(dir)
}

// applyTemplate replaces the default Next.js page with a clean branded template.
func applyTemplate(dir string) error {
	// Remove default Next.js/Vercel assets
	for _, f := range []string{"next.svg", "vercel.svg", "file.svg", "globe.svg", "window.svg"} {
		os.Remove(filepath.Join(dir, "public", f))
	}

	// Ensure public/ has at least one file so Docker COPY doesn't fail
	if err := writeRobotsTxt(dir); err != nil {
		return err
	}

	if err := writePageTSX(dir); err != nil {
		return err
	}
	if err := writeLayoutTSX(dir); err != nil {
		return err
	}
	return writeGlobalCSS(dir)
}

func writePageTSX(dir string) error {
	content := `export default function Home() {
  return (
    <div className="flex flex-col flex-1 items-center justify-center px-6">
      <main className="flex w-full max-w-xl flex-col items-center gap-8 text-center">
        <div className="flex items-center gap-2.5">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
          </span>
          <span className="font-mono text-xs text-emerald-600 dark:text-emerald-400">
            Live
          </span>
        </div>

        <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">
          Preview is live.
        </h1>

        <p className="text-base leading-relaxed text-muted-foreground">
          Your project is set up and development has started. This is your
          preview link &mdash; check back anytime to follow the progress and
          leave feedback. When the project is ready, it will move to your
          own domain.
        </p>

        <p className="font-mono text-xs text-muted-foreground/50 pt-4">
          Developed by Monolinie
        </p>
      </main>
    </div>
  );
}
`
	return os.WriteFile(filepath.Join(dir, "src", "app", "page.tsx"), []byte(content), 0644)
}

func writeLayoutTSX(dir string) error {
	content := `import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Monolinie Project",
  description: "Built and deployed with Monolinie",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={` + "`" + `${geistSans.variable} ${geistMono.variable} min-h-screen flex flex-col antialiased` + "`" + `}
      >
        {children}
      </body>
    </html>
  );
}
`
	return os.WriteFile(filepath.Join(dir, "src", "app", "layout.tsx"), []byte(content), 0644)
}

func writeGlobalCSS(dir string) error {
	content := `@import "tailwindcss";

:root {
  --background: oklch(0.98 0.005 85);
  --foreground: oklch(0.14 0.01 60);
  --muted: oklch(0.95 0.005 85);
  --muted-foreground: oklch(0.45 0.01 60);
  --border: oklch(0.91 0.01 80);
}

@media (prefers-color-scheme: dark) {
  :root {
    --background: oklch(0.0 0 0);
    --foreground: oklch(0.93 0 0);
    --muted: oklch(0.16 0 0);
    --muted-foreground: oklch(0.6 0 0);
    --border: oklch(0.22 0 0);
  }
}

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-border: var(--border);
  --font-sans: var(--font-geist-sans);
  --font-mono: var(--font-geist-mono);
}

body {
  background: var(--background);
  color: var(--foreground);
  font-family: var(--font-sans), system-ui, sans-serif;
}
`
	return os.WriteFile(filepath.Join(dir, "src", "app", "globals.css"), []byte(content), 0644)
}

func writeRobotsTxt(dir string) error {
	content := `User-agent: *
Allow: /
`
	return os.WriteFile(filepath.Join(dir, "public", "robots.txt"), []byte(content), 0644)
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
