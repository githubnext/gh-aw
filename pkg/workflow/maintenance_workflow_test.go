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
			err := GenerateMaintenanceWorkflow(tt.workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil)

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
			err := GenerateMaintenanceWorkflow(tt.workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil)
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
	err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	operationSkipCondition := `github.event_name != 'workflow_dispatch' || github.event.inputs.operation == ''`
	operationRunCondition := `github.event_name == 'workflow_dispatch' && github.event.inputs.operation != '' && github.event.inputs.operation != 'safe_outputs' && github.event.inputs.operation != 'create_labels' && github.event.inputs.operation != 'clean_cache_memories' && github.event.inputs.operation != 'validate'`
	applySafeOutputsCondition := `github.event_name == 'workflow_dispatch' && github.event.inputs.operation == 'safe_outputs'`
	createLabelsCondition := `github.event_name == 'workflow_dispatch' && github.event.inputs.operation == 'create_labels'`
	cleanCacheMemoriesCondition := `github.event_name != 'workflow_dispatch' || github.event.inputs.operation == '' || github.event.inputs.operation == 'clean_cache_memories'`

	const jobSectionSearchRange = 300
	const runOpSectionSearchRange = 500

	// Jobs that should be disabled when any non-dedicated operation is set (cleanup-cache-memory has its own dedicated operation)
	disabledJobs := []string{"close-expired-entities:", "compile-workflows:", "secret-validation:"}
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

	// cleanup-cache-memory job should run on schedule, empty operation, or clean_cache_memories operation
	cleanupCacheIdx := strings.Index(yaml, "\n  cleanup-cache-memory:")
	if cleanupCacheIdx == -1 {
		t.Errorf("Job cleanup-cache-memory not found in generated workflow")
	} else {
		cleanupCacheSection := yaml[cleanupCacheIdx : cleanupCacheIdx+jobSectionSearchRange]
		if !strings.Contains(cleanupCacheSection, cleanCacheMemoriesCondition) {
			t.Errorf("Job cleanup-cache-memory should have the clean_cache_memories condition %q in:\n%s", cleanCacheMemoriesCondition, cleanupCacheSection)
		}
	}

	// run_operation job should NOT have the skip condition but should have its own activation condition
	// and should exclude safe_outputs
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

	// apply_safe_outputs job should be triggered when operation == 'safe_outputs'
	applyIdx := strings.Index(yaml, "\n  apply_safe_outputs:")
	if applyIdx == -1 {
		t.Errorf("Job apply_safe_outputs not found in generated workflow")
	} else {
		applySection := yaml[applyIdx : applyIdx+runOpSectionSearchRange]
		if !strings.Contains(applySection, applySafeOutputsCondition) {
			t.Errorf("Job apply_safe_outputs should have the activation condition %q in:\n%s", applySafeOutputsCondition, applySection)
		}
	}

	// create_labels job should be triggered when operation == 'create_labels'
	createLabelsIdx := strings.Index(yaml, "\n  create_labels:")
	if createLabelsIdx == -1 {
		t.Errorf("Job create_labels not found in generated workflow")
	} else {
		createLabelsSection := yaml[createLabelsIdx : createLabelsIdx+runOpSectionSearchRange]
		if !strings.Contains(createLabelsSection, createLabelsCondition) {
			t.Errorf("Job create_labels should have the activation condition %q in:\n%s", createLabelsCondition, createLabelsSection)
		}
	}

	// validate_workflows job should be triggered when operation == 'validate'
	validateCondition := `github.event_name == 'workflow_dispatch' && github.event.inputs.operation == 'validate'`
	validateIdx := strings.Index(yaml, "\n  validate_workflows:")
	if validateIdx == -1 {
		t.Errorf("Job validate_workflows not found in generated workflow")
	} else {
		validateSection := yaml[validateIdx : validateIdx+runOpSectionSearchRange]
		if !strings.Contains(validateSection, validateCondition) {
			t.Errorf("Job validate_workflows should have the activation condition %q in:\n%s", validateCondition, validateSection)
		}
	}

	// Verify create_labels is an option in the operation choices
	if !strings.Contains(yaml, "- 'create_labels'") {
		t.Error("workflow_dispatch operation choices should include 'create_labels'")
	}

	// Verify safe_outputs is an option in the operation choices
	if !strings.Contains(yaml, "- 'safe_outputs'") {
		t.Error("workflow_dispatch operation choices should include 'safe_outputs'")
	}

	// Verify clean_cache_memories is an option in the operation choices
	if !strings.Contains(yaml, "- 'clean_cache_memories'") {
		t.Error("workflow_dispatch operation choices should include 'clean_cache_memories'")
	}

	// Verify validate is an option in the operation choices
	if !strings.Contains(yaml, "- 'validate'") {
		t.Error("workflow_dispatch operation choices should include 'validate'")
	}

	// Verify run_url input exists in workflow_dispatch
	if !strings.Contains(yaml, "run_url:") {
		t.Error("workflow_dispatch should include run_url input")
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
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeRelease, "v0.47.4", false, nil)
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

		err := GenerateMaintenanceWorkflow(workflowDataListWithResolver, tmpDir, "v1.0.0", ActionModeRelease, "v0.47.4", false, nil)
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
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "v0.47.4", false, nil)
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
		result := generateInstallCLISteps(ActionModeDev, "v1.0.0", "", nil)
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
		result := generateInstallCLISteps(ActionModeRelease, "v1.0.0", "", nil)
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
		result := generateInstallCLISteps(ActionModeRelease, "v1.0.0", "v2.0.0", nil)
		if !strings.Contains(result, "setup-cli@v2.0.0") {
			t.Errorf("Release mode should use actionTag v2.0.0, got:\n%s", result)
		}
	})

	t.Run("release mode with resolver uses SHA-pinned setup-cli reference", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw/actions/setup-cli", "v1.0.0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		resolver := NewActionResolver(cache)

		result := generateInstallCLISteps(ActionModeRelease, "v1.0.0", "", resolver)
		expectedRef := "github/gh-aw/actions/setup-cli@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa # v1.0.0"
		if !strings.Contains(result, expectedRef) {
			t.Errorf("Release mode with resolver should use SHA-pinned setup-cli reference %q, got:\n%s", expectedRef, result)
		}
		// Must not contain the bare mutable tag
		if strings.Contains(result, "setup-cli@v1.0.0") {
			t.Errorf("Release mode with resolver must not use mutable tag setup-cli@v1.0.0, got:\n%s", result)
		}
	})

	t.Run("action mode with resolver uses SHA-pinned setup-cli reference", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw-actions/setup-cli", "v1.0.0", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		resolver := NewActionResolver(cache)

		result := generateInstallCLISteps(ActionModeAction, "v1.0.0", "", resolver)
		expectedRef := "github/gh-aw-actions/setup-cli@bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb # v1.0.0"
		if !strings.Contains(result, expectedRef) {
			t.Errorf("Action mode with resolver should use SHA-pinned setup-cli reference %q, got:\n%s", expectedRef, result)
		}
		// Must not contain the bare mutable tag
		if strings.Contains(result, "setup-cli@v1.0.0") {
			t.Errorf("Action mode with resolver must not use mutable tag setup-cli@v1.0.0, got:\n%s", result)
		}
	})

	t.Run("release mode without resolver falls back to tag reference", func(t *testing.T) {
		result := generateInstallCLISteps(ActionModeRelease, "v1.0.0", "", nil)
		if !strings.Contains(result, "github/gh-aw/actions/setup-cli@v1.0.0") {
			t.Errorf("Release mode without resolver should fall back to tag reference, got:\n%s", result)
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
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil)
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
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeRelease, "v1.0.0", false, nil)
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
		err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", ActionModeDev, "", false, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		// run_operation, create_labels, validate_workflows, and compile_workflows should use the same setup-go version
		// (all use GetActionPin, not hardcoded pins). Exactly 4 occurrences expected.
		setupGoPin := GetActionPin("actions/setup-go")
		occurrences := strings.Count(yaml, setupGoPin)
		if occurrences != 4 {
			t.Errorf("Expected exactly 4 occurrences of pinned setup-go ref %q (run_operation + create_labels + validate_workflows + compile_workflows), got %d in:\n%s",
				setupGoPin, occurrences, yaml)
		}
	})
}

