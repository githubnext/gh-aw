//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestBuildStandardNpmEngineInstallSteps(t *testing.T) {
	tests := []struct {
		name           string
		workflowData   *WorkflowData
		expectedSteps  int // Number of steps expected (Node.js setup + npm install)
		expectedInStep string
	}{
		{
			name:           "with default version",
			workflowData:   &WorkflowData{},
			expectedSteps:  2, // Node.js setup + npm install
			expectedInStep: string(constants.DefaultCopilotVersion),
		},
		{
			name: "with custom version from engine config",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Version: "1.2.3",
				},
			},
			expectedSteps:  2,
			expectedInStep: "1.2.3",
		},
		{
			name: "with empty version in engine config (use default)",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Version: "",
				},
			},
			expectedSteps:  2,
			expectedInStep: string(constants.DefaultCopilotVersion),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := BuildStandardNpmEngineInstallSteps(
				"@github/copilot",
				string(constants.DefaultCopilotVersion),
				"Install GitHub Copilot CLI",
				"copilot",
				tt.workflowData,
			)

			if len(steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(steps))
			}

			// Verify that the expected version appears in the steps
			found := false
			for _, step := range steps {
				for _, line := range step {
					if strings.Contains(line, tt.expectedInStep) {
						found = true
						break
					}
				}
			}

			if !found {
				t.Errorf("Expected version %s not found in steps", tt.expectedInStep)
			}
		})
	}
}

func TestBuildStandardNpmEngineInstallSteps_AllEngines(t *testing.T) {
	tests := []struct {
		name           string
		packageName    string
		defaultVersion string
		stepName       string
		cacheKeyPrefix string
	}{
		{
			name:           "copilot engine",
			packageName:    "@github/copilot",
			defaultVersion: string(constants.DefaultCopilotVersion),
			stepName:       "Install GitHub Copilot CLI",
			cacheKeyPrefix: "copilot",
		},
		{
			name:           "codex engine",
			packageName:    "@openai/codex",
			defaultVersion: string(constants.DefaultCodexVersion),
			stepName:       "Install Codex CLI",
			cacheKeyPrefix: "codex",
		},
		{
			name:           "claude engine",
			packageName:    "@anthropic-ai/claude-code",
			defaultVersion: string(constants.DefaultClaudeCodeVersion),
			stepName:       "Install Claude Code CLI",
			cacheKeyPrefix: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{}

			steps := BuildStandardNpmEngineInstallSteps(
				tt.packageName,
				tt.defaultVersion,
				tt.stepName,
				tt.cacheKeyPrefix,
				workflowData,
			)

			if len(steps) < 1 {
				t.Errorf("Expected at least 1 step, got %d", len(steps))
			}

			// Verify package name appears in steps
			found := false
			for _, step := range steps {
				for _, line := range step {
					if strings.Contains(line, tt.packageName) {
						found = true
						break
					}
				}
			}

			if !found {
				t.Errorf("Expected package name %s not found in steps", tt.packageName)
			}
		})
	}
}

func TestShellEscapeArgWithFullyQuotedAgentPath(t *testing.T) {
	// After the bypass removal, a double-quoted string is treated as any other argument
	// containing special characters and gets properly single-quoted.
	agentPath := "\"${GITHUB_WORKSPACE}/.github/agents/test-agent.md\""

	result := shellEscapeArg(agentPath)

	// Should be single-quoted (the bypass was removed for security)
	if !strings.HasPrefix(result, "'") {
		t.Errorf("shellEscapeArg should single-quote the agent path after bypass removal, got: %s", result)
	}

	// Should NOT be left as-is (that was the vulnerable bypass behavior)
	if result == agentPath {
		t.Errorf("shellEscapeArg should not leave double-quoted agent path unchanged (bypass removed), got: %s", result)
	}
}

func TestGetNpmBinPathSetup(t *testing.T) {
	pathSetup := GetNpmBinPathSetup()

	// Should use find command to locate bin directories in hostedtoolcache
	if !strings.Contains(pathSetup, "/opt/hostedtoolcache") {
		t.Errorf("PATH setup should reference /opt/hostedtoolcache, got: %s", pathSetup)
	}

	// Should search for bin directories
	if !strings.Contains(pathSetup, "-name bin") {
		t.Errorf("PATH setup should search for bin directories, got: %s", pathSetup)
	}

	// Should preserve existing PATH
	if !strings.Contains(pathSetup, "$PATH") {
		t.Errorf("PATH setup should include $PATH, got: %s", pathSetup)
	}

	// Should re-prepend GOROOT/bin after the find to preserve correct Go version ordering
	// (find returns alphabetically, so go/1.23 can shadow go/1.25)
	if !strings.Contains(pathSetup, "$GOROOT") {
		t.Errorf("PATH setup should re-prepend GOROOT/bin after find, got: %s", pathSetup)
	}

	// GOROOT re-prepend should come AFTER the find command
	findIdx := strings.Index(pathSetup, "find /opt/hostedtoolcache")
	gorootIdx := strings.Index(pathSetup, "$GOROOT")
	if gorootIdx < findIdx {
		t.Errorf("GOROOT re-prepend should come after find command, got: %s", pathSetup)
	}
}

