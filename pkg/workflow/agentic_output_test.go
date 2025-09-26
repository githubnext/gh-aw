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

	// Verify GITHUB_AW_SAFE_OUTPUTS functionality (should be present for all engines)
	if !strings.Contains(lockContent, "- name: Setup agent output") {
		t.Error("Expected 'Setup agent output' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Ingest agent output") {
		t.Error("Expected 'Ingest agent output' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Upload agentic output file") {
		t.Error("Expected 'Upload agentic output file' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Upload sanitized agent output") {
		t.Error("Expected 'Upload sanitized agent output' step to be in generated workflow")
	}

	// Verify job output declaration for GITHUB_AW_SAFE_OUTPUTS
	if !strings.Contains(lockContent, "outputs:\n      output: ${{ steps.collect_output.outputs.output }}") {
		t.Error("Expected job output declaration for 'output'")
	}

	// Verify GITHUB_AW_SAFE_OUTPUTS is passed to Claude
	if !strings.Contains(lockContent, "GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS environment variable to be passed to engine")
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

	// Verify Print Agent output step has file existence check
	if !strings.Contains(lockContent, "if [ -f ${{ env.GITHUB_AW_SAFE_OUTPUTS }} ]; then") {
		t.Error("Expected Print Agent output step to check if output file exists before reading it")
	}

	if !strings.Contains(lockContent, "No agent output file found") {
		t.Error("Expected Print Agent output step to provide message when no output file found")
	}

	// Verify that both artifacts are uploaded
	if !strings.Contains(lockContent, fmt.Sprintf("name: %s", constants.SafeOutputArtifactName)) {
		t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS artifact name to be '%s'", constants.SafeOutputArtifactName)
	}

	t.Log("Claude workflow correctly includes both GITHUB_AW_SAFE_OUTPUTS and engine output collection")
}

func TestCodexEngineNoOutputSteps(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "codex-no-output-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with Codex engine (should have GITHUB_AW_SAFE_OUTPUTS but no engine output collection)
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

This workflow tests that Codex engine gets GITHUB_AW_SAFE_OUTPUTS but not engine output collection.
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

	// Verify that Codex workflow DOES have GITHUB_AW_SAFE_OUTPUTS functionality
	if !strings.Contains(lockContent, "- name: Setup agent output") {
		t.Error("Codex workflow should have 'Setup agent output' step (GITHUB_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "- name: Ingest agent output") {
		t.Error("Codex workflow should have 'Ingest agent output' step (GITHUB_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "- name: Upload agentic output file") {
		t.Error("Codex workflow should have 'Upload agentic output file' step (GITHUB_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "- name: Upload sanitized agent output") {
		t.Error("Codex workflow should have 'Upload sanitized agent output' step (GITHUB_AW_SAFE_OUTPUTS functionality)")
	}

	if !strings.Contains(lockContent, "GITHUB_AW_SAFE_OUTPUTS") {
		t.Error("Codex workflow should reference GITHUB_AW_SAFE_OUTPUTS environment variable")
	}

	if !strings.Contains(lockContent, fmt.Sprintf("name: %s", constants.SafeOutputArtifactName)) {
		t.Errorf("Codex workflow should reference %s artifact (GITHUB_AW_SAFE_OUTPUTS)", constants.SafeOutputArtifactName)
	}

	// Verify that job outputs section includes output for GITHUB_AW_SAFE_OUTPUTS
	if !strings.Contains(lockContent, "outputs:\n      output: ${{ steps.collect_output.outputs.output }}") {
		t.Error("Codex workflow should have job output declaration for 'output' (GITHUB_AW_SAFE_OUTPUTS)")
	}

	// Verify that Codex workflow does NOT have engine output collection steps
	if strings.Contains(lockContent, "- name: Collect engine output files") {
		t.Error("Codex workflow should NOT have 'Collect engine output files' step")
	}

	if strings.Contains(lockContent, "- name: Upload engine output files") {
		t.Error("Codex workflow should NOT have 'Upload engine output files' step")
	}

	if strings.Contains(lockContent, "name: agent_outputs") {
		t.Error("Codex workflow should NOT reference 'agent_outputs' artifact")
	}

	// Verify that the Codex execution step is still present
	if !strings.Contains(lockContent, "- name: Run Codex") {
		t.Error("Expected 'Run Codex' step to be in generated workflow")
	}

	t.Log("Codex workflow correctly includes GITHUB_AW_SAFE_OUTPUTS functionality but excludes engine output collection")
}

func TestEngineOutputFileDeclarations(t *testing.T) {
	// Test Claude engine declares no output files (Claude CLI no longer produces output.txt)
	claudeEngine := NewClaudeEngine()
	claudeOutputFiles := claudeEngine.GetDeclaredOutputFiles()

	if len(claudeOutputFiles) != 0 {
		t.Errorf("Claude engine should declare no output files (Claude CLI no longer produces output.txt), got: %v", claudeOutputFiles)
	}

	// Test Codex engine declares no output files
	codexEngine := NewCodexEngine()
	codexOutputFiles := codexEngine.GetDeclaredOutputFiles()

	if len(codexOutputFiles) != 0 {
		t.Errorf("Codex engine should declare no output files, got: %v", codexOutputFiles)
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

	// Create a test markdown file with Copilot engine (which declares /tmp/.copilot/logs/ as output file)
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

This workflow tests that /tmp/ files are excluded from cleanup.
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

	// Verify that the upload step includes the /tmp/ path (artifact should still be uploaded)
	if !strings.Contains(lockStr, "/tmp/.copilot/logs/") {
		t.Error("Expected upload artifact path to include '/tmp/.copilot/logs/' in generated workflow")
	}

	// Verify that the cleanup step does NOT include rm commands for /tmp/ paths
	if strings.Contains(lockStr, "rm -fr /tmp/.copilot/logs/") {
		t.Error("Cleanup step should NOT include 'rm -fr /tmp/.copilot/logs/' command")
	}

	// Verify that cleanup step does NOT exist when all files are in /tmp/
	if strings.Contains(lockStr, "- name: Clean up engine output files") {
		t.Error("Cleanup step should NOT be present when all output files are in /tmp/")
	}

	t.Log("Successfully verified that /tmp/ files are excluded from cleanup step while still being uploaded as artifacts")
}

func TestEngineOutputCleanupWithMixedPaths(t *testing.T) {
	// Test the cleanup logic directly with mixed paths to ensure proper filtering
	var yaml strings.Builder

	// Simulate mixed output files: some in /tmp/, some in workspace
	mockOutputFiles := []string{
		"/tmp/logs/debug.log",
		"workspace-output/results.txt",
		"/tmp/.cache/data.json",
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

	// Add cleanup step with the same logic as the actual implementation
	// Only emit cleanup step if there are workspace files to delete
	var workspaceFiles []string
	for _, file := range mockOutputFiles {
		if !strings.HasPrefix(file, "/tmp/") {
			workspaceFiles = append(workspaceFiles, file)
		}
	}

	// Only emit cleanup step if there are workspace files to delete
	if len(workspaceFiles) > 0 {
		yaml.WriteString("      - name: Clean up engine output files\n")
		yaml.WriteString("        run: |\n")
		for _, file := range workspaceFiles {
			yaml.WriteString("          rm -fr " + file + "\n")
		}
	}

	result := yaml.String()

	// Verify that all files are included in the upload step
	if !strings.Contains(result, "/tmp/logs/debug.log") {
		t.Error("Expected /tmp/logs/debug.log to be included in upload step")
	}
	if !strings.Contains(result, "workspace-output/results.txt") {
		t.Error("Expected workspace-output/results.txt to be included in upload step")
	}
	if !strings.Contains(result, "/tmp/.cache/data.json") {
		t.Error("Expected /tmp/.cache/data.json to be included in upload step")
	}
	if !strings.Contains(result, "build/artifacts.zip") {
		t.Error("Expected build/artifacts.zip to be included in upload step")
	}

	// Verify that only workspace files are included in cleanup step
	if strings.Contains(result, "rm -fr /tmp/logs/debug.log") {
		t.Error("Cleanup step should NOT include 'rm -fr /tmp/logs/debug.log' command")
	}
	if strings.Contains(result, "rm -fr /tmp/.cache/data.json") {
		t.Error("Cleanup step should NOT include 'rm -fr /tmp/.cache/data.json' command")
	}
	if !strings.Contains(result, "rm -fr workspace-output/results.txt") {
		t.Error("Cleanup step should include 'rm -fr workspace-output/results.txt' command")
	}
	if !strings.Contains(result, "rm -fr build/artifacts.zip") {
		t.Error("Cleanup step should include 'rm -fr build/artifacts.zip' command")
	}

	t.Log("Successfully verified that mixed path cleanup properly filters /tmp/ files")
}
