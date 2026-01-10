package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSafeInputsStepCodeGenerationStability verifies that the MCP setup step code generation
// for safe-inputs produces stable, deterministic output when called multiple times.
// This test ensures that tools are sorted before generating cat commands.
func TestSafeInputsStepCodeGenerationStability(t *testing.T) {
	// Create a config with multiple tools to ensure sorting is tested
	safeInputsConfig := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"zebra-shell": {
				Name:        "zebra-shell",
				Description: "A shell tool that starts with Z",
				Run:         "echo zebra",
			},
			"alpha-js": {
				Name:        "alpha-js",
				Description: "A JS tool that starts with A",
				Script:      "return 'alpha';",
			},
			"middle-shell": {
				Name:        "middle-shell",
				Description: "A shell tool in the middle",
				Run:         "echo middle",
			},
			"beta-js": {
				Name:        "beta-js",
				Description: "A JS tool that starts with B",
				Script:      "return 'beta';",
			},
		},
	}

	workflowData := &WorkflowData{
		SafeInputs: safeInputsConfig,
		Tools:      make(map[string]any),
		Features: map[string]any{
			"safe-inputs": true, // Feature flag is optional now
		},
	}

	// Generate MCP setup code multiple times using the actual compiler method
	iterations := 10
	outputs := make([]string, iterations)
	compiler := &Compiler{}

	// Create a mock engine that does nothing for MCP config
	mockEngine := &CustomEngine{}

	for i := 0; i < iterations; i++ {
		var yaml strings.Builder
		compiler.generateMCPSetup(&yaml, workflowData.Tools, mockEngine, workflowData)
		outputs[i] = yaml.String()
	}

	// All iterations should produce identical output
	for i := 1; i < iterations; i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("generateMCPSetup produced different output on iteration %d", i+1)
			// Find first difference for debugging
			for j := 0; j < len(outputs[0]) && j < len(outputs[i]); j++ {
				if outputs[0][j] != outputs[i][j] {
					start := j - 100
					if start < 0 {
						start = 0
					}
					end := j + 100
					if end > len(outputs[0]) {
						end = len(outputs[0])
					}
					if end > len(outputs[i]) {
						end = len(outputs[i])
					}
					t.Errorf("First difference at position %d:\n  Expected: %q\n  Got: %q", j, outputs[0][start:end], outputs[i][start:end])
					break
				}
			}
		}
	}

	// Verify tools appear in sorted order in the output
	// All tools are sorted alphabetically regardless of type (JavaScript or shell):
	// alpha-js, beta-js, middle-shell, zebra-shell
	alphaPos := strings.Index(outputs[0], "alpha-js")
	betaPos := strings.Index(outputs[0], "beta-js")
	middlePos := strings.Index(outputs[0], "middle-shell")
	zebraPos := strings.Index(outputs[0], "zebra-shell")

	if alphaPos == -1 || betaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Error("Output should contain all tool names")
	}

	// Verify alphabetical sorting: alpha < beta < middle < zebra
	if alphaPos >= betaPos || betaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("Tools should be sorted alphabetically in step code: alpha(%d) < beta(%d) < middle(%d) < zebra(%d)",
			alphaPos, betaPos, middlePos, zebraPos)
	}
}

// TestMCPGatewayVersionFromFrontmatter tests that sandbox.mcp.version specified in frontmatter
// is correctly used in both the docker predownload step and the MCP gateway setup command
func TestMCPGatewayVersionFromFrontmatter(t *testing.T) {
	tests := []struct {
		name            string
		sandboxConfig   *SandboxConfig
		expectedVersion string
		description     string
	}{
		{
			name: "custom version specified in frontmatter",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: constants.DefaultMCPGatewayContainer,
					Version:   "v0.0.5",
					Port:      8080,
				},
			},
			expectedVersion: "v0.0.5",
			description:     "should use custom version v0.0.5",
		},
		{
			name: "no version specified - should use default",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: constants.DefaultMCPGatewayContainer,
					Port:      8080,
				},
			},
			expectedVersion: string(constants.DefaultMCPGatewayVersion),
			description:     "should use default version when not specified",
		},
		{
			name: "empty version string - should use default",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: constants.DefaultMCPGatewayContainer,
					Version:   "",
					Port:      8080,
				},
			},
			expectedVersion: string(constants.DefaultMCPGatewayVersion),
			description:     "should use default version when version is empty string",
		},
		{
			name: "version 'latest' replaced with default",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: constants.DefaultMCPGatewayContainer,
					Version:   "latest",
					Port:      8080,
				},
			},
			expectedVersion: string(constants.DefaultMCPGatewayVersion),
			description:     "should replace 'latest' with default version",
		},
		{
			name: "custom version with different format",
			sandboxConfig: &SandboxConfig{
				MCP: &MCPGatewayRuntimeConfig{
					Container: constants.DefaultMCPGatewayContainer,
					Version:   "1.2.3",
					Port:      8080,
				},
			},
			expectedVersion: "1.2.3",
			description:     "should use custom version 1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				SandboxConfig: tt.sandboxConfig,
				Tools:         map[string]any{"github": map[string]any{}},
			}

			// Ensure MCP gateway config is applied (includes normalization of "latest")
			ensureDefaultMCPGatewayConfig(workflowData)

			// After normalization, verify the version matches expected
			require.NotNil(t, workflowData.SandboxConfig, "SandboxConfig should not be nil")
			require.NotNil(t, workflowData.SandboxConfig.MCP, "MCP gateway config should not be nil")

			actualVersion := workflowData.SandboxConfig.MCP.Version
			assert.Equal(t, tt.expectedVersion, actualVersion,
				"Version after normalization should be %s (%s)", tt.expectedVersion, tt.description)

			// Test 1: Verify docker image collection uses the correct version
			dockerImages := collectDockerImages(workflowData.Tools, workflowData)
			expectedImage := constants.DefaultMCPGatewayContainer + ":" + tt.expectedVersion

			found := false
			for _, img := range dockerImages {
				if strings.Contains(img, constants.DefaultMCPGatewayContainer) {
					assert.Equal(t, expectedImage, img,
						"Docker image should include correct version (%s)", tt.description)
					found = true
					break
				}
			}
			assert.True(t, found, "MCP gateway container should be in docker images list")

			// Test 2: Verify MCP gateway setup command uses the correct version
			compiler := &Compiler{}
			var yaml strings.Builder
			mockEngine := &CustomEngine{}

			compiler.generateMCPSetup(&yaml, workflowData.Tools, mockEngine, workflowData)
			setupOutput := yaml.String()

			// The setup output should contain the container image with the correct version
			assert.Contains(t, setupOutput, expectedImage,
				"MCP gateway setup should use correct container version (%s)", tt.description)
		})
	}
}
