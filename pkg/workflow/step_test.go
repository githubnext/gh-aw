package workflow

import (
	"strings"
	"testing"
)

func TestNewStepWithRun(t *testing.T) {
	step := NewStepWithRun("Run Test", "echo hello")

	if step.Name != "Run Test" {
		t.Errorf("Expected Name 'Run Test', got '%s'", step.Name)
	}

	if step.Run != "echo hello" {
		t.Errorf("Expected Run 'echo hello', got '%s'", step.Run)
	}
}

func TestNewStepWithUses(t *testing.T) {
	step := NewStepWithUses("Checkout", "actions/checkout@v4")

	if step.Name != "Checkout" {
		t.Errorf("Expected Name 'Checkout', got '%s'", step.Name)
	}

	if step.Uses != "actions/checkout@v4" {
		t.Errorf("Expected Uses 'actions/checkout@v4', got '%s'", step.Uses)
	}
}

func TestNewGitHubScriptStep(t *testing.T) {
	env := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	}

	step := NewGitHubScriptStep("Test Script", "console.log('test');", env)

	if step.Name != "Test Script" {
		t.Errorf("Expected Name 'Test Script', got '%s'", step.Name)
	}

	if step.Uses != "actions/github-script@v8" {
		t.Errorf("Expected Uses 'actions/github-script@v8', got '%s'", step.Uses)
	}

	if step.Env["FOO"] != "bar" {
		t.Errorf("Expected Env['FOO'] 'bar', got '%s'", step.Env["FOO"])
	}

	if step.With["script"] != "console.log('test');" {
		t.Errorf("Expected With['script'] to contain JavaScript")
	}
}

func TestStepSetters(t *testing.T) {
	step := NewStepWithRun("Test", "echo test")

	step.SetID("test-id")
	if step.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", step.ID)
	}

	step.SetIf("success()")
	if step.If != "success()" {
		t.Errorf("Expected If 'success()', got '%s'", step.If)
	}

	step.AddEnv("KEY", "value")
	if step.Env["KEY"] != "value" {
		t.Errorf("Expected Env['KEY'] 'value', got '%s'", step.Env["KEY"])
	}

	envMap := map[string]string{
		"KEY2": "value2",
		"KEY3": "value3",
	}
	step.AddEnvMap(envMap)
	if step.Env["KEY2"] != "value2" {
		t.Errorf("Expected Env['KEY2'] 'value2', got '%s'", step.Env["KEY2"])
	}

	step.AddWith("param", "value")
	if step.With["param"] != "value" {
		t.Errorf("Expected With['param'] 'value', got '%v'", step.With["param"])
	}

	step.SetGitHubToken("${{ secrets.TOKEN }}")
	if step.With["github-token"] != "${{ secrets.TOKEN }}" {
		t.Errorf("Expected github-token in With")
	}
}

func TestStepToYAML(t *testing.T) {
	step := NewStepWithRun("Test Step", "echo hello")
	step.SetID("test-id")

	yaml, err := step.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// Check for proper indentation (6 spaces)
	if !strings.Contains(yaml, "      - name: Test Step") {
		t.Errorf("Expected proper indentation in YAML output: %s", yaml)
	}

	// Check that fields are present
	if !strings.Contains(yaml, "name: Test Step") {
		t.Errorf("Expected 'name: Test Step' in YAML")
	}

	if !strings.Contains(yaml, "id: test-id") {
		t.Errorf("Expected 'id: test-id' in YAML")
	}

	if !strings.Contains(yaml, "run: echo hello") {
		t.Errorf("Expected 'run: echo hello' in YAML")
	}
}

func TestStepToYAMLFieldOrdering(t *testing.T) {
	// Create a step with all major fields to test ordering
	step := NewStepWithRun("Complex Step", "echo test")
	step.SetID("step-id")
	step.SetIf("success()")
	step.AddEnv("KEY", "value")
	step.AddWith("param", "value")

	yaml, err := step.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// Find positions of each field in the YAML output
	namePos := strings.Index(yaml, "name:")
	idPos := strings.Index(yaml, "id:")
	ifPos := strings.Index(yaml, "if:")
	runPos := strings.Index(yaml, "run:")
	envPos := strings.Index(yaml, "env:")
	withPos := strings.Index(yaml, "with:")

	// Verify field order: name, id, if, run, env, with
	if namePos == -1 || idPos == -1 || ifPos == -1 || runPos == -1 || envPos == -1 || withPos == -1 {
		t.Fatalf("Not all fields found in YAML: %s", yaml)
	}

	if !(namePos < idPos && idPos < ifPos && ifPos < runPos && runPos < envPos && envPos < withPos) {
		t.Errorf("Field order incorrect in YAML. Expected: name, id, if, run, env, with. Got:\n%s", yaml)
	}
}

