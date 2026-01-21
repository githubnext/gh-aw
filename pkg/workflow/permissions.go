package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var permissionsLog = logger.New("workflow:permissions")

// PermissionsParser provides functionality to parse and analyze GitHub Actions permissions
type PermissionsParser struct {
	rawPermissions string
	parsedPerms    map[string]string
	isShorthand    bool
	shorthandValue string
	hasAll         bool
	allLevel       string
}

// NewPermissionsParser creates a new PermissionsParser instance
func NewPermissionsParser(permissionsYAML string) *PermissionsParser {
	permissionsLog.Print("Creating new permissions parser")

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
		permissionsLog.Print("No permissions to parse")
		return
	}

	permissionsLog.Printf("Parsing permissions YAML: length=%d", len(p.rawPermissions))

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

	// Check if it's a shorthand permission (read-all, write-all, none)
	// Note: "read" and "write" are no longer valid shorthands as they create invalid GitHub Actions YAML
	shorthandPerms := []string{"read-all", "write-all", "none"}
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
		permissionsLog.Printf("Successfully parsed permissions map with %d keys", len(perms))

		// Handle 'all' key specially
		if allValue, exists := perms["all"]; exists {
			if strValue, ok := allValue.(string); ok {
				permissionsLog.Printf("Found 'all' permission with value: %s", strValue)
				if strValue == "write" {
					permissionsLog.Print("Invalid 'all: write' not allowed, ignoring permissions")
					// all: write is not allowed - don't set any permissions
					return
				}
				if strValue == "read" {
					// Check that no other permissions are set to 'none' when all: read is used
					for key, value := range perms {
						if key != "all" {
							if permValue, ok := value.(string); ok && permValue == "none" {
								permissionsLog.Printf("Invalid combination: all: read with %s: none", key)
								// all: read cannot be combined with : none - don't set any permissions
								return
							}
						}
					}
					p.hasAll = true
					p.allLevel = strValue
					permissionsLog.Print("Set hasAll=true with level=read")
				}
			}
		} // Convert any values to strings
		for key, value := range perms {
			if strValue, ok := value.(string); ok {
				p.parsedPerms[key] = strValue
			}
		}
		permissionsLog.Printf("Parsed %d permission entries", len(p.parsedPerms))
	} else {
		permissionsLog.Printf("Failed to parse permissions as YAML: %v", err)
	}
}

