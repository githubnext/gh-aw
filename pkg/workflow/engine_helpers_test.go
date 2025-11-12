package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
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
			expectedInStep: constants.DefaultCopilotVersion,
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
			expectedInStep: constants.DefaultCopilotVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := BuildStandardNpmEngineInstallSteps(
				"@github/copilot",
				constants.DefaultCopilotVersion,
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
			defaultVersion: constants.DefaultCopilotVersion,
			stepName:       "Install GitHub Copilot CLI",
			cacheKeyPrefix: "copilot",
		},
		{
			name:           "codex engine",
			packageName:    "@openai/codex",
			defaultVersion: constants.DefaultCodexVersion,
			stepName:       "Install Codex",
			cacheKeyPrefix: "codex",
		},
		{
			name:           "claude engine",
			packageName:    "@anthropic-ai/claude-code",
			defaultVersion: constants.DefaultClaudeCodeVersion,
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

// TestResolveAgentFilePath tests the shared agent file path resolution helper
func TestResolveAgentFilePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic agent file path",
			input:    ".github/agents/test-agent.md",
			expected: "\"${GITHUB_WORKSPACE}/.github/agents/test-agent.md\"",
		},
		{
			name:     "path with spaces",
			input:    ".github/agents/my agent file.md",
			expected: "\"${GITHUB_WORKSPACE}/.github/agents/my agent file.md\"",
		},
		{
			name:     "deeply nested path",
			input:    ".github/copilot/instructions/deep/nested/agent.md",
			expected: "\"${GITHUB_WORKSPACE}/.github/copilot/instructions/deep/nested/agent.md\"",
		},
		{
			name:     "simple filename",
			input:    "agent.md",
			expected: "\"${GITHUB_WORKSPACE}/agent.md\"",
		},
		{
			name:     "path with special characters",
			input:    ".github/agents/test-agent_v2.0.md",
			expected: "\"${GITHUB_WORKSPACE}/.github/agents/test-agent_v2.0.md\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveAgentFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveAgentFilePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestResolveAgentFilePathFormat tests that the output format is consistent
func TestResolveAgentFilePathFormat(t *testing.T) {
	input := ".github/agents/test.md"
	result := ResolveAgentFilePath(input)

	// Verify it starts with opening quote, GITHUB_WORKSPACE variable, and forward slash
	expectedPrefix := "\"${GITHUB_WORKSPACE}/"
	if !strings.HasPrefix(result, expectedPrefix) {
		t.Errorf("Expected path to start with %q, got: %s", expectedPrefix, result)
	}

	// Verify it ends with the input path and a closing quote
	expectedSuffix := input + "\""
	if !strings.HasSuffix(result, expectedSuffix) {
		t.Errorf("Expected path to end with %q, got: %q", expectedSuffix, result)
	}

	// Verify the complete expected format
	expected := "\"${GITHUB_WORKSPACE}/" + input + "\""
	if result != expected {
		t.Errorf("Expected %q, got: %q", expected, result)
	}
}