func TestWriteStepsToString(t *testing.T) {
	step1 := NewStepWithRun("Step 1", "echo one")
	step2 := NewStepWithRun("Step 2", "echo two")

	yaml, err := WriteStepsToString(step1, step2)
	if err != nil {
		t.Fatalf("WriteStepsToString failed: %v", err)
	}

	if !strings.Contains(yaml, "Step 1") {
		t.Errorf("Expected 'Step 1' in output")
	}

	if !strings.Contains(yaml, "Step 2") {
		t.Errorf("Expected 'Step 2' in output")
	}

	if !strings.Contains(yaml, "echo one") {
		t.Errorf("Expected 'echo one' in output")
	}

	if !strings.Contains(yaml, "echo two") {
		t.Errorf("Expected 'echo two' in output")
	}
}

func TestStepToYAMLLines(t *testing.T) {
	step := NewStepWithRun("Test Step", "echo test")
	step.SetID("test-id")

	lines, err := StepToYAMLLines(step)
	if err != nil {
		t.Fatalf("StepToYAMLLines failed: %v", err)
	}

	if len(lines) == 0 {
		t.Fatalf("Expected non-empty lines")
	}

	// Join lines to check content
	yaml := strings.Join(lines, "\n")

	if !strings.Contains(yaml, "name: Test Step") {
		t.Errorf("Expected 'name: Test Step' in lines")
	}

	if !strings.Contains(yaml, "id: test-id") {
		t.Errorf("Expected 'id: test-id' in lines")
	}
}

func TestStepsToYAMLLines(t *testing.T) {
	step1 := NewStepWithRun("Step 1", "echo one")
	step2 := NewStepWithUses("Step 2", "actions/checkout@v4")

	steps, err := StepsToYAMLLines(step1, step2)
	if err != nil {
		t.Fatalf("StepsToYAMLLines failed: %v", err)
	}

	if len(steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(steps))
	}

	// Check first step
	step1YAML := strings.Join(steps[0], "\n")
	if !strings.Contains(step1YAML, "Step 1") {
		t.Errorf("Expected 'Step 1' in first step")
	}

	// Check second step
	step2YAML := strings.Join(steps[1], "\n")
	if !strings.Contains(step2YAML, "actions/checkout@v4") {
		t.Errorf("Expected 'actions/checkout@v4' in second step")
	}
}

func TestGitHubScriptStepWithMultilineScript(t *testing.T) {
	script := `const issue = context.issue;
console.log('Issue number:', issue.number);
return 'done';`

	env := map[string]string{
		"ISSUE_NUM": "${{ github.event.issue.number }}",
	}

	step := NewGitHubScriptStep("Test Script", script, env)

	yaml, err := step.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// Verify the script is included (may be literal block scalar or folded)
	if !strings.Contains(yaml, "script:") {
		t.Errorf("Expected 'script:' in YAML")
	}

	if !strings.Contains(yaml, "uses: actions/github-script@v8") {
		t.Errorf("Expected 'uses: actions/github-script@v8' in YAML")
	}

	if !strings.Contains(yaml, "ISSUE_NUM") {
		t.Errorf("Expected 'ISSUE_NUM' env var in YAML")
	}
}

func TestStepWithEmptyFields(t *testing.T) {
	// Test that empty fields are omitted from YAML output
	step := NewStepWithRun("Simple Step", "echo test")

	yaml, err := step.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// These fields should not be in the YAML
	if strings.Contains(yaml, "id:") {
		t.Errorf("Expected 'id' to be omitted from YAML")
	}

	if strings.Contains(yaml, "if:") {
		t.Errorf("Expected 'if' to be omitted from YAML")
	}

	if strings.Contains(yaml, "uses:") {
		t.Errorf("Expected 'uses' to be omitted from YAML")
	}

	if strings.Contains(yaml, "env:") {
		t.Errorf("Expected 'env' to be omitted from YAML")
	}

	if strings.Contains(yaml, "with:") {
		t.Errorf("Expected 'with' to be omitted from YAML")
	}

	// These fields should be present
	if !strings.Contains(yaml, "name: Simple Step") {
		t.Errorf("Expected 'name' to be in YAML")
	}

	if !strings.Contains(yaml, "run: echo test") {
		t.Errorf("Expected 'run' to be in YAML")
	}
}

func TestStepChaining(t *testing.T) {
	// Test that setter methods can be chained
	step := NewStepWithRun("Chained Step", "echo test").
		SetID("chain-id").
		SetIf("success()").
		AddEnv("KEY1", "value1").
		AddEnv("KEY2", "value2")

	if step.ID != "chain-id" {
		t.Errorf("Expected ID 'chain-id' after chaining")
	}

	if step.If != "success()" {
		t.Errorf("Expected If 'success()' after chaining")
	}

	if step.Env["KEY1"] != "value1" || step.Env["KEY2"] != "value2" {
		t.Errorf("Expected both env vars to be set after chaining")
	}
}

