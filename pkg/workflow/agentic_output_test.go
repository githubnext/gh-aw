package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestAgenticOutputCollection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "agentic-output-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with agentic output collection for Claude engine
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: claude
safe-outputs:
  add-labels:
    allowed: ["bug", "enhancement"]
---

# Test Agentic Output Collection

This workflow tests the agentic output collection functionality.
`

	testFile := filepath.Join(tmpDir, "test-agentic-output.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with agentic output: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-agentic-output.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify GH_AW_SAFE_OUTPUTS is set at job level with fixed path
	if !strings.Contains(lockContent, "GH_AW_SAFE_OUTPUTS: /tmp/gh-aw/safeoutputs/outputs.jsonl") {
		t.Error("Expected 'GH_AW_SAFE_OUTPUTS: /tmp/gh-aw/safeoutputs/outputs.jsonl' environment variable in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Ingest agent output") {
		t.Error("Expected 'Ingest agent output' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Upload Safe Outputs") {
		t.Error("Expected 'Upload Safe Outputs' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Upload sanitized agent output") {
		t.Error("Expected 'Upload sanitized agent output' step to be in generated workflow")
	}

	// Verify job output declaration for GH_AW_SAFE_OUTPUTS
	if !strings.Contains(lockContent, "outputs:\n      output: ${{ steps.collect_output.outputs.output }}") {
		t.Error("Expected job output declaration for 'output'")
	}

	// Verify GH_AW_SAFE_OUTPUTS is passed to Claude
	if !strings.Contains(lockContent, "GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}") {
		t.Error("Expected GH_AW_SAFE_OUTPUTS environment variable to be passed to engine")
	}

	// Verify prompt contains output instructions
	if !strings.Contains(lockContent, "## Adding Labels to Issues or Pull Requests") {
		t.Error("Expected output instructions to be injected into prompt")
	}

	// Verify Claude engine no longer has upload steps (Claude CLI no longer produces output.txt)
	if strings.Contains(lockContent, "- name: Upload engine output files") {
		t.Error("Claude workflow should NOT have 'Upload engine output files' step (Claude CLI no longer produces output.txt)")
	}

	if strings.Contains(lockContent, "name: agent_outputs") {
		t.Error("Claude workflow should NOT reference 'agent_outputs' artifact (Claude CLI no longer produces output.txt)")
	}

	// Verify that both artifacts are uploaded
	if !strings.Contains(lockContent, fmt.Sprintf("name: %s", constants.SafeOutputArtifactName)) {
		t.Errorf("Expected GH_AW_SAFE_OUTPUTS artifact name to be '%s'", constants.SafeOutputArtifactName)
	}

	t.Log("Claude workflow correctly includes both GH_AW_SAFE_OUTPUTS and engine output collection")
}

func TestCodexEngineWithOutputSteps(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "codex-no-output-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with Codex engine (should have GH_AW_SAFE_OUTPUTS but no engine output collection)
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: codex
safe-outputs:
  add-labels:
    allowed: ["bug", "enhancement"]
---

# Test Codex No Engine Output Collection

This workflow tests that Codex engine gets GH_AW_SAFE_OUTPUTS but not engine output collection.
`

	testFile := filepath.Join(tmpDir, "test-codex-no-output.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with Codex: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-codex-no-output.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify that Codex workflow DOES have GH_AW_SAFE_OUTPUTS functionality at job level
	if !strings.Contains(lockContent, "GH_AW_SAFE_OUTPUTS: /tmp/gh-aw/safeoutputs/outputs.jsonl") {
		t.Error("Codex workflow should have 'GH_AW_SAFE_OUTPUTS: /tmp/gh-aw/safeoutputs/outputs.jsonl' environment variable (GH_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "- name: Ingest agent output") {
		t.Error("Codex workflow should have 'Ingest agent output' step (GH_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "- name: Upload Safe Outputs") {
		t.Error("Codex workflow should have 'Upload Safe Outputs' step (GH_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "- name: Upload sanitized agent output") {
		t.Error("Codex workflow should have 'Upload sanitized agent output' step (GH_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "GH_AW_SAFE_OUTPUTS") {
		t.Error("Codex workflow should reference GH_AW_SAFE_OUTPUTS environment variable")
	}

	if !strings.Contains(lockContent, fmt.Sprintf("name: %s", constants.SafeOutputArtifactName)) {
		t.Errorf("Codex workflow should reference %s artifact (GH_AW_SAFE_OUTPUTS)", constants.SafeOutputArtifactName)
	}

	// Verify that job outputs section includes output for GH_AW_SAFE_OUTPUTS
	if !strings.Contains(lockContent, "outputs:\n      output: ${{ steps.collect_output.outputs.output }}") {
		t.Error("Codex workflow should have job output declaration for 'output' (GH_AW_SAFE_OUTPUTS)")
	}

	// Verify that Codex workflow DOES have engine output collection steps
	// (because GetDeclaredOutputFiles returns a non-empty list)
	if !strings.Contains(lockContent, "- name: Upload engine output files") {
		t.Error("Codex workflow should have 'Upload engine output files' step")
	}

	if !strings.Contains(lockContent, "name: agent_outputs") {
		t.Error("Codex workflow should reference 'agent_outputs' artifact")
	}

	// Verify that the Codex execution step is still present
	if !strings.Contains(lockContent, "- name: Run Codex") {
		t.Error("Expected 'Run Codex' step to be in generated workflow")
	}

	t.Log("Codex workflow correctly includes both GH_AW_SAFE_OUTPUTS functionality and engine output collection")
}

func TestEngineOutputFileDeclarations(t *testing.T) {
	// Test Claude engine declares no output files (Claude CLI no longer produces output.txt)
	claudeEngine := NewClaudeEngine()
	claudeOutputFiles := claudeEngine.GetDeclaredOutputFiles()

	if len(claudeOutputFiles) != 0 {
		t.Errorf("Claude engine should declare no output files (Claude CLI no longer produces output.txt), got: %v", claudeOutputFiles)
	}

	// Test Codex engine declares output files for log collection
	codexEngine := NewCodexEngine()
	codexOutputFiles := codexEngine.GetDeclaredOutputFiles()

	if len(codexOutputFiles) == 0 {
		t.Errorf("Codex engine should declare output files for log collection, got: %v", codexOutputFiles)
	}

	if len(codexOutputFiles) > 0 && codexOutputFiles[0] != "/tmp/gh-aw/mcp-config/logs/" {
		t.Errorf("Codex engine should declare /tmp/gh-aw/mcp-config/logs/, got: %v", codexOutputFiles[0])
	}

	t.Logf("Claude engine declares: %v", claudeOutputFiles)
	t.Logf("Codex engine declares: %v", codexOutputFiles)
}

func TestEngineOutputCleanupExcludesTmpFiles(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "engine-output-cleanup-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with Copilot engine (which declares /tmp/gh-aw/.h-aw/.copilot/logs/ as output file)
	testContent := `---
on: push
permissions:
  contents: read
tools:
  github:
    allowed: [list_issues]
engine: copilot
---

# Test Engine Output Cleanup

This workflow tests that /tmp/gh-aw/ h-aw/ files are excluded from cleanup.
`

	testFile := filepath.Join(tmpDir, "test-engine-output-cleanup.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify that the upload step includes the /tmp/gh-aw/ path (artifact should still be uploaded)
	if !strings.Contains(lockStr, "/tmp/gh-aw/.copilot/logs/") {
		t.Error("Expected upload artifact path to include '/tmp/gh-aw/.copilot/logs/' in generated workflow")
	}

	// Verify that the cleanup step does NOT include rm commands for /tmp/gh-aw/ paths
	if strings.Contains(lockStr, "rm -fr /tmp/gh-aw/.copilot/logs/") {
		t.Error("Cleanup step should NOT include 'rm -fr /tmp/gh-aw/.copilot/logs/' command")
	}

	// Verify that cleanup step does NOT exist when all files are in /tmp/gh-aw/
	if strings.Contains(lockStr, "- name: Clean up engine output files") {
		t.Error("Cleanup step should NOT be present when all output files are in /tmp/gh-aw/")
	}

	t.Log("Successfully verified that /tmp/gh-aw/ files are excluded from cleanup step while still being uploaded as artifacts")
}

func TestClaudeEngineNetworkHookCleanup(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("Network hook cleanup with Claude engine and network permissions", func(t *testing.T) {
		// Test data with Claude engine and network permissions
		data := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"example.com", "*.trusted.com"},
			},
		}

		steps := engine.GetExecutionSteps(data, "/tmp/gh-aw/th-aw/test.log")

		// Convert all steps to string for analysis
		var allStepsStr strings.Builder
		for _, step := range steps {
			allStepsStr.WriteString(strings.Join(step, "\n"))
			allStepsStr.WriteString("\n")
		}
		result := allStepsStr.String()

		// Verify cleanup step is generated
		if !strings.Contains(result, "- name: Clean up network proxy hook files") {
			t.Error("Expected cleanup step to be generated with Claude engine and network permissions")
		}

		// Verify if: always() condition
		if !strings.Contains(result, "if: always()") {
			t.Error("Expected cleanup step to have 'if: always()' condition")
		}

		// Verify cleanup commands
		if !strings.Contains(result, "rm -rf .claude/hooks/network_permissions.py || true") {
			t.Error("Expected cleanup step to remove network_permissions.py")
		}

		if !strings.Contains(result, "rm -rf .claude/hooks || true") {
			t.Error("Expected cleanup step to remove hooks directory")
		}

		if !strings.Contains(result, "rm -rf .claude || true") {
			t.Error("Expected cleanup step to remove .claude directory")
		}
	})

	t.Run("Cleanup with Claude engine and defaults network permissions", func(t *testing.T) {
		// Test data with Claude engine and defaults network permissions
		// (This simulates what happens when no network section is specified - defaults to "defaults" mode)
		data := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
			NetworkPermissions: &NetworkPermissions{
				Mode: "defaults", // Default network mode
			},
		}

		steps := engine.GetExecutionSteps(data, "/tmp/gh-aw/th-aw/test.log")

		// Convert all steps to string for analysis
		var allStepsStr strings.Builder
		for _, step := range steps {
			allStepsStr.WriteString(strings.Join(step, "\n"))
			allStepsStr.WriteString("\n")
		}
		result := allStepsStr.String()

		// Verify cleanup step is generated for defaults mode
		if !strings.Contains(result, "- name: Clean up network proxy hook files") {
			t.Error("Expected cleanup step to be generated with defaults network permissions")
		}
	})

	t.Run("No cleanup with Claude engine but no network permissions", func(t *testing.T) {
		// Test data with Claude engine but no network permissions
		data := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
			NetworkPermissions: nil, // No network permissions
		}

		steps := engine.GetExecutionSteps(data, "/tmp/gh-aw/th-aw/test.log")

		// Convert all steps to string for analysis
		var allStepsStr strings.Builder
		for _, step := range steps {
			allStepsStr.WriteString(strings.Join(step, "\n"))
			allStepsStr.WriteString("\n")
		}
		result := allStepsStr.String()

		// Verify no cleanup step is generated
		if strings.Contains(result, "- name: Clean up network proxy hook files") {
			t.Error("Expected no cleanup step to be generated without network permissions")
		}
	})

	t.Run("Cleanup with empty network permissions (deny-all)", func(t *testing.T) {
		// Test data with Claude engine and empty network permissions (deny-all)
		data := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID:    "claude",
				Model: "claude-3-5-sonnet-20241022",
			},
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{}, // Empty allowed list (deny-all, but still uses hooks)
			},
		}

		steps := engine.GetExecutionSteps(data, "/tmp/gh-aw/th-aw/test.log")

		// Convert all steps to string for analysis
		var allStepsStr strings.Builder
		for _, step := range steps {
			allStepsStr.WriteString(strings.Join(step, "\n"))
			allStepsStr.WriteString("\n")
		}
		result := allStepsStr.String()

		// Verify cleanup step is generated even for deny-all policy
		// because hooks are still created for deny-all enforcement
		if !strings.Contains(result, "- name: Clean up network proxy hook files") {
			t.Error("Expected cleanup step to be generated even with deny-all network permissions")
		}
	})
}

