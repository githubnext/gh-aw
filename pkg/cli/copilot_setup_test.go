//go:build !integration

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"

	"github.com/goccy/go-yaml"
)

// mockSHAResolver is a test double for workflow.ActionSHAResolver that returns a fixed SHA
type mockSHAResolver struct {
	sha string
	err error
}

func (m *mockSHAResolver) ResolveSHA(_, _ string) (string, error) {
	return m.sha, m.err
}
func TestEnsureCopilotSetupSteps(t *testing.T) {
	tests := []struct {
		name             string
		existingWorkflow *Workflow
		verbose          bool
		wantErr          bool
		validateContent  func(*testing.T, []byte)
	}{
		{
			name:    "creates new copilot-setup-steps.yml",
			verbose: false,
			wantErr: false,
			validateContent: func(t *testing.T, content []byte) {
				if !strings.Contains(string(content), "copilot-setup-steps") {
					t.Error("Expected workflow to contain 'copilot-setup-steps' job name")
				}
				if !strings.Contains(string(content), "install-gh-aw.sh") {
					t.Error("Expected workflow to contain install-gh-aw.sh bash script")
				}
				if !strings.Contains(string(content), "curl -fsSL") {
					t.Error("Expected workflow to contain curl command")
				}
			},
		},
		{
			name: "skips update when extension install already exists",
			existingWorkflow: &Workflow{
				Name: "Copilot Setup Steps",
				On:   "workflow_dispatch",
				Jobs: map[string]WorkflowJob{
					"copilot-setup-steps": {
						RunsOn: "ubuntu-latest",
						Steps: []CopilotWorkflowStep{
							{
								Name: "Checkout code",
								Uses: "actions/checkout@v5",
							},
							{
								Name: "Install gh-aw extension",
								Run:  "curl -fsSL https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh | bash",
							},
						},
					},
				},
			},
			verbose: true,
			wantErr: false,
			validateContent: func(t *testing.T, content []byte) {
				// Should not modify existing correct config
				count := strings.Count(string(content), "Install gh-aw extension")
				if count != 1 {
					t.Errorf("Expected exactly 1 occurrence of 'Install gh-aw extension', got %d", count)
				}
			},
		},
		{
			name: "renders instructions for existing workflow without install step",
			existingWorkflow: &Workflow{
				Name: "Copilot Setup Steps",
				On:   "workflow_dispatch",
				Jobs: map[string]WorkflowJob{
					"copilot-setup-steps": {
						RunsOn: "ubuntu-latest",
						Steps: []CopilotWorkflowStep{
							{
								Name: "Some existing step",
								Run:  "echo 'existing'",
							},
							{
								Name: "Build",
								Run:  "echo 'build'",
							},
						},
					},
				},
			},
			verbose: false,
			wantErr: false,
			validateContent: func(t *testing.T, content []byte) {
				// File should NOT be modified - should remain with only 2 steps
				var wf Workflow
				if err := yaml.Unmarshal(content, &wf); err != nil {
					t.Fatalf("Failed to unmarshal workflow YAML: %v", err)
				}
				job, ok := wf.Jobs["copilot-setup-steps"]
				if !ok {
					t.Fatalf("Expected job 'copilot-setup-steps' not found")
				}

				// File should remain unchanged with only 2 existing steps
				if len(job.Steps) != 2 {
					t.Errorf("Expected 2 steps (file should not be modified), got %d", len(job.Steps))
				}

				// Verify the install step was NOT injected
				if job.Steps[0].Name == "Install gh-aw extension" {
					t.Errorf("Expected 'Install gh-aw extension' step to NOT be injected (instructions should be rendered)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Create existing workflow if specified
			if tt.existingWorkflow != nil {
				workflowsDir := filepath.Join(".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				data, err := yaml.Marshal(tt.existingWorkflow)
				if err != nil {
					t.Fatalf("Failed to marshal existing workflow: %v", err)
				}

				setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
				if err := os.WriteFile(setupStepsPath, data, 0644); err != nil {
					t.Fatalf("Failed to write existing workflow: %v", err)
				}
			}

			// Call the function
			err = ensureCopilotSetupSteps(tt.verbose, workflow.ActionModeDev, "dev")

			if (err != nil) != tt.wantErr {
				t.Errorf("ensureCopilotSetupSteps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify the file was created/updated
			setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
			content, err := os.ReadFile(setupStepsPath)
			if err != nil {
				t.Fatalf("Failed to read copilot-setup-steps.yml: %v", err)
			}

			// Run custom validation if provided
			if tt.validateContent != nil {
				tt.validateContent(t, content)
			}
		})
	}
}

func TestWorkflowStructMarshaling(t *testing.T) {
	t.Parallel()

	workflow := Workflow{
		Name: "Test Workflow",
		On:   "push",
		Jobs: map[string]WorkflowJob{
			"test-job": {
				RunsOn: "ubuntu-latest",
				Permissions: map[string]any{
					"contents": "read",
				},
				Steps: []CopilotWorkflowStep{
					{
						Name: "Checkout",
						Uses: "actions/checkout@v5",
					},
					{
						Name: "Run script",
						Run:  "echo 'test'",
						Env: map[string]any{
							"TEST_VAR": "value",
						},
					},
				},
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&workflow)
	if err != nil {
		t.Fatalf("Failed to marshal workflow: %v", err)
	}

	// Unmarshal back
	var unmarshaledWorkflow Workflow
	if err := yaml.Unmarshal(data, &unmarshaledWorkflow); err != nil {
		t.Fatalf("Failed to unmarshal workflow: %v", err)
	}

	// Verify structure
	if unmarshaledWorkflow.Name != "Test Workflow" {
		t.Errorf("Expected name 'Test Workflow', got %q", unmarshaledWorkflow.Name)
	}

	job, exists := unmarshaledWorkflow.Jobs["test-job"]
	if !exists {
		t.Fatal("Expected 'test-job' to exist")
	}

	if len(job.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(job.Steps))
	}
}

func TestCopilotSetupStepsYAMLConstant(t *testing.T) {
	t.Parallel()

	// Verify the constant can be parsed
	var workflow Workflow
	if err := yaml.Unmarshal([]byte(copilotSetupStepsYAML), &workflow); err != nil {
		t.Fatalf("Failed to parse copilotSetupStepsYAML constant: %v", err)
	}

	// Verify key elements
	if workflow.Name != "Copilot Setup Steps" {
		t.Errorf("Expected workflow name 'Copilot Setup Steps', got %q", workflow.Name)
	}

	job, exists := workflow.Jobs["copilot-setup-steps"]
	if !exists {
		t.Fatal("Expected 'copilot-setup-steps' job to exist")
	}

	// Verify it has the extension install step
	hasExtensionInstall := false
	for _, step := range job.Steps {
		if strings.Contains(step.Run, "install-gh-aw.sh") || strings.Contains(step.Run, "curl -fsSL") {
			hasExtensionInstall = true
			break
		}
	}

	if !hasExtensionInstall {
		t.Error("Expected copilotSetupStepsYAML to contain extension install step with bash script")
	}

	// Verify it does NOT have checkout, Go setup or build steps (for universal use)
	for _, step := range job.Steps {
		if strings.Contains(step.Name, "Checkout") || strings.Contains(step.Uses, "checkout@") {
			t.Error("Template should not contain 'Checkout' step - not mandatory for extension install")
		}
		if strings.Contains(step.Name, "Set up Go") {
			t.Error("Template should not contain 'Set up Go' step for universal use")
		}
		if strings.Contains(step.Name, "Build gh-aw from source") {
			t.Error("Template should not contain 'Build gh-aw from source' step for universal use")
		}
		if strings.Contains(step.Run, "make build") {
			t.Error("Template should not contain 'make build' command for universal use")
		}
	}
}

func TestEnsureCopilotSetupStepsFilePermissions(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	err = ensureCopilotSetupSteps(false, workflow.ActionModeDev, "dev")
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Check file permissions
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	info, err := os.Stat(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to stat copilot-setup-steps.yml: %v", err)
	}

	// Verify file is readable and writable
	mode := info.Mode()
	if mode.Perm()&0600 != 0600 {
		t.Errorf("Expected file to have at least 0600 permissions, got %o", mode.Perm())
	}
}

func TestCopilotWorkflowStepStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		step CopilotWorkflowStep
	}{
		{
			name: "step with uses",
			step: CopilotWorkflowStep{
				Name: "Checkout",
				Uses: "actions/checkout@v5",
			},
		},
		{
			name: "step with run",
			step: CopilotWorkflowStep{
				Name: "Run command",
				Run:  "echo 'test'",
			},
		},
		{
			name: "step with environment",
			step: CopilotWorkflowStep{
				Name: "Run with env",
				Run:  "echo $TEST",
				Env: map[string]any{
					"TEST": "value",
				},
			},
		},
		{
			name: "step with with parameters",
			step: CopilotWorkflowStep{
				Name: "Setup",
				Uses: "actions/setup-go@v6",
				With: map[string]any{
					"go-version": "1.21",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to YAML
			data, err := yaml.Marshal(&tt.step)
			if err != nil {
				t.Fatalf("Failed to marshal step: %v", err)
			}

			// Unmarshal back
			var unmarshaledStep CopilotWorkflowStep
			if err := yaml.Unmarshal(data, &unmarshaledStep); err != nil {
				t.Fatalf("Failed to unmarshal step: %v", err)
			}

			// Verify name is preserved
			if unmarshaledStep.Name != tt.step.Name {
				t.Errorf("Expected name %q, got %q", tt.step.Name, unmarshaledStep.Name)
			}
		})
	}
}

func TestEnsureCopilotSetupStepsDirectoryCreation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Call function when .github/workflows doesn't exist
	err = ensureCopilotSetupSteps(false, workflow.ActionModeDev, "dev")
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Verify directory structure was created
	workflowsDir := filepath.Join(".github", "workflows")
	info, err := os.Stat(workflowsDir)
	if os.IsNotExist(err) {
		t.Error("Expected .github/workflows directory to be created")
		return
	}

	if !info.IsDir() {
		t.Error("Expected .github/workflows to be a directory")
	}

	// Verify file was created
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Error("Expected copilot-setup-steps.yml to be created")
	}
}

// TestEnsureCopilotSetupSteps_ReleaseMode tests that release mode uses the actions/setup-cli action
func TestEnsureCopilotSetupSteps_ReleaseMode(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Call function with release mode
	testVersion := "v1.2.3"
	err = ensureCopilotSetupSteps(false, workflow.ActionModeRelease, testVersion)
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Read generated file
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read copilot-setup-steps.yml: %v", err)
	}

	contentStr := string(content)

	// Verify it uses actions/setup-cli with the correct version tag
	if !strings.Contains(contentStr, "actions/setup-cli@v1.2.3") {
		t.Errorf("Expected copilot-setup-steps.yml to use actions/setup-cli@v1.2.3 in release mode, got:\n%s", contentStr)
	}

	// Verify it uses the correct version in the with parameter
	if !strings.Contains(contentStr, "version: v1.2.3") {
		t.Errorf("Expected copilot-setup-steps.yml to have version: v1.2.3, got:\n%s", contentStr)
	}

	// Verify it has checkout step
	if !strings.Contains(contentStr, "actions/checkout@v6") {
		t.Error("Expected copilot-setup-steps.yml to have checkout step in release mode")
	}

	// Verify it doesn't use curl/install-gh-aw.sh
	if strings.Contains(contentStr, "install-gh-aw.sh") || strings.Contains(contentStr, "curl -fsSL") {
		t.Error("Expected copilot-setup-steps.yml to NOT use curl method in release mode")
	}
}

// TestEnsureCopilotSetupSteps_DevMode tests that dev mode uses curl install method
func TestEnsureCopilotSetupSteps_DevMode(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Call function with dev mode
	err = ensureCopilotSetupSteps(false, workflow.ActionModeDev, "dev")
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Read generated file
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read copilot-setup-steps.yml: %v", err)
	}

	contentStr := string(content)

	// Verify it uses curl method
	if !strings.Contains(contentStr, "install-gh-aw.sh") {
		t.Error("Expected copilot-setup-steps.yml to use install-gh-aw.sh in dev mode")
	}

	// Verify it doesn't use actions/setup-cli
	if strings.Contains(contentStr, "actions/setup-cli") {
		t.Error("Expected copilot-setup-steps.yml to NOT use actions/setup-cli in dev mode")
	}
}

// TestEnsureCopilotSetupSteps_CreateWithReleaseMode tests creating a new file with release mode
func TestEnsureCopilotSetupSteps_CreateWithReleaseMode(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create new file with release mode and specific version
	testVersion := "v2.0.0"
	err = ensureCopilotSetupSteps(false, workflow.ActionModeRelease, testVersion)
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read copilot-setup-steps.yml: %v", err)
	}

	contentStr := string(content)

	// Verify release mode characteristics
	if !strings.Contains(contentStr, "actions/setup-cli@v2.0.0") {
		t.Errorf("Expected action reference with version tag @v2.0.0, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "version: v2.0.0") {
		t.Errorf("Expected version parameter v2.0.0, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "actions/checkout@v6") {
		t.Errorf("Expected checkout step in release mode")
	}
}

// TestEnsureCopilotSetupSteps_CreateWithDevMode tests creating a new file with dev mode
func TestEnsureCopilotSetupSteps_CreateWithDevMode(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create new file with dev mode
	err = ensureCopilotSetupSteps(false, workflow.ActionModeDev, "dev")
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read copilot-setup-steps.yml: %v", err)
	}

	contentStr := string(content)

	// Verify dev mode characteristics
	if !strings.Contains(contentStr, "curl -fsSL") {
		t.Errorf("Expected curl command in dev mode")
	}
	if !strings.Contains(contentStr, "install-gh-aw.sh") {
		t.Errorf("Expected install-gh-aw.sh reference in dev mode")
	}
	if strings.Contains(contentStr, "actions/setup-cli") {
		t.Errorf("Did not expect actions/setup-cli in dev mode")
	}
	if strings.Contains(contentStr, "actions/checkout") {
		t.Errorf("Did not expect checkout step in dev mode")
	}
}

// TestEnsureCopilotSetupSteps_UpdateExistingWithReleaseMode tests updating an existing file with release mode
func TestEnsureCopilotSetupSteps_UpdateExistingWithReleaseMode(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write existing workflow without gh-aw install step
	existingContent := `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - name: Some other step
        run: echo "test"
`
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := os.WriteFile(setupStepsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing workflow: %v", err)
	}

	// Call with release mode - should render instructions instead of modifying
	testVersion := "v3.0.0"
	err = ensureCopilotSetupSteps(false, workflow.ActionModeRelease, testVersion)
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Read file - should remain unchanged
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Verify file was NOT modified - should remain identical to existingContent
	if contentStr != existingContent {
		t.Errorf("Expected file to remain unchanged (instructions should be rendered instead), got:\n%s", contentStr)
	}

	// Verify the install step was NOT injected
	if strings.Contains(contentStr, "actions/setup-cli") {
		t.Errorf("Expected 'actions/setup-cli' to NOT be injected (instructions should be rendered)")
	}
	if strings.Contains(contentStr, "Install gh-aw extension") {
		t.Errorf("Expected 'Install gh-aw extension' step to NOT be injected (instructions should be rendered)")
	}
}

// TestEnsureCopilotSetupSteps_UpdateExistingWithDevMode tests updating an existing file with dev mode
func TestEnsureCopilotSetupSteps_UpdateExistingWithDevMode(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write existing workflow without gh-aw install step
	existingContent := `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - name: Some other step
        run: echo "test"
`
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := os.WriteFile(setupStepsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing workflow: %v", err)
	}

	// Call with dev mode - should render instructions instead of modifying
	err = ensureCopilotSetupSteps(false, workflow.ActionModeDev, "dev")
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Read file - should remain unchanged
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Verify file was NOT modified - should remain identical to existingContent
	if contentStr != existingContent {
		t.Errorf("Expected file to remain unchanged (instructions should be rendered instead), got:\n%s", contentStr)
	}

	// Verify the install step was NOT injected
	if strings.Contains(contentStr, "curl -fsSL") {
		t.Errorf("Expected 'curl' command to NOT be injected (instructions should be rendered)")
	}
	if strings.Contains(contentStr, "install-gh-aw.sh") {
		t.Errorf("Expected 'install-gh-aw.sh' to NOT be injected (instructions should be rendered)")
	}
	if strings.Contains(contentStr, "actions/setup-cli") {
		t.Errorf("Did not expect actions/setup-cli in dev mode")
	}
	// Verify original step is preserved
	if !strings.Contains(contentStr, "Some other step") {
		t.Errorf("Expected original step to be preserved")
	}
}

// TestEnsureCopilotSetupSteps_SkipsUpdateWhenActionExists tests that update is skipped when action already exists
func TestEnsureCopilotSetupSteps_SkipsUpdateWhenActionExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write existing workflow WITH actions/setup-cli (release mode)
	existingContent := `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    steps:
      - uses: github/gh-aw/actions/setup-cli@v1.0.0
        with:
          version: v1.0.0
`
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := os.WriteFile(setupStepsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing workflow: %v", err)
	}

	// Attempt to update - should skip
	err = ensureCopilotSetupSteps(false, workflow.ActionModeRelease, "v2.0.0")
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Read file - should be unchanged
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(content)

	// Verify file was not modified (still has v1.0.0)
	if !strings.Contains(contentStr, "v1.0.0") {
		t.Errorf("Expected file to remain unchanged with v1.0.0")
	}
	if strings.Contains(contentStr, "v2.0.0") {
		t.Errorf("File should not have been updated to v2.0.0")
	}
}

