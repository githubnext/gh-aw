//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCompileCustomMCPWorkflow is a helper that compiles a workflow markdown string and returns
// the generated lock-file content.
func testCompileCustomMCPWorkflow(t *testing.T, markdown string) string {
	t.Helper()
	compiler := NewCompilerWithVersion("1.0.0")
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(markdown), 0644))
	err := compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "CompileWorkflow should not fail")
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	data, err := os.ReadFile(lockFile)
	require.NoError(t, err, "lock file should exist")
	return string(data)
}

// =============================================================================
// github-app on custom HTTP MCP server
// =============================================================================

// TestCustomMCPServerGitHubAppTokenMintStep verifies that a custom HTTP MCP server with
// github-app configured generates token-mint and token-invalidate workflow steps.
func TestCustomMCPServerGitHubAppTokenMintStep(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  my-server:
    url: "https://api.example.com/mcp"
    github-app:
      app-id: "${{ vars.APP_ID }}"
      private-key: "${{ secrets.APP_PRIVATE_KEY }}"
---

# Test Workflow

Test workflow with custom MCP server using GitHub App.
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	// Mint step must be present with the correct ID
	assert.Contains(t, lock, "id: my-server-mcp-app-token", "Should generate mint step with server-specific ID")
	assert.Contains(t, lock, "uses: actions/create-github-app-token", "Should use create-github-app-token action")
	assert.Contains(t, lock, "app-id: ${{ vars.APP_ID }}", "Should include app-id")
	assert.Contains(t, lock, "private-key: ${{ secrets.APP_PRIVATE_KEY }}", "Should include private-key")

	// Invalidation step must be present
	assert.Contains(t, lock, "steps.my-server-mcp-app-token.outputs.token", "Should reference token in invalidation step")
	assert.Contains(t, lock, "if: always() && steps.my-server-mcp-app-token.outputs.token != ''", "Should have conditional invalidation")

	// The gateway env var must export the minted token
	assert.Contains(t, lock, "MCP_MY_SERVER_APP_TOKEN: ${{ steps.my-server-mcp-app-token.outputs.token }}",
		"Should export minted token as MCP_MY_SERVER_APP_TOKEN env var")

	// The gateway config should have the Authorization header injected
	assert.Contains(t, lock, `"Authorization": "Bearer ${MCP_MY_SERVER_APP_TOKEN}"`,
		"Should auto-inject Authorization header in gateway config")
}

// TestCustomMCPServerGitHubAppOrgWide verifies org-wide access (repositories: ['*']) compiles correctly.
func TestCustomMCPServerGitHubAppOrgWide(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  corp-server:
    url: "https://mcp.corp.example.com/mcp"
    github-app:
      app-id: "${{ vars.CORP_APP_ID }}"
      private-key: "${{ secrets.CORP_APP_KEY }}"
      repositories:
        - "*"
---

# Org-wide App Test
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	assert.Contains(t, lock, "id: corp-server-mcp-app-token", "Should use corp-server-mcp-app-token step ID")
	assert.Contains(t, lock, "MCP_CORP_SERVER_APP_TOKEN", "Should use MCP_CORP_SERVER_APP_TOKEN env var")
	// Org-wide access omits the repositories field
	assert.NotContains(t, lock, "repositories: *", "Should not include wildcard repositories field")
}

// TestCustomMCPServerGitHubAppMultipleServers verifies multiple servers each get their own mint/invalidate steps.
func TestCustomMCPServerGitHubAppMultipleServers(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  alpha-server:
    url: "https://alpha.example.com/mcp"
    github-app:
      app-id: "${{ vars.ALPHA_APP_ID }}"
      private-key: "${{ secrets.ALPHA_PRIVATE_KEY }}"
  beta-server:
    url: "https://beta.example.com/mcp"
    github-app:
      app-id: "${{ vars.BETA_APP_ID }}"
      private-key: "${{ secrets.BETA_PRIVATE_KEY }}"
---

# Multi-server App Test
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	assert.Contains(t, lock, "id: alpha-server-mcp-app-token", "Should generate step for alpha-server")
	assert.Contains(t, lock, "id: beta-server-mcp-app-token", "Should generate step for beta-server")
	assert.Contains(t, lock, "MCP_ALPHA_SERVER_APP_TOKEN", "Should export alpha token env var")
	assert.Contains(t, lock, "MCP_BETA_SERVER_APP_TOKEN", "Should export beta token env var")
}