func TestEngineOutputCleanupWithMixedPaths(t *testing.T) {
	// Test the cleanup logic directly with mixed paths to ensure proper filtering
	var yaml strings.Builder

	// Simulate mixed output files: some in /tmp/gh-aw/, some in workspace
	mockOutputFiles := []string{
		"/tmp/gh-aw/logs/debug.log",
		"workspace-output/results.txt",
		"/tmp/gh-aw/.cache/data.json",
		"build/artifacts.zip",
	}

	// Generate the engine output collection manually to test the logic
	yaml.WriteString("      - name: Upload engine output files\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: agent_outputs\n")
	yaml.WriteString("          path: |\n")
	for _, file := range mockOutputFiles {
		yaml.WriteString("            " + file + "\n")
	}
	yaml.WriteString("          if-no-files-found: ignore\n")

	// Add cleanup step using the same function as the actual implementation
	cleanupYaml, hasCleanup := generateCleanupStep(mockOutputFiles)
	if hasCleanup {
		yaml.WriteString(cleanupYaml)
	}

	result := yaml.String()

	// Verify that all files are included in the upload step
	if !strings.Contains(result, "/tmp/gh-aw/logs/debug.log") {
		t.Error("Expected /tmp/gh-aw/logs/debug.log to be included in upload step")
	}
	if !strings.Contains(result, "workspace-output/results.txt") {
		t.Error("Expected workspace-output/results.txt to be included in upload step")
	}
	if !strings.Contains(result, "/tmp/gh-aw/.cache/data.json") {
		t.Error("Expected /tmp/gh-aw/.cache/data.json to be included in upload step")
	}
	if !strings.Contains(result, "build/artifacts.zip") {
		t.Error("Expected build/artifacts.zip to be included in upload step")
	}

	// Verify that only workspace files are included in cleanup step
	if strings.Contains(result, "rm -fr /tmp/gh-aw/logs/debug.log") {
		t.Error("Cleanup step should NOT include 'rm -fr /tmp/gh-aw/logs/debug.log' command")
	}
	if strings.Contains(result, "rm -fr /tmp/gh-aw/.cache/data.json") {
		t.Error("Cleanup step should NOT include 'rm -fr /tmp/gh-aw/.cache/data.json' command")
	}
	if !strings.Contains(result, "rm -fr workspace-output/results.txt") {
		t.Error("Cleanup step should include 'rm -fr workspace-output/results.txt' command")
	}
	if !strings.Contains(result, "rm -fr build/artifacts.zip") {
		t.Error("Cleanup step should include 'rm -fr build/artifacts.zip' command")
	}

	t.Log("Successfully verified that mixed path cleanup properly filters /tmp/gh-aw/ files")
}

