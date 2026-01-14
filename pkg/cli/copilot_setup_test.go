//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/goccy/go-yaml"
)

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
				if !strings.Contains(string(content), "gh aw version") {
					t.Error("Expected workflow to contain gh aw version verification")
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
								Run:  "curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash",
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
			name: "injects extension install into existing workflow",
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
				// Unmarshal YAML content into Workflow struct for structured validation
				var wf Workflow
				if err := yaml.Unmarshal(content, &wf); err != nil {
					t.Fatalf("Failed to unmarshal workflow YAML: %v", err)
				}
				job, ok := wf.Jobs["copilot-setup-steps"]
				if !ok {
					t.Fatalf("Expected job 'copilot-setup-steps' not found")
				}

				// Extension install and verify steps should be injected at the beginning
				if len(job.Steps) < 4 {
					t.Fatalf("Expected at least 4 steps after injection (2 injected + 2 existing), got %d", len(job.Steps))
				}

				if job.Steps[0].Name != "Install gh-aw extension" {
					t.Errorf("Expected first step to be 'Install gh-aw extension', got %q", job.Steps[0].Name)
				}

				if job.Steps[1].Name != "Verify gh-aw installation" {
					t.Errorf("Expected second step to be 'Verify gh-aw installation', got %q", job.Steps[1].Name)
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
			err = ensureCopilotSetupSteps(tt.verbose)

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

func TestInjectExtensionInstallStep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		workflow      *Workflow
		wantErr       bool
		expectedSteps int
		validateFunc  func(*testing.T, *Workflow)
	}{
		{
			name: "injects at beginning of existing steps",
			workflow: &Workflow{
				Jobs: map[string]WorkflowJob{
					"copilot-setup-steps": {
						Steps: []CopilotWorkflowStep{
							{Name: "Some step"},
							{Name: "Build"},
						},
					},
				},
			},
			wantErr:       false,
			expectedSteps: 4, // 2 injected steps + 2 existing steps
			validateFunc: func(t *testing.T, w *Workflow) {
				steps := w.Jobs["copilot-setup-steps"].Steps
				// Extension install should be at index 0 (beginning)
				if steps[0].Name != "Install gh-aw extension" {
					t.Errorf("Expected step 0 to be 'Install gh-aw extension', got %q", steps[0].Name)
				}
				// Verify step should be at index 1
				if steps[1].Name != "Verify gh-aw installation" {
					t.Errorf("Expected step 1 to be 'Verify gh-aw installation', got %q", steps[1].Name)
				}
			},
		},
		{
			name: "injects when no existing steps",
			workflow: &Workflow{
				Jobs: map[string]WorkflowJob{
					"copilot-setup-steps": {
						Steps: []CopilotWorkflowStep{},
					},
				},
			},
			wantErr:       false,
			expectedSteps: 2, // 2 injected steps (install + verify)
			validateFunc: func(t *testing.T, w *Workflow) {
				steps := w.Jobs["copilot-setup-steps"].Steps
				// Should have 2 steps
				if len(steps) != 2 {
					t.Errorf("Expected 2 steps, got %d", len(steps))
				}
				if steps[0].Name != "Install gh-aw extension" {
					t.Errorf("Expected step 0 to be 'Install gh-aw extension', got %q", steps[0].Name)
				}
				if steps[1].Name != "Verify gh-aw installation" {
					t.Errorf("Expected step 1 to be 'Verify gh-aw installation', got %q", steps[1].Name)
				}
			},
		},
		{
			name: "returns error when job not found",
			workflow: &Workflow{
				Jobs: map[string]WorkflowJob{
					"other-job": {
						Steps: []CopilotWorkflowStep{},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := injectExtensionInstallStep(tt.workflow)

			if (err != nil) != tt.wantErr {
				t.Errorf("injectExtensionInstallStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			job := tt.workflow.Jobs["copilot-setup-steps"]
			if len(job.Steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d", tt.expectedSteps, len(job.Steps))
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, tt.workflow)
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
		if strings.Contains(step.Name, "Setup Go") {
			t.Error("Template should not contain 'Setup Go' step for universal use")
		}
		if strings.Contains(step.Name, "Build gh-aw from source") {
			t.Error("Template should not contain 'Build gh-aw from source' step for universal use")
		}
		if strings.Contains(step.Run, "make build") {
			t.Error("Template should not contain 'make build' command for universal use")
		}
	}

	// Verify verification step uses 'gh aw version' (works via GitHub CLI after bash install)
	hasVerification := false
	for _, step := range job.Steps {
		if strings.Contains(step.Name, "Verify") {
			hasVerification = true
			if !strings.Contains(step.Run, "gh aw") {
				t.Error("Verification step should use 'gh aw version' after bash install")
			}
		}
	}

	if !hasVerification {
		t.Error("Expected template to contain verification step")
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

	err = ensureCopilotSetupSteps(false)
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
	err = ensureCopilotSetupSteps(false)
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
