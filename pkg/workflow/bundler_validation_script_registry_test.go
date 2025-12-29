package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNoExecSync_GitHubScriptMode(t *testing.T) {
	tests := []struct {
		name        string
		scriptName  string
		content     string
		mode        RuntimeMode
		expectError bool
	}{
		{
			name:       "GitHub Script mode with execSync should fail",
			scriptName: "test_script",
			content: `
const { execSync } = require("child_process");
const result = execSync("ls -la");
`,
			mode:        RuntimeModeGitHubScript,
			expectError: true,
		},
		{
			name:       "GitHub Script mode with exec should pass",
			scriptName: "test_script",
			content: `
const { exec } = require("@actions/exec");
await exec.exec("ls -la");
`,
			mode:        RuntimeModeGitHubScript,
			expectError: false,
		},
		{
			name:       "GitHub Script mode without exec should pass",
			scriptName: "test_script",
			content: `
const fs = require("fs");
const data = fs.readFileSync("file.txt");
`,
			mode:        RuntimeModeGitHubScript,
			expectError: false,
		},
		{
			name:       "Node.js mode with execSync should pass (not checked)",
			scriptName: "test_script",
			content: `
const { execSync } = require("child_process");
const result = execSync("ls -la");
`,
			mode:        RuntimeModeNodeJS,
			expectError: false,
		},
		{
			name:       "GitHub Script mode with execSync in comment should pass",
			scriptName: "test_script",
			content: `
// Don't use execSync, use exec instead
const { exec } = require("@actions/exec");
`,
			mode:        RuntimeModeGitHubScript,
			expectError: false,
		},
		{
			name:       "GitHub Script mode with multiple execSync calls should fail",
			scriptName: "test_script",
			content: `
const { execSync } = require("child_process");
execSync("git status");
const output = execSync("git diff");
`,
			mode:        RuntimeModeGitHubScript,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoExecSync(tt.scriptName, tt.content, tt.mode)
			if tt.expectError {
				require.Error(t, err, "Expected validation to fail")
				assert.Contains(t, err.Error(), "execSync", "Error should mention execSync")
			} else {
				assert.NoError(t, err, "Expected validation to pass")
			}
		})
	}
}

func TestValidateNoGitHubScriptGlobals_NodeJSMode(t *testing.T) {
	tests := []struct {
		name        string
		scriptName  string
		content     string
		mode        RuntimeMode
		expectError bool
	}{
		{
			name:       "Node.js mode with core.* should fail",
			scriptName: "test_script",
			content: `
const fs = require("fs");
core.info("This is a message");
`,
			mode:        RuntimeModeNodeJS,
			expectError: true,
		},
		{
			name:       "Node.js mode with exec.* should fail",
			scriptName: "test_script",
			content: `
const fs = require("fs");
await exec.exec("ls -la");
`,
			mode:        RuntimeModeNodeJS,
			expectError: true,
		},
		{
			name:       "Node.js mode with github.* should fail",
			scriptName: "test_script",
			content: `
const fs = require("fs");
const repo = github.context.repo;
`,
			mode:        RuntimeModeNodeJS,
			expectError: true,
		},
		{
			name:       "Node.js mode without GitHub Actions globals should pass",
			scriptName: "test_script",
			content: `
const fs = require("fs");
const data = fs.readFileSync("file.txt");
console.log("Processing data");
`,
			mode:        RuntimeModeNodeJS,
			expectError: false,
		},
		{
			name:       "GitHub Script mode with core.* should pass (not checked)",
			scriptName: "test_script",
			content: `
core.info("This is a message");
core.setOutput("result", "value");
`,
			mode:        RuntimeModeGitHubScript,
			expectError: false,
		},
		{
			name:       "Node.js mode with GitHub Actions globals in comment should pass",
			scriptName: "test_script",
			content: `
// Don't use core.info in Node.js scripts
console.log("Use console.log instead");
`,
			mode:        RuntimeModeNodeJS,
			expectError: false,
		},
		{
			name:       "Node.js mode with type reference should pass",
			scriptName: "test_script",
			content: `
/// <reference types="@actions/github-script" />
const fs = require("fs");
`,
			mode:        RuntimeModeNodeJS,
			expectError: false,
		},
		{
			name:       "Node.js mode with multiple GitHub Actions globals should fail",
			scriptName: "test_script",
			content: `
const fs = require("fs");
core.info("Message");
exec.exec("ls");
const repo = github.context.repo;
`,
			mode:        RuntimeModeNodeJS,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoGitHubScriptGlobals(tt.scriptName, tt.content, tt.mode)
			if tt.expectError {
				assert.Error(t, err, "Expected validation to fail")
			} else {
				assert.NoError(t, err, "Expected validation to pass")
			}
		})
	}
}

func TestScriptRegistry_RegisterWithMode_Validation(t *testing.T) {
	t.Run("GitHub Script mode with execSync should panic", func(t *testing.T) {
		registry := NewScriptRegistry()
		invalidScript := `
const { execSync } = require("child_process");
execSync("ls -la");
`
		assert.Panics(t, func() {
			registry.RegisterWithMode("invalid_script", invalidScript, RuntimeModeGitHubScript)
		}, "Should panic when registering GitHub Script with execSync")
	})

	t.Run("Node.js mode with GitHub Actions globals should panic", func(t *testing.T) {
		registry := NewScriptRegistry()
		invalidScript := `
const fs = require("fs");
core.info("This should not be here");
`
		assert.Panics(t, func() {
			registry.RegisterWithMode("invalid_script", invalidScript, RuntimeModeNodeJS)
		}, "Should panic when registering Node.js script with GitHub Actions globals")
	})

	t.Run("Valid GitHub Script mode should not panic", func(t *testing.T) {
		registry := NewScriptRegistry()
		validScript := `
const { exec } = require("@actions/exec");
core.info("This is valid for GitHub Script mode");
`
		assert.NotPanics(t, func() {
			registry.RegisterWithMode("valid_script", validScript, RuntimeModeGitHubScript)
		}, "Should not panic with valid GitHub Script")
	})

	t.Run("Valid Node.js mode should not panic", func(t *testing.T) {
		registry := NewScriptRegistry()
		validScript := `
const fs = require("fs");
const { execSync } = require("child_process");
console.log("This is valid for Node.js mode");
`
		assert.NotPanics(t, func() {
			registry.RegisterWithMode("valid_script", validScript, RuntimeModeNodeJS)
		}, "Should not panic with valid Node.js script")
	})
}
