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
			stepName:       "Install Codex",
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

// TestExtractAgentIdentifier tests extracting agent identifier from file paths
func TestExtractAgentIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic agent file path",
			input:    ".github/agents/test-agent.md",
			expected: "test-agent",
		},
		{
			name:     "path with spaces",
			input:    ".github/agents/my agent file.md",
			expected: "my agent file",
		},
		{
			name:     "deeply nested path",
			input:    ".github/copilot/instructions/deep/nested/agent.md",
			expected: "agent",
		},
		{
			name:     "simple filename",
			input:    "agent.md",
			expected: "agent",
		},
		{
			name:     "path with special characters",
			input:    ".github/agents/test-agent_v2.0.md",
			expected: "test-agent_v2.0",
		},
		{
			name:     "cli-consistency-checker example",
			input:    ".github/agents/cli-consistency-checker.md",
			expected: "cli-consistency-checker",
		},
		{
			name:     "path without extension",
			input:    ".github/agents/test-agent",
			expected: "test-agent",
		},
		{
			name:     "custom agent file with .agent.md extension",
			input:    ".github/agents/speckit-dispatcher.agent.md",
			expected: "speckit-dispatcher",
		},
		{
			name:     "custom agent file simple path",
			input:    ".github/agents/test-agent.agent.md",
			expected: "test-agent",
		},
		{
			name:     "custom agent file with path",
			input:    "../agents/technical-doc-writer.agent.md",
			expected: "technical-doc-writer",
		},
		{
			name:     "custom agent file with underscores",
			input:    ".github/agents/my_custom_agent.agent.md",
			expected: "my_custom_agent",
		},
		{
			name:     "agent file with only .agent extension",
			input:    ".github/agents/test-agent.agent",
			expected: "test-agent",
		},
		{
			name:     "agent file with .agent extension in path",
			input:    "../agents/my-agent.agent",
			expected: "my-agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractAgentIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractAgentIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestShellVariableExpansionInAgentPath tests that agent paths allow shell variable expansion
func TestShellVariableExpansionInAgentPath(t *testing.T) {
	agentFile := ".github/agents/test-agent.md"
	result := ResolveAgentFilePath(agentFile)

	// The result should be fully wrapped in double quotes (not single quotes)
	// Format: "${GITHUB_WORKSPACE}/.github/agents/test-agent.md"
	expected := "\"${GITHUB_WORKSPACE}/.github/agents/test-agent.md\""

	if result != expected {
		t.Errorf("ResolveAgentFilePath(%q) = %q, want %q", agentFile, result, expected)
	}

	// Verify it's properly quoted for shell variable expansion
	// Should start with double quote (not single quote)
	if !strings.HasPrefix(result, "\"") {
		t.Errorf("Agent path should start with double quote for variable expansion, got: %s", result)
	}

	// Should end with double quote (not single quote)
	if !strings.HasSuffix(result, "\"") {
		t.Errorf("Agent path should end with double quote for variable expansion, got: %s", result)
	}

	// Should NOT contain single quotes around the double-quoted section
	// Old broken format was: '"${GITHUB_WORKSPACE}"/.github/agents/test.md'
	if strings.Contains(result, "'\"") || strings.Contains(result, "\"'") {
		t.Errorf("Agent path should not mix single and double quotes, got: %s", result)
	}

	// Should contain the variable placeholder without internal quotes
	// Correct: "${GITHUB_WORKSPACE}/path"
	// Incorrect: "${GITHUB_WORKSPACE}"/path
	if strings.Contains(result, "\"/") && !strings.HasSuffix(result, "\"/\"") {
		t.Errorf("Variable should be inside the double quotes with path, got: %s", result)
	}
}

// TestShellEscapeArgWithFullyQuotedAgentPath tests that fully quoted agent paths are not re-escaped
func TestShellEscapeArgWithFullyQuotedAgentPath(t *testing.T) {
	// This simulates what happens when ResolveAgentFilePath output goes through shellEscapeArg
	agentPath := "\"${GITHUB_WORKSPACE}/.github/agents/test-agent.md\""

	result := shellEscapeArg(agentPath)

	// Should be left as-is because it's already fully double-quoted
	if result != agentPath {
		t.Errorf("shellEscapeArg should leave fully quoted path as-is, got: %s, want: %s", result, agentPath)
	}

	// Should NOT wrap it in additional single quotes
	if strings.HasPrefix(result, "'") {
		t.Errorf("shellEscapeArg should not add single quotes to already double-quoted string, got: %s", result)
	}
}

// TestBuildAWFArgs tests the AWF argument builder helper function
func TestBuildAWFArgs(t *testing.T) {
	tests := []struct {
		name          string
		workflowData  *WorkflowData
		config        AWFConfig
		expectTTY     bool
		expectDomains string
		expectMounts  int // Expected number of mount arguments (not counting --mount flag)
	}{
		{
			name: "basic configuration without TTY",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{},
			},
			config: AWFConfig{
				AllowedDomains: "example.com",
				EnableTTY:      false,
			},
			expectTTY:     false,
			expectDomains: "example.com",
			expectMounts:  3, // /tmp, workspace, hostedtoolcache
		},
		{
			name: "configuration with TTY enabled",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{},
			},
			config: AWFConfig{
				AllowedDomains: "api.anthropic.com",
				EnableTTY:      true,
			},
			expectTTY:     true,
			expectDomains: "api.anthropic.com",
			expectMounts:  3,
		},
		{
			name: "configuration with custom mounts",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Mounts: []string{"/custom:/custom:ro", "/data:/data:rw"},
					},
				},
			},
			config: AWFConfig{
				AllowedDomains: "example.com",
				EnableTTY:      false,
			},
			expectTTY:     false,
			expectDomains: "example.com",
			expectMounts:  5, // /tmp, workspace, hostedtoolcache + 2 custom
		},
		{
			name: "configuration with custom firewall log level",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						LogLevel: "debug",
					},
				},
			},
			config: AWFConfig{
				AllowedDomains: "example.com",
				EnableTTY:      false,
			},
			expectTTY:     false,
			expectDomains: "example.com",
			expectMounts:  3,
		},
		{
			name: "configuration with custom AWF command",
			workflowData: &WorkflowData{
				NetworkPermissions: &NetworkPermissions{},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Command: "custom-awf",
					},
				},
			},
			config: AWFConfig{
				AllowedDomains: "example.com",
				EnableTTY:      false,
			},
			expectTTY:     false,
			expectDomains: "example.com",
			expectMounts:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, command := buildAWFArgs(tt.workflowData, tt.config)

			// Check for TTY flag
			hasTTY := false
			for _, arg := range args {
				if arg == "--tty" {
					hasTTY = true
					break
				}
			}
			if hasTTY != tt.expectTTY {
				t.Errorf("Expected TTY=%v, got TTY=%v", tt.expectTTY, hasTTY)
			}

			// Check for allowed domains
			foundDomains := false
			for i, arg := range args {
				if arg == "--allow-domains" && i+1 < len(args) {
					if args[i+1] != tt.expectDomains {
						t.Errorf("Expected domains %q, got %q", tt.expectDomains, args[i+1])
					}
					foundDomains = true
					break
				}
			}
			if !foundDomains {
				t.Error("Expected --allow-domains flag not found")
			}

			// Count mount arguments (each mount appears after a --mount flag)
			mountCount := 0
			for i, arg := range args {
				if arg == "--mount" && i+1 < len(args) {
					mountCount++
				}
			}
			if mountCount != tt.expectMounts {
				t.Errorf("Expected %d mounts, got %d", tt.expectMounts, mountCount)
			}

			// Check command
			if tt.workflowData.SandboxConfig != nil && tt.workflowData.SandboxConfig.Agent != nil && tt.workflowData.SandboxConfig.Agent.Command != "" {
				if command != tt.workflowData.SandboxConfig.Agent.Command {
					t.Errorf("Expected custom command %q, got %q", tt.workflowData.SandboxConfig.Agent.Command, command)
				}
			} else {
				if command != "sudo -E awf" {
					t.Errorf("Expected standard command 'sudo -E awf', got %q", command)
				}
			}

			// Verify standard arguments are present
			expectedArgs := []string{
				"--env-all",
				"--container-workdir",
				"--log-level",
				"--proxy-logs-dir",
				"--image-tag",
			}
			for _, expected := range expectedArgs {
				found := false
				for _, arg := range args {
					if arg == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected argument %q not found in args", expected)
				}
			}
		})
	}
}

