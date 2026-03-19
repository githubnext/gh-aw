//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/stringutil"
)

// TestIsPlaywrightCLIMode tests the isPlaywrightCLIMode helper function
func TestIsPlaywrightCLIMode(t *testing.T) {
	tests := []struct {
		name     string
		config   *PlaywrightToolConfig
		expected bool
	}{
		{
			name:     "nil config is not CLI mode",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty config is not CLI mode",
			config:   &PlaywrightToolConfig{},
			expected: false,
		},
		{
			name:     "mode mcp is not CLI mode",
			config:   &PlaywrightToolConfig{Mode: "mcp"},
			expected: false,
		},
		{
			name:     "mode cli is CLI mode",
			config:   &PlaywrightToolConfig{Mode: "cli"},
			expected: true,
		},
		{
			name:     "unknown mode is not CLI mode",
			config:   &PlaywrightToolConfig{Mode: "unknown"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPlaywrightCLIMode(tt.config)
			assert.Equal(t, tt.expected, result, "isPlaywrightCLIMode(%+v)", tt.config)
		})
	}
}

// TestParsePlaywrightToolWithMode tests that the mode field is parsed correctly
func TestParsePlaywrightToolWithMode(t *testing.T) {
	tests := []struct {
		name         string
		input        any
		expectedMode string
	}{
		{
			name:         "nil input has empty mode",
			input:        nil,
			expectedMode: "",
		},
		{
			name: "mode cli is parsed",
			input: map[string]any{
				"mode": "cli",
			},
			expectedMode: "cli",
		},
		{
			name: "mode mcp is parsed",
			input: map[string]any{
				"mode": "mcp",
			},
			expectedMode: "mcp",
		},
		{
			name: "no mode field has empty mode",
			input: map[string]any{
				"version": "1.0.0",
			},
			expectedMode: "",
		},
		{
			name: "mode with other fields is parsed",
			input: map[string]any{
				"mode":    "cli",
				"version": "1.0.0",
				"args":    []any{"--browser", "chromium"},
			},
			expectedMode: "cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := parsePlaywrightTool(tt.input)
			require.NotNil(t, config, "parsePlaywrightTool should not return nil")
			assert.Equal(t, tt.expectedMode, config.Mode, "Mode should be parsed correctly")
		})
	}
}

// TestValidatePlaywrightToolConfig tests playwright mode validation
func TestValidatePlaywrightToolConfig(t *testing.T) {
	tests := []struct {
		name      string
		tools     *Tools
		expectErr bool
		errMsg    string
	}{
		{
			name:      "nil tools is valid",
			tools:     nil,
			expectErr: false,
		},
		{
			name: "nil playwright is valid",
			tools: &Tools{
				Custom: make(map[string]MCPServerConfig),
				raw:    make(map[string]any),
			},
			expectErr: false,
		},
		{
			name: "mode cli is valid",
			tools: &Tools{
				Playwright: &PlaywrightToolConfig{Mode: "cli"},
				Custom:     make(map[string]MCPServerConfig),
				raw:        make(map[string]any),
			},
			expectErr: false,
		},
		{
			name: "mode mcp is valid",
			tools: &Tools{
				Playwright: &PlaywrightToolConfig{Mode: "mcp"},
				Custom:     make(map[string]MCPServerConfig),
				raw:        make(map[string]any),
			},
			expectErr: false,
		},
		{
			name: "empty mode is valid (defaults to mcp)",
			tools: &Tools{
				Playwright: &PlaywrightToolConfig{},
				Custom:     make(map[string]MCPServerConfig),
				raw:        make(map[string]any),
			},
			expectErr: false,
		},
		{
			name: "invalid mode is rejected",
			tools: &Tools{
				Playwright: &PlaywrightToolConfig{Mode: "invalid"},
				Custom:     make(map[string]MCPServerConfig),
				raw:        make(map[string]any),
			},
			expectErr: true,
			errMsg:    "tools.playwright.mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlaywrightToolConfig(tt.tools, "test-workflow")
			if tt.expectErr {
				require.Error(t, err, "Expected an error but got none")
				assert.Contains(t, err.Error(), tt.errMsg, "Error message should contain expected text")
			} else {
				assert.NoError(t, err, "Expected no error")
			}
		})
	}
}

