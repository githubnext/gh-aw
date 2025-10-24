package workflow

import (
	"strings"
	"testing"
)

func TestCopilotEngine_HTTPMCPWithHeaderSecrets_Integration(t *testing.T) {
	// Create workflow data with HTTP MCP tool using header secrets
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"datadog": map[string]any{
				"type": "http",
				"url":  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
				"headers": map[string]any{
					"DD_API_KEY":         "${{ secrets.DD_API_KEY }}",
					"DD_APPLICATION_KEY": "${{ secrets.DD_APPLICATION_KEY }}",
					"DD_SITE":            "${{ secrets.DD_SITE || 'datadoghq.com' }}",
				},
				"allowed": []string{
					"search_datadog_dashboards",
					"search_datadog_slos",
					"search_datadog_metrics",
					"get_datadog_metric",
				},
			},
		},
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
	}

	engine := NewCopilotEngine()

	// Test that RenderMCPConfig produces no output (it's now handled via --additional-mcp-config)
	var mcpConfig strings.Builder
	mcpTools := []string{"datadog"}
	engine.RenderMCPConfig(&mcpConfig, workflowData.Tools, mcpTools, workflowData)

	mcpOutput := mcpConfig.String()
	if mcpOutput != "" {
		t.Errorf("Expected RenderMCPConfig to produce no output, but got:\n%s", mcpOutput)
	}

	// Test execution steps to verify --additional-mcp-config contains the MCP config
	steps := engine.GetExecutionSteps(workflowData, "/tmp/log.txt")

	// Find the execution step
	var executionStepFound bool
	var stepContent string
	for _, step := range steps {
		stepStr := strings.Join(step, "\n")
		if strings.Contains(stepStr, "Execute GitHub Copilot CLI") {
			executionStepFound = true
			stepContent = stepStr
			break
		}
	}

	if !executionStepFound {
		t.Fatal("Could not find execution step")
	}

	// Verify --additional-mcp-config argument is present
	if !strings.Contains(stepContent, "--additional-mcp-config") {
		t.Errorf("Expected step to contain '--additional-mcp-config', but it didn't")
	}

	// Verify MCP config contains headers with env var references (not secret expressions)
	// JSON is now directly in the command (quoted), not base64-encoded
	expectedMCPChecks := []string{
		`"datadog": {`,
		`"type": "http"`,
		`"url": "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"`,
		`"headers": {`,
		`"DD_API_KEY": "\${DD_API_KEY}"`,
		`"DD_APPLICATION_KEY": "\${DD_APPLICATION_KEY}"`,
		`"DD_SITE": "\${DD_SITE}"`,
		`"tools": [`,
		`"search_datadog_dashboards"`,
		`"env": {`,
		`"DD_API_KEY": "\${DD_API_KEY}"`,
		`"DD_APPLICATION_KEY": "\${DD_APPLICATION_KEY}"`,
		`"DD_SITE": "\${DD_SITE}"`,
	}

	for _, expected := range expectedMCPChecks {
		if !strings.Contains(stepContent, expected) {
			t.Errorf("Expected MCP config content not found: %q\nStep content:\n%s", expected, stepContent)
		}
	}

	// Extract just the MCP config part (between --additional-mcp-config and the next --)
	mcpConfigStart := strings.Index(stepContent, "--additional-mcp-config")
	mcpConfigEnd := strings.Index(stepContent[mcpConfigStart:], "' --")
	if mcpConfigStart == -1 || mcpConfigEnd == -1 {
		t.Fatal("Could not extract MCP config from step content")
	}
	mcpConfigPart := stepContent[mcpConfigStart : mcpConfigStart+mcpConfigEnd]

	// Verify secret expressions are NOT in the MCP config portion (they should be in env section only)
	unexpectedMCPChecks := []string{
		`${{ secrets.DD_API_KEY }}`,
		`${{ secrets.DD_APPLICATION_KEY }}`,
		`${{ secrets.DD_SITE || 'datadoghq.com' }}`,
	}

	for _, unexpected := range unexpectedMCPChecks {
		if strings.Contains(mcpConfigPart, unexpected) {
			t.Errorf("Unexpected secret expression in MCP config: %q\nMCP config part:\n%s", unexpected, mcpConfigPart)
		}
	}

	// Verify env variables are declared with secret expressions (already have stepContent from above)
	expectedEnvChecks := []string{
		`DD_API_KEY: ${{ secrets.DD_API_KEY }}`,
		`DD_APPLICATION_KEY: ${{ secrets.DD_APPLICATION_KEY }}`,
		`DD_SITE: ${{ secrets.DD_SITE || 'datadoghq.com' }}`,
	}

	for _, expected := range expectedEnvChecks {
		if !strings.Contains(stepContent, expected) {
			t.Errorf("Expected env declaration not found: %q\nActual execution step:\n%s", expected, stepContent)
		}
	}
}

