package workflow

import (
	"encoding/base64"
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

	// Find the execution step and extract the base64-encoded MCP config
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

	// Extract and decode the base64 value
	parts := strings.Split(stepContent, "--additional-mcp-config")
	if len(parts) < 2 {
		t.Fatal("Could not find --additional-mcp-config argument")
	}

	afterFlag := strings.TrimSpace(parts[1])
	base64Value := strings.Fields(afterFlag)[0]

	decoded, err := base64.StdEncoding.DecodeString(base64Value)
	if err != nil {
		t.Fatalf("Failed to decode base64 MCP config: %v", err)
	}

	decodedStr := string(decoded)

	// Verify MCP config contains headers with env var references (not secret expressions)
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
		if !strings.Contains(decodedStr, expected) {
			t.Errorf("Expected MCP config content not found: %q\nActual MCP config:\n%s", expected, decodedStr)
		}
	}

	// Verify secret expressions are NOT in MCP config
	unexpectedMCPChecks := []string{
		`${{ secrets.DD_API_KEY }}`,
		`${{ secrets.DD_APPLICATION_KEY }}`,
		`${{ secrets.DD_SITE || 'datadoghq.com' }}`,
	}

	for _, unexpected := range unexpectedMCPChecks {
		if strings.Contains(decodedStr, unexpected) {
			t.Errorf("Unexpected secret expression in MCP config: %q\nActual MCP config:\n%s", unexpected, decodedStr)
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

	// Extract and decode the base64-encoded MCP config
	parts := strings.Split(executionStepContent, "--additional-mcp-config")
	if len(parts) < 2 {
		t.Fatal("Could not find --additional-mcp-config argument")
	}

	afterFlag := strings.TrimSpace(parts[1])
	base64Value := strings.Fields(afterFlag)[0]

	decoded, err := base64.StdEncoding.DecodeString(base64Value)
	if err != nil {
		t.Fatalf("Failed to decode base64 MCP config: %v", err)
	}

	decodedStr := string(decoded)

	// Verify static header is present
	if !strings.Contains(decodedStr, `"X-Static-Header": "static-value"`) {
		t.Errorf("Expected static header not found in MCP config:\n%s", decodedStr)
	}

	// Verify no env section is added when there are no secrets
	if strings.Contains(decodedStr, `"custom": {`) && strings.Contains(decodedStr, `"env": {`) {
		// Check if env section is specifically for the custom tool (after its opening brace)
		customIdx := strings.Index(decodedStr, `"custom": {`)
		nextToolIdx := strings.Index(decodedStr[customIdx+12:], `": {`)
		if nextToolIdx == -1 {
			nextToolIdx = len(decodedStr) - customIdx - 12
		}
		customSection := decodedStr[customIdx : customIdx+12+nextToolIdx]

		if strings.Contains(customSection, `"env": {`) {
			t.Errorf("Unexpected env section found in MCP config for tool without secrets:\n%s", decodedStr)
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
