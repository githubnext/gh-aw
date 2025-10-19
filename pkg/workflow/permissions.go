package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

// PermissionsParser provides functionality to parse and analyze GitHub Actions permissions
type PermissionsParser struct {
	rawPermissions string
	parsedPerms    map[string]string
	isShorthand    bool
	shorthandValue string
}

// NewPermissionsParser creates a new PermissionsParser instance
func NewPermissionsParser(permissionsYAML string) *PermissionsParser {
	parser := &PermissionsParser{
		rawPermissions: permissionsYAML,
		parsedPerms:    make(map[string]string),
	}
	parser.parse()
	return parser
}

// parse parses the permissions YAML and populates the internal structures
func (p *PermissionsParser) parse() {
	if p.rawPermissions == "" {
		return
	}

	// Remove the "permissions:" prefix if present and get just the YAML content
	yamlContent := strings.TrimSpace(p.rawPermissions)
	if strings.HasPrefix(yamlContent, "permissions:") {
		// Extract everything after "permissions:"
		lines := strings.Split(yamlContent, "\n")
		if len(lines) > 1 {
			// Get the lines after the first, and normalize indentation
			contentLines := lines[1:]
			var normalizedLines []string

			// Find the common indentation to remove
			minIndent := -1
			for _, line := range contentLines {
				if strings.TrimSpace(line) == "" {
					continue // Skip empty lines
				}
				indent := 0
				for _, r := range line {
					if r == ' ' || r == '\t' {
						indent++
					} else {
						break
					}
				}
				if minIndent == -1 || indent < minIndent {
					minIndent = indent
				}
			}

			// Remove common indentation from all lines
			if minIndent > 0 {
				for _, line := range contentLines {
					if strings.TrimSpace(line) == "" {
						normalizedLines = append(normalizedLines, "")
					} else if len(line) > minIndent {
						normalizedLines = append(normalizedLines, line[minIndent:])
					} else {
						normalizedLines = append(normalizedLines, line)
					}
				}
			} else {
				normalizedLines = contentLines
			}

			yamlContent = strings.Join(normalizedLines, "\n")
		} else {
			// Single line format like "permissions: read-all"
			parts := strings.SplitN(lines[0], ":", 2)
			if len(parts) == 2 {
				yamlContent = strings.TrimSpace(parts[1])
			}
		}
	}

	yamlContent = strings.TrimSpace(yamlContent)
	if yamlContent == "" {
		return
	}

	// Check if it's a shorthand permission (read-all, write-all, read, write, none)
	shorthandPerms := []string{"read-all", "write-all", "read", "write", "none"}
	for _, shorthand := range shorthandPerms {
		if yamlContent == shorthand {
			p.isShorthand = true
			p.shorthandValue = shorthand
			return
		}
	}

	// Try to parse as YAML map
	var perms map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &perms); err == nil {
		// Convert any values to strings
		for key, value := range perms {
			if strValue, ok := value.(string); ok {
				p.parsedPerms[key] = strValue
			}
		}
	}
}

// HasContentsReadAccess returns true if the permissions allow reading contents
func (p *PermissionsParser) HasContentsReadAccess() bool {
	// Handle shorthand permissions
	if p.isShorthand {
		switch p.shorthandValue {
		case "read-all", "write-all", "read", "write":
			return true
		case "none":
			return false
		}
		return false
	}

	// Handle explicit permissions map
	if contentsLevel, exists := p.parsedPerms["contents"]; exists {
		return contentsLevel == "read" || contentsLevel == "write"
	}

	// Default: if no contents permission is specified, assume no access
	return false
}

