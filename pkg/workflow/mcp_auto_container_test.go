package workflow

import (
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetWellKnownContainer tests the well-known container mapping for common commands
func TestGetWellKnownContainer(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		expectedImage   string
		expectedEntry   string
		shouldBeNil     bool
	}{
		{
			name:          "npx command maps to Node Alpine LTS",
			command:       "npx",
			expectedImage: constants.DefaultNodeAlpineLTSImage,
			expectedEntry: "npx",
			shouldBeNil:   false,
		},
		{
			name:          "uvx command maps to Python Alpine LTS",
			command:       "uvx",
			expectedImage: constants.DefaultPythonAlpineLTSImage,
			expectedEntry: "uvx",
			shouldBeNil:   false,
		},
		{
			name:        "unknown command returns nil",
			command:     "unknown-command",
			shouldBeNil: true,
		},
		{
			name:        "docker command returns nil (already containerized)",
			command:     "docker",
			shouldBeNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWellKnownContainer(tt.command)

			if tt.shouldBeNil {
				assert.Nil(t, result, "Expected nil for command: %s", tt.command)
			} else {
				require.NotNil(t, result, "Expected non-nil result for command: %s", tt.command)
				assert.Equal(t, tt.expectedImage, result.Image, "Image mismatch for command: %s", tt.command)
				assert.Equal(t, tt.expectedEntry, result.Entrypoint, "Entrypoint mismatch for command: %s", tt.command)
			}
		})
	}
}

// TestAutoContainerAssignment tests that stdio MCP servers with well-known commands
// are automatically assigned appropriate containers
func TestAutoContainerAssignment(t *testing.T) {
	tests := []struct {
		name              string
		toolConfig        map[string]any
		toolName          string
		expectedContainer string
		expectedEntrypoint string
		expectError       bool
	}{
		{
			name: "npx command gets Node Alpine container",
			toolConfig: map[string]any{
				"command": "npx",
				"args":    []any{"-y", "@microsoft/markitdown"},
			},
			toolName:           "markitdown",
			expectedContainer:  constants.DefaultNodeAlpineLTSImage,
			expectedEntrypoint: "npx",
			expectError:        false,
		},
		{
			name: "uvx command gets Python Alpine container",
			toolConfig: map[string]any{
				"command": "uvx",
				"args":    []any{"--from", "git+https://github.com/oraios/serena", "serena"},
			},
			toolName:           "serena",
			expectedContainer:  constants.DefaultPythonAlpineLTSImage,
			expectedEntrypoint: "uvx",
			expectError:        false,
		},
		{
			name: "explicit container overrides auto-assignment",
			toolConfig: map[string]any{
				"command":   "npx",
				"args":      []any{"-y", "some-tool"},
				"container": "custom-image:latest",
			},
			toolName:           "custom-tool",
			expectedContainer:  "custom-image:latest",
			expectedEntrypoint: "",
			expectError:        false,
		},
		{
			name: "unknown command does not get auto-container",
			toolConfig: map[string]any{
				"command": "unknown-cmd",
				"args":    []any{"--help"},
			},
			toolName:          "unknown-tool",
			expectedContainer: "",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getMCPConfig(tt.toolConfig, tt.toolName)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err, "Unexpected error for test: %s", tt.name)
			require.NotNil(t, result, "Expected non-nil result for test: %s", tt.name)

			// After getMCPConfig processing, check the Docker command was generated correctly
			if tt.expectedContainer != "" && tt.expectedEntrypoint != "" {
				// The container should have been transformed to docker command
				assert.Equal(t, "docker", result.Command, "Expected docker command after container transformation")
				
				// Verify the container image is in the args
				containerImageFound := false
				for _, arg := range result.Args {
					if arg == tt.expectedContainer || arg == tt.expectedContainer+":"+result.Version {
						containerImageFound = true
						break
					}
				}
				assert.True(t, containerImageFound, "Expected container image %s in docker args: %v", tt.expectedContainer, result.Args)

				// Verify entrypoint args include the original command
				entrypointFound := false
				for _, arg := range result.Args {
					if arg == tt.expectedEntrypoint {
						entrypointFound = true
						break
					}
				}
				assert.True(t, entrypointFound, "Expected entrypoint %s in docker args: %v", tt.expectedEntrypoint, result.Args)
			} else if tt.expectedContainer != "" && tt.expectedEntrypoint == "" {
				// Explicit container override case - check the specific container is used
				containerImageFound := false
				for _, arg := range result.Args {
					if arg == tt.expectedContainer {
						containerImageFound = true
						break
					}
				}
				assert.True(t, containerImageFound, "Expected explicit container %s in docker args: %v", tt.expectedContainer, result.Args)
			}
		})
	}
}

// TestAutoContainerWithArgs verifies that original args are preserved after auto-container assignment
func TestAutoContainerWithArgs(t *testing.T) {
	toolConfig := map[string]any{
		"command": "npx",
		"args":    []any{"-y", "@microsoft/markitdown", "--format", "gfm"},
	}

	result, err := getMCPConfig(toolConfig, "markitdown")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify that after transformation, all original args are present
	// The structure should be: docker run ... node:lts-alpine npx -y @microsoft/markitdown --format gfm
	argsStr := ""
	for _, arg := range result.Args {
		argsStr += arg + " "
	}

	assert.Contains(t, argsStr, "npx", "Expected npx command in docker args")
	assert.Contains(t, argsStr, "-y", "Expected -y flag preserved")
	assert.Contains(t, argsStr, "@microsoft/markitdown", "Expected package name preserved")
	assert.Contains(t, argsStr, "--format", "Expected --format flag preserved")
	assert.Contains(t, argsStr, "gfm", "Expected gfm value preserved")
}

// TestNoAutoContainerForHTTPMCP verifies that HTTP MCP servers don't get auto-containerization
func TestNoAutoContainerForHTTPMCP(t *testing.T) {
	toolConfig := map[string]any{
		"type": "http",
		"url":  "https://api.example.com/mcp",
	}

	result, err := getMCPConfig(toolConfig, "http-tool")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "http", result.Type)
	assert.Empty(t, result.Container, "HTTP MCP should not get container assignment")
	assert.Empty(t, result.Command, "HTTP MCP should not get command assignment")
}