// TestGetNpmBinPathSetup_GorootOrdering verifies that GOROOT/bin takes precedence
// over alphabetically-ordered Go versions in hostedtoolcache.
func TestGetNpmBinPathSetup_GorootOrdering(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping shell-based test on non-Linux platform")
	}

	// Create a temporary hostedtoolcache structure with two Go versions
	tmpDir := t.TempDir()
	goOld := filepath.Join(tmpDir, "go", "1.23.12", "x64", "bin")
	goNew := filepath.Join(tmpDir, "go", "1.25.0", "x64", "bin")
	os.MkdirAll(goOld, 0o755)
	os.MkdirAll(goNew, 0o755)

	// Write fake go scripts: old reports 1.23, new reports 1.25
	os.WriteFile(filepath.Join(goOld, "go"), []byte("#!/bin/bash\necho 'go version go1.23.12 linux/amd64'\n"), 0o755)
	os.WriteFile(filepath.Join(goNew, "go"), []byte("#!/bin/bash\necho 'go version go1.25.0 linux/amd64'\n"), 0o755)

	// Simulate the PATH setup with GOROOT pointing to the newer version
	shellCmd := fmt.Sprintf(
		`export GOROOT=%q; export PATH="$(find %q -maxdepth 4 -type d -name bin 2>/dev/null | tr '\n' ':')$PATH"; [ -n "$GOROOT" ] && export PATH="$GOROOT/bin:$PATH" || true; go version`,
		filepath.Join(tmpDir, "go", "1.25.0", "x64"),
		tmpDir,
	)

	cmd := exec.Command("bash", "-c", shellCmd)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to execute shell command: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if !strings.Contains(result, "go1.25.0") {
		t.Errorf("Expected go1.25.0 to take precedence, but got: %s", result)
	}
}

// TestGetNpmBinPathSetup_NoGorootDoesNotBreakChain verifies that when GOROOT is
// not set, the command chain continues (the || true prevents short-circuit).
func TestGetNpmBinPathSetup_NoGorootDoesNotBreakChain(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping shell-based test on non-Linux platform")
	}

	// The full command pattern used by engines:
	//   GetNpmBinPathSetup() && INSTRUCTION="..." && codex exec ...
	// When GOROOT is empty, [ -n "$GOROOT" ] is false. Without || true,
	// the && chain short-circuits and INSTRUCTION is never set.
	shellCmd := `unset GOROOT; export PATH="$(find /opt/hostedtoolcache -maxdepth 4 -type d -name bin 2>/dev/null | tr '\n' ':')$PATH"; [ -n "$GOROOT" ] && export PATH="$GOROOT/bin:$PATH" || true && echo "chain-continued"`

	cmd := exec.Command("bash", "-c", shellCmd)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command chain should not fail when GOROOT is empty: %v", err)
	}

	result := strings.TrimSpace(string(output))
	if !strings.Contains(result, "chain-continued") {
		t.Errorf("Expected command chain to continue when GOROOT is empty, got: %q", result)
	}
}

func TestYamlStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain string unchanged",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			expected: "",
		},
		{
			name:     "github actions expression unchanged",
			input:    "${{ secrets.TOKEN }}",
			expected: "${{ secrets.TOKEN }}",
		},
		{
			name:     "json object gets single-quoted",
			input:    `{"key":"value"}`,
			expected: `'{"key":"value"}'`,
		},
		{
			name:     "json array gets single-quoted",
			input:    `["a","b"]`,
			expected: `'["a","b"]'`,
		},
		{
			name:     "json object with embedded single quote gets escaped",
			input:    `{"key":"it's"}`,
			expected: `'{"key":"it''s"}'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := yamlStringValue(tt.input)
			if result != tt.expected {
				t.Errorf("yamlStringValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatStepWithCommandAndEnvYAMLSafe(t *testing.T) {
	t.Run("json env var is single-quoted for valid YAML", func(t *testing.T) {
		stepLines := []string{"      - name: Test step"}
		env := map[string]string{
			"MY_JSON": `{"key":"value","nested":{"a":1}}`,
		}
		result := FormatStepWithCommandAndEnv(stepLines, "echo test", env)
		output := strings.Join(result, "\n")

		// The JSON value must be single-quoted so YAML treats it as a string
		if !strings.Contains(output, `MY_JSON: '{"key":"value","nested":{"a":1}}'`) {
			t.Errorf("Expected single-quoted JSON in env, got:\n%s", output)
		}
	})

	t.Run("github expression env var is not quoted", func(t *testing.T) {
		stepLines := []string{"      - name: Test step"}
		env := map[string]string{
			"MY_TOKEN": "${{ secrets.TOKEN }}",
		}
		result := FormatStepWithCommandAndEnv(stepLines, "echo test", env)
		output := strings.Join(result, "\n")

		// GitHub Actions expressions should not be wrapped in extra quotes
		if !strings.Contains(output, "MY_TOKEN: ${{ secrets.TOKEN }}") {
			t.Errorf("Expected unquoted github expression in env, got:\n%s", output)
		}
	})
}
