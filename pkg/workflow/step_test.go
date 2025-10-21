package workflow

import (
	"testing"

	"github.com/goccy/go-yaml"
)

func TestParseStepsFromFrontmatter_LegacyArrayFormat(t *testing.T) {
	// Test legacy array format (should go into PreAgent position)
	yamlContent := `
- name: Test Step
  run: echo "hello"
- name: Another Step
  uses: actions/checkout@v4
`

	var data []any
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	steps, err := ParseStepsFromFrontmatter(data)
	if err != nil {
		t.Fatalf("Failed to parse steps: %v", err)
	}

	if steps == nil {
		t.Fatal("Expected steps to be non-nil")
	}

	if len(steps.PreAgent) != 2 {
		t.Errorf("Expected 2 pre-agent steps, got %d", len(steps.PreAgent))
	}

	if len(steps.Pre) != 0 || len(steps.PostAgent) != 0 || len(steps.Post) != 0 {
		t.Error("Expected only pre-agent steps to be populated in legacy format")
	}

	// Verify first step
	if steps.PreAgent[0].Name != "Test Step" {
		t.Errorf("Expected first step name 'Test Step', got '%s'", steps.PreAgent[0].Name)
	}
	if steps.PreAgent[0].Run != "echo \"hello\"" {
		t.Errorf("Expected first step run 'echo \"hello\"', got '%s'", steps.PreAgent[0].Run)
	}

	// Verify second step
	if steps.PreAgent[1].Name != "Another Step" {
		t.Errorf("Expected second step name 'Another Step', got '%s'", steps.PreAgent[1].Name)
	}
	if steps.PreAgent[1].Uses != "actions/checkout@v4" {
		t.Errorf("Expected second step uses 'actions/checkout@v4', got '%s'", steps.PreAgent[1].Uses)
	}
}

func TestParseStepsFromFrontmatter_NewObjectFormat(t *testing.T) {
	// Test new object format with named positions
	yamlContent := `
pre:
  - name: Pre Step
    run: echo "pre"
pre-agent:
  - name: Pre Agent Step
    run: echo "pre-agent"
post-agent:
  - name: Post Agent Step
    run: echo "post-agent"
post:
  - name: Post Step
    run: echo "post"
`

	var data map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	steps, err := ParseStepsFromFrontmatter(data)
	if err != nil {
		t.Fatalf("Failed to parse steps: %v", err)
	}

	if steps == nil {
		t.Fatal("Expected steps to be non-nil")
	}

	// Verify all positions are populated
	if len(steps.Pre) != 1 {
		t.Errorf("Expected 1 pre step, got %d", len(steps.Pre))
	}
	if len(steps.PreAgent) != 1 {
		t.Errorf("Expected 1 pre-agent step, got %d", len(steps.PreAgent))
	}
	if len(steps.PostAgent) != 1 {
		t.Errorf("Expected 1 post-agent step, got %d", len(steps.PostAgent))
	}
	if len(steps.Post) != 1 {
		t.Errorf("Expected 1 post step, got %d", len(steps.Post))
	}

	// Verify step names
	if steps.Pre[0].Name != "Pre Step" {
		t.Errorf("Expected pre step name 'Pre Step', got '%s'", steps.Pre[0].Name)
	}
	if steps.PreAgent[0].Name != "Pre Agent Step" {
		t.Errorf("Expected pre-agent step name 'Pre Agent Step', got '%s'", steps.PreAgent[0].Name)
	}
	if steps.PostAgent[0].Name != "Post Agent Step" {
		t.Errorf("Expected post-agent step name 'Post Agent Step', got '%s'", steps.PostAgent[0].Name)
	}
	if steps.Post[0].Name != "Post Step" {
		t.Errorf("Expected post step name 'Post Step', got '%s'", steps.Post[0].Name)
	}
}

func TestParseStepsFromFrontmatter_PartialObjectFormat(t *testing.T) {
	// Test object format with only some positions defined
	yamlContent := `
pre-agent:
  - name: Pre Agent Step
    run: echo "pre-agent"
post-agent:
  - name: Post Agent Step
    run: echo "post-agent"
`

	var data map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	steps, err := ParseStepsFromFrontmatter(data)
	if err != nil {
		t.Fatalf("Failed to parse steps: %v", err)
	}

	if steps == nil {
		t.Fatal("Expected steps to be non-nil")
	}

	// Verify only specified positions are populated
	if len(steps.Pre) != 0 {
		t.Errorf("Expected 0 pre steps, got %d", len(steps.Pre))
	}
	if len(steps.PreAgent) != 1 {
		t.Errorf("Expected 1 pre-agent step, got %d", len(steps.PreAgent))
	}
	if len(steps.PostAgent) != 1 {
		t.Errorf("Expected 1 post-agent step, got %d", len(steps.PostAgent))
	}
	if len(steps.Post) != 0 {
		t.Errorf("Expected 0 post steps, got %d", len(steps.Post))
	}
}

func TestParseStepsFromFrontmatter_NilInput(t *testing.T) {
	steps, err := ParseStepsFromFrontmatter(nil)
	if err != nil {
		t.Fatalf("Expected no error for nil input, got: %v", err)
	}
	if steps != nil {
		t.Error("Expected nil steps for nil input")
	}
}