func TestGenerateMaintenanceWorkflow_SetupCLISHAPinning(t *testing.T) {
	setupCLISHA := "cccccccccccccccccccccccccccccccccccccccc"

	workflowDataListWithResolver := func(resolver *ActionResolver) []*WorkflowData {
		return []*WorkflowData{
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
	}

	t.Run("release mode with resolver SHA-pins setup-cli in run_operation", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw/actions/setup-cli", "v1.0.0", setupCLISHA)
		// Also seed the setup action to keep the test hermetic (GenerateMaintenanceWorkflow
		// calls ResolveSetupActionReference with the same resolver, which would otherwise
		// attempt a real gh api call on a cache miss).
		cache.Set("github/gh-aw/actions/setup", "v1.0.0", "dddddddddddddddddddddddddddddddddddddddd")
		resolver := NewActionResolver(cache)

		err := GenerateMaintenanceWorkflow(workflowDataListWithResolver(resolver), tmpDir, "v1.0.0", ActionModeRelease, "v1.0.0", false, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		expectedRef := "github/gh-aw/actions/setup-cli@" + setupCLISHA + " # v1.0.0"
		if !strings.Contains(yaml, expectedRef) {
			t.Errorf("Expected SHA-pinned setup-cli reference %q in generated workflow, got:\n%s", expectedRef, yaml)
		}
		// Bare tag must not appear
		if strings.Contains(yaml, "setup-cli@v1.0.0") {
			t.Errorf("Generated workflow must not use mutable tag setup-cli@v1.0.0; got:\n%s", yaml)
		}
	})
}

func TestGenerateMaintenanceWorkflow_RepoConfig(t *testing.T) {
	// makeList returns a fresh workflow data list for each sub-test to avoid
	// shared-state issues between parallel or repeated sub-tests.
	makeList := func() []*WorkflowData {
		return []*WorkflowData{
			{
				Name: "test-workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{Expires: 24},
				},
			},
		}
	}

	t.Run("custom string runs_on is used in all jobs", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &RepoConfig{
			Maintenance: &MaintenanceConfig{RunsOn: RunsOnValue{"my-custom-runner"}},
		}
		err := GenerateMaintenanceWorkflow(makeList(), tmpDir, "v1.0.0", ActionModeDev, "", false, cfg)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		if !strings.Contains(yaml, "runs-on: my-custom-runner") {
			t.Errorf("Expected 'runs-on: my-custom-runner' in generated workflow, got:\n%s", yaml)
		}
		// Default runner must not appear
		if strings.Contains(yaml, "runs-on: ubuntu-slim") {
			t.Errorf("Generated workflow must not use default runner 'ubuntu-slim' when overridden; got:\n%s", yaml)
		}
	})

	t.Run("array runs_on is used in all jobs", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &RepoConfig{
			Maintenance: &MaintenanceConfig{RunsOn: RunsOnValue{"self-hosted", "linux"}},
		}
		err := GenerateMaintenanceWorkflow(makeList(), tmpDir, "v1.0.0", ActionModeDev, "", false, cfg)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		if !strings.Contains(yaml, `runs-on: ["self-hosted","linux"]`) {
			t.Errorf("Expected array runs-on in generated workflow, got:\n%s", yaml)
		}
	})

	t.Run("maintenance disabled deletes existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a pre-existing maintenance file to be deleted
		maintenanceFile := filepath.Join(tmpDir, "agentics-maintenance.yml")
		if err := os.WriteFile(maintenanceFile, []byte("existing content"), 0o600); err != nil {
			t.Fatalf("Failed to write pre-existing file: %v", err)
		}
		cfg := &RepoConfig{MaintenanceDisabled: true}
		err := GenerateMaintenanceWorkflow(makeList(), tmpDir, "v1.0.0", ActionModeDev, "", false, cfg)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if _, statErr := os.Stat(maintenanceFile); !os.IsNotExist(statErr) {
			t.Errorf("Expected maintenance workflow to be deleted when disabled, but file still exists")
		}
	})

	t.Run("maintenance disabled skips generation even with expires", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &RepoConfig{MaintenanceDisabled: true}
		err := GenerateMaintenanceWorkflow(makeList(), tmpDir, "v1.0.0", ActionModeDev, "", false, cfg)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); !os.IsNotExist(statErr) {
			t.Errorf("Expected no maintenance workflow to be generated when disabled")
		}
	})

	t.Run("maintenance disabled with expires emits warning (no error)", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Workflow with expires configured – maintenance is disabled in aw.json.
		list := []*WorkflowData{
			{
				Name: "my-workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{Expires: 48},
				},
			},
		}
		cfg := &RepoConfig{MaintenanceDisabled: true}
		// The function must succeed (no error), even though a warning is printed.
		err := GenerateMaintenanceWorkflow(list, tmpDir, "v1.0.0", ActionModeDev, "", false, cfg)
		if err != nil {
			t.Fatalf("Expected no error when maintenance is disabled with expires, got: %v", err)
		}
		// The maintenance workflow must not be generated.
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); !os.IsNotExist(statErr) {
			t.Errorf("Expected no maintenance workflow file when maintenance is disabled")
		}
	})
}
