package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAutoContainerizationIntegration tests end-to-end workflow compilation
// with automatic containerization for npx and uvx commands
func TestAutoContainerizationIntegration(t *testing.T) {
	tests := []struct {
		name            string
		workflowContent string
		expectedStrings []string
		engine          string
	}{
		{
			name: "npx command auto-containerized in Copilot workflow",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network: defaults
mcp-servers:
  markitdown:
    command: npx
    args: ["-y", "@microsoft/markitdown"]
---

# Test Auto-Containerization

Test that npx MCP server is automatically containerized.
`,
			expectedStrings: []string{
				`"markitdown": {`,
				`"command": "docker"`,
				`"node:lts-alpine"`,
				`"npx"`,
				`"@microsoft/markitdown"`,
			},
			engine: "copilot",
		},
		{
			name: "uvx command auto-containerized in simple custom tool",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
network: defaults
tools:
  python-tool:
    command: uvx
    args: ["python-package"]
---

# Test UVX Auto-Containerization

Test that uvx MCP server is automatically containerized.
`,
			expectedStrings: []string{
				`"python-tool": {`,
				`"command": "docker"`,
				`"python:alpine"`,
				`"uvx"`,
				`"python-package"`,
			},
			engine: "claude",
		},
		{
			name: "explicit container overrides auto-assignment",
			workflowContent: `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network: defaults
tools:
  custom-tool:
    container: custom-image:v1
    entrypoint: npx
    entrypointArgs: ["-y", "some-tool"]
---

# Test Explicit Container Override

Test that explicit container configuration overrides auto-assignment.
`,
			expectedStrings: []string{
				`"custom-tool": {`,
				`"command": "docker"`,
				`"custom-image:v1"`,
				// Should NOT contain auto-assigned node:lts-alpine
			},
			engine: "copilot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "test-auto-container-*.md")
			require.NoError(t, err, "Failed to create temp file")
			defer os.Remove(tmpFile.Name())

			// Write content to file
			_, err = tmpFile.WriteString(tt.workflowContent)
			require.NoError(t, err, "Failed to write to temp file")
			tmpFile.Close()

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			compiler.SetSkipValidation(true) // Skip validation for test

			workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
			require.NoError(t, err, "Failed to parse workflow file")

			yamlContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
			require.NoError(t, err, "Failed to generate YAML")

			// Verify expected strings are present
			for _, expected := range tt.expectedStrings {
				assert.Contains(t, yamlContent, expected,
					"Expected string '%s' not found in generated YAML", expected)
			}

			// For explicit container test, verify node:lts-alpine is NOT present
			if strings.Contains(tt.name, "explicit container") {
				assert.NotContains(t, yamlContent, "node:lts-alpine",
					"Auto-assigned container should not be present when explicit container is specified")
			}
		})
	}
}

// TestAutoContainerizationWithMultipleServers tests that multiple MCP servers
// can have different auto-assigned containers in the same workflow
func TestAutoContainerizationWithMultipleServers(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network: defaults
tools:
  node-server:
    command: npx
    args: ["-y", "node-package"]
  python-server:
    command: uvx
    args: ["python-package"]
---

# Test Multiple Auto-Containerization

Test that multiple MCP servers get appropriate containers.
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-multi-container-*.md")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(workflowContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)

	workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
	require.NoError(t, err)

	yamlContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
	require.NoError(t, err)

	// Verify both containers are present
	assert.Contains(t, yamlContent, "node:lts-alpine",
		"Node Alpine container should be used for npx server")
	assert.Contains(t, yamlContent, "python:alpine",
		"Python Alpine container should be used for uvx server")

	// Verify both server configurations
	assert.Contains(t, yamlContent, `"node-server"`)
	assert.Contains(t, yamlContent, `"python-server"`)
}

// TestAutoContainerizationPreservesEnvironmentVars tests that environment
// variables are properly passed through to the auto-assigned container
func TestAutoContainerizationPreservesEnvironmentVars(t *testing.T) {
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network: defaults
tools:
  env-test:
    command: npx
    args: ["-y", "some-tool"]
    env:
      NODE_ENV: "production"
      API_KEY: "secret-value"
---

# Test Environment Variables

Test that environment variables are preserved with auto-containerization.
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-env-container-*.md")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(workflowContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)

	workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
	require.NoError(t, err)

	yamlContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
	require.NoError(t, err)

	// Verify environment variables are included in the docker command
	assert.Contains(t, yamlContent, `"-e"`,
		"Docker -e flag should be present for environment variables")
	assert.Contains(t, yamlContent, `"NODE_ENV"`,
		"NODE_ENV should be passed to container")
	assert.Contains(t, yamlContent, `"API_KEY"`,
		"API_KEY should be passed to container")
}