// IsAllowed checks if a specific permission scope has the specified access level
// scope: "contents", "issues", "pull-requests", etc.
// level: "read", "write", "none"
func (p *PermissionsParser) IsAllowed(scope, level string) bool {
	// Handle shorthand permissions
	if p.isShorthand {
		switch p.shorthandValue {
		case "read-all":
			return level == "read"
		case "write-all":
			return level == "read" || level == "write"
		case "read":
			return level == "read"
		case "write":
			return level == "read" || level == "write"
		case "none":
			return false
		default:
			return false
		}
	}

	// Handle explicit permissions map
	if permLevel, exists := p.parsedPerms[scope]; exists {
		if level == "read" {
			// Read access is allowed if permission is "read" or "write"
			return permLevel == "read" || permLevel == "write"
		}
		return permLevel == level
	}

	// Default: permission not specified means no access
	return false
}

// NewPermissionsParserFromValue creates a PermissionsParser from a frontmatter value (any type)
func NewPermissionsParserFromValue(permissionsValue any) *PermissionsParser {
	parser := &PermissionsParser{
		parsedPerms: make(map[string]string),
	}

	if permissionsValue == nil {
		return parser
	}

	// Handle string shorthand (read-all, write-all, etc.)
	if strValue, ok := permissionsValue.(string); ok {
		parser.isShorthand = true
		parser.shorthandValue = strValue
		return parser
	}

	// Handle map format
	if mapValue, ok := permissionsValue.(map[string]any); ok {
		for key, value := range mapValue {
			if strValue, ok := value.(string); ok {
				parser.parsedPerms[key] = strValue
			}
		}
	}

	return parser
}

