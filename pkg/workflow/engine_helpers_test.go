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

// TestBuildStandardPipInstallSteps tests the pip package installation helper
func TestBuildStandardPipInstallSteps(t *testing.T) {
	tests := []struct {
		name          string
		packages      []string
		useUv         bool
		expectedSteps int
		checkContains []string
	}{
		{
			name:          "single package with pip",
			packages:      []string{"requests"},
			useUv:         false,
			expectedSteps: 1, // Only install step - runtime detection adds Python setup automatically
			checkContains: []string{"pip install requests"},
		},
		{
			name:          "multiple packages with pip",
			packages:      []string{"requests", "numpy", "pandas"},
			useUv:         false,
			expectedSteps: 1,
			checkContains: []string{"pip install requests numpy pandas"},
		},
		{
			name:          "single package with uv",
			packages:      []string{"requests"},
			useUv:         true,
			expectedSteps: 1, // Only install step - runtime detection adds Python/uv setup automatically
			checkContains: []string{"uv pip install --system requests"},
		},
		{
			name:          "multiple packages with uv",
			packages:      []string{"requests", "numpy"},
			useUv:         true,
			expectedSteps: 1,
			checkContains: []string{"uv pip install --system requests numpy"},
		},
		{
			name:          "empty package list",
			packages:      []string{},
			useUv:         false,
			expectedSteps: 0,
			checkContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := BuildStandardPipInstallSteps(tt.packages, tt.useUv)

			if len(steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(steps))
			}

			// Check that expected strings appear in the steps
			allStepsText := ""
			for _, step := range steps {
				for _, line := range step {
					allStepsText += line + "\n"
				}
			}

			for _, expected := range tt.checkContains {
				if !strings.Contains(allStepsText, expected) {
					t.Errorf("Expected to find %q in steps, but it was not found", expected)
				}
			}
		})
	}
}

// TestBuildStandardPipInstallSteps_RuntimeDetection tests that the helper relies on runtime detection
func TestBuildStandardPipInstallSteps_RuntimeDetection(t *testing.T) {
	steps := BuildStandardPipInstallSteps([]string{"requests"}, false)

	// Convert steps to text
	allStepsText := ""
	for _, step := range steps {
		for _, line := range step {
			allStepsText += line + "\n"
		}
	}

	// Should NOT contain Python setup - runtime detection handles that
	if strings.Contains(allStepsText, "Setup Python") {
		t.Error("Should not include Python setup - runtime detection handles that automatically")
	}

	// Should contain the pip install command
	if !strings.Contains(allStepsText, "pip install requests") {
		t.Error("Should contain pip install command")
	}
}

// TestBuildStandardPipInstallSteps_UvRuntimeDetection tests uv with runtime detection
func TestBuildStandardPipInstallSteps_UvRuntimeDetection(t *testing.T) {
	steps := BuildStandardPipInstallSteps([]string{"requests"}, true)

	// Convert steps to text
	allStepsText := ""
	for _, step := range steps {
		for _, line := range step {
			allStepsText += line + "\n"
		}
	}

	// Should NOT contain uv setup - runtime detection handles that
	if strings.Contains(allStepsText, "Setup uv") {
		t.Error("Should not include uv setup - runtime detection handles that automatically")
	}

	// Should contain the uv pip install command
	if !strings.Contains(allStepsText, "uv pip install --system requests") {
		t.Error("Should contain uv pip install command")
	}
}

// TestBuildStandardDockerSetupSteps tests the Docker image pre-download helper
func TestBuildStandardDockerSetupSteps(t *testing.T) {
	tests := []struct {
		name          string
		images        []string
		expectedSteps int
		checkContains []string
	}{
		{
			name:          "single image",
			images:        []string{"ghcr.io/github/github-mcp-server:v1.0.0"},
			expectedSteps: 1,
			checkContains: []string{"Download Docker images", "docker pull ghcr.io/github/github-mcp-server:v1.0.0"},
		},
		{
			name:          "multiple images",
			images:        []string{"alpine:latest", "ubuntu:22.04", "nginx:stable"},
			expectedSteps: 1,
			checkContains: []string{"Download Docker images", "docker pull alpine:latest", "docker pull nginx:stable", "docker pull ubuntu:22.04"},
		},
		{
			name:          "empty image list",
			images:        []string{},
			expectedSteps: 0,
			checkContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := BuildStandardDockerSetupSteps(tt.images)

			if len(steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(steps))
			}

			// Check that expected strings appear in the steps
			allStepsText := ""
			for _, step := range steps {
				for _, line := range step {
					allStepsText += line + "\n"
				}
			}

			for _, expected := range tt.checkContains {
				if !strings.Contains(allStepsText, expected) {
					t.Errorf("Expected to find %q in steps, but it was not found", expected)
				}
			}
		})
	}
}

