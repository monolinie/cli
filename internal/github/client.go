package github

import (
	"encoding/json"
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
// If the key is already in use on another repo in the org, it removes it first.
func AddDeployKey(org, name, title, publicKey string) error {
	cmd := exec.Command("gh", "repo", "deploy-key", "add", "-",
		"-R", org+"/"+name,
		"--title", title,
	)
	cmd.Stdin = strings.NewReader(publicKey)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "key is already in use") {
			removed := removeDeployKeyFromOrg(org, publicKey)
			if !removed {
				return fmt.Errorf("deploy key is already in use on another repo (possibly deleted) and could not be removed automatically — generate a new SSH key in Dokploy, then retry")
			}
			// Retry after removing from old repo
			retryCmd := exec.Command("gh", "repo", "deploy-key", "add", "-",
				"-R", org+"/"+name,
				"--title", title,
			)
			retryCmd.Stdin = strings.NewReader(publicKey)
			retryOutput, retryErr := retryCmd.CombinedOutput()
			if retryErr != nil {
				return fmt.Errorf("add deploy key (retry): %s\n%s", retryErr, string(retryOutput))
			}
			return nil
		}
		return fmt.Errorf("add deploy key: %s\n%s", err, string(output))
	}
	return nil
}

type deployKey struct {
	ID  int    `json:"id"`
	Key string `json:"key"`
}

// RemoveAllDeployKeys removes all deploy keys from a repo (call before deleting to prevent orphans).
func RemoveAllDeployKeys(org, name string) {
	keysCmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/%s/keys", org, name))
	keysOutput, err := keysCmd.CombinedOutput()
	if err != nil {
		return
	}

	var keys []deployKey
	if err := json.Unmarshal(keysOutput, &keys); err != nil {
		return
	}

	for _, k := range keys {
		delCmd := exec.Command("gh", "api", "-X", "DELETE",
			fmt.Sprintf("/repos/%s/%s/keys/%d", org, name, k.ID),
		)
		delCmd.CombinedOutput() // best-effort
	}
}

// removeDeployKeyFromOrg finds and removes a deploy key across all repos in the org.
// Returns true if the key was found and removed, false if orphaned (e.g. from a deleted repo).
func removeDeployKeyFromOrg(org, publicKey string) bool {
	cmd := exec.Command("gh", "api", "--paginate",
		fmt.Sprintf("/orgs/%s/repos?per_page=100", org),
		"-q", ".[].name",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	targetParts := strings.Fields(strings.TrimSpace(publicKey))
	targetKeyData := ""
	if len(targetParts) >= 2 {
		targetKeyData = targetParts[1]
	}

	repos := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, repo := range repos {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}

		keysCmd := exec.Command("gh", "api",
			fmt.Sprintf("/repos/%s/%s/keys", org, repo),
		)
		keysOutput, err := keysCmd.CombinedOutput()
		if err != nil {
			continue
		}

		var keys []deployKey
		if err := json.Unmarshal(keysOutput, &keys); err != nil {
			continue
		}

		for _, k := range keys {
			keyParts := strings.Fields(k.Key)
			if len(keyParts) >= 2 && keyParts[1] == targetKeyData {
				delCmd := exec.Command("gh", "api", "-X", "DELETE",
					fmt.Sprintf("/repos/%s/%s/keys/%d", org, repo, k.ID),
				)
				if _, err := delCmd.CombinedOutput(); err != nil {
					return false
				}
				return true
			}
		}
	}

	return false
}

// DeleteRepo deletes a GitHub repository using the gh CLI.
func DeleteRepo(org, name string) error {
	cmd := exec.Command("gh", "repo", "delete", org+"/"+name, "--yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh repo delete: %s\n%s", err, string(output))
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