// TestEnsureCopilotSetupSteps_SkipsUpdateWhenCurlExists tests that update is skipped when curl install exists
func TestEnsureCopilotSetupSteps_SkipsUpdateWhenCurlExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write existing workflow WITH curl install (dev mode)
	existingContent := `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    steps:
      - name: Install gh-aw extension
        run: curl -fsSL https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh | bash
`
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := os.WriteFile(setupStepsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing workflow: %v", err)
	}

	// Attempt to update - should skip
	err = ensureCopilotSetupSteps(false, workflow.ActionModeDev, "dev")
	if err != nil {
		t.Fatalf("ensureCopilotSetupSteps() failed: %v", err)
	}

	// Verify file content matches expected (should be unchanged)
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != existingContent {
		t.Errorf("Expected file to remain unchanged")
	}
}

// TestUpgradeCopilotSetupSteps tests upgrading version in existing copilot-setup-steps.yml
func TestUpgradeCopilotSetupSteps(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write existing workflow WITH actions/setup-cli at v1.0.0
	existingContent := `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
      - name: Install gh-aw extension
        uses: github/gh-aw/actions/setup-cli@v1.0.0
        with:
          version: v1.0.0
      - name: Verify gh-aw installation
        run: gh aw version
`
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := os.WriteFile(setupStepsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing workflow: %v", err)
	}

	// Upgrade to v2.0.0
	err = upgradeCopilotSetupSteps(false, workflow.ActionModeRelease, "v2.0.0")
	if err != nil {
		t.Fatalf("upgradeCopilotSetupSteps() failed: %v", err)
	}

	// Read updated file
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	contentStr := string(content)

	// Verify version was upgraded
	if !strings.Contains(contentStr, "actions/setup-cli@v2.0.0") {
		t.Errorf("Expected action reference to be upgraded to @v2.0.0, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "version: v2.0.0") {
		t.Errorf("Expected version parameter to be v2.0.0, got:\n%s", contentStr)
	}

	// Verify old version is gone
	if strings.Contains(contentStr, "v1.0.0") {
		t.Errorf("Old version v1.0.0 should not be present, got:\n%s", contentStr)
	}
}

// TestUpgradeCopilotSetupSteps_NoFile tests upgrading when file doesn't exist
func TestUpgradeCopilotSetupSteps_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Attempt to upgrade when file doesn't exist - should create new file
	err = upgradeCopilotSetupSteps(false, workflow.ActionModeRelease, "v2.0.0")
	if err != nil {
		t.Fatalf("upgradeCopilotSetupSteps() failed: %v", err)
	}

	// Verify file was created with the new version
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "actions/setup-cli@v2.0.0") {
		t.Errorf("Expected new file to have @v2.0.0, got:\n%s", contentStr)
	}
}