// TestBuildAWFArgs_CustomArgs tests custom args from firewall and agent configs
func TestBuildAWFArgs_CustomArgs(t *testing.T) {
	workflowData := &WorkflowData{
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Args: []string{"--firewall-arg1", "--firewall-arg2"},
			},
		},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Args: []string{"--agent-arg1", "--agent-arg2"},
			},
		},
	}

	config := AWFConfig{
		AllowedDomains: "example.com",
		EnableTTY:      false,
	}

	args, _ := buildAWFArgs(workflowData, config)

	// Check that custom args are included
	customArgs := []string{"--firewall-arg1", "--firewall-arg2", "--agent-arg1", "--agent-arg2"}
	for _, customArg := range customArgs {
		found := false
		for _, arg := range args {
			if arg == customArg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected custom arg %q not found in args", customArg)
		}
	}
}

// TestBuildAWFArgs_MountSorting tests that custom mounts are sorted for consistency
func TestBuildAWFArgs_MountSorting(t *testing.T) {
	workflowData := &WorkflowData{
		NetworkPermissions: &NetworkPermissions{},
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Mounts: []string{"/z:/z:ro", "/a:/a:ro", "/m:/m:ro"},
			},
		},
	}

	config := AWFConfig{
		AllowedDomains: "example.com",
		EnableTTY:      false,
	}

	args, _ := buildAWFArgs(workflowData, config)

	// Find the custom mounts in args (they come after the standard 3 mounts)
	var customMounts []string
	mountCount := 0
	for i, arg := range args {
		if arg == "--mount" && i+1 < len(args) {
			mountCount++
			// Skip the first 3 standard mounts
			if mountCount > 3 {
				customMounts = append(customMounts, args[i+1])
			}
		}
	}

	// Verify they are sorted
	expectedMounts := []string{"/a:/a:ro", "/m:/m:ro", "/z:/z:ro"}
	if len(customMounts) != len(expectedMounts) {
		t.Errorf("Expected %d custom mounts, got %d", len(expectedMounts), len(customMounts))
	}

	for i, mount := range customMounts {
		if mount != expectedMounts[i] {
			t.Errorf("Mount at index %d: expected %q, got %q", i, expectedMounts[i], mount)
		}
	}
}
