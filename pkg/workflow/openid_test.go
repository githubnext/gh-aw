package workflow

import (
	"strings"
	"testing"
)

func TestOIDCConfigExtraction(t *testing.T) {
	compiler := &Compiler{
		engineRegistry: GetGlobalEngineRegistry(),
	}

	// Test OIDC configuration extraction
	frontmatter := map[string]any{
		"engine": map[string]any{
			"id": "claude",
			"oidc": map[string]any{
				"enabled":            true,
				"audience":           "test-audience",
				"token_exchange_url": "https://api.example.com/token-exchange",
				"token_revoke_url":   "https://api.example.com/token-revoke",
				"env_var_name":       "TEST_TOKEN",
				"fallback_env_var":   "TEST_FALLBACK",
			},
		},
	}

	engineID, config := compiler.ExtractEngineConfig(frontmatter)

	if engineID != "claude" {
		t.Errorf("Expected engine ID 'claude', got '%s'", engineID)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.OIDC == nil {
		t.Fatal("Expected OIDC config to be non-nil")
	}

	if !config.OIDC.Enabled {
		t.Error("Expected OIDC to be enabled")
	}

	if config.OIDC.Audience != "test-audience" {
		t.Errorf("Expected audience 'test-audience', got '%s'", config.OIDC.Audience)
	}

	if config.OIDC.TokenExchangeURL != "https://api.example.com/token-exchange" {
		t.Errorf("Expected token exchange URL 'https://api.example.com/token-exchange', got '%s'", config.OIDC.TokenExchangeURL)
	}

	if config.OIDC.TokenRevokeURL != "https://api.example.com/token-revoke" {
		t.Errorf("Expected token revoke URL 'https://api.example.com/token-revoke', got '%s'", config.OIDC.TokenRevokeURL)
	}

	if config.OIDC.EnvVarName != "TEST_TOKEN" {
		t.Errorf("Expected env var name 'TEST_TOKEN', got '%s'", config.OIDC.EnvVarName)
	}

	if config.OIDC.FallbackEnvVar != "TEST_FALLBACK" {
		t.Errorf("Expected fallback env var 'TEST_FALLBACK', got '%s'", config.OIDC.FallbackEnvVar)
	}
}

func TestOIDCConfigDefaults(t *testing.T) {
	// Test with minimal OIDC configuration
	oidcConfig := &OIDCConfig{
		Enabled:          true,
		TokenExchangeURL: "https://api.example.com/exchange",
	}

	// Test default audience for Claude
	audience := oidcConfig.GetOIDCAudience("claude")
	if audience != "claude-code-github-action" {
		t.Errorf("Expected default audience 'claude-code-github-action', got '%s'", audience)
	}

	// Test engine method for token env var name
	claudeEngine := NewClaudeEngine()
	envVarName := claudeEngine.GetTokenEnvVarName()
	if envVarName != "ANTHROPIC_API_KEY" {
		t.Errorf("Expected Claude env var name 'ANTHROPIC_API_KEY', got '%s'", envVarName)
	}

	// Test for other engines
	copilotEngine := NewCopilotEngine()
	copilotEnvVar := copilotEngine.GetTokenEnvVarName()
	if copilotEnvVar != "GITHUB_TOKEN" {
		t.Errorf("Expected Copilot env var name 'GITHUB_TOKEN', got '%s'", copilotEnvVar)
	}

	codexEngine := NewCodexEngine()
	codexEnvVar := codexEngine.GetTokenEnvVarName()
	if codexEnvVar != "OPENAI_API_KEY" {
		t.Errorf("Expected Codex env var name 'OPENAI_API_KEY', got '%s'", codexEnvVar)
	}
}

func TestClaudeEngineWithOIDC(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{
		Name:            "test-workflow",
		MarkdownContent: "Test workflow",
		EngineConfig: &EngineConfig{
			ID: "claude",
			OIDC: &OIDCConfig{
				Enabled:          true,
				Audience:         "claude-code-github-action",
				TokenExchangeURL: "https://api.anthropic.com/api/github/github-app-token-exchange",
				TokenRevokeURL:   "https://api.anthropic.com/api/github/github-app-token-revoke",
			},
		},
		Tools: map[string]any{},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// Convert steps to string for easier inspection
	var stepsStr string
	for _, step := range steps {
		stepsStr += strings.Join(step, "\n") + "\n"
	}

	// Verify OIDC setup step is present
	if !strings.Contains(stepsStr, "Setup OIDC token") {
		t.Error("Expected OIDC setup step to be present")
	}

	// Verify OIDC revoke step is present
	if !strings.Contains(stepsStr, "Revoke OIDC token") {
		t.Error("Expected OIDC revoke step to be present")
	}

	// Verify token is used from OIDC setup step
	if !strings.Contains(stepsStr, "ANTHROPIC_API_KEY: ${{ steps.setup_oidc_token.outputs.token }}") {
		t.Error("Expected ANTHROPIC_API_KEY to use token from OIDC setup step")
	}

	// Verify setup step uses github-script
	if !strings.Contains(stepsStr, "uses: actions/github-script@v8") {
		t.Error("Expected OIDC setup step to use actions/github-script@v8")
	}

	// Verify revoke step has if: always()
	if !strings.Contains(stepsStr, "if: always()") {
		t.Error("Expected OIDC revoke step to have 'if: always()' condition")
	}
}

func TestClaudeEngineWithoutOIDC(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{
		Name:            "test-workflow",
		MarkdownContent: "Test workflow",
		EngineConfig: &EngineConfig{
			ID: "claude",
		},
		Tools: map[string]any{},
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// Convert steps to string for easier inspection
	var stepsStr string
	for _, step := range steps {
		stepsStr += strings.Join(step, "\n") + "\n"
	}

	// Verify OIDC setup step is NOT present
	if strings.Contains(stepsStr, "Setup OIDC token") {
		t.Error("Expected OIDC setup step to NOT be present when OIDC is not configured")
	}

	// Verify OIDC revoke step is NOT present
	if strings.Contains(stepsStr, "Revoke OIDC token") {
		t.Error("Expected OIDC revoke step to NOT be present when OIDC is not configured")
	}

	// Verify token uses direct secret
	if !strings.Contains(stepsStr, "ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}") {
		t.Error("Expected ANTHROPIC_API_KEY to use direct secret when OIDC is not configured")
	}
}

func TestHasOIDCConfig(t *testing.T) {
	// Test with nil config
	if HasOIDCConfig(nil) {
		t.Error("Expected HasOIDCConfig to return false for nil config")
	}

	// Test with config but no OIDC
	config := &EngineConfig{
		ID: "claude",
	}
	if HasOIDCConfig(config) {
		t.Error("Expected HasOIDCConfig to return false when OIDC is nil")
	}

	// Test with OIDC disabled
	config.OIDC = &OIDCConfig{
		Enabled: false,
	}
	if HasOIDCConfig(config) {
		t.Error("Expected HasOIDCConfig to return false when OIDC is disabled")
	}

	// Test with OIDC enabled
	config.OIDC.Enabled = true
	if !HasOIDCConfig(config) {
		t.Error("Expected HasOIDCConfig to return true when OIDC is enabled")
	}
}