// TestUpgradeCopilotSetupSteps_DevMode tests that dev mode doesn't use actions/setup-cli
func TestUpgradeCopilotSetupSteps_DevMode(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Write existing workflow with curl install (dev mode)
	existingContent := `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    steps:
      - name: Install gh-aw extension
        run: curl -fsSL https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh | bash
      - name: Verify gh-aw installation
        run: gh aw version
`
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := os.WriteFile(setupStepsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing workflow: %v", err)
	}

	// Attempt upgrade in dev mode - should not modify file
	err = upgradeCopilotSetupSteps(false, workflow.ActionModeDev, "dev")
	if err != nil {
		t.Fatalf("upgradeCopilotSetupSteps() failed: %v", err)
	}

	// Verify file was not changed (dev mode doesn't upgrade curl-based installs)
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != existingContent {
		t.Errorf("File should remain unchanged in dev mode")
	}
}

// TestUpgradeSetupCliVersionInContent tests the regex-based content upgrade helper.
func TestUpgradeSetupCliVersionInContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		content       string
		actionMode    workflow.ActionMode
		version       string
		resolver      workflow.ActionSHAResolver
		expectUpgrade bool
		validate      func(*testing.T, string)
	}{
		{
			name: "upgrades version-tag ref",
			content: `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Install gh-aw extension
        uses: github/gh-aw/actions/setup-cli@v1.0.0
        with:
          version: v1.0.0
`,
			actionMode:    workflow.ActionModeRelease,
			version:       "v2.0.0",
			resolver:      nil,
			expectUpgrade: true,
			validate: func(t *testing.T, got string) {
				if !strings.Contains(got, "uses: github/gh-aw/actions/setup-cli@v2.0.0") {
					t.Errorf("Expected updated uses: line, got:\n%s", got)
				}
				if !strings.Contains(got, "version: v2.0.0") {
					t.Errorf("Expected updated version: parameter, got:\n%s", got)
				}
				if strings.Contains(got, "v1.0.0") {
					t.Errorf("Old version v1.0.0 should be gone, got:\n%s", got)
				}
				// File structure must be preserved (comment line, on: key, etc.)
				if !strings.Contains(got, "on: workflow_dispatch") {
					t.Errorf("Expected on: field to be preserved, got:\n%s", got)
				}
			},
		},
		{
			name: "upgrades SHA-pinned ref and produces unquoted uses: value",
			content: `name: "Copilot Setup Steps"
on: workflow_dispatch
jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    steps:
      - name: Install gh-aw extension
        uses: github/gh-aw/actions/setup-cli@v1.0.0
        with:
          version: v1.0.0
`,
			actionMode:    workflow.ActionModeRelease,
			version:       "v2.0.0",
			resolver:      &mockSHAResolver{sha: "bd9c0ca491e6334a2797ef56ad6ee89958d54ab9"},
			expectUpgrade: true,
			validate: func(t *testing.T, got string) {
				want := "uses: github/gh-aw/actions/setup-cli@bd9c0ca491e6334a2797ef56ad6ee89958d54ab9 # v2.0.0"
				if !strings.Contains(got, want) {
					t.Errorf("Expected unquoted SHA-pinned uses: line %q, got:\n%s", want, got)
				}
				// Confirm NO quoted form is present
				if strings.Contains(got, `uses: "github/gh-aw`) {
					t.Errorf("uses: value must not be quoted, got:\n%s", got)
				}
				if !strings.Contains(got, "version: v2.0.0") {
					t.Errorf("Expected updated version: parameter, got:\n%s", got)
				}
			},
		},
		{
			name: "strips existing quotes from uses: value",
			content: `jobs:
  copilot-setup-steps:
    steps:
      - name: Install gh-aw extension
        uses: "github/gh-aw/actions/setup-cli@oldsha # v0.53.2"
        with:
          version: v0.53.2
`,
			actionMode:    workflow.ActionModeRelease,
			version:       "v2.0.0",
			resolver:      nil,
			expectUpgrade: true,
			validate: func(t *testing.T, got string) {
				if strings.Contains(got, `"github/gh-aw`) {
					t.Errorf("Quotes must be stripped from uses: value, got:\n%s", got)
				}
				if !strings.Contains(got, "uses: github/gh-aw/actions/setup-cli@v2.0.0") {
					t.Errorf("Expected updated unquoted uses: line, got:\n%s", got)
				}
				if !strings.Contains(got, "version: v2.0.0") {
					t.Errorf("Expected version: to be updated to v2.0.0, got:\n%s", got)
				}
			},
		},
		{
			name: "no upgrade when no setup-cli step",
			content: `jobs:
  copilot-setup-steps:
    steps:
      - run: echo hello
`,
			actionMode:    workflow.ActionModeRelease,
			version:       "v2.0.0",
			resolver:      nil,
			expectUpgrade: false,
		},
		{
			name: "no upgrade in dev mode",
			content: `jobs:
  copilot-setup-steps:
    steps:
      - uses: github/gh-aw/actions/setup-cli@v1.0.0
        with:
          version: v1.0.0
`,
			actionMode:    workflow.ActionModeDev,
			version:       "v2.0.0",
			resolver:      nil,
			expectUpgrade: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgraded, got, err := upgradeSetupCliVersionInContent([]byte(tt.content), tt.actionMode, tt.version, tt.resolver)
			if err != nil {
				t.Fatalf("upgradeSetupCliVersionInContent() error: %v", err)
			}
			if upgraded != tt.expectUpgrade {
				t.Errorf("upgraded = %v, want %v", upgraded, tt.expectUpgrade)
			}
			if tt.validate != nil {
				tt.validate(t, string(got))
			}
		})
	}
}