func TestCopilotEngine_MultipleHTTPMCPTools_Integration(t *testing.T) {
	// Create workflow data with multiple HTTP MCP tools
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"datadog": map[string]any{
				"type": "http",
				"url":  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
				"headers": map[string]any{
					"DD_API_KEY": "${{ secrets.DD_API_KEY }}",
				},
			},
			"custom": map[string]any{
				"type": "http",
				"url":  "https://api.custom.com/mcp",
				"headers": map[string]any{
					"CUSTOM_TOKEN": "${{ secrets.CUSTOM_TOKEN }}",
					"X-API-Key":    "${{ secrets.CUSTOM_API_KEY }}",
				},
			},
			"github": map[string]any{
				"allowed": []string{"get_repository"},
			},
		},
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
	}

	engine := NewCopilotEngine()

	// Test execution steps
	steps := engine.GetExecutionSteps(workflowData, "/tmp/log.txt")

	// Find the execution step
	var executionStepContent string
	for _, step := range steps {
		stepStr := strings.Join(step, "\n")
		if strings.Contains(stepStr, "Execute GitHub Copilot CLI") {
			executionStepContent = stepStr
			break
		}
	}

	if executionStepContent == "" {
		t.Fatal("Execution step not found")
	}

	// Verify all env variables from both tools are declared
	expectedEnvChecks := []string{
		`CUSTOM_API_KEY: ${{ secrets.CUSTOM_API_KEY }}`,
		`CUSTOM_TOKEN: ${{ secrets.CUSTOM_TOKEN }}`,
		`DD_API_KEY: ${{ secrets.DD_API_KEY }}`,
	}

	for _, expected := range expectedEnvChecks {
		if !strings.Contains(executionStepContent, expected) {
			t.Errorf("Expected env declaration not found: %q\nActual execution step:\n%s", expected, executionStepContent)
		}
	}

	// Verify env variables are sorted alphabetically
	ddIdx := strings.Index(executionStepContent, "DD_API_KEY:")
	customApiIdx := strings.Index(executionStepContent, "CUSTOM_API_KEY:")
	customTokenIdx := strings.Index(executionStepContent, "CUSTOM_TOKEN:")

	if !(customApiIdx < customTokenIdx && customTokenIdx < ddIdx) {
		t.Errorf("Env variables are not sorted alphabetically in execution step")
	}
}

func TestCopilotEngine_HTTPMCPWithoutSecrets_Integration(t *testing.T) {
	// Create workflow data with HTTP MCP tool without secrets
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"custom": map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
				"headers": map[string]any{
					"X-Static-Header": "static-value",
				},
			},
		},
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
	}

	engine := NewCopilotEngine()

	// Test that RenderMCPConfig produces no output
	var mcpConfig strings.Builder
	mcpTools := []string{"custom"}
	engine.RenderMCPConfig(&mcpConfig, workflowData.Tools, mcpTools, workflowData)

	mcpOutput := mcpConfig.String()
	if mcpOutput != "" {
		t.Errorf("Expected RenderMCPConfig to produce no output, but got:\n%s", mcpOutput)
	}

	// Test execution steps to verify --additional-mcp-config contains the MCP config
	steps := engine.GetExecutionSteps(workflowData, "/tmp/log.txt")

	var executionStepContent string
	for _, step := range steps {
		stepStr := strings.Join(step, "\n")
		if strings.Contains(stepStr, "Execute GitHub Copilot CLI") {
			executionStepContent = stepStr
			break
		}
	}

	if executionStepContent == "" {
		t.Fatal("Execution step not found")
	}

	// Verify static header is present (JSON is now directly in the command, not base64-encoded)
	if !strings.Contains(executionStepContent, `"X-Static-Header": "static-value"`) {
		t.Errorf("Expected static header not found in MCP config:\n%s", executionStepContent)
	}

	// Verify no env section is added when there are no secrets
	if strings.Contains(executionStepContent, `"custom": {`) && strings.Contains(executionStepContent, `"env": {`) {
		// Check if env section is specifically for the custom tool (after its opening brace)
		customIdx := strings.Index(executionStepContent, `"custom": {`)
		nextToolIdx := strings.Index(executionStepContent[customIdx+12:], `": {`)
		if nextToolIdx == -1 {
			nextToolIdx = len(executionStepContent) - customIdx - 12
		}
		customSection := executionStepContent[customIdx : customIdx+12+nextToolIdx]

		if strings.Contains(customSection, `"env": {`) {
			t.Errorf("Unexpected env section found in MCP config for tool without secrets:\n%s", executionStepContent)
		}
	}

	// Verify no header-related env variables are added
	unexpectedEnvChecks := []string{
		`X_STATIC_HEADER:`,
		`STATIC_VALUE:`,
	}

	for _, unexpected := range unexpectedEnvChecks {
		if strings.Contains(executionStepContent, unexpected) {
			t.Errorf("Unexpected env variable found: %q\nActual execution step:\n%s", unexpected, executionStepContent)
		}
	}
}
