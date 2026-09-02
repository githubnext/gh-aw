package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandJiraToolConfigServiceAccount(t *testing.T) {
	tools := map[string]any{
		"jira": map[string]any{
			"auth": map[string]any{
				"type":  jiraServiceAccountAuth,
				"token": "${{ secrets.ATLASSIAN_SERVICE_ACCOUNT_API_KEY }}",
			},
			"allowed": []any{"getJiraIssue", "searchJiraIssuesUsingJql"},
		},
	}

	require.NoError(t, expandJiraToolConfig(tools))
	config := tools["jira"].(map[string]any)
	assert.Equal(t, "http", config["type"])
	assert.Equal(t, defaultJiraMCPURL, config["url"])
	assert.Equal(t, "Bear"+"er ${{ secrets.ATLASSIAN_SERVICE_ACCOUNT_API_KEY }}", config["headers"].(map[string]any)["Authorization"])

	mcpConfig, err := getMCPConfig(config, "jira")
	require.NoError(t, err)
	assert.Nil(t, mcpConfig.Auth)
	assert.Equal(t, []string{"getJiraIssue", "searchJiraIssuesUsingJql"}, mcpConfig.Allowed)
}

func TestExpandJiraToolConfigAPIToken(t *testing.T) {
	tools := map[string]any{
		"jira": map[string]any{
			"url": "https://example.atlassian.net/mcp",
			"auth": map[string]any{
				"type":  jiraAPITokenAuth,
				"email": "${{ secrets.ATLASSIAN_EMAIL }}",
				"token": "${{ secrets.ATLASSIAN_API_TOKEN }}",
			},
		},
	}

	require.NoError(t, expandJiraToolConfig(tools))
	config := tools["jira"].(map[string]any)
	assert.Equal(t, "https://example.atlassian.net/mcp", config["url"])
	assert.Equal(t, "Basic ${{ env.GH_AW_JIRA_BASIC_AUTH }}", config["headers"].(map[string]any)["Authorization"])

	stepEnv := jiraAPIAuthStepEnv(tools)
	assert.Equal(t, "${{ secrets.ATLASSIAN_EMAIL }}", stepEnv[jiraEmailEnvVar])
	assert.Equal(t, "${{ secrets.ATLASSIAN_API_TOKEN }}", stepEnv[jiraTokenEnvVar])

	var setup strings.Builder
	writeJiraAPIAuthPreparation(&setup, tools)
	assert.Contains(t, setup.String(), `printf '%s:%s' "$GH_AW_JIRA_EMAIL" "$GH_AW_JIRA_TOKEN"`)
	assert.Contains(t, setup.String(), `echo "::add-mask::${GH_AW_JIRA_BASIC_AUTH}"`)
	assert.NotContains(t, setup.String(), "ATLASSIAN_EMAIL")
	assert.NotContains(t, setup.String(), "ATLASSIAN_API_TOKEN")
}

func TestExpandJiraToolConfigRejectsInvalidAuth(t *testing.T) {
	tests := []struct {
		name    string
		auth    map[string]any
		message string
	}{
		{
			name:    "unsupported auth type",
			auth:    map[string]any{"type": "oauth", "token": "${{ secrets.TOKEN }}"},
			message: "tools.jira.auth.type must be",
		},
		{
			name:    "missing token",
			auth:    map[string]any{"type": jiraServiceAccountAuth},
			message: "tools.jira.auth.token is required",
		},
		{
			name:    "missing email",
			auth:    map[string]any{"type": jiraAPITokenAuth, "token": "${{ secrets.TOKEN }}"},
			message: "tools.jira.auth.email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := map[string]any{"jira": map[string]any{"auth": tt.auth}}
			err := expandJiraToolConfig(tools)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestJiraHeadersUseRuntimeEnvironmentReferences(t *testing.T) {
	serviceHeaders := renderCustomMCPHeadersTOML(
		map[string]string{"Authorization": "Bear" + "er ${{ secrets.JIRA_TOKEN }}"},
		map[string]string{"JIRA_TOKEN": "${{ secrets.JIRA_TOKEN }}"},
	)
	assert.Equal(t, "Bear"+"er ${JIRA_TOKEN}", serviceHeaders["Authorization"])

	apiHeaders := renderCustomMCPHeadersTOML(
		map[string]string{"Authorization": "Basic ${{ env.GH_AW_JIRA_BASIC_AUTH }}"},
		nil,
	)
	assert.Equal(t, "Basic ${GH_AW_JIRA_BASIC_AUTH}", apiHeaders["Authorization"])
}

func TestJiraAPITokenCompilation(t *testing.T) {
	workflow := `---
on:
  workflow_dispatch:
strict: false
engine: copilot
tools:
  jira:
    auth:
      type: api-token
      email: ${{ secrets.ATLASSIAN_EMAIL }}
      token: ${{ secrets.ATLASSIAN_API_TOKEN }}
    allowed:
      - getJiraIssue
      - searchJiraIssuesUsingJql
---

Read Jira issues.
`
	file, err := os.CreateTemp("", "jira-api-token-*.md")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	_, err = file.WriteString(workflow)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	compiler := NewCompiler()
	compiler.SetSkipValidation(true)
	data, err := compiler.ParseWorkflowFile(file.Name())
	require.NoError(t, err)
	compiled, _, _, err := compiler.generateYAML(data, file.Name())
	require.NoError(t, err)

	assert.Contains(t, compiled, defaultJiraMCPURL)
	assert.Contains(t, compiled, `"getJiraIssue"`)
	assert.Contains(t, compiled, `"searchJiraIssuesUsingJql"`)
	assert.Contains(t, compiled, `GH_AW_JIRA_EMAIL: ${{ secrets.ATLASSIAN_EMAIL }}`)
	assert.Contains(t, compiled, `GH_AW_JIRA_TOKEN: ${{ secrets.ATLASSIAN_API_TOKEN }}`)
	assert.Contains(t, compiled, `printf '%s:%s' "$GH_AW_JIRA_EMAIL" "$GH_AW_JIRA_TOKEN"`)
	assert.Contains(t, compiled, `"Authorization": "Basic \${GH_AW_JIRA_BASIC_AUTH}"`)
	assert.NotContains(t, compiled, `"email": "${{ secrets.ATLASSIAN_EMAIL }}"`)
	assert.NotContains(t, compiled, `"token": "${{ secrets.ATLASSIAN_API_TOKEN }}"`)
}