// TestCustomMCPServerGitHubAppAndHeadersCanCoexist verifies that user-specified headers
// (other than Authorization) are preserved alongside the auto-injected Authorization.
func TestCustomMCPServerGitHubAppAndHeadersCanCoexist(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  custom:
    url: "https://api.example.com/mcp"
    headers:
      X-Tenant-ID: "my-tenant"
    github-app:
      app-id: "${{ vars.APP_ID }}"
      private-key: "${{ secrets.APP_PRIVATE_KEY }}"
---

# Test With Explicit Headers
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	assert.Contains(t, lock, `"X-Tenant-ID": "my-tenant"`, "Should preserve user-defined headers")
	assert.Contains(t, lock, `"Authorization": "Bearer ${MCP_CUSTOM_APP_TOKEN}"`, "Should inject Authorization header")
}

// TestCustomMCPServerNoGitHubApp verifies that a plain HTTP MCP server without github-app
// compiles correctly without any token-mint steps.
func TestCustomMCPServerNoGitHubApp(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  plain-server:
    url: "https://api.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_KEY }}"
---

# Plain HTTP MCP server
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	assert.NotContains(t, lock, "plain-server-mcp-app-token", "Should not generate token step for plain server")
	assert.NotContains(t, lock, "MCP_PLAIN_SERVER_APP_TOKEN", "Should not export token env var")
}

// =============================================================================
// Helper utilities
// =============================================================================

// TestCustomMCPServerAppTokenStepID verifies the step-ID naming function.
func TestCustomMCPServerAppTokenStepID(t *testing.T) {
	tests := []struct {
		serverName string
		wantID     string
	}{
		{"my-server", "my-server-mcp-app-token"},
		{"alpha", "alpha-mcp-app-token"},
		{"corp-mcp", "corp-mcp-mcp-app-token"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.wantID, customMCPServerAppTokenStepID(tt.serverName))
	}
}

// TestCustomMCPServerAppTokenEnvVar verifies the env-var naming function.
func TestCustomMCPServerAppTokenEnvVar(t *testing.T) {
	tests := []struct {
		serverName string
		wantVar    string
	}{
		{"my-server", "MCP_MY_SERVER_APP_TOKEN"},
		{"alpha", "MCP_ALPHA_APP_TOKEN"},
		{"corp-api", "MCP_CORP_API_APP_TOKEN"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.wantVar, customMCPServerAppTokenEnvVar(tt.serverName))
	}
}

// =============================================================================
// github-oidc auth on custom HTTP MCP server
// =============================================================================

// TestCustomMCPServerOIDCAuthPermission verifies that auth.type: github-oidc
// automatically injects id-token: write into the agent job permissions.
func TestCustomMCPServerOIDCAuthPermission(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  my-server:
    url: "https://api.example.com/mcp"
    auth:
      type: github-oidc
      audience: "https://api.example.com"
---

# OIDC Auth Test
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	// id-token: write must be in the agent job permissions
	assert.Contains(t, lock, "id-token: write", "id-token: write should be auto-injected")

	// OIDC env vars must be passed to the gateway
	assert.Contains(t, lock, "ACTIONS_ID_TOKEN_REQUEST_URL", "Should pass ACTIONS_ID_TOKEN_REQUEST_URL to gateway")
	assert.Contains(t, lock, "ACTIONS_ID_TOKEN_REQUEST_TOKEN", "Should pass ACTIONS_ID_TOKEN_REQUEST_TOKEN to gateway")
}

// TestCustomMCPServerOIDCAuthGatewayConfig verifies that the auth object is rendered in the gateway config.
func TestCustomMCPServerOIDCAuthGatewayConfig(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  my-server:
    url: "https://mcp.example.com/mcp"
    auth:
      type: github-oidc
      audience: "https://mcp.example.com"
---

# OIDC Gateway Config Test
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	// The gateway config JSON should contain the auth block
	assert.Contains(t, lock, `"auth":`, "Gateway config should include auth block")
	assert.Contains(t, lock, `"type": "github-oidc"`, "Gateway config should include auth type")
	assert.Contains(t, lock, `"audience": "https://mcp.example.com"`, "Gateway config should include audience")
}

// TestCustomMCPServerOIDCAuthNoAudience verifies auth without audience compiles fine.
func TestCustomMCPServerOIDCAuthNoAudience(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  my-server:
    url: "https://api.example.com/mcp"
    auth:
      type: github-oidc
---

# OIDC no audience
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	assert.Contains(t, lock, `"type": "github-oidc"`, "Should include auth type")
	assert.NotContains(t, lock, `"audience"`, "Should not include audience when not specified")
	assert.Contains(t, lock, "id-token: write", "Should still inject id-token: write")
}

