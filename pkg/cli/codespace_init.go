package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var codespaceInitLog = logger.New("cli:codespace_init")

// DevcontainerRepositoryPermissions represents the permissions for a repository in devcontainer.json
type DevcontainerRepositoryPermissions struct {
	Actions      string `json:"actions,omitempty"`
	Contents     string `json:"contents,omitempty"`
	Workflows    string `json:"workflows,omitempty"`
	Issues       string `json:"issues,omitempty"`
	PullRequests string `json:"pull-requests,omitempty"`
	Discussions  string `json:"discussions,omitempty"`
	Metadata     string `json:"metadata,omitempty"`
}

// DevcontainerRepositoryConfig represents repository configuration in devcontainer.json
type DevcontainerRepositoryConfig struct {
	Permissions DevcontainerRepositoryPermissions `json:"permissions"`
}

// DevcontainerCodespaces represents the codespaces section of devcontainer.json
type DevcontainerCodespaces struct {
	Repositories map[string]DevcontainerRepositoryConfig `json:"repositories"`
}

// DevcontainerCustomizations represents the customizations section of devcontainer.json
// Uses map[string]any to preserve all custom fields
type DevcontainerCustomizations struct {
	Codespaces *DevcontainerCodespaces `json:"codespaces,omitempty"`
	// Store additional fields not explicitly defined
	Extra map[string]any `json:"-"`
}

