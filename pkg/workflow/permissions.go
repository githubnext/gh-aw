package workflow

import (
	"strings"

	"gopkg.in/yaml.v3"
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
