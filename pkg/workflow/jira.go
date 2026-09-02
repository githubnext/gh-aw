package workflow

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	defaultJiraMCPURL      = "https://mcp.atlassian.com/v1/mcp"
	jiraBasicAuthEnvVar    = "GH_AW_JIRA_BASIC_AUTH"
	jiraEmailEnvVar        = "GH_AW_JIRA_EMAIL"
	jiraTokenEnvVar        = "GH_AW_JIRA_TOKEN"
	jiraServiceAccountAuth = "service-account"
	jiraAPITokenAuth       = "api-token"
)

var jiraSecretExpressionPattern = regexp.MustCompile(`^\$\{\{\s*secrets\.[A-Z_][A-Z0-9_]*\s*\}\}$`)

// expandJiraToolConfig converts the first-class Jira configuration into the
// generic HTTP MCP shape consumed by the existing gateway pipeline.
func expandJiraToolConfig(tools map[string]any) error {
	raw, exists := tools["jira"]
	if !exists {
		return nil
	}
	if enabled, ok := raw.(bool); ok && !enabled {
		delete(tools, "jira")
		return nil
	}

	config, ok := raw.(map[string]any)
	if !ok {
		return errors.New("tools.jira must be an object with an auth configuration")
	}
	if err := validateJiraToolConfig(config); err != nil {
		return err
	}
	auth, ok := config["auth"].(map[string]any)
	if !ok {
		return errors.New("tools.jira.auth is required")
	}
	authType, ok := auth["type"].(string)
	if !ok {
		return fmt.Errorf("tools.jira.auth.type must be %q or %q", jiraServiceAccountAuth, jiraAPITokenAuth)
	}
	token, ok := auth["token"].(string)
	if !ok {
		return errors.New("tools.jira.auth.token is required")
	}

	headers := map[string]any{}
	switch authType {
	case jiraServiceAccountAuth:
		headers["Authorization"] = "Bearer " + token
	case jiraAPITokenAuth:
		headers["Authorization"] = "Basic ${{ env." + jiraBasicAuthEnvVar + " }}"
	}

	url, _ := config["url"].(string)
	if strings.TrimSpace(url) == "" {
		url = defaultJiraMCPURL
	}

	expanded := map[string]any{
		"type":    "http",
		"url":     url,
		"headers": headers,
		"auth":    auth,
	}
	if allowed, exists := config["allowed"]; exists {
		expanded["allowed"] = allowed
	}
	tools["jira"] = expanded
	return nil
}

func validateJiraToolConfig(config map[string]any) error {
	for field := range config {
		switch field {
		case "auth", "url", "allowed":
		default:
			return fmt.Errorf("tools.jira.%s is not supported; valid fields are: allowed, auth, url", field)
		}
	}

	if rawURL, exists := config["url"]; exists {
		endpoint, ok := rawURL.(string)
		if !ok {
			return errors.New("tools.jira.url must be an HTTPS URL")
		}
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return errors.New("tools.jira.url must be an HTTPS URL without embedded credentials")
		}
	}

	if allowed, exists := config["allowed"]; exists {
		if err := validateJiraAllowedTools(allowed); err != nil {
			return err
		}
	}

	auth, ok := config["auth"].(map[string]any)
	if !ok {
		return errors.New("tools.jira.auth is required")
	}
	return validateJiraAuthConfig(auth)
}

func validateJiraAllowedTools(value any) error {
	var allowed []string
	switch values := value.(type) {
	case []string:
		allowed = values
	case []any:
		allowed = make([]string, 0, len(values))
		for _, item := range values {
			tool, ok := item.(string)
			if !ok {
				return errors.New("tools.jira.allowed must contain only non-empty tool names")
			}
			allowed = append(allowed, tool)
		}
	default:
		return errors.New("tools.jira.allowed must be a non-empty array of tool names")
	}
	if len(allowed) == 0 {
		return errors.New("tools.jira.allowed must be a non-empty array of tool names")
	}
	for _, tool := range allowed {
		if strings.TrimSpace(tool) == "" {
			return errors.New("tools.jira.allowed must contain only non-empty tool names")
		}
	}
	return nil
}

func validateJiraAuthConfig(auth map[string]any) error {
	authType, _ := auth["type"].(string)
	for field := range auth {
		if field != "type" && field != "token" && (authType != jiraAPITokenAuth || field != "email") {
			return fmt.Errorf("tools.jira.auth.%s is not supported for %s authentication", field, authType)
		}
	}

	token, _ := auth["token"].(string)
	if token == "" {
		return errors.New("tools.jira.auth.token is required")
	}
	if !jiraSecretExpressionPattern.MatchString(token) {
		return errors.New("tools.jira.auth.token must be a direct GitHub Actions secret expression")
	}
	switch authType {
	case jiraServiceAccountAuth:
		return nil
	case jiraAPITokenAuth:
		email, _ := auth["email"].(string)
		if email == "" {
			return errors.New("tools.jira.auth.email is required for api-token authentication")
		}
		if !jiraSecretExpressionPattern.MatchString(email) {
			return errors.New("tools.jira.auth.email must be a direct GitHub Actions secret expression for api-token authentication")
		}
		return nil
	default:
		return fmt.Errorf("tools.jira.auth.type must be %q or %q", jiraServiceAccountAuth, jiraAPITokenAuth)
	}
}

func jiraAuthConfig(tools map[string]any) (map[string]any, bool) {
	config, ok := tools["jira"].(map[string]any)
	if !ok {
		return nil, false
	}
	auth, ok := config["auth"].(map[string]any)
	return auth, ok
}

func jiraAPIAuthStepEnv(tools map[string]any) map[string]string {
	auth, ok := jiraAuthConfig(tools)
	if !ok || auth["type"] != jiraAPITokenAuth {
		return nil
	}
	email, emailOK := auth["email"].(string)
	token, tokenOK := auth["token"].(string)
	if !emailOK || !tokenOK {
		return nil
	}
	return map[string]string{
		jiraEmailEnvVar: email,
		jiraTokenEnvVar: token,
	}
}

func writeJiraAPIAuthPreparation(yaml *strings.Builder, tools map[string]any) {
	if len(jiraAPIAuthStepEnv(tools)) == 0 {
		return
	}
	fmt.Fprintf(yaml, "          %s=\"$(printf '%%s:%%s' \"$%s\" \"$%s\" | base64 | tr -d '\\n')\"\n", jiraBasicAuthEnvVar, jiraEmailEnvVar, jiraTokenEnvVar)
	fmt.Fprintf(yaml, "          echo \"::add-mask::${%s}\"\n", jiraBasicAuthEnvVar)
	fmt.Fprintf(yaml, "          export %s\n", jiraBasicAuthEnvVar)
}