// MarshalJSON implements json.Marshaler for DevcontainerCustomizations
func (c *DevcontainerCustomizations) MarshalJSON() ([]byte, error) {
	// Create a map to hold all fields
	result := make(map[string]any)

	// Add extra fields first
	for k, v := range c.Extra {
		result[k] = v
	}

	// Add codespaces if present
	if c.Codespaces != nil {
		result["codespaces"] = c.Codespaces
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements json.Unmarshaler for DevcontainerCustomizations
func (c *DevcontainerCustomizations) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Extra = make(map[string]any)

	// Process each field
	for key, value := range raw {
		if key == "codespaces" {
			// Handle codespaces specially
			c.Codespaces = &DevcontainerCodespaces{}
			if err := json.Unmarshal(value, c.Codespaces); err != nil {
				return err
			}
		} else {
			// Store other fields in Extra
			var v any
			if err := json.Unmarshal(value, &v); err != nil {
				return err
			}
			c.Extra[key] = v
		}
	}

	return nil
}

// Devcontainer represents the structure of devcontainer.json
// Uses map[string]any to preserve all custom fields
type Devcontainer struct {
	Image          string                      `json:"image,omitempty"`
	Name           string                      `json:"name,omitempty"`
	Customizations *DevcontainerCustomizations `json:"customizations,omitempty"`
	// Store additional fields not explicitly defined
	Extra map[string]any `json:"-"`
}

// MarshalJSON implements json.Marshaler for Devcontainer
func (d *Devcontainer) MarshalJSON() ([]byte, error) {
	// Create a map to hold all fields
	result := make(map[string]any)

	// Add extra fields first
	for k, v := range d.Extra {
		result[k] = v
	}

	// Add known fields if present
	if d.Image != "" {
		result["image"] = d.Image
	}
	if d.Name != "" {
		result["name"] = d.Name
	}
	if d.Customizations != nil {
		result["customizations"] = d.Customizations
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements json.Unmarshaler for Devcontainer
func (d *Devcontainer) UnmarshalJSON(data []byte) error {
	// First unmarshal into a map
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	d.Extra = make(map[string]any)

	// Process each field
	for key, value := range raw {
		switch key {
		case "image":
			if err := json.Unmarshal(value, &d.Image); err != nil {
				return err
			}
		case "name":
			if err := json.Unmarshal(value, &d.Name); err != nil {
				return err
			}
		case "customizations":
			d.Customizations = &DevcontainerCustomizations{}
			if err := json.Unmarshal(value, d.Customizations); err != nil {
				return err
			}
		default:
			// Store other fields in Extra
			var v any
			if err := json.Unmarshal(value, &v); err != nil {
				return err
			}
			d.Extra[key] = v
		}
	}

	return nil
}

// ensureDevcontainerCodespace creates or updates .devcontainer/devcontainer.json with codespace permissions
func ensureDevcontainerCodespace(verbose bool) error {
	codespaceInitLog.Print("Ensuring devcontainer.json exists with codespace permissions")

	// Get the current repository slug
	repoSlug, err := GetCurrentRepoSlug()
	if err != nil {
		codespaceInitLog.Printf("Failed to get current repository slug: %v", err)
		return fmt.Errorf("failed to get current repository: %w", err)
	}
	codespaceInitLog.Printf("Current repository: %s", repoSlug)

	// Check for devcontainer.json in multiple locations
	devcontainerPaths := []string{
		".devcontainer/devcontainer.json",
		".devcontainer.json",
	}

	var devcontainerPath string
	var existingConfig *Devcontainer

	// Look for existing devcontainer.json
	for _, path := range devcontainerPaths {
		if data, err := os.ReadFile(path); err == nil {
			codespaceInitLog.Printf("Found existing devcontainer.json at: %s", path)
			devcontainerPath = path
			existingConfig = &Devcontainer{}
			if err := json.Unmarshal(data, existingConfig); err != nil {
				codespaceInitLog.Printf("Failed to parse existing devcontainer.json, will create new one: %v", err)
				existingConfig = nil
			}
			break
		}
	}

	// If no existing config found, create a new one in .devcontainer/
	if devcontainerPath == "" {
		devcontainerPath = ".devcontainer/devcontainer.json"
		codespaceInitLog.Printf("No existing devcontainer.json found, will create at: %s", devcontainerPath)
	}

	// Create or update the configuration
	var config *Devcontainer
	if existingConfig != nil {
		config = existingConfig
		codespaceInitLog.Print("Using existing configuration")
	} else {
		config = &Devcontainer{
			Image: "mcr.microsoft.com/devcontainers/universal:2",
		}
		codespaceInitLog.Print("Creating new basic configuration")
	}

	// Ensure customizations exists
	if config.Customizations == nil {
		config.Customizations = &DevcontainerCustomizations{}
	}

	// Ensure codespaces exists
	if config.Customizations.Codespaces == nil {
		config.Customizations.Codespaces = &DevcontainerCodespaces{
			Repositories: make(map[string]DevcontainerRepositoryConfig),
		}
	}

	// Ensure repositories map exists
	if config.Customizations.Codespaces.Repositories == nil {
		config.Customizations.Codespaces.Repositories = make(map[string]DevcontainerRepositoryConfig)
	}

	// Add permissions for the current repository
	currentRepoPerms := DevcontainerRepositoryConfig{
		Permissions: DevcontainerRepositoryPermissions{
			Actions:      "write",
			Contents:     "write",
			Workflows:    "write",
			Issues:       "write",
			PullRequests: "write",
			Discussions:  "write",
		},
	}

	// Check if current repo already has permissions configured
	if existingPerms, exists := config.Customizations.Codespaces.Repositories[repoSlug]; exists {
		codespaceInitLog.Printf("Repository %s already has permissions configured", repoSlug)
		// Merge permissions - only update if not already set
		if existingPerms.Permissions.Actions == "" {
			existingPerms.Permissions.Actions = "write"
		}
		if existingPerms.Permissions.Contents == "" {
			existingPerms.Permissions.Contents = "write"
		}
		if existingPerms.Permissions.Workflows == "" {
			existingPerms.Permissions.Workflows = "write"
		}
		if existingPerms.Permissions.Issues == "" {
			existingPerms.Permissions.Issues = "write"
		}
		if existingPerms.Permissions.PullRequests == "" {
			existingPerms.Permissions.PullRequests = "write"
		}
		if existingPerms.Permissions.Discussions == "" {
			existingPerms.Permissions.Discussions = "write"
		}
		config.Customizations.Codespaces.Repositories[repoSlug] = existingPerms
	} else {
		codespaceInitLog.Printf("Adding permissions for repository: %s", repoSlug)
		config.Customizations.Codespaces.Repositories[repoSlug] = currentRepoPerms
	}

	// Add read permissions for githubnext/gh-aw releases
	ghAwRepo := "githubnext/gh-aw"
	ghAwPerms := DevcontainerRepositoryConfig{
		Permissions: DevcontainerRepositoryPermissions{
			Contents: "read",
			Metadata: "read",
		},
	}

	// Check if gh-aw repo already has permissions configured
	if existingPerms, exists := config.Customizations.Codespaces.Repositories[ghAwRepo]; exists {
		codespaceInitLog.Printf("Repository %s already has permissions configured", ghAwRepo)
		// Only update if not already set to read or write
		if existingPerms.Permissions.Contents == "" {
			existingPerms.Permissions.Contents = "read"
		}
		if existingPerms.Permissions.Metadata == "" {
			existingPerms.Permissions.Metadata = "read"
		}
		config.Customizations.Codespaces.Repositories[ghAwRepo] = existingPerms
	} else {
		codespaceInitLog.Printf("Adding read permissions for repository: %s", ghAwRepo)
		config.Customizations.Codespaces.Repositories[ghAwRepo] = ghAwPerms
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal devcontainer.json: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(devcontainerPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		codespaceInitLog.Printf("Ensured directory exists: %s", dir)
	}

	// Write the file
	if err := os.WriteFile(devcontainerPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write devcontainer.json: %w", err)
	}

	codespaceInitLog.Printf("Successfully wrote devcontainer.json to: %s", devcontainerPath)

	if verbose {
		fmt.Fprintf(os.Stderr, "Configured %s with codespace permissions\n", devcontainerPath)
		fmt.Fprintf(os.Stderr, "  - Added permissions for %s\n", repoSlug)
		fmt.Fprintf(os.Stderr, "  - Added read permissions for %s\n", ghAwRepo)
	}

	return nil
}
