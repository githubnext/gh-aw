package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

// ensureDevcontainerConfig creates or updates .devcontainer/devcontainer.json
func ensureDevcontainerConfig(verbose bool, additionalRepos []string) error {
	devcontainerLog.Printf("Creating or updating .devcontainer/devcontainer.json with additional repos: %v", additionalRepos)

	// Create .devcontainer directory if it doesn't exist
	devcontainerDir := ".devcontainer"
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
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

	// Create repository permissions map
	repositories := map[string]DevcontainerRepoPermissions{
		repoName: {
			Permissions: map[string]string{
				"contents":      "write",
				"pull-requests": "write",
				"workflows":     "write",
			},
		},
		"githubnext/gh-aw": {
			Permissions: map[string]string{
				"contents": "read",
			},
		},
	}

	// Add additional repositories with read permissions
	for _, repo := range additionalRepos {
		if repo != "" && repo != repoName && repo != "githubnext/gh-aw" {
			repositories[repo] = DevcontainerRepoPermissions{
				Permissions: map[string]string{
					"contents": "read",
				},
			}
			devcontainerLog.Printf("Added read permission for additional repo: %s", repo)
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
			"ghcr.io/devcontainers/features/github-cli:1": map[string]any{},
		},
		PostCreateCommand: "curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash && npm install -g @github/copilot",
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

// getCurrentRepoName gets the current repository name from git remote
func getCurrentRepoName() string {
	// Try to get the repository name from git remote
	// This is a simple implementation that may not work in all cases
	// but provides a reasonable default
	gitRoot, err := findGitRoot()
	if err != nil {
		return ""
	}

	// Get the directory name as fallback
	return filepath.Base(gitRoot)
}
