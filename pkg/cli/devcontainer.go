package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var devcontainerLog = logger.New("cli:devcontainer")

// DevcontainerCustomizations represents VSCode customizations in devcontainer.json
type DevcontainerCustomizations struct {
	VSCode     *DevcontainerVSCode     `json:"vscode,omitempty"`
	Codespaces *DevcontainerCodespaces `json:"codespaces,omitempty"`
}

// DevcontainerVSCode represents VSCode-specific settings
type DevcontainerVSCode struct {
	Settings   map[string]any `json:"settings,omitempty"`
	Extensions []string       `json:"extensions,omitempty"`
}

// DevcontainerCodespaces represents GitHub Codespaces-specific settings
type DevcontainerCodespaces struct {
	Repositories map[string]DevcontainerRepoPermissions `json:"repositories"`
}

// DevcontainerRepoPermissions represents permissions for a repository
type DevcontainerRepoPermissions struct {
	Permissions map[string]string `json:"permissions"`
}

// DevcontainerFeatures represents features to install in the devcontainer
type DevcontainerFeatures map[string]any

// DevcontainerConfig represents the structure of devcontainer.json
type DevcontainerConfig struct {
	Name              string                      `json:"name"`
	Image             string                      `json:"image"`
	Customizations    *DevcontainerCustomizations `json:"customizations,omitempty"`
	Features          DevcontainerFeatures        `json:"features,omitempty"`
	PostCreateCommand string                      `json:"postCreateCommand,omitempty"`
}

// ensureDevcontainerConfig creates or updates .devcontainer/gh-aw/devcontainer.json
func ensureDevcontainerConfig(verbose bool, additionalRepos []string) error {
	devcontainerLog.Printf("Creating or updating .devcontainer/gh-aw/devcontainer.json with additional repos: %v", additionalRepos)

	// Create .devcontainer/gh-aw directory if it doesn't exist
	// Using a subdirectory to avoid overriding existing devcontainer.json files
	devcontainerDir := filepath.Join(".devcontainer", "gh-aw")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer/gh-aw directory: %w", err)
	}
	devcontainerLog.Printf("Ensured directory exists: %s", devcontainerDir)

	devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")

	// Check if file already exists
	if _, err := os.Stat(devcontainerPath); err == nil {
		devcontainerLog.Printf("File already exists: %s", devcontainerPath)
		if verbose {
			fmt.Fprintf(os.Stderr, "Devcontainer already exists at %s (skipping)\n", devcontainerPath)
		}
		return nil
	}

	// Get current repository name from git remote
	repoName := getCurrentRepoName()
	if repoName == "" {
		repoName = "current-repo"
	}

	// Get the owner from the current repository
	owner := getRepoOwner()

	// Create repository permissions map
	// Reference: https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces
	// Default codespace permissions are read/write to the repository from which it was created.
	// For the current repo, we grant the standard codespace write permissions plus workflows:write
	// to enable triggering GitHub Actions workflows.
	// Note: Repository permissions can only be set for repositories in the same organization.
	repositories := map[string]DevcontainerRepoPermissions{
		repoName: {
			Permissions: map[string]string{
				"actions":       "write",
				"contents":      "write",
				"discussions":   "read",
				"issues":        "read",
				"pull-requests": "write",
				"workflows":     "write",
			},
		},
	}

	// Add additional repositories with read permissions
	// For additional repos, we grant default codespace read permissions plus workflows:read
	// to allow reading workflow definitions without write access.
	// Since permissions must be in the same organization, we automatically prepend the owner.
	// Reference: https://docs.github.com/en/codespaces/managing-your-codespaces/managing-repository-access-for-your-codespaces#setting-additional-repository-permissions
	for _, repo := range additionalRepos {
		if repo == "" {
			continue
		}

		// If repo already contains '/', validate that the owner matches
		// Otherwise, prepend the owner
		fullRepoName := repo
		if strings.Contains(repo, "/") {
			// Validate that the owner matches the current repo's owner
			parts := strings.Split(repo, "/")
			if len(parts) >= 2 {
				repoOwner := parts[0]
				if owner != "" && repoOwner != owner {
					return fmt.Errorf("repository '%s' is not in the same organization as the current repository (expected owner: '%s')", repo, owner)
				}
			}
		} else if owner != "" {
			fullRepoName = owner + "/" + repo
		}

		if fullRepoName != repoName {
			repositories[fullRepoName] = DevcontainerRepoPermissions{
				Permissions: map[string]string{
					"actions":       "read",
					"contents":      "read",
					"discussions":   "read",
					"issues":        "read",
					"pull-requests": "read",
					"workflows":     "read",
				},
			}
			devcontainerLog.Printf("Added read permissions for additional repo: %s", fullRepoName)
		}
	}

	// Create devcontainer configuration
	config := DevcontainerConfig{
		Name:  "Agentic Workflows Development",
		Image: "mcr.microsoft.com/devcontainers/universal:latest",
		Customizations: &DevcontainerCustomizations{
			VSCode: &DevcontainerVSCode{
				Extensions: []string{
					"GitHub.copilot",
					"GitHub.copilot-chat",
				},
			},
			Codespaces: &DevcontainerCodespaces{
				Repositories: repositories,
			},
		},
		Features: DevcontainerFeatures{
			"ghcr.io/devcontainers/features/github-cli:1":       map[string]any{},
			"ghcr.io/devcontainers/features/copilot-cli:latest": map[string]any{},
		},
		PostCreateCommand: "curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash",
	}

	// Write config file with proper indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal devcontainer.json: %w", err)
	}

	// Add newline at end of file
	data = append(data, '\n')

	if err := os.WriteFile(devcontainerPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write devcontainer.json: %w", err)
	}
	devcontainerLog.Printf("Created file: %s", devcontainerPath)

	return nil
}

// getCurrentRepoName gets the current repository name from git remote in owner/repo format
func getCurrentRepoName() string {
	// Try to get the repository name from git remote
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to directory name
		gitRoot, err := findGitRoot()
		if err != nil {
			return ""
		}
		return filepath.Base(gitRoot)
	}

	remoteURL := strings.TrimSpace(string(output))
	return parseGitHubRepoFromURL(remoteURL)
}

// getRepoOwner extracts the owner from the git remote URL
func getRepoOwner() string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	remoteURL := strings.TrimSpace(string(output))
	fullRepo := parseGitHubRepoFromURL(remoteURL)

	// Extract owner from "owner/repo" format
	parts := strings.Split(fullRepo, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// parseGitHubRepoFromURL extracts owner/repo from a GitHub URL
func parseGitHubRepoFromURL(url string) string {
	// Remove .git suffix if present
	url = strings.TrimSuffix(url, ".git")

	// Handle HTTPS URLs: https://github.com/owner/repo
	if strings.Contains(url, "github.com/") {
		parts := strings.Split(url, "github.com/")
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// Handle SSH URLs: git@github.com:owner/repo
	if strings.Contains(url, "git@github.com:") {
		parts := strings.Split(url, "git@github.com:")
		if len(parts) == 2 {
			return parts[1]
		}
	}

	return ""
}