// TestBuildStandardDockerSetupSteps_Sorted tests that images are sorted for consistency
func TestBuildStandardDockerSetupSteps_Sorted(t *testing.T) {
	images := []string{"ubuntu:22.04", "alpine:latest", "nginx:stable"}
	steps := BuildStandardDockerSetupSteps(images)

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Convert to single string
	allText := strings.Join(steps[0], "\n")

	// Find positions of each image in the output
	alpinePos := strings.Index(allText, "alpine:latest")
	nginxPos := strings.Index(allText, "nginx:stable")
	ubuntuPos := strings.Index(allText, "ubuntu:22.04")

	// Verify alphabetical order (alpine < nginx < ubuntu)
	if alpinePos == -1 || nginxPos == -1 || ubuntuPos == -1 {
		t.Fatal("Not all images found in output")
	}

	if alpinePos > nginxPos || nginxPos > ubuntuPos {
		t.Error("Images should be sorted alphabetically")
	}
}

// TestBuildStandardEngineCleanupSteps tests the cleanup helper
func TestBuildStandardEngineCleanupSteps(t *testing.T) {
	tests := []struct {
		name          string
		cleanupPaths  []string
		expectedSteps int
		checkContains []string
	}{
		{
			name:          "single path",
			cleanupPaths:  []string{"/tmp/gh-aw/.copilot/"},
			expectedSteps: 1,
			checkContains: []string{"Cleanup temporary files", "if: always()", "rm -rf /tmp/gh-aw/.copilot/"},
		},
		{
			name:          "multiple paths",
			cleanupPaths:  []string{"/tmp/gh-aw/.copilot/", "/tmp/gh-aw/mcp-config/", "/tmp/gh-aw/logs/"},
			expectedSteps: 1,
			checkContains: []string{"Cleanup temporary files", "if: always()", "rm -rf /tmp/gh-aw/.copilot/", "rm -rf /tmp/gh-aw/mcp-config/", "rm -rf /tmp/gh-aw/logs/"},
		},
		{
			name:          "empty path list",
			cleanupPaths:  []string{},
			expectedSteps: 0,
			checkContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := BuildStandardEngineCleanupSteps(tt.cleanupPaths)

			if len(steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(steps))
			}

			// Check that expected strings appear in the steps
			allStepsText := ""
			for _, step := range steps {
				for _, line := range step {
					allStepsText += line + "\n"
				}
			}

			for _, expected := range tt.checkContains {
				if !strings.Contains(allStepsText, expected) {
					t.Errorf("Expected to find %q in steps, but it was not found", expected)
				}
			}
		})
	}
}

// TestBuildStandardEngineCleanupSteps_Sorted tests that paths are sorted for consistency
func TestBuildStandardEngineCleanupSteps_Sorted(t *testing.T) {
	paths := []string{"/tmp/z", "/tmp/a", "/tmp/m"}
	steps := BuildStandardEngineCleanupSteps(paths)

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Convert to single string
	allText := strings.Join(steps[0], "\n")

	// Find positions of each path in the output
	aPos := strings.Index(allText, "rm -rf /tmp/a")
	mPos := strings.Index(allText, "rm -rf /tmp/m")
	zPos := strings.Index(allText, "rm -rf /tmp/z")

	// Verify alphabetical order (a < m < z)
	if aPos == -1 || mPos == -1 || zPos == -1 {
		t.Fatal("Not all paths found in output")
	}

	if aPos > mPos || mPos > zPos {
		t.Error("Paths should be sorted alphabetically")
	}
}

// TestBuildStandardEngineCleanupSteps_AlwaysCondition tests that cleanup has always() condition
func TestBuildStandardEngineCleanupSteps_AlwaysCondition(t *testing.T) {
	steps := BuildStandardEngineCleanupSteps([]string{"/tmp/test"})

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Convert to single string
	allText := strings.Join(steps[0], "\n")

	// Verify that if: always() is present
	if !strings.Contains(allText, "if: always()") {
		t.Error("Cleanup step should have 'if: always()' condition")
	}
}

// TestBuildStandardEngineCleanupSteps_ErrorHandling tests that cleanup commands have || true
func TestBuildStandardEngineCleanupSteps_ErrorHandling(t *testing.T) {
	steps := BuildStandardEngineCleanupSteps([]string{"/tmp/test1", "/tmp/test2"})

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Convert to single string
	allText := strings.Join(steps[0], "\n")

	// Verify that || true is present for each rm command
	if !strings.Contains(allText, "rm -rf /tmp/test1 || true") {
		t.Error("Cleanup commands should have '|| true' suffix to prevent failures")
	}

	if !strings.Contains(allText, "rm -rf /tmp/test2 || true") {
		t.Error("Cleanup commands should have '|| true' suffix to prevent failures")
	}
}