// TestUpgradeSetupCliVersionInContent_ExactPreservation verifies that
// upgradeSetupCliVersionInContent changes ONLY the two target lines
// (uses: and version:) and leaves every other byte of the file intact —
// including YAML comments at all positions, blank lines, field ordering,
// indentation, and unrelated step entries.
func TestUpgradeSetupCliVersionInContent_ExactPreservation(t *testing.T) {
	t.Parallel()

	// A deliberately rich workflow file:
	// - top-level comment before the name field
	// - inline comment on the on: trigger
	// - comment inside the jobs block
	// - multiple steps with their own comments
	// - a step after setup-cli with its own comment
	// - trailing comment at end of file
	input := `# Top-level workflow comment — must survive the upgrade.
name: "Copilot Setup Steps"

# Trigger comment: dispatched manually or on push.
on: # inline comment on on:
  workflow_dispatch:
  push:
    paths:
      - .github/workflows/copilot-setup-steps.yml # path filter comment

jobs:
  # Job-level comment that must not be lost.
  copilot-setup-steps:
    runs-on: ubuntu-latest
    # Permission comment.
    permissions:
      contents: read # read-only is sufficient

    steps:
      # Step 1 comment.
      - name: Checkout repository
        uses: actions/checkout@v4 # pin to stable tag
        with:
          fetch-depth: 0 # full history

      # Step 2 comment — this step should be updated.
      - name: Install gh-aw extension
        uses: github/gh-aw/actions/setup-cli@v1.0.0
        with:
          version: v1.0.0
          extra-param: keep-me # this param must not be touched

      # Step 3 comment — must be fully preserved.
      - name: Run something else
        run: echo "hello" # inline run comment
`

	// Expected output: identical to input except the two target lines.
	expected := `# Top-level workflow comment — must survive the upgrade.
name: "Copilot Setup Steps"

# Trigger comment: dispatched manually or on push.
on: # inline comment on on:
  workflow_dispatch:
  push:
    paths:
      - .github/workflows/copilot-setup-steps.yml # path filter comment

jobs:
  # Job-level comment that must not be lost.
  copilot-setup-steps:
    runs-on: ubuntu-latest
    # Permission comment.
    permissions:
      contents: read # read-only is sufficient

    steps:
      # Step 1 comment.
      - name: Checkout repository
        uses: actions/checkout@v4 # pin to stable tag
        with:
          fetch-depth: 0 # full history

      # Step 2 comment — this step should be updated.
      - name: Install gh-aw extension
        uses: github/gh-aw/actions/setup-cli@v2.0.0
        with:
          version: v2.0.0
          extra-param: keep-me # this param must not be touched

      # Step 3 comment — must be fully preserved.
      - name: Run something else
        run: echo "hello" # inline run comment
`

	upgraded, got, err := upgradeSetupCliVersionInContent([]byte(input), workflow.ActionModeRelease, "v2.0.0", nil)
	if err != nil {
		t.Fatalf("upgradeSetupCliVersionInContent() error: %v", err)
	}
	if !upgraded {
		t.Fatal("Expected upgrade to occur")
	}

	gotStr := string(got)
	if gotStr != expected {
		// Show a line-by-line diff to make failures easy to diagnose.
		inputLines := strings.Split(input, "\n")
		expectedLines := strings.Split(expected, "\n")
		gotLines := strings.Split(gotStr, "\n")

		t.Errorf("Output does not match expected (only uses: and version: lines should differ).\n")
		for i := 0; i < len(expectedLines) || i < len(gotLines); i++ {
			var exp, act string
			if i < len(expectedLines) {
				exp = expectedLines[i]
			}
			if i < len(gotLines) {
				act = gotLines[i]
			}
			if exp != act {
				orig := ""
				if i < len(inputLines) {
					orig = inputLines[i]
				}
				t.Errorf("  line %d:\n    input:    %q\n    expected: %q\n    got:      %q", i+1, orig, exp, act)
			}
		}
	}
}