// ContainsCheckout returns true if the given custom steps contain an actions/checkout step
func ContainsCheckout(customSteps string) bool {
	if customSteps == "" {
		return false
	}

	// Look for actions/checkout usage patterns
	checkoutPatterns := []string{
		"actions/checkout@",
		"uses: actions/checkout",
		"- uses: actions/checkout",
	}

	lowerSteps := strings.ToLower(customSteps)
	for _, pattern := range checkoutPatterns {
		if strings.Contains(lowerSteps, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// PermissionLevel represents the level of access (read, write, none)
type PermissionLevel string

const (
	PermissionRead  PermissionLevel = "read"
	PermissionWrite PermissionLevel = "write"
	PermissionNone  PermissionLevel = "none"
)

// PermissionScope represents a GitHub Actions permission scope
type PermissionScope string

const (
	PermissionActions        PermissionScope = "actions"
	PermissionChecks         PermissionScope = "checks"
	PermissionContents       PermissionScope = "contents"
	PermissionDeployments    PermissionScope = "deployments"
	PermissionDiscussions    PermissionScope = "discussions"
	PermissionIssues         PermissionScope = "issues"
	PermissionPackages       PermissionScope = "packages"
	PermissionPages          PermissionScope = "pages"
	PermissionPullRequests   PermissionScope = "pull-requests"
	PermissionRepositoryProj PermissionScope = "repository-projects"
	PermissionSecurityEvents PermissionScope = "security-events"
	PermissionStatuses       PermissionScope = "statuses"
	PermissionModels         PermissionScope = "models"
)

// Permissions represents GitHub Actions permissions
// It can be a shorthand (read-all, write-all, read, write, none) or a map of scopes to levels
type Permissions struct {
	shorthand   string
	permissions map[PermissionScope]PermissionLevel
}

// NewPermissions creates a new Permissions with an empty map
func NewPermissions() *Permissions {
	return &Permissions{
		permissions: make(map[PermissionScope]PermissionLevel),
	}
}

// NewPermissionsReadAll creates a Permissions with read-all shorthand
func NewPermissionsReadAll() *Permissions {
	return &Permissions{
		shorthand: "read-all",
	}
}

// NewPermissionsWriteAll creates a Permissions with write-all shorthand
func NewPermissionsWriteAll() *Permissions {
	return &Permissions{
		shorthand: "write-all",
	}
}

// NewPermissionsRead creates a Permissions with read shorthand
func NewPermissionsRead() *Permissions {
	return &Permissions{
		shorthand: "read",
	}
}

// NewPermissionsWrite creates a Permissions with write shorthand
func NewPermissionsWrite() *Permissions {
	return &Permissions{
		shorthand: "write",
	}
}

// NewPermissionsNone creates a Permissions with none shorthand
func NewPermissionsNone() *Permissions {
	return &Permissions{
		shorthand: "none",
	}
}

// NewPermissionsFromMap creates a Permissions from a map of scopes to levels
func NewPermissionsFromMap(perms map[PermissionScope]PermissionLevel) *Permissions {
	p := NewPermissions()
	for scope, level := range perms {
		p.permissions[scope] = level
	}
	return p
}

// Set sets a permission for a specific scope
func (p *Permissions) Set(scope PermissionScope, level PermissionLevel) {
	if p.shorthand != "" {
		// Convert from shorthand to map
		p.shorthand = ""
		if p.permissions == nil {
			p.permissions = make(map[PermissionScope]PermissionLevel)
		}
	}
	p.permissions[scope] = level
}

// Get gets the permission level for a specific scope
func (p *Permissions) Get(scope PermissionScope) (PermissionLevel, bool) {
	if p.shorthand != "" {
		// Shorthand permissions apply to all scopes
		switch p.shorthand {
		case "read-all", "read":
			return PermissionRead, true
		case "write-all", "write":
			return PermissionWrite, true
		case "none":
			return PermissionNone, true
		}
		return "", false
	}
	level, exists := p.permissions[scope]
	return level, exists
}

// Merge merges another Permissions into this one
// Write permission takes precedence over read (write implies read)
// Individual scope permissions override shorthand
func (p *Permissions) Merge(other *Permissions) {
	if other == nil {
		return
	}

	// If other has shorthand, we need to handle it specially
	if other.shorthand != "" {
		// If we also have shorthand, resolve the conflict
		if p.shorthand != "" {
			// Promote to the higher permission level
			if other.shorthand == "write-all" || p.shorthand == "write-all" {
				p.shorthand = "write-all"
			} else if other.shorthand == "write" || p.shorthand == "write" {
				p.shorthand = "write"
			} else if other.shorthand == "read-all" || p.shorthand == "read-all" {
				p.shorthand = "read-all"
			} else if other.shorthand == "read" || p.shorthand == "read" {
				p.shorthand = "read"
			}
			// none is lowest, so only keep if both are none
			return
		}
		// We have map, other has shorthand - expand our map
		// Apply other's shorthand as baseline, then our specific permissions override
		otherLevel := PermissionNone
		switch other.shorthand {
		case "read-all", "read":
			otherLevel = PermissionRead
		case "write-all", "write":
			otherLevel = PermissionWrite
		}

		// For all scopes we don't have, set to other's shorthand level
		allScopes := []PermissionScope{
			PermissionActions, PermissionChecks, PermissionContents,
			PermissionDeployments, PermissionDiscussions, PermissionIssues,
			PermissionPackages, PermissionPages, PermissionPullRequests,
			PermissionRepositoryProj, PermissionSecurityEvents, PermissionStatuses,
			PermissionModels,
		}
		for _, scope := range allScopes {
			if _, exists := p.permissions[scope]; !exists && otherLevel != PermissionNone {
				p.permissions[scope] = otherLevel
			}
		}
		return
	}

	// Both have maps, merge them
	if p.shorthand != "" {
		// We have shorthand, other has map - convert to map first
		p.shorthand = ""
		if p.permissions == nil {
			p.permissions = make(map[PermissionScope]PermissionLevel)
		}
	}

	// Merge permissions - write overrides read
	for scope, otherLevel := range other.permissions {
		currentLevel, exists := p.permissions[scope]
		if !exists {
			p.permissions[scope] = otherLevel
		} else {
			// Write takes precedence
			if otherLevel == PermissionWrite || currentLevel == PermissionWrite {
				p.permissions[scope] = PermissionWrite
			} else if otherLevel == PermissionRead || currentLevel == PermissionRead {
				p.permissions[scope] = PermissionRead
			} else {
				p.permissions[scope] = PermissionNone
			}
		}
	}
}

// RenderToYAML renders the Permissions to GitHub Actions YAML format
func (p *Permissions) RenderToYAML() string {
	if p == nil {
		return ""
	}

	if p.shorthand != "" {
		return fmt.Sprintf("permissions: %s", p.shorthand)
	}

	if len(p.permissions) == 0 {
		return ""
	}

	// Sort scopes for consistent output
	var scopes []string
	for scope := range p.permissions {
		scopes = append(scopes, string(scope))
	}
	sort.Strings(scopes)

	var lines []string
	lines = append(lines, "permissions:")
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := p.permissions[scope]
		// Add 2 spaces for proper indentation under permissions:
		// When rendered in a job, the job renderer adds 4 spaces to the first line only,
		// so we need to pre-indent continuation lines with 4 additional spaces
		// to get 6 total spaces (4 from job + 2 for being under permissions)
		lines = append(lines, fmt.Sprintf("      %s: %s", scope, level))
	}

	return strings.Join(lines, "\n")
}

// Helper functions for common permission patterns

// NewPermissionsContentsRead creates permissions with contents: read
func NewPermissionsContentsRead() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
	})
}

