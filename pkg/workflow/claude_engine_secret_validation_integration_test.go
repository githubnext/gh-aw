package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeEngine_SecretValidationScriptPath_Integration tests that Claude engine
// generates secret validation steps with the correct script path
func TestClaudeEngine_SecretValidationScriptPath_Integration(t *testing.T) {
	engine := NewClaudeEngine()
	workflowData := &WorkflowData{
		Name: "test-claude-workflow",
	}

	steps := engine.GetInstallationSteps(workflowData)
	require.NotEmpty(t, steps, "Expected at least one installation step")

	// First step should be secret validation
	firstStep := strings.Join(steps[0], "\n")

	// Verify the step name
	assert.Contains(t, firstStep, "Validate CLAUDE_CODE_OAUTH_TOKEN or ANTHROPIC_API_KEY secret",
		"First installation step should validate Claude secrets")

	// Verify it calls the validate_multi_secret.sh script with correct path
	assert.Contains(t, firstStep, "/opt/gh-aw/actions/validate_multi_secret.sh",
		"Expected step to call validate_multi_secret.sh script at /opt/gh-aw/actions/")

	// Verify it passes both secret names to the script
	assert.Contains(t, firstStep, "CLAUDE_CODE_OAUTH_TOKEN ANTHROPIC_API_KEY",
		"Should pass both CLAUDE_CODE_OAUTH_TOKEN and ANTHROPIC_API_KEY to the script")

	// Verify it passes the engine name
	assert.Contains(t, firstStep, "Claude Code",
		"Should pass engine name 'Claude Code' to the script")

	// Verify it passes the docs URL
	assert.Contains(t, firstStep, "https://githubnext.github.io/gh-aw/reference/engines/#anthropic-claude-code",
		"Should pass docs URL to the script")

	// Verify both secrets are in the env section
	assert.Contains(t, firstStep, "CLAUDE_CODE_OAUTH_TOKEN: ${{ secrets.CLAUDE_CODE_OAUTH_TOKEN }}",
		"Secret validation step should reference secrets.CLAUDE_CODE_OAUTH_TOKEN")
	assert.Contains(t, firstStep, "ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}",
		"Secret validation step should reference secrets.ANTHROPIC_API_KEY")
}

// TestSmokeClaude_UsesClaudeEngine_Integration tests that the smoke-claude workflow
// uses the Claude engine (not Copilot or other engines) by checking the compiled lock file
func TestSmokeClaude_UsesClaudeEngine_Integration(t *testing.T) {
	// Find the smoke-claude.md and lock.yml files
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "smoke-claude.md")
	lockFilePath := filepath.Join("..", "..", ".github", "workflows", "smoke-claude.lock.yml")
	
	// Check if files exist (skip test if not in CI environment)
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Skip("smoke-claude.md not found, skipping integration test")
	}
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		t.Skip("smoke-claude.lock.yml not found, skipping integration test")
	}

	// Read the workflow markdown file to verify engine config
	mdContent, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Failed to read smoke-claude.md")

	workflowStr := string(mdContent)

	// Verify the workflow specifies Claude engine in frontmatter
	assert.Contains(t, workflowStr, "engine:",
		"Workflow should have an engine configuration")
	assert.Contains(t, workflowStr, "id: claude",
		"Workflow should specify 'id: claude' engine")

	// Read the compiled lock file to verify it uses Claude
	lockContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "Failed to read smoke-claude.lock.yml")

	yamlOutput := string(lockContent)

	// Verify the compiled workflow uses Claude CLI installation
	assert.Contains(t, yamlOutput, "Install Claude Code CLI",
		"Compiled workflow should install Claude Code CLI")
	assert.Contains(t, yamlOutput, "@anthropic-ai/claude-code",
		"Compiled workflow should reference @anthropic-ai/claude-code package")

	// Verify the compiled workflow uses Claude secret validation (not Copilot)
	assert.Contains(t, yamlOutput, "Validate CLAUDE_CODE_OAUTH_TOKEN or ANTHROPIC_API_KEY secret",
		"Compiled workflow should validate Claude secrets")
	assert.Contains(t, yamlOutput, "/opt/gh-aw/actions/validate_multi_secret.sh",
		"Compiled workflow should use validate_multi_secret.sh script")
	assert.Contains(t, yamlOutput, "CLAUDE_CODE_OAUTH_TOKEN ANTHROPIC_API_KEY",
		"Compiled workflow should pass Claude secrets to validation script")

	// Verify it does NOT use Copilot secrets (this was the bug reported)
	assert.NotContains(t, yamlOutput, "Validate COPILOT_GITHUB_TOKEN",
		"Compiled workflow should NOT validate COPILOT_GITHUB_TOKEN (wrong engine)")

	// Verify the compiled workflow executes Claude CLI
	assert.Contains(t, yamlOutput, "Execute Claude Code CLI",
		"Compiled workflow should execute Claude Code CLI")
}

// TestValidateMultiSecretScript_Exists_Integration verifies that the
// validate_multi_secret.sh script exists in the expected location
func TestValidateMultiSecretScript_Exists_Integration(t *testing.T) {
	// Check if the script exists in the source location
	scriptPath := filepath.Join("..", "..", "actions", "setup", "sh", "validate_multi_secret.sh")
	
	info, err := os.Stat(scriptPath)
	require.NoError(t, err, "validate_multi_secret.sh should exist at actions/setup/sh/")
	
	// Verify it's a file, not a directory
	assert.False(t, info.IsDir(), "validate_multi_secret.sh should be a file, not a directory")
	
	// Verify it has executable permissions (on Unix systems)
	if info.Mode()&0111 != 0 {
		t.Logf("✓ validate_multi_secret.sh has executable permissions")
	}
	
	// Read the script to verify it has the expected content
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err, "Failed to read validate_multi_secret.sh")
	
	scriptStr := string(content)
	
	// Verify it's a bash script
	assert.True(t, strings.HasPrefix(scriptStr, "#!/bin/bash") || strings.HasPrefix(scriptStr, "#!/usr/bin/env bash"),
		"Script should start with bash shebang")
	
	// Verify it has the expected usage pattern
	assert.Contains(t, scriptStr, "Usage: $0 SECRET_NAME1 [SECRET_NAME2 ...] ENGINE_NAME DOCS_URL",
		"Script should have expected usage pattern")
	
	// Verify it validates secrets
	assert.Contains(t, scriptStr, "all_empty=true",
		"Script should check if all secrets are empty")
}