// SHA-pinned reference writes an unquoted uses: line, preserving the rest of the file.
// Regression test for: gh aw upgrade wraps uses value in quotes including inline comment.
func TestUpgradeCopilotSetupSteps_SHAPinnedNoQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Pre-existing file with a version-tagged reference and extra comments/fields
	// that must be preserved unchanged.
	existingContent := `name: "Copilot Setup Steps"

# This workflow configures the environment for GitHub Copilot Agent
on:
  workflow_dispatch:
  push:
    paths:
      - .github/workflows/copilot-setup-steps.yml

jobs:
  copilot-setup-steps:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
      - name: Install gh-aw extension
        uses: github/gh-aw/actions/setup-cli@v1.0.0
        with:
          version: v1.0.0
`
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := os.WriteFile(setupStepsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing workflow: %v", err)
	}

	// upgradeSetupCliVersionInContent with a SHA resolver — the result must be unquoted
	sha := "bd9c0ca491e6334a2797ef56ad6ee89958d54ab9"
	resolver := &mockSHAResolver{sha: sha}
	upgraded, updated, err := upgradeSetupCliVersionInContent([]byte(existingContent), workflow.ActionModeRelease, "v2.0.0", resolver)
	if err != nil {
		t.Fatalf("upgradeSetupCliVersionInContent() error: %v", err)
	}
	if !upgraded {
		t.Fatal("Expected upgrade to occur")
	}

	updatedStr := string(updated)

	// The uses: line must be unquoted
	wantUses := "uses: github/gh-aw/actions/setup-cli@" + sha + " # v2.0.0"
	if !strings.Contains(updatedStr, wantUses) {
		t.Errorf("Expected unquoted uses: line %q, got:\n%s", wantUses, updatedStr)
	}
	if strings.Contains(updatedStr, `uses: "github/gh-aw`) {
		t.Errorf("uses: value must not be quoted, got:\n%s", updatedStr)
	}

	// version: parameter updated
	if !strings.Contains(updatedStr, "version: v2.0.0") {
		t.Errorf("Expected version: v2.0.0, got:\n%s", updatedStr)
	}

	// All other content must be preserved exactly
	for _, preserved := range []string{
		`# This workflow configures the environment for GitHub Copilot Agent`,
		`workflow_dispatch:`,
		`- .github/workflows/copilot-setup-steps.yml`,
		`permissions:`,
		`contents: read`,
		`uses: actions/checkout@v4`,
	} {
		if !strings.Contains(updatedStr, preserved) {
			t.Errorf("Expected content %q to be preserved, got:\n%s", preserved, updatedStr)
		}
	}
}