// TestPlaywrightCLIModeCompilation tests that playwright CLI mode compiles correctly
func TestPlaywrightCLIModeCompilation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gh-aw-playwright-cli-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name             string
		workflowContent  string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "CLI mode does not include playwright MCP Docker image",
			workflowContent: `---
on: push
engine: claude
tools:
  playwright:
    mode: cli
---

# Test Workflow

Test playwright CLI mode.
`,
			shouldNotContain: []string{
				"mcr.microsoft.com/playwright/mcp",
			},
			shouldContain: []string{
				"@playwright/cli",
				"Install Playwright CLI",
			},
		},
		{
			name: "MCP mode (default) includes playwright MCP Docker image",
			workflowContent: `---
on: push
engine: claude
tools:
  playwright:
---

# Test Workflow

Test playwright MCP mode.
`,
			shouldContain: []string{
				"mcr.microsoft.com/playwright/mcp",
			},
			shouldNotContain: []string{
				"@playwright/cli",
				"Install Playwright CLI",
			},
		},
		{
			name: "Explicit MCP mode includes playwright MCP Docker image",
			workflowContent: `---
on: push
engine: claude
tools:
  playwright:
    mode: mcp
---

# Test Workflow

Test playwright explicit MCP mode.
`,
			shouldContain: []string{
				"mcr.microsoft.com/playwright/mcp",
			},
			shouldNotContain: []string{
				"@playwright/cli",
				"Install Playwright CLI",
			},
		},
		{
			name: "CLI mode with copilot engine",
			workflowContent: `---
on: push
engine: copilot
tools:
  playwright:
    mode: cli
---

# Test Workflow

Test playwright CLI mode with copilot.
`,
			shouldNotContain: []string{
				"mcr.microsoft.com/playwright/mcp",
			},
			shouldContain: []string{
				"@playwright/cli",
				"Install Playwright CLI",
			},
		},
		{
			name: "CLI mode with codex engine",
			workflowContent: `---
on: push
engine: codex
tools:
  playwright:
    mode: cli
---

# Test Workflow

Test playwright CLI mode with codex.
`,
			shouldNotContain: []string{
				"mcr.microsoft.com/playwright/mcp",
			},
			shouldContain: []string{
				"@playwright/cli",
				"Install Playwright CLI",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test-"+strings.ReplaceAll(tt.name, " ", "-")+".md")
			err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644)
			require.NoError(t, err, "Failed to create test workflow file")

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(testFile)
			require.NoError(t, err, "Failed to compile workflow")

			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Failed to read generated lock file")

			lockStr := string(lockContent)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, lockStr, expected, "Lock file should contain: %s", expected)
			}
			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, lockStr, notExpected, "Lock file should NOT contain: %s", notExpected)
			}
		})
	}
}

// TestPlaywrightCLIModePrompt tests that the CLI mode prompt is included instead of MCP prompt
func TestPlaywrightCLIModePrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gh-aw-playwright-cli-prompt-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name             string
		workflowContent  string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "CLI mode uses playwright-cli prompt",
			workflowContent: `---
on: push
engine: claude
tools:
  playwright:
    mode: cli
---

# Test Workflow

Test playwright CLI prompt.
`,
			shouldContain: []string{
				"playwright_cli_prompt.md",
			},
			shouldNotContain: []string{
				"playwright_prompt.md",
			},
		},
		{
			name: "MCP mode uses playwright MCP prompt",
			workflowContent: `---
on: push
engine: claude
tools:
  playwright:
---

# Test Workflow

Test playwright MCP prompt.
`,
			shouldContain: []string{
				"playwright_prompt.md",
			},
			shouldNotContain: []string{
				"playwright_cli_prompt.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test-"+strings.ReplaceAll(tt.name, " ", "-")+".md")
			err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644)
			require.NoError(t, err, "Failed to create test workflow file")

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(testFile)
			require.NoError(t, err, "Failed to compile workflow")

			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Failed to read generated lock file")

			lockStr := string(lockContent)

			for _, expected := range tt.shouldContain {
				assert.Contains(t, lockStr, expected, "Lock file should contain: %s", expected)
			}
			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, lockStr, notExpected, "Lock file should NOT contain: %s", notExpected)
			}
		})
	}
}

// TestPlaywrightCLIModeValidationError tests that invalid mode values are rejected
func TestPlaywrightCLIModeValidationError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gh-aw-playwright-cli-validation-test-*")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir)

	workflowContent := `---
on: push
engine: claude
tools:
  playwright:
    mode: invalid
---

# Test Workflow

Test playwright invalid mode.
`
	testFile := filepath.Join(tmpDir, "test-invalid-mode.md")
	err = os.WriteFile(testFile, []byte(workflowContent), 0644)
	require.NoError(t, err, "Failed to create test workflow file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.Error(t, err, "Expected compilation to fail with invalid mode")
	// Either the schema validator or our custom validator should reject the invalid mode
	assert.Contains(t, err.Error(), "mode", "Error should mention the mode field")
}
