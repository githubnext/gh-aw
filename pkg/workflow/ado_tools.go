package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	defaultADOMCPURL     = "https://mcp.dev.azure.com/"
	adoReadOnlyHeader    = "X-MCP-Readonly"
	adoToolsetsHeader    = "X-MCP-Toolsets"
	adoReadOnlyValue     = "true"
	adoOrganizationLimit = 49
)

var adoOrganizationPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,47}[A-Za-z0-9])?$`)

var adoSupportedToolsets = map[string]struct{}{
	"all":       {},
	"advsec":    {},
	"elm":       {},
	"pipelines": {},
	"repos":     {},
	"testplan":  {},
	"wiki":      {},
	"wit":       {},
	"work":      {},
}

var adoRequiredDomains = []string{
	"*.dev.azure.com",
	"*.microsoftonline.com",
	"*.visualstudio.com",
	"app.vssps.visualstudio.com",
}

// expandADOToolConfig converts the first-class Azure DevOps configuration into
// the generic HTTP MCP shape consumed by the MCP gateway pipeline.
func expandADOToolConfig(tools map[string]any) error {
	raw, exists := tools["ado"]
	if !exists {
		return nil
	}
	if enabled, ok := raw.(bool); ok && !enabled {
		delete(tools, "ado")
		return nil
	}

	config, ok := raw.(map[string]any)
	if !ok {
		return errors.New("tools.ado must be an object with an organization")
	}
	if err := validateADOToolConfig(config); err != nil {
		return err
	}

	organization, ok := config["organization"].(string)
	if !ok {
		return errors.New("tools.ado.organization is required")
	}
	headers := map[string]any{
		adoReadOnlyHeader: adoReadOnlyValue,
	}
	if toolsets, exists := config["toolsets"]; exists {
		values, _ := parseADOStringList(toolsets, "toolsets")
		headers[adoToolsetsHeader] = strings.Join(values, ",")
	}

	expanded := map[string]any{
		"type":    "http",
		"url":     defaultADOMCPURL + organization,
		"headers": headers,
	}
	if allowed, exists := config["allowed"]; exists {
		expanded["allowed"] = allowed
	}
	tools["ado"] = expanded
	return nil
}

func validateADOToolConfig(config map[string]any) error {
	for field := range config {
		switch field {
		case "organization", "toolsets", "allowed":
		default:
			return fmt.Errorf("tools.ado.%s is not supported; valid fields are: allowed, organization, toolsets", field)
		}
	}

	organization, ok := config["organization"].(string)
	if !ok || organization == "" {
		return errors.New("tools.ado.organization is required")
	}
	if len(organization) > adoOrganizationLimit || !adoOrganizationPattern.MatchString(organization) {
		return errors.New("tools.ado.organization must contain only letters, numbers, and interior hyphens, and be under 50 characters")
	}

	if value, exists := config["toolsets"]; exists {
		toolsets, err := parseADOStringList(value, "toolsets")
		if err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(toolsets))
		for _, toolset := range toolsets {
			if _, supported := adoSupportedToolsets[toolset]; !supported {
				return fmt.Errorf("tools.ado.toolsets contains unsupported toolset %q", toolset)
			}
			if _, duplicate := seen[toolset]; duplicate {
				return fmt.Errorf("tools.ado.toolsets contains duplicate toolset %q", toolset)
			}
			seen[toolset] = struct{}{}
		}
		if len(toolsets) > 1 && slices.Contains(toolsets, "all") {
			return errors.New(`tools.ado.toolsets must contain only "all" when the all toolset is used`)
		}
	}

	if value, exists := config["allowed"]; exists {
		allowed, err := parseADOStringList(value, "allowed")
		if err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(allowed))
		for _, tool := range allowed {
			if !isValidADOToolName(tool) {
				return fmt.Errorf("tools.ado.allowed contains invalid tool name %q", tool)
			}
			if _, duplicate := seen[tool]; duplicate {
				return fmt.Errorf("tools.ado.allowed contains duplicate tool %q", tool)
			}
			seen[tool] = struct{}{}
		}
	}
	return nil
}

func parseADOStringList(value any, field string) ([]string, error) {
	var values []string
	switch raw := value.(type) {
	case []string:
		values = raw
	case []any:
		values = make([]string, 0, len(raw))
		for _, item := range raw {
			value, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("tools.ado.%s must contain only non-empty strings", field)
			}
			values = append(values, value)
		}
	default:
		return nil, fmt.Errorf("tools.ado.%s must be a non-empty array of strings", field)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("tools.ado.%s must be a non-empty array of strings", field)
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("tools.ado.%s must contain only non-empty strings", field)
		}
	}
	return values, nil
}

func isValidADOToolName(name string) bool {
	if name == "" {
		return false
	}
	for i, char := range name {
		if (char >= 'a' && char <= 'z') || (i > 0 && char >= '0' && char <= '9') || (i > 0 && char == '_') {
			continue
		}
		return false
	}
	return true
}
