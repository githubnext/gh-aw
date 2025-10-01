package workflow

import (
	"strings"
	"testing"
)

func TestGitHubModelsEngine(t *testing.T) {
	engine := NewGitHubModelsEngine()

	if engine.GetID() != "github-models" {
		t.Errorf("Expected engine ID 'github-models', got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "GitHub Models" {
		t.Errorf("Expected display name 'GitHub Models', got '%s'", engine.GetDisplayName())
	}

	if engine.IsExperimental() {
		t.Error("Expected GitHub Models engine to not be experimental")
	}

	if !engine.SupportsToolsAllowlist() {
		t.Error("Expected GitHub Models engine to support tools allowlist")
	}

	if !engine.SupportsHTTPTransport() {
		t.Error("Expected GitHub Models engine to support HTTP transport")
	}

	if engine.SupportsMaxTurns() {
		t.Error("Expected GitHub Models engine to not support max-turns")
	}
}

func TestGitHubModelsEngineGetInstallationSteps(t *testing.T) {
	engine := NewGitHubModelsEngine()

	steps := engine.GetInstallationSteps(&WorkflowData{})
	if len(steps) != 0 {
		t.Errorf("Expected 0 installation steps for GitHub Models engine, got %d", len(steps))
	}
}

func TestGitHubModelsEngineGetExecutionSteps(t *testing.T) {
	engine := NewGitHubModelsEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Should have at least 2 steps: inference and process response
	if len(steps) < 2 {
		t.Errorf("Expected at least 2 steps, got %d", len(steps))
	}

	// Convert steps to string for checking
	allSteps := ""
	for _, step := range steps {
		allSteps += strings.Join(step, "\n") + "\n"
	}

	// Check for key elements in the generated steps
	if !strings.Contains(allSteps, "actions/ai-inference@v1") {
		t.Error("Expected steps to contain 'actions/ai-inference@v1'")
	}

	if !strings.Contains(allSteps, "prompt-file: '/tmp/aw-prompts/prompt.txt'") {
		t.Error("Expected steps to contain prompt-file parameter")
	}

	if !strings.Contains(allSteps, "Process AI Response") {
		t.Error("Expected steps to contain 'Process AI Response' step")
	}

	if !strings.Contains(allSteps, "steps.inference.outputs.response-file") {
		t.Error("Expected steps to reference response-file output")
	}
}

func TestGitHubModelsEngineGetExecutionStepsWithModel(t *testing.T) {
	engine := NewGitHubModelsEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:    "github-models",
			Model: "gpt-4o",
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Convert steps to string
	allSteps := ""
	for _, step := range steps {
		allSteps += strings.Join(step, "\n") + "\n"
	}

	// Check for model parameter
	if !strings.Contains(allSteps, "model: 'gpt-4o'") {
		t.Error("Expected steps to contain model parameter 'gpt-4o'")
	}
}

func TestGitHubModelsEngineGetExecutionStepsWithGitHubTool(t *testing.T) {
	engine := NewGitHubModelsEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{
				"allowed": []string{"get_repository"},
			},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Convert steps to string
	allSteps := ""
	for _, step := range steps {
		allSteps += strings.Join(step, "\n") + "\n"
	}

	// Check for GitHub MCP configuration
	if !strings.Contains(allSteps, "enable-github-mcp: true") {
		t.Error("Expected steps to contain 'enable-github-mcp: true'")
	}

	if !strings.Contains(allSteps, "github-mcp-token: ${{ secrets.GITHUB_MCP_TOKEN }}") {
		t.Error("Expected steps to contain github-mcp-token parameter")
	}
}

func TestGitHubModelsEngineGetExecutionStepsWithCustomVersion(t *testing.T) {
	engine := NewGitHubModelsEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:      "github-models",
			Version: "v1.2",
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Convert steps to string
	allSteps := ""
	for _, step := range steps {
		allSteps += strings.Join(step, "\n") + "\n"
	}

	// Check for custom version
	if !strings.Contains(allSteps, "actions/ai-inference@v1.2") {
		t.Error("Expected steps to contain 'actions/ai-inference@v1.2'")
	}
}

func TestGitHubModelsEngineParseLogMetrics(t *testing.T) {
	engine := NewGitHubModelsEngine()

	logContent := `This is a test response
Error: Something went wrong
Warning: This is a warning
Some normal content`

	metrics := engine.ParseLogMetrics(logContent, false)

	if metrics.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", metrics.ErrorCount)
	}

	if metrics.WarningCount != 1 {
		t.Errorf("Expected 1 warning, got %d", metrics.WarningCount)
	}
}

func TestGitHubModelsEngineGetLogParserScriptId(t *testing.T) {
	engine := NewGitHubModelsEngine()

	scriptId := engine.GetLogParserScriptId()
	if scriptId != "parse_github_models_log" {
		t.Errorf("Expected script ID 'parse_github_models_log', got '%s'", scriptId)
	}
}

func TestGitHubModelsEngineGetVersionCommand(t *testing.T) {
	engine := NewGitHubModelsEngine()

	versionCmd := engine.GetVersionCommand()
	if versionCmd != "" {
		t.Errorf("Expected empty version command, got '%s'", versionCmd)
	}
}

func TestGitHubModelsEngineGetDeclaredOutputFiles(t *testing.T) {
	engine := NewGitHubModelsEngine()

	outputFiles := engine.GetDeclaredOutputFiles()
	if len(outputFiles) != 0 {
		t.Errorf("Expected 0 output files, got %d", len(outputFiles))
	}
}

func TestGitHubModelsEngineRenderMCPConfig(t *testing.T) {
	engine := NewGitHubModelsEngine()

	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, map[string]any{}, []string{}, &WorkflowData{})

	// GitHub Models handles MCP via action parameters, so this should be empty
	result := yaml.String()
	if result != "" {
		t.Errorf("Expected empty MCP config, got: %s", result)
	}
}

func TestGitHubModelsEngineGetErrorPatterns(t *testing.T) {
	engine := NewGitHubModelsEngine()

	patterns := engine.GetErrorPatterns()
	if len(patterns) == 0 {
		t.Error("Expected some error patterns, got none")
	}

	// Verify at least one pattern exists for errors
	foundErrorPattern := false
	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(pattern.Description), "error") {
			foundErrorPattern = true
			break
		}
	}

	if !foundErrorPattern {
		t.Error("Expected to find at least one error pattern")
	}
}