// TestCustomMCPServerOIDCAuthNoNoMintStep verifies that OIDC auth does NOT generate a token-mint step
// (the gateway handles token acquisition itself).
func TestCustomMCPServerOIDCAuthNoNoMintStep(t *testing.T) {
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  my-server:
    url: "https://api.example.com/mcp"
    auth:
      type: github-oidc
      audience: "https://api.example.com"
---

# OIDC no mint step
`
	lock := testCompileCustomMCPWorkflow(t, markdown)

	assert.NotContains(t, lock, "create-github-app-token", "OIDC auth should NOT generate a token-mint step")
	assert.NotContains(t, lock, "my-server-mcp-app-token", "OIDC auth should NOT generate an app-token step")
}

// TestCustomMCPServerOIDCAndGitHubAppMutuallyExclusive verifies compilation fails when
// both auth and github-app are configured on the same server.
func TestCustomMCPServerOIDCAndGitHubAppMutuallyExclusive(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")
	markdown := `---
on: issues
permissions:
  contents: read
strict: false
mcp-servers:
  bad-server:
    url: "https://api.example.com/mcp"
    github-app:
      app-id: "${{ vars.APP_ID }}"
      private-key: "${{ secrets.APP_PRIVATE_KEY }}"
    auth:
      type: github-oidc
      audience: "https://api.example.com"
---

# Mutual exclusion test
`
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	require.NoError(t, os.WriteFile(testFile, []byte(markdown), 0644))
	err := compiler.CompileWorkflow(testFile)
	// Compilation must fail - either the schema validator or the Go-level validator catches
	// the illegal combination of auth + github-app on the same server.
	require.Error(t, err, "Should fail when both auth and github-app are configured")
}

// TestParseCustomMCPGitHubApp verifies the github-app parsing helper.
func TestParseCustomMCPGitHubApp(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		toolConfig := map[string]any{
			"url": "https://api.example.com/mcp",
			"github-app": map[string]any{
				"app-id":      "${{ vars.APP_ID }}",
				"private-key": "${{ secrets.PRIVATE_KEY }}",
				"owner":       "my-org",
			},
		}
		app := parseCustomMCPGitHubApp(toolConfig)
		require.NotNil(t, app)
		assert.Equal(t, "${{ vars.APP_ID }}", app.AppID)
		assert.Equal(t, "${{ secrets.PRIVATE_KEY }}", app.PrivateKey)
		assert.Equal(t, "my-org", app.Owner)
	})

	t.Run("missing private-key returns nil", func(t *testing.T) {
		toolConfig := map[string]any{
			"url": "https://api.example.com/mcp",
			"github-app": map[string]any{
				"app-id": "${{ vars.APP_ID }}",
			},
		}
		app := parseCustomMCPGitHubApp(toolConfig)
		assert.Nil(t, app, "Should return nil when private-key is missing")
	})

	t.Run("no github-app returns nil", func(t *testing.T) {
		toolConfig := map[string]any{
			"url": "https://api.example.com/mcp",
		}
		app := parseCustomMCPGitHubApp(toolConfig)
		assert.Nil(t, app)
	})
}

// TestHasCustomMCPServerOIDCAuth verifies the OIDC-detection helper.
func TestHasCustomMCPServerOIDCAuth(t *testing.T) {
	t.Run("server with github-oidc", func(t *testing.T) {
		tools := map[string]any{
			"my-server": map[string]any{
				"url": "https://api.example.com/mcp",
				"auth": map[string]any{
					"type":     "github-oidc",
					"audience": "https://api.example.com",
				},
			},
		}
		assert.True(t, hasCustomMCPServerOIDCAuth(tools), "Should detect OIDC auth")
	})

	t.Run("server without auth", func(t *testing.T) {
		tools := map[string]any{
			"my-server": map[string]any{
				"url": "https://api.example.com/mcp",
			},
		}
		assert.False(t, hasCustomMCPServerOIDCAuth(tools), "Should not detect OIDC auth")
	})

	t.Run("server with github-app only", func(t *testing.T) {
		tools := map[string]any{
			"my-server": map[string]any{
				"url": "https://api.example.com/mcp",
				"github-app": map[string]any{
					"app-id":      "${{ vars.APP_ID }}",
					"private-key": "${{ secrets.KEY }}",
				},
			},
		}
		assert.False(t, hasCustomMCPServerOIDCAuth(tools), "github-app alone should not trigger OIDC detection")
	})

	t.Run("empty tools", func(t *testing.T) {
		assert.False(t, hasCustomMCPServerOIDCAuth(map[string]any{}))
	})
}
