package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateRepo creates a new GitHub repository using the gh CLI.
func CreateRepo(org, name string, private bool) error {
	args := []string{"repo", "create", org + "/" + name}
	if private {
		args = append(args, "--private")
	} else {
		args = append(args, "--public")
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh repo create: %s\n%s", err, string(output))
	}
	return nil
}

// CloneRepo clones a GitHub repository using gh CLI (uses HTTPS + gh auth).
func CloneRepo(org, name, dir string) error {
	repo := fmt.Sprintf("%s/%s", org, name)
	cmd := exec.Command("gh", "repo", "clone", repo, dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %s\n%s", err, string(output))
	}
	return nil
}

// AddDeployKey adds an SSH public key as a deploy key to the repository.
func AddDeployKey(org, name, title, publicKey string) error {
	cmd := exec.Command("gh", "repo", "deploy-key", "add", "-",
		"-R", org+"/"+name,
		"--title", title,
	)
	cmd.Stdin = strings.NewReader(publicKey)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "already in use" error
		if strings.Contains(string(output), "key is already in use") {
			return nil
		}
		return fmt.Errorf("add deploy key: %s\n%s", err, string(output))
	}
	return nil
}

// CommitAndPush stages all files, commits, and pushes to the remote.
func CommitAndPush(dir, message string) error {
	commands := [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", message},
		{"git", "push", "-u", "origin", "main"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %s\n%s", strings.Join(args, " "), err, string(output))
		}
	}

	return nil
}