// HasContentsReadAccess returns true if the permissions allow reading contents
func (p *PermissionsParser) HasContentsReadAccess() bool {
	permissionsLog.Print("Checking contents read access")

	// Handle shorthand permissions
	if p.isShorthand {
		switch p.shorthandValue {
		case "read-all", "write-all":
			permissionsLog.Printf("Shorthand permissions grant contents read: %s", p.shorthandValue)
			return true
		case "none":
			permissionsLog.Print("Shorthand 'none' denies contents read")
			return false
		}
		return false
	}

	// Handle all: read case
	if p.hasAll && p.allLevel == "read" {
		// all: read grants contents access unless explicitly overridden
		if contentsLevel, exists := p.parsedPerms["contents"]; exists {
			return contentsLevel == "read" || contentsLevel == "write"
		}
		return true
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
	permissionsLog.Printf("Checking if scope=%s has level=%s", scope, level)

	// Handle shorthand permissions
	if p.isShorthand {
		permissionsLog.Printf("Using shorthand permission: %s", p.shorthandValue)
		switch p.shorthandValue {
		case "read-all":
			return level == "read"
		case "write-all":
			return level == "read" || level == "write"
		case "none":
			return false
		default:
			return false
		}
	}

	// Handle all: read case
	if p.hasAll && p.allLevel == "read" {
		// Check if there's an explicit permission for this scope
		if permLevel, exists := p.parsedPerms[scope]; exists {
			if level == "read" {
				// Read access is allowed if permission is "read" or "write"
				return permLevel == "read" || permLevel == "write"
			}
			return permLevel == level
		}
		// No explicit permission, use the "all" default
		// Special case: id-token doesn't support read level
		if scope == "id-token" && level == "read" {
			return false
		}
		return level == "read"
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
		// Handle 'all' key specially
		if allValue, exists := mapValue["all"]; exists {
			if strValue, ok := allValue.(string); ok {
				if strValue == "write" {
					// all: write is not allowed, return empty parser
					return parser
				}
				if strValue == "read" {
					// Check that no other permissions are set to 'none' when all: read is used
					for key, value := range mapValue {
						if key != "all" {
							if permValue, ok := value.(string); ok && permValue == "none" {
								// all: read cannot be combined with : none, return empty parser
								return parser
							}
						}
					}
					parser.hasAll = true
					parser.allLevel = strValue
				}
			}
		}

		for key, value := range mapValue {
			if strValue, ok := value.(string); ok {
				parser.parsedPerms[key] = strValue
			}
		}
	}

	return parser
}

// ToPermissions converts a PermissionsParser to a Permissions object
func (p *PermissionsParser) ToPermissions() *Permissions {
	if p == nil {
		return NewPermissions()
	}

	// Handle shorthand permissions
	if p.isShorthand {
		switch p.shorthandValue {
		case "read-all":
			return NewPermissionsReadAll()
		case "write-all":
			return NewPermissionsWriteAll()
		case "none":
			return NewPermissionsNone()
		default:
			return NewPermissions()
		}
	}

	// Handle all: read case
	if p.hasAll && p.allLevel == "read" {
		perms := NewPermissionsAllRead()

		// Apply explicit overrides from parsedPerms
		for key, value := range p.parsedPerms {
			if key == "all" {
				continue // Skip the "all" key itself
			}
			scope := convertStringToPermissionScope(key)
			if scope != "" {
				perms.Set(scope, PermissionLevel(value))
			}
		}

		return perms
	}

	// Handle explicit permissions map
	permsMap := make(map[PermissionScope]PermissionLevel)
	for key, value := range p.parsedPerms {
		if key == "all" {
			continue // Skip the "all" key
		}
		scope := convertStringToPermissionScope(key)
		if scope != "" {
			permsMap[scope] = PermissionLevel(value)
		}
	}

	return NewPermissionsFromMap(permsMap)
}

// convertStringToPermissionScope converts a string key to a PermissionScope
func convertStringToPermissionScope(key string) PermissionScope {
	switch key {
	case "actions":
		return PermissionActions
	case "attestations":
		return PermissionAttestations
	case "checks":
		return PermissionChecks
	case "contents":
		return PermissionContents
	case "deployments":
		return PermissionDeployments
	case "discussions":
		return PermissionDiscussions
	case "id-token":
		return PermissionIdToken
	case "issues":
		return PermissionIssues
	case "models":
		return PermissionModels
	case "packages":
		return PermissionPackages
	case "pages":
		return PermissionPages
	case "pull-requests":
		return PermissionPullRequests
	case "repository-projects":
		return PermissionRepositoryProj
	case "organization-projects":
		return PermissionOrganizationProj
	case "security-events":
		return PermissionSecurityEvents
	case "statuses":
		return PermissionStatuses
	default:
		return ""
	}
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
	PermissionActions          PermissionScope = "actions"
	PermissionAttestations     PermissionScope = "attestations"
	PermissionChecks           PermissionScope = "checks"
	PermissionContents         PermissionScope = "contents"
	PermissionDeployments      PermissionScope = "deployments"
	PermissionDiscussions      PermissionScope = "discussions"
	PermissionIdToken          PermissionScope = "id-token"
	PermissionIssues           PermissionScope = "issues"
	PermissionModels           PermissionScope = "models"
	PermissionPackages         PermissionScope = "packages"
	PermissionPages            PermissionScope = "pages"
	PermissionPullRequests     PermissionScope = "pull-requests"
	PermissionRepositoryProj   PermissionScope = "repository-projects"
	PermissionOrganizationProj PermissionScope = "organization-projects"
	PermissionSecurityEvents   PermissionScope = "security-events"
	PermissionStatuses         PermissionScope = "statuses"
)

// GetAllPermissionScopes returns all available permission scopes
func GetAllPermissionScopes() []PermissionScope {
	return []PermissionScope{
		PermissionActions,
		PermissionAttestations,
		PermissionChecks,
		PermissionContents,
		PermissionDeployments,
		PermissionDiscussions,
		PermissionIdToken,
		PermissionIssues,
		PermissionModels,
		PermissionPackages,
		PermissionPages,
		PermissionPullRequests,
		PermissionRepositoryProj,
		PermissionOrganizationProj,
		PermissionSecurityEvents,
		PermissionStatuses,
	}
}

// Permissions represents GitHub Actions permissions
// It can be a shorthand (read-all, write-all, read, write, none) or a map of scopes to levels
// It can also have an "all" permission that expands to all scopes
type Permissions struct {
	shorthand     string
	permissions   map[PermissionScope]PermissionLevel
	hasAll        bool
	allLevel      PermissionLevel
	explicitEmpty bool // When true, renders "permissions: {}" even if no permissions are set
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

// NewPermissionsNone creates a Permissions with none shorthand
func NewPermissionsNone() *Permissions {
	return &Permissions{
		shorthand: "none",
	}
}

// NewPermissionsEmpty creates a Permissions that explicitly renders as "permissions: {}"
func NewPermissionsEmpty() *Permissions {
	return &Permissions{
		permissions:   make(map[PermissionScope]PermissionLevel),
		explicitEmpty: true,
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

// NewPermissionsAllRead creates a Permissions with all: read
func NewPermissionsAllRead() *Permissions {
	return &Permissions{
		hasAll:   true,
		allLevel: PermissionRead,
	}
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
	if p.hasAll {
		// Convert from all to explicit map
		if p.permissions == nil {
			p.permissions = make(map[PermissionScope]PermissionLevel)
		}
		// Expand all permissions to explicit permissions first
		for _, s := range GetAllPermissionScopes() {
			if _, exists := p.permissions[s]; !exists {
				p.permissions[s] = p.allLevel
			}
		}
		p.hasAll = false
		p.allLevel = ""
	}
	p.permissions[scope] = level
}

// Get gets the permission level for a specific scope
func (p *Permissions) Get(scope PermissionScope) (PermissionLevel, bool) {
	if p.shorthand != "" {
		// Shorthand permissions apply to all scopes
		switch p.shorthand {
		case "read-all":
			return PermissionRead, true
		case "write-all":
			return PermissionWrite, true
		case "none":
			return PermissionNone, true
		}
		return "", false
	}

	// Check explicit permission first
	if level, exists := p.permissions[scope]; exists {
		return level, true
	}

	// If we have all: read, return that as default for any scope not explicitly set
	if p.hasAll {
		// Special case: id-token doesn't support read level
		if scope == PermissionIdToken && p.allLevel == PermissionRead {
			return "", false
		}
		return p.allLevel, true
	}

	return "", false
}

// Merge merges another Permissions into this one
// Write permission takes precedence over read (write implies read)
// Individual scope permissions override shorthand
func (p *Permissions) Merge(other *Permissions) {
	if other == nil {
		return
	}

	// Handle all permissions - convert to explicit first if needed
	if p.hasAll || other.hasAll {
		// Convert both to explicit maps
		if p.hasAll {
			if p.permissions == nil {
				p.permissions = make(map[PermissionScope]PermissionLevel)
			}
			for _, scope := range GetAllPermissionScopes() {
				if _, exists := p.permissions[scope]; !exists {
					// Skip id-token when level is read since it doesn't support read
					if scope == PermissionIdToken && p.allLevel == PermissionRead {
						continue
					}
					p.permissions[scope] = p.allLevel
				}
			}
			p.hasAll = false
			p.allLevel = ""
		}
		if other.hasAll {
			if other.permissions == nil {
				// Create a temporary map for merging
				tempPerms := make(map[PermissionScope]PermissionLevel)
				for _, scope := range GetAllPermissionScopes() {
					// Skip id-token when level is read since it doesn't support read
					if scope == PermissionIdToken && other.allLevel == PermissionRead {
						continue
					}
					tempPerms[scope] = other.allLevel
				}
				// Merge the temporary map
				for scope, otherLevel := range tempPerms {
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
				// Also merge explicit permissions from other if any
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
				return
			}
		}
	}

	// If other has shorthand, we need to handle it specially
	if other.shorthand != "" {
		// If we also have shorthand, resolve the conflict
		if p.shorthand != "" {
			// Promote to the higher permission level
			if other.shorthand == "write-all" || p.shorthand == "write-all" {
				p.shorthand = "write-all"
			} else if other.shorthand == "read-all" || p.shorthand == "read-all" {
				p.shorthand = "read-all"
			}
			// none is lowest, so only keep if both are none
			return
		}
		// We have map, other has shorthand - expand our map
		// Apply other's shorthand as baseline, then our specific permissions override
		otherLevel := PermissionNone
		switch other.shorthand {
		case "read-all":
			otherLevel = PermissionRead
		case "write-all":
			otherLevel = PermissionWrite
		}

		// For all scopes we don't have, set to other's shorthand level
		allScopes := GetAllPermissionScopes()
		for _, scope := range allScopes {
			if _, exists := p.permissions[scope]; !exists && otherLevel != PermissionNone {
				// Skip id-token when level is read since it doesn't support read
				if scope == PermissionIdToken && otherLevel == PermissionRead {
					continue
				}
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

	// Collect all permissions to render
	allPerms := make(map[PermissionScope]PermissionLevel)

	if p.hasAll {
		// Expand all: read/write to individual permissions
		for _, scope := range GetAllPermissionScopes() {
			// Skip id-token when expanding all: read since id-token doesn't support read level
			if scope == PermissionIdToken && p.allLevel == PermissionRead {
				continue
			}
			allPerms[scope] = p.allLevel
		}
	}

	// Override with explicit permissions
	for scope, level := range p.permissions {
		allPerms[scope] = level
	}

	if len(allPerms) == 0 {
		// If explicitEmpty is true, render "permissions: {}"
		if p.explicitEmpty {
			return "permissions: {}"
		}
		return ""
	}

	// Sort scopes for consistent output
	var scopes []string
	for scope := range allPerms {
		scopes = append(scopes, string(scope))
	}
	sort.Strings(scopes)

	var lines []string
	lines = append(lines, "permissions:")
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := allPerms[scope]

		// Skip organization-projects - it's only valid for GitHub App tokens, not workflow permissions
		if scope == PermissionOrganizationProj {
			continue
		}

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

// NewPermissionsActionsWrite creates permissions with actions: write
// This is required for dispatching workflows via workflow_dispatch
func NewPermissionsActionsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionActions: PermissionWrite,
	})
}

// NewPermissionsActionsWriteContentsWriteIssuesWritePRWrite creates permissions with actions: write, contents: write, issues: write, pull-requests: write
// This is required for the replaceActorsForAssignable GraphQL mutation used to assign GitHub Copilot agents to issues
func NewPermissionsActionsWriteContentsWriteIssuesWritePRWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionActions:      PermissionWrite,
		PermissionContents:     PermissionWrite,
		PermissionIssues:       PermissionWrite,
		PermissionPullRequests: PermissionWrite,
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

// NewPermissionsContentsReadProjectsWrite creates permissions with contents: read and organization-projects: write
// Note: organization-projects is only valid for GitHub App tokens, not workflow permissions
func NewPermissionsContentsReadProjectsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:         PermissionRead,
		PermissionOrganizationProj: PermissionWrite,
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

// NewPermissionsContentsWriteIssuesWritePRWriteDiscussionsWrite creates permissions with contents: write, issues: write, pull-requests: write, discussions: write
func NewPermissionsContentsWriteIssuesWritePRWriteDiscussionsWrite() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:     PermissionWrite,
		PermissionIssues:       PermissionWrite,
		PermissionPullRequests: PermissionWrite,
		PermissionDiscussions:  PermissionWrite,
	})
}