func TestMergeSteps(t *testing.T) {
	main := &WorkflowSteps{
		Pre: []Step{
			{Name: "Main Pre", Run: "echo main-pre"},
		},
		PreAgent: []Step{
			{Name: "Main PreAgent", Run: "echo main-pre-agent"},
		},
		PostAgent: []Step{
			{Name: "Main PostAgent", Run: "echo main-post-agent"},
		},
		Post: []Step{
			{Name: "Main Post", Run: "echo main-post"},
		},
	}

	imported := &WorkflowSteps{
		Pre: []Step{
			{Name: "Imported Pre", Run: "echo imported-pre"},
		},
		PreAgent: []Step{
			{Name: "Imported PreAgent", Run: "echo imported-pre-agent"},
		},
		PostAgent: []Step{
			{Name: "Imported PostAgent", Run: "echo imported-post-agent"},
		},
		Post: []Step{
			{Name: "Imported Post", Run: "echo imported-post"},
		},
	}

	merged := MergeSteps(main, imported)

	// Verify imported steps come first
	if len(merged.Pre) != 2 {
		t.Errorf("Expected 2 pre steps, got %d", len(merged.Pre))
	}
	if merged.Pre[0].Name != "Imported Pre" {
		t.Errorf("Expected first pre step to be imported, got '%s'", merged.Pre[0].Name)
	}
	if merged.Pre[1].Name != "Main Pre" {
		t.Errorf("Expected second pre step to be main, got '%s'", merged.Pre[1].Name)
	}

	// Verify same for other positions
	if len(merged.PreAgent) != 2 {
		t.Errorf("Expected 2 pre-agent steps, got %d", len(merged.PreAgent))
	}
	if len(merged.PostAgent) != 2 {
		t.Errorf("Expected 2 post-agent steps, got %d", len(merged.PostAgent))
	}
	if len(merged.Post) != 2 {
		t.Errorf("Expected 2 post steps, got %d", len(merged.Post))
	}
}

func TestMergeSteps_NilInputs(t *testing.T) {
	// Test nil main
	imported := &WorkflowSteps{
		PreAgent: []Step{{Name: "Test", Run: "echo test"}},
	}
	merged := MergeSteps(nil, imported)
	if merged != imported {
		t.Error("Expected imported steps when main is nil")
	}

	// Test nil imported
	main := &WorkflowSteps{
		PreAgent: []Step{{Name: "Test", Run: "echo test"}},
	}
	merged = MergeSteps(main, nil)
	if merged != main {
		t.Error("Expected main steps when imported is nil")
	}

	// Test both nil
	merged = MergeSteps(nil, nil)
	if merged != nil {
		t.Error("Expected nil when both inputs are nil")
	}
}

func TestWorkflowSteps_IsEmpty(t *testing.T) {
	// Test nil
	var steps *WorkflowSteps
	if !steps.IsEmpty() {
		t.Error("Expected nil steps to be empty")
	}

	// Test empty struct
	steps = &WorkflowSteps{}
	if !steps.IsEmpty() {
		t.Error("Expected empty struct to be empty")
	}

	// Test with steps
	steps = &WorkflowSteps{
		PreAgent: []Step{{Name: "Test", Run: "echo test"}},
	}
	if steps.IsEmpty() {
		t.Error("Expected non-empty steps")
	}
}

func TestStep_AllFields(t *testing.T) {
	// Test parsing a step with all fields
	yamlContent := `
id: test-step
name: Test Step
run: echo "test"
if: success()
continue-on-error: true
env:
  KEY1: value1
  KEY2: value2
working-directory: /tmp
shell: bash
timeout-minutes: 5
`

	var step Step
	if err := yaml.Unmarshal([]byte(yamlContent), &step); err != nil {
		t.Fatalf("Failed to unmarshal step: %v", err)
	}

	if step.ID != "test-step" {
		t.Errorf("Expected ID 'test-step', got '%s'", step.ID)
	}
	if step.Name != "Test Step" {
		t.Errorf("Expected name 'Test Step', got '%s'", step.Name)
	}
	if step.Run != "echo \"test\"" {
		t.Errorf("Expected run 'echo \"test\"', got '%s'", step.Run)
	}
	if step.If != "success()" {
		t.Errorf("Expected if 'success()', got '%s'", step.If)
	}
	if step.Continue != "true" {
		t.Errorf("Expected continue 'true', got '%s'", step.Continue)
	}
	if len(step.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(step.Env))
	}
	if step.WorkingDirectory != "/tmp" {
		t.Errorf("Expected working directory '/tmp', got '%s'", step.WorkingDirectory)
	}
	if step.Shell != "bash" {
		t.Errorf("Expected shell 'bash', got '%s'", step.Shell)
	}
	if step.TimeoutMinutes != 5 {
		t.Errorf("Expected timeout 5, got %d", step.TimeoutMinutes)
	}
}

func TestStep_UsesFormat(t *testing.T) {
	// Test parsing a step with uses and with fields
	yamlContent := `
name: Checkout
uses: actions/checkout@v4
with:
  fetch-depth: 0
  token: ${{ secrets.GITHUB_TOKEN }}
`

	var step Step
	if err := yaml.Unmarshal([]byte(yamlContent), &step); err != nil {
		t.Fatalf("Failed to unmarshal step: %v", err)
	}

	if step.Uses != "actions/checkout@v4" {
		t.Errorf("Expected uses 'actions/checkout@v4', got '%s'", step.Uses)
	}
	if len(step.With) != 2 {
		t.Errorf("Expected 2 with fields, got %d", len(step.With))
	}
	// fetch-depth should be present and be a number (YAML might parse as int or float)
	if _, ok := step.With["fetch-depth"]; !ok {
		t.Error("Expected fetch-depth to be present in with fields")
	}
}