// NewPermissionsContentsReadIssuesWrite creates permissions with contents: read and issues: write
func NewPermissionsContentsReadIssuesWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionRead,
		PermissionIssues:   PermissionWrite,
	})
}

// NewPermissionsContentsReadIssuesWritePRWrite creates permissions with contents: read, issues: write, pull-requests: write
func NewPermissionsContentsReadIssuesWritePRWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:     PermissionRead,
		PermissionIssues:       PermissionWrite,
		PermissionPullRequests: PermissionWrite,
	})
}

// NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite creates permissions with contents: read, issues: write, pull-requests: write, discussions: write
func NewPermissionsContentsReadIssuesWritePRWriteDiscussionsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:     PermissionRead,
		PermissionIssues:       PermissionWrite,
		PermissionPullRequests: PermissionWrite,
		PermissionDiscussions:  PermissionWrite,
	})
}

// NewPermissionsContentsWrite creates permissions with contents: write
func NewPermissionsContentsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents: PermissionWrite,
	})
}

// NewPermissionsContentsWriteIssuesWritePRWrite creates permissions with contents: write, issues: write, pull-requests: write
func NewPermissionsContentsWriteIssuesWritePRWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:     PermissionWrite,
		PermissionIssues:       PermissionWrite,
		PermissionPullRequests: PermissionWrite,
	})
}

// NewPermissionsDiscussionsWrite creates permissions with discussions: write
func NewPermissionsDiscussionsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionDiscussions: PermissionWrite,
	})
}

// NewPermissionsContentsReadDiscussionsWrite creates permissions with contents: read and discussions: write
func NewPermissionsContentsReadDiscussionsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:    PermissionRead,
		PermissionDiscussions: PermissionWrite,
	})
}

// NewPermissionsContentsReadPRWrite creates permissions with contents: read and pull-requests: write
func NewPermissionsContentsReadPRWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:     PermissionRead,
		PermissionPullRequests: PermissionWrite,
	})
}

// NewPermissionsContentsReadSecurityEventsWrite creates permissions with contents: read and security-events: write
func NewPermissionsContentsReadSecurityEventsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:       PermissionRead,
		PermissionSecurityEvents: PermissionWrite,
	})
}

// NewPermissionsContentsReadSecurityEventsWriteActionsRead creates permissions with contents: read, security-events: write, actions: read
func NewPermissionsContentsReadSecurityEventsWriteActionsRead() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:       PermissionRead,
		PermissionSecurityEvents: PermissionWrite,
		PermissionActions:        PermissionRead,
	})
}

// NewPermissionsContentsWritePRReadIssuesRead creates permissions with contents: write, pull-requests: read, issues: read
func NewPermissionsContentsWritePRReadIssuesRead() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:     PermissionWrite,
		PermissionPullRequests: PermissionRead,
		PermissionIssues:       PermissionRead,
	})
}
