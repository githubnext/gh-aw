package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

// Permissions represents GitHub Actions workflow permissions
type Permissions struct {
	// Global permission mode (if any)
	Global string // "read", "write", "read-all", "write-all", or "" for individual permissions

	// Individual permissions
	Actions         string // read, write, none
	Checks          string // read, write, none
	Contents        string // read, write, none
	Deployments     string // read, write, none
	Discussions     string // read, write, none
	Issues          string // read, write, none
	Metadata        string // read, write, none
	Models          string // read, write, none
	Packages        string // read, write, none
	Pages           string // read, write, none
	PullRequests    string // read, write, none
	RepositoryProjects string // read, write, none
	SecurityEvents  string // read, write, none
	Statuses        string // read, write, none
}

// ParsePermissions parses permissions from frontmatter
func ParsePermissions(frontmatter map[string]any) (*Permissions, error) {
	perms := &Permissions{}

	permissionsValue, exists := frontmatter["permissions"]
	if !exists {
		// No permissions specified, return nil so we can set defaults
		return nil, nil
	}

	switch v := permissionsValue.(type) {
	case string:
		// Handle global permission strings
		switch strings.TrimSpace(v) {
		case "read":
			perms.Global = "read"
		case "write":
			perms.Global = "write"
		case "read-all":
			perms.Global = "read-all"
		case "write-all":
			perms.Global = "write-all"
		default:
			return nil, fmt.Errorf("invalid global permission: %s", v)
		}
	case map[string]any:
		// Handle individual permissions object
		for key, value := range v {
			if valueStr, ok := value.(string); ok {
				if err := perms.setPermission(key, valueStr); err != nil {
					return nil, fmt.Errorf("invalid permission %s: %w", key, err)
				}
			}
		}
	case nil:
		// Empty permissions: {} case - YAML parses as nil
		// This means no permissions at all
		return perms, nil
	default:
		return nil, fmt.Errorf("invalid permissions format: expected string or object, got %T", v)
	}

	return perms, nil
}

// setPermission sets an individual permission
func (p *Permissions) setPermission(key, value string) error {
	// Normalize permission value
	value = strings.TrimSpace(strings.ToLower(value))
	if value != "read" && value != "write" && value != "none" {
		return fmt.Errorf("invalid permission value: %s (must be read, write, or none)", value)
	}

	// Set the appropriate field
	switch strings.ToLower(key) {
	case "actions":
		p.Actions = value
	case "checks":
		p.Checks = value
	case "contents":
		p.Contents = value
	case "deployments":
		p.Deployments = value
	case "discussions":
		p.Discussions = value
	case "issues":
		p.Issues = value
	case "metadata":
		p.Metadata = value
	case "models":
		p.Models = value
	case "packages":
		p.Packages = value
	case "pages":
		p.Pages = value
	case "pull-requests":
		p.PullRequests = value
	case "repository-projects":
		p.RepositoryProjects = value
	case "security-events":
		p.SecurityEvents = value
	case "statuses":
		p.Statuses = value
	default:
		return fmt.Errorf("unknown permission: %s", key)
	}

	return nil
}

// HasContentsAccess returns true if permissions include contents access
func (p *Permissions) HasContentsAccess() bool {
	// Check global permissions that include contents access
	if p.Global == "read" || p.Global == "write" || p.Global == "read-all" || p.Global == "write-all" {
		return true
	}

	// Check explicit contents permissions
	if p.Contents == "read" || p.Contents == "write" {
		return true
	}

	return false
}

// IsEmpty returns true if no permissions are set (equivalent to permissions: {})
func (p *Permissions) IsEmpty() bool {
	// Check if global permission is empty and all individual permissions are empty
	return p.Global == "" &&
		p.Actions == "" && p.Checks == "" && p.Contents == "" && p.Deployments == "" &&
		p.Discussions == "" && p.Issues == "" && p.Metadata == "" && p.Models == "" &&
		p.Packages == "" && p.Pages == "" && p.PullRequests == "" && p.RepositoryProjects == "" &&
		p.SecurityEvents == "" && p.Statuses == ""
}

// ToYAML serializes permissions back to YAML string format for GitHub Actions
func (p *Permissions) ToYAML() string {
	// Handle global permissions first
	if p.Global != "" {
		return fmt.Sprintf("permissions: %s", p.Global)
	}

	// Handle individual permissions
	perms := make(map[string]string)
	
	if p.Actions != "" {
		perms["actions"] = p.Actions
	}
	if p.Checks != "" {
		perms["checks"] = p.Checks
	}
	if p.Contents != "" {
		perms["contents"] = p.Contents
	}
	if p.Deployments != "" {
		perms["deployments"] = p.Deployments
	}
	if p.Discussions != "" {
		perms["discussions"] = p.Discussions
	}
	if p.Issues != "" {
		perms["issues"] = p.Issues
	}
	if p.Metadata != "" {
		perms["metadata"] = p.Metadata
	}
	if p.Models != "" {
		perms["models"] = p.Models
	}
	if p.Packages != "" {
		perms["packages"] = p.Packages
	}
	if p.Pages != "" {
		perms["pages"] = p.Pages
	}
	if p.PullRequests != "" {
		perms["pull-requests"] = p.PullRequests
	}
	if p.RepositoryProjects != "" {
		perms["repository-projects"] = p.RepositoryProjects
	}
	if p.SecurityEvents != "" {
		perms["security-events"] = p.SecurityEvents
	}
	if p.Statuses != "" {
		perms["statuses"] = p.Statuses
	}

	// If no permissions set, return empty permissions
	if len(perms) == 0 {
		return "permissions: {}"
	}

	// Convert to YAML
	yamlData := map[string]map[string]string{
		"permissions": perms,
	}

	bytes, err := yaml.Marshal(yamlData)
	if err != nil {
		// Fallback to manual formatting if YAML marshal fails
		return p.toYAMLManual(perms)
	}

	return strings.TrimSpace(string(bytes))
}

// toYAMLManual creates YAML manually as a fallback
func (p *Permissions) toYAMLManual(perms map[string]string) string {
	if len(perms) == 0 {
		return "permissions: {}"
	}

	var lines []string
	lines = append(lines, "permissions:")

	// Sort keys for consistent output
	keys := make([]string, 0, len(perms))
	for k := range perms {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("  %s: %s", key, perms[key]))
	}

	return strings.Join(lines, "\n")
}