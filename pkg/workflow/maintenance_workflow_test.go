//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMaintenanceCron(t *testing.T) {
	tests := []struct {
		name           string
		minExpiresDays int
		expectedCron   string
		expectedDesc   string
	}{
		{
			name:           "1 day or less - every 2 hours",
			minExpiresDays: 1,
			expectedCron:   "37 */2 * * *",
			expectedDesc:   "Every 2 hours",
		},
		{
			name:           "2 days - every 6 hours",
			minExpiresDays: 2,
			expectedCron:   "37 */6 * * *",
			expectedDesc:   "Every 6 hours",
		},
		{
			name:           "3 days - every 12 hours",
			minExpiresDays: 3,
			expectedCron:   "37 */12 * * *",
			expectedDesc:   "Every 12 hours",
		},
		{
			name:           "4 days - every 12 hours",
			minExpiresDays: 4,
			expectedCron:   "37 */12 * * *",
			expectedDesc:   "Every 12 hours",
		},
		{
			name:           "5 days - daily",
			minExpiresDays: 5,
			expectedCron:   "37 0 * * *",
			expectedDesc:   "Daily",
		},
		{
			name:           "7 days - daily",
			minExpiresDays: 7,
			expectedCron:   "37 0 * * *",
			expectedDesc:   "Daily",
		},
		{
			name:           "30 days - daily",
			minExpiresDays: 30,
			expectedCron:   "37 0 * * *",
			expectedDesc:   "Daily",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cron, desc := generateMaintenanceCron(tt.minExpiresDays)
			if cron != tt.expectedCron {
				t.Errorf("generateMaintenanceCron(%d) cron = %q, expected %q", tt.minExpiresDays, cron, tt.expectedCron)
			}
			if desc != tt.expectedDesc {
				t.Errorf("generateMaintenanceCron(%d) desc = %q, expected %q", tt.minExpiresDays, desc, tt.expectedDesc)
			}
		})
	}
}

func TestGenerateMaintenanceWorkflow_WithExpires(t *testing.T) {
	tests := []struct {
		name                    string
		workflowDataList        []*WorkflowData
		expectWorkflowGenerated bool
		expectError             bool
	}{
		{
			name: "with expires in discussions - should generate workflow",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{
							Expires: 168, // 7 days
						},
					},
				},
			},
			expectWorkflowGenerated: true,
			expectError:             false,
		},
		{
			name: "with expires in issues - should generate workflow",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow-issues",
					SafeOutputs: &SafeOutputsConfig{
						CreateIssues: &CreateIssuesConfig{
							Expires: 48, // 2 days
						},
					},
				},
			},
			expectWorkflowGenerated: true,
			expectError:             false,
		},
		{
			name: "without expires field - should NOT generate workflow",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{},
					},
				},
			},
			expectWorkflowGenerated: false,
			expectError:             false,
		},
		{
			name: "with both discussions and issues expires - should generate workflow",
			workflowDataList: []*WorkflowData{
				{
					Name: "multi-expires-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{
							Expires: 168,
						},
						CreateIssues: &CreateIssuesConfig{
							Expires: 48,
						},
					},
				},
			},
			expectWorkflowGenerated: true,
			expectError:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the workflow
			tmpDir := t.TempDir()

			// Call GenerateMaintenanceWorkflow
			err := GenerateMaintenanceWorkflow(tt.workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check if workflow file was generated
			maintenanceFile := filepath.Join(tmpDir, "agentics-maintenance.yml")
			_, statErr := os.Stat(maintenanceFile)
			workflowExists := statErr == nil

			if tt.expectWorkflowGenerated && !workflowExists {
				t.Errorf("Expected maintenance workflow to be generated but it was not")
			}
			if !tt.expectWorkflowGenerated && workflowExists {
				t.Errorf("Expected maintenance workflow NOT to be generated but it was")
			}
		})
	}
}

