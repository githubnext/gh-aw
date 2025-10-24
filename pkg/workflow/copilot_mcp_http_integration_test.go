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

	// Test execution steps to verify --additional-mcp-config argument and env variables
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

	// Verify --additional-mcp-config argument is present
	if !strings.Contains(executionStepContent, "--additional-mcp-config") {
		t.Error("Expected --additional-mcp-config argument not found in execution step")
	}

	// Verify MCP config JSON contains headers with env var references (not secret expressions)
	// The JSON is passed as an argument, so check for the content within single quotes
	// Note: JSON.Marshal properly escapes backslashes, so \${VAR} becomes \\${VAR} in the JSON
	expectedMCPChecks := []string{
		`"datadog"`,
		`"type": "http"`,
		`"url": "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"`,
		`"headers"`,
		`"DD_API_KEY": "\\${DD_API_KEY}"`, // Double backslash in JSON due to proper escaping
		`"DD_APPLICATION_KEY": "\\${DD_APPLICATION_KEY}"`,
		`"DD_SITE": "\\${DD_SITE}"`,
		`"tools"`,
		`"search_datadog_dashboards"`,
		`"env"`,
	}

	for _, expected := range expectedMCPChecks {
		if !strings.Contains(executionStepContent, expected) {
			t.Errorf("Expected MCP config content not found: %q\nActual execution step:\n%s", expected, executionStepContent)
		}
	}

	// Verify secret expressions are NOT in the MCP config JSON (which is the --additional-mcp-config argument value)
	// The secret expressions should only be in the env: section at the end of the step
	// Extract just the --additional-mcp-config argument value
	configStartIdx := strings.Index(executionStepContent, "--additional-mcp-config")
	if configStartIdx == -1 {
		t.Fatal("--additional-mcp-config argument not found")
	}
	// Find the end of the JSON argument (look for the closing quote and next argument)
	configEndIdx := strings.Index(executionStepContent[configStartIdx:], "' --")
	if configEndIdx == -1 {
		configEndIdx = len(executionStepContent) - configStartIdx
	}
	mcpConfigSection := executionStepContent[configStartIdx : configStartIdx+configEndIdx]

	unexpectedMCPChecks := []string{
		`${{ secrets.DD_API_KEY }}`,
		`${{ secrets.DD_APPLICATION_KEY }}`,
	}

	for _, unexpected := range unexpectedMCPChecks {
		if strings.Contains(mcpConfigSection, unexpected) {
			t.Errorf("Unexpected secret expression in MCP config JSON: %q\nActual MCP config section:\n%s", unexpected, mcpConfigSection)
		}
	}

	// Verify env variables are declared with secret expressions
	expectedEnvChecks := []string{
		`DD_API_KEY: ${{ secrets.DD_API_KEY }}`,
		`DD_APPLICATION_KEY: ${{ secrets.DD_APPLICATION_KEY }}`,
		`DD_SITE: ${{ secrets.DD_SITE || 'datadoghq.com' }}`,
	}

	for _, expected := range expectedEnvChecks {
		if !strings.Contains(executionStepContent, expected) {
			t.Errorf("Expected env declaration not found: %q\nActual execution step:\n%s", expected, executionStepContent)
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

	// Verify --additional-mcp-config contains both tools
	if !strings.Contains(executionStepContent, "--additional-mcp-config") {
		t.Error("Expected --additional-mcp-config argument not found")
	}

	// Verify both tools are in the MCP config JSON
	expectedMCPChecks := []string{
		`"datadog"`,
		`"custom"`,
		`"github"`,
	}

	for _, expected := range expectedMCPChecks {
		if !strings.Contains(executionStepContent, expected) {
			t.Errorf("Expected MCP config content not found: %q", expected)
		}
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

	// Test execution steps
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

	// Verify --additional-mcp-config is present
	if !strings.Contains(executionStepContent, "--additional-mcp-config") {
		t.Error("Expected --additional-mcp-config argument not found")
	}

	// Verify static header is present in the MCP config JSON
	if !strings.Contains(executionStepContent, `"X-Static-Header": "static-value"`) {
		t.Errorf("Expected static header not found in MCP config")
	}

	// Verify no header-related env variables are added to the execution step
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
