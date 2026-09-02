package workflow

import (
	"errors"
	"fmt"
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

// expandJiraToolConfig converts the first-class Jira configuration into the
// generic HTTP MCP shape consumed by the existing gateway pipeline.
func expandJiraToolConfig(tools map[string]any) error {
	raw, exists := tools["jira"]
	if !exists {
		return nil
	}
	if enabled, ok := raw.(bool); ok && !enabled {
		return nil
	}

	config, ok := raw.(map[string]any)
	if !ok {
		return errors.New("tools.jira must be an object with an auth configuration")
	}
	auth, ok := config["auth"].(map[string]any)
	if !ok {
		return errors.New("tools.jira.auth is required")
	}

	authType, _ := auth["type"].(string)
	token, _ := auth["token"].(string)
	if token == "" {
		return errors.New("tools.jira.auth.token is required")
	}

	headers := map[string]any{}
	switch authType {
	case jiraServiceAccountAuth:
		headers["Authorization"] = "Bearer " + token
	case jiraAPITokenAuth:
		email, _ := auth["email"].(string)
		if email == "" {
			return errors.New("tools.jira.auth.email is required for api-token authentication")
		}
		headers["Authorization"] = "Basic ${{ env." + jiraBasicAuthEnvVar + " }}"
	default:
		return fmt.Errorf("tools.jira.auth.type must be %q or %q", jiraServiceAccountAuth, jiraAPITokenAuth)
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