func TestGenerateMaintenanceWorkflow_DeletesExistingFile(t *testing.T) {
	tests := []struct {
		name             string
		workflowDataList []*WorkflowData
		createFileBefore bool
		expectFileExists bool
	}{
		{
			name: "no expires field - should delete existing file",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{},
					},
				},
			},
			createFileBefore: true,
			expectFileExists: false,
		},
		{
			name: "with expires - should create file",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{
							Expires: 168,
						},
					},
				},
			},
			createFileBefore: false,
			expectFileExists: true,
		},
		{
			name: "no expires without existing file - should not error",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{},
					},
				},
			},
			createFileBefore: false,
			expectFileExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			maintenanceFile := filepath.Join(tmpDir, "agentics-maintenance.yml")

			// Create the maintenance file if requested
			if tt.createFileBefore {
				err := os.WriteFile(maintenanceFile, []byte("# Existing maintenance workflow\n"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Call GenerateMaintenanceWorkflow
			err := GenerateMaintenanceWorkflow(tt.workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check if file exists
			_, statErr := os.Stat(maintenanceFile)
			fileExists := statErr == nil

			if tt.expectFileExists && !fileExists {
				t.Errorf("Expected maintenance workflow file to exist but it does not")
			}
			if !tt.expectFileExists && fileExists {
				t.Errorf("Expected maintenance workflow file NOT to exist but it does")
			}
		})
	}
}

func TestGenerateMaintenanceWorkflow_OperationJobConditions(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					Expires: 48,
				},
			},
		},
	}

	tmpDir := t.TempDir()
	err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	operationSkipCondition := `github.event_name != 'workflow_dispatch' || github.event.inputs.operation == ''`
	operationRunCondition := `github.event_name == 'workflow_dispatch' && github.event.inputs.operation != ''`

	const jobSectionSearchRange = 300
	const runOpSectionSearchRange = 200

	// Jobs that should be disabled when operation is set
	disabledJobs := []string{"close-expired-entities:", "compile-workflows:", "zizmor-scan:", "secret-validation:"}
	for _, job := range disabledJobs {
		// Find the if: condition for each job
		jobIdx := strings.Index(yaml, "\n  "+job)
		if jobIdx == -1 {
			t.Errorf("Job %q not found in generated workflow", job)
			continue
		}
		// Check that the operation skip condition appears after the job name (within a reasonable range)
		jobSection := yaml[jobIdx : jobIdx+jobSectionSearchRange]
		if !strings.Contains(jobSection, operationSkipCondition) {
			t.Errorf("Job %q is missing the operation skip condition %q in:\n%s", job, operationSkipCondition, jobSection)
		}
	}

	// run_operation job should NOT have the skip condition but should have its own activation condition
	runOpIdx := strings.Index(yaml, "\n  run_operation:")
	if runOpIdx == -1 {
		t.Errorf("Job run_operation not found in generated workflow")
	} else {
		runOpSection := yaml[runOpIdx : runOpIdx+runOpSectionSearchRange]
		if strings.Contains(runOpSection, operationSkipCondition) {
			t.Errorf("Job run_operation should NOT have the operation skip condition")
		}
		if !strings.Contains(runOpSection, operationRunCondition) {
			t.Errorf("Job run_operation should have the activation condition %q", operationRunCondition)
		}
	}
}

func TestGenerateMaintenanceWorkflow_ActionTag(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					Expires: 48,
				},
			},
		},
	}

	t.Run("release mode with action-tag uses remote ref", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeRelease, "v0.47.4", false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		if !strings.Contains(string(content), "github/gh-aw/actions/setup@v0.47.4") {
			t.Errorf("Expected remote ref with action-tag v0.47.4, got:\n%s", string(content))
		}
		if strings.Contains(string(content), "uses: ./actions/setup") {
			t.Errorf("Expected no local path in release mode with action-tag, got:\n%s", string(content))
		}
	})

	t.Run("release mode with action-tag and resolver uses SHA-pinned ref", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Set up an action resolver with a cached SHA for the setup action
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw/actions/setup", "v0.47.4", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		resolver := NewActionResolver(cache)

		workflowDataListWithResolver := []*WorkflowData{
			{
				Name:              "test-workflow",
				ActionResolver:    resolver,
				ActionPinWarnings: make(map[string]bool),
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						Expires: 48,
					},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(workflowDataListWithResolver, tmpDir, "v1.0.0", ActionModeRelease, "v0.47.4", false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		expectedRef := "github/gh-aw/actions/setup@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa # v0.47.4"
		if !strings.Contains(string(content), expectedRef) {
			t.Errorf("Expected SHA-pinned ref %q, got:\n%s", expectedRef, string(content))
		}
		if strings.Contains(string(content), "uses: ./actions/setup") {
			t.Errorf("Expected no local path in release mode with action-tag, got:\n%s", string(content))
		}
	})

	t.Run("dev mode ignores action-tag and uses local path", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "v0.47.4", false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		if !strings.Contains(string(content), "uses: ./actions/setup") {
			t.Errorf("Expected local path in dev mode, got:\n%s", string(content))
		}
	})
}