// TestGetActionRef tests the getActionRef helper with and without a resolver
func TestGetActionRef(t *testing.T) {
	tests := []struct {
		name        string
		actionMode  workflow.ActionMode
		version     string
		resolver    workflow.ActionSHAResolver
		expectedRef string
	}{
		{
			name:        "release mode without resolver uses version tag",
			actionMode:  workflow.ActionModeRelease,
			version:     "v1.2.3",
			resolver:    nil,
			expectedRef: "@v1.2.3",
		},
		{
			name:        "release mode with resolver uses SHA-pinned reference",
			actionMode:  workflow.ActionModeRelease,
			version:     "v1.2.3",
			resolver:    &mockSHAResolver{sha: "abc1234567890123456789012345678901234567890"},
			expectedRef: "@abc1234567890123456789012345678901234567890 # v1.2.3",
		},
		{
			name:        "release mode with failing resolver falls back to version tag",
			actionMode:  workflow.ActionModeRelease,
			version:     "v1.2.3",
			resolver:    &mockSHAResolver{sha: "", err: errors.New("resolution failed")},
			expectedRef: "@v1.2.3",
		},
		{
			name:        "dev mode uses @main",
			actionMode:  workflow.ActionModeDev,
			version:     "v1.2.3",
			resolver:    nil,
			expectedRef: "@main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := getActionRef(tt.actionMode, tt.version, tt.resolver)
			if ref != tt.expectedRef {
				t.Errorf("getActionRef() = %q, want %q", ref, tt.expectedRef)
			}
		})
	}
}
