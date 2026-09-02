package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	defaultADOMCPURL     = "https://mcp.dev.azure.com"
	adoReadonlyHeader    = "X-MCP-Readonly"
	adoToolsetsHeader    = "X-MCP-Toolsets"
	adoAuthorizationName = "Authorization"
)

var (
	adoOrganizationPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,48}[A-Za-z0-9])?$`)
	adoToolsetNames        = map[string]struct{}{
		"all": {}, "repos": {}, "advsec": {}, "wit": {}, "pipelines": {},
		"wiki": {}, "work": {}, "testplan": {}, "elm": {},
	}
)

// expandADOToolConfig converts the first-class Azure DevOps configuration into
// the generic HTTP MCP shape consumed by the existing gateway pipeline.
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
		return errors.New("tools.ado must be an object with organization and token fields")
	}
	organization, token, toolsets, err := validateADOToolConfig(config)
	if err != nil {
		return err
	}

	headers := map[string]any{
		adoAuthorizationName: "Bearer " + token,
		adoReadonlyHeader:    "true",
	}
	if len(toolsets) > 0 {
		headers[adoToolsetsHeader] = strings.Join(toolsets, ",")
	}
	tools["ado"] = map[string]any{
		"type":    "http",
		"url":     fmt.Sprintf("%s/%s", defaultADOMCPURL, organization),
		"headers": headers,
	}
	return nil
}

func validateADOToolConfig(config map[string]any) (string, string, []string, error) {
	for field := range config {
		switch field {
		case "organization", "token", "toolsets":
		default:
			return "", "", nil, fmt.Errorf("tools.ado.%s is not supported; valid fields are: organization, token, toolsets", field)
		}
	}

	organization, _ := config["organization"].(string)
	if !adoOrganizationPattern.MatchString(organization) {
		return "", "", nil, errors.New("tools.ado.organization must be an Azure DevOps organization name containing only letters, numbers, and hyphens, with no leading or trailing hyphen")
	}

	token, _ := config["token"].(string)
	if token == "" {
		return "", "", nil, errors.New("tools.ado.token is required")
	}
	if !jiraSecretExpressionPattern.MatchString(token) {
		return "", "", nil, errors.New("tools.ado.token must be a direct GitHub Actions secret expression")
	}

	toolsets, err := parseADOToolsets(config["toolsets"])
	if err != nil {
		return "", "", nil, err
	}
	return organization, token, toolsets, nil
}

func parseADOToolsets(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	var toolsets []string
	switch values := value.(type) {
	case []string:
		toolsets = values
	case []any:
		toolsets = make([]string, 0, len(values))
		for _, value := range values {
			toolset, ok := value.(string)
			if !ok {
				return nil, errors.New("tools.ado.toolsets must contain only supported toolset names")
			}
			toolsets = append(toolsets, toolset)
		}
	default:
		return nil, errors.New("tools.ado.toolsets must be a non-empty array of supported toolset names")
	}
	if len(toolsets) == 0 {
		return nil, errors.New("tools.ado.toolsets must be a non-empty array of supported toolset names")
	}

	seen := make(map[string]struct{}, len(toolsets))
	for _, toolset := range toolsets {
		if _, supported := adoToolsetNames[toolset]; !supported {
			return nil, fmt.Errorf("tools.ado.toolsets contains unsupported toolset %q", toolset)
		}
		if _, duplicate := seen[toolset]; duplicate {
			return nil, fmt.Errorf("tools.ado.toolsets contains duplicate toolset %q", toolset)
		}
		seen[toolset] = struct{}{}
	}
	if len(toolsets) > 1 && slices.Contains(toolsets, "all") {
		return nil, errors.New(`tools.ado.toolsets must contain only "all" when the all toolset is used`)
	}
	return toolsets, nil
}