func TestGenerateInstallCLISteps(t *testing.T) {
	t.Run("dev mode generates Setup Go and Build gh-aw steps", func(t *testing.T) {
		result := generateInstallCLISteps(ActionModeDev, "v1.0.0", "")
		if !strings.Contains(result, "Setup Go") {
			t.Errorf("Dev mode should include Setup Go step, got:\n%s", result)
		}
		if !strings.Contains(result, "make build") {
			t.Errorf("Dev mode should include make build step, got:\n%s", result)
		}
		if strings.Contains(result, "setup-cli") {
			t.Errorf("Dev mode should NOT use setup-cli action, got:\n%s", result)
		}
	})

	t.Run("release mode generates setup-cli action step", func(t *testing.T) {
		result := generateInstallCLISteps(ActionModeRelease, "v1.0.0", "")
		if !strings.Contains(result, "github/gh-aw/actions/setup-cli@v1.0.0") {
			t.Errorf("Release mode should use setup-cli action with version, got:\n%s", result)
		}
		if !strings.Contains(result, "version: v1.0.0") {
			t.Errorf("Release mode should pass version to setup-cli, got:\n%s", result)
		}
		if strings.Contains(result, "make build") {
			t.Errorf("Release mode should NOT build from source, got:\n%s", result)
		}
	})

	t.Run("release mode uses actionTag over version", func(t *testing.T) {
		result := generateInstallCLISteps(ActionModeRelease, "v1.0.0", "v2.0.0")
		if !strings.Contains(result, "setup-cli@v2.0.0") {
			t.Errorf("Release mode should use actionTag v2.0.0, got:\n%s", result)
		}
	})
}

func TestGetCLICmdPrefix(t *testing.T) {
	if getCLICmdPrefix(ActionModeDev) != "./gh-aw" {
		t.Errorf("Dev mode should use ./gh-aw prefix")
	}
	if getCLICmdPrefix(ActionModeRelease) != "gh aw" {
		t.Errorf("Release mode should use 'gh aw' prefix")
	}
}

func TestGenerateMaintenanceWorkflow_RunOperationCLICodegen(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					Expires: 48,
				},
			},
		},
	}

	t.Run("dev mode run_operation uses build from source", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		if !strings.Contains(yaml, "make build") {
			t.Errorf("Dev mode run_operation should build from source, got:\n%s", yaml)
		}
		if !strings.Contains(yaml, "GH_AW_CMD_PREFIX: ./gh-aw") {
			t.Errorf("Dev mode run_operation should use ./gh-aw prefix, got:\n%s", yaml)
		}
	})

	t.Run("release mode run_operation uses setup-cli action not gh extension install", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeRelease, "v1.0.0", false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		if strings.Contains(yaml, "gh extension install") {
			t.Errorf("Release mode should NOT use gh extension install, got:\n%s", yaml)
		}
		if !strings.Contains(yaml, "github/gh-aw/actions/setup-cli@v1.0.0") {
			t.Errorf("Release mode run_operation should use setup-cli action, got:\n%s", yaml)
		}
		if !strings.Contains(yaml, "GH_AW_CMD_PREFIX: gh aw") {
			t.Errorf("Release mode run_operation should use 'gh aw' prefix, got:\n%s", yaml)
		}
	})

	t.Run("dev mode compile_workflows uses same codegen as run_operation", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		// Both run_operation and compile_workflows should use the same setup-go version
		// (both use GetActionPin, not hardcoded pins). Exactly 2 occurrences expected.
		setupGoPin := GetActionPin("actions/setup-go")
		occurrences := strings.Count(yaml, setupGoPin)
		if occurrences != 2 {
			t.Errorf("Expected exactly 2 occurrences of pinned setup-go ref %q (run_operation + compile_workflows), got %d in:\n%s",
				setupGoPin, occurrences, yaml)
		}
	})
}