func TestGenerateCleanupStep(t *testing.T) {
	// Test the generateCleanupStep function directly to demonstrate its testability

	// Test case 1: Only /tmp/gh-aw/ files - should not generate cleanup step
	tmpOnlyFiles := []string{"/tmp/gh-aw/logs/debug.log", "/tmp/gh-aw/.cache/data.json"}
	cleanupYaml, hasCleanup := generateCleanupStep(tmpOnlyFiles)

	if hasCleanup {
		t.Error("Expected no cleanup step for /tmp/gh-aw/ only files")
	}
	if cleanupYaml != "" {
		t.Error("Expected empty cleanup YAML for /tmp/gh-aw/ only files")
	}

	// Test case 2: Only workspace files - should generate cleanup step
	workspaceOnlyFiles := []string{"output.txt", "build/artifacts.zip"}
	cleanupYaml, hasCleanup = generateCleanupStep(workspaceOnlyFiles)

	if !hasCleanup {
		t.Error("Expected cleanup step for workspace files")
	}
	if !strings.Contains(cleanupYaml, "rm -fr output.txt") {
		t.Error("Expected cleanup YAML to contain 'rm -fr output.txt'")
	}
	if !strings.Contains(cleanupYaml, "rm -fr build/artifacts.zip") {
		t.Error("Expected cleanup YAML to contain 'rm -fr build/artifacts.zip'")
	}

	// Test case 3: Mixed files - should generate cleanup step only for workspace files
	mixedFiles := []string{"/tmp/gh-aw/debug.log", "workspace/output.txt", "/tmp/gh-aw/.cache/data.json"}
	cleanupYaml, hasCleanup = generateCleanupStep(mixedFiles)

	if !hasCleanup {
		t.Error("Expected cleanup step for mixed files containing workspace files")
	}
	if strings.Contains(cleanupYaml, "rm -fr /tmp/gh-aw/debug.log") {
		t.Error("Cleanup YAML should NOT contain /tmp/gh-aw/ files")
	}
	if strings.Contains(cleanupYaml, "rm -fr /tmp/gh-aw/.cache/data.json") {
		t.Error("Cleanup YAML should NOT contain /tmp/gh-aw/ files")
	}
	if !strings.Contains(cleanupYaml, "rm -fr workspace/output.txt") {
		t.Error("Expected cleanup YAML to contain workspace files")
	}

	// Test case 4: Empty input - should not generate cleanup step
	emptyFiles := []string{}
	cleanupYaml, hasCleanup = generateCleanupStep(emptyFiles)

	if hasCleanup {
		t.Error("Expected no cleanup step for empty files list")
	}
	if cleanupYaml != "" {
		t.Error("Expected empty cleanup YAML for empty files list")
	}

	t.Log("Successfully verified generateCleanupStep function behavior in all scenarios")
}
