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

	// Test execution steps to verify MCP config is in --additional-mcp-config
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

	// Verify --additional-mcp-config is present
	if !strings.Contains(executionStepContent, "--additional-mcp-config") {
		t.Error("Expected --additional-mcp-config flag in execution step")
	}

	// Verify the MCP config contains the datadog server with inlined secrets
	expectedMCPChecks := []string{
		`"datadog"`,
		`"type":"http"`,
		`"url":"https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"`,
		`"headers"`,
		`"DD_API_KEY":"${{ secrets.DD_API_KEY }}"`,
		`"DD_APPLICATION_KEY":"${{ secrets.DD_APPLICATION_KEY }}"`,
		`"DD_SITE":"${{ secrets.DD_SITE || 'datadoghq.com' }}"`,
	}

	for _, expected := range expectedMCPChecks {
		if !strings.Contains(executionStepContent, expected) {
			t.Errorf("Expected MCP config content not found in execution step: %q", expected)
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

	// Verify --additional-mcp-config is present
	if !strings.Contains(executionStepContent, "--additional-mcp-config") {
		t.Error("Expected --additional-mcp-config flag in execution step")
	}

	// Verify all tools are in the MCP config with their inlined secrets
	expectedMCPChecks := []string{
		`"custom"`,
		`"CUSTOM_TOKEN":"${{ secrets.CUSTOM_TOKEN }}"`,
		`"X-API-Key":"${{ secrets.CUSTOM_API_KEY }}"`,
		`"datadog"`,
		`"DD_API_KEY":"${{ secrets.DD_API_KEY }}"`,
		`"github"`,
	}

	for _, expected := range expectedMCPChecks {
		if !strings.Contains(executionStepContent, expected) {
			t.Errorf("Expected MCP config content not found in execution step: %q", expected)
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

	// Verify static header is present in --additional-mcp-config
	if !strings.Contains(executionStepContent, `"X-Static-Header":"static-value"`) {
		t.Errorf("Expected static header not found in MCP config")
	}
}