func TestBuildGitHubScriptStepLines(t *testing.T) {
	env := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	}

	withParams := map[string]string{
		"github-token": "${{ secrets.GITHUB_TOKEN }}",
	}

	lines := BuildGitHubScriptStepLines("Test Script", "test-id", "", env, withParams)

	// Join lines to check content
	yaml := strings.Join(lines, "")

	if !strings.Contains(yaml, "- name: Test Script") {
		t.Errorf("Expected 'name: Test Script' in output")
	}

	if !strings.Contains(yaml, "id: test-id") {
		t.Errorf("Expected 'id: test-id' in output")
	}

	if !strings.Contains(yaml, "uses: actions/github-script@v8") {
		t.Errorf("Expected 'uses: actions/github-script@v8' in output")
	}

	if !strings.Contains(yaml, "env:") {
		t.Errorf("Expected 'env:' section in output")
	}

	if !strings.Contains(yaml, "FOO: bar") {
		t.Errorf("Expected 'FOO: bar' in env section")
	}

	if !strings.Contains(yaml, "BAZ: qux") {
		t.Errorf("Expected 'BAZ: qux' in env section")
	}

	if !strings.Contains(yaml, "with:") {
		t.Errorf("Expected 'with:' section in output")
	}

	if !strings.Contains(yaml, "github-token:") {
		t.Errorf("Expected 'github-token' in with section")
	}

	if !strings.Contains(yaml, "script: |") {
		t.Errorf("Expected 'script: |' in with section")
	}
}

func TestBuildGitHubScriptStepLinesMinimal(t *testing.T) {
	// Test with minimal parameters (no env, no with params, no id)
	lines := BuildGitHubScriptStepLines("Minimal Script", "", "", nil, nil)

	yaml := strings.Join(lines, "")

	if !strings.Contains(yaml, "- name: Minimal Script") {
		t.Errorf("Expected 'name: Minimal Script' in output")
	}

	if strings.Contains(yaml, "id:") {
		t.Errorf("Expected no 'id:' field when id is empty")
	}

	if !strings.Contains(yaml, "uses: actions/github-script@v8") {
		t.Errorf("Expected 'uses: actions/github-script@v8' in output")
	}

	if strings.Contains(yaml, "env:") {
		t.Errorf("Expected no 'env:' section when env is nil")
	}

	if !strings.Contains(yaml, "with:") {
		t.Errorf("Expected 'with:' section in output")
	}

	if !strings.Contains(yaml, "script: |") {
		t.Errorf("Expected 'script: |' in with section")
	}
}

func TestBuildGitHubScriptStepLinesWithScript(t *testing.T) {
	// Test with script content included
	script := `const issue = context.issue;
console.log('Issue:', issue.number);
return 'done';`

	env := map[string]string{
		"ISSUE_NUM": "${{ github.event.issue.number }}",
	}

	lines := BuildGitHubScriptStepLines("Script With Content", "test-id", script, env, nil)

	yaml := strings.Join(lines, "")

	if !strings.Contains(yaml, "- name: Script With Content") {
		t.Errorf("Expected 'name: Script With Content' in output")
	}

	if !strings.Contains(yaml, "id: test-id") {
		t.Errorf("Expected 'id: test-id' in output")
	}

	if !strings.Contains(yaml, "script: |") {
		t.Errorf("Expected 'script: |' in with section")
	}

	// Check that script content is included and properly indented
	if !strings.Contains(yaml, "const issue = context.issue") {
		t.Errorf("Expected script content to be included")
	}

	if !strings.Contains(yaml, "console.log") {
		t.Errorf("Expected script content with console.log")
	}

	// Verify the script lines have proper indentation (12 spaces)
	if !strings.Contains(yaml, "            const issue") {
		t.Errorf("Expected script lines to have 12-space indentation")
	}
}

func TestAppendScriptLines(t *testing.T) {
	// Create initial step lines
	lines := []string{
		"      - name: Test\n",
		"        uses: actions/github-script@v8\n",
		"        with:\n",
		"          script: |\n",
	}

	script := `console.log('test');
return true;`

	// Append script lines
	lines = AppendScriptLines(lines, script)

	yaml := strings.Join(lines, "")

	if !strings.Contains(yaml, "console.log('test')") {
		t.Errorf("Expected script content to be appended")
	}

	if !strings.Contains(yaml, "return true") {
		t.Errorf("Expected script content with return statement")
	}

	// Verify indentation
	if !strings.Contains(yaml, "            console.log") {
		t.Errorf("Expected appended script to have proper indentation")
	}
}
