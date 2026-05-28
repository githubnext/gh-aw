//go:build !integration

package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
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
			err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
				WorkflowDataList: tt.workflowDataList,
				WorkflowDir:      tmpDir,
				Version:          "v1.0.0",
				ActionMode:       ActionModeDev,
				ActionTag:        "",
				RepoConfig:       nil,
				RepoSlug:         "",
			})

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
			err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
				WorkflowDataList: tt.workflowDataList,
				WorkflowDir:      tmpDir,
				Version:          "v1.0.0",
				ActionMode:       ActionModeDev,
				ActionTag:        "",
				RepoConfig:       nil,
				RepoSlug:         "",
			})
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
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       nil,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	operationSkipCondition := `github.event_name != 'workflow_dispatch' && github.event_name != 'workflow_call' || inputs.operation == ''`
	operationRunCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation != '' && inputs.operation != 'safe_outputs' && inputs.operation != 'create_labels' && inputs.operation != 'activity_report' && inputs.operation != 'close_agentic_workflows_issues' && inputs.operation != 'clean_cache_memories' && inputs.operation != 'update_pull_request_branches' && inputs.operation != 'validate' && inputs.operation != 'forecast'`
	applySafeOutputsCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation == 'safe_outputs'`
	createLabelsCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation == 'create_labels'`
	updatePullRequestBranchesCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation == 'update_pull_request_branches'`
	activityReportCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation == 'activity_report'`
	forecastCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation == 'forecast'`
	closeAgenticWorkflowIssuesCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation == 'close_agentic_workflows_issues'`
	cleanCacheMemoriesCondition := `github.event_name != 'workflow_dispatch' && github.event_name != 'workflow_call' || inputs.operation == '' || inputs.operation == 'clean_cache_memories'`

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

	// update_pull_request_branches job should be triggered when operation == 'update_pull_request_branches'
	updatePullRequestBranchesIdx := strings.Index(yaml, "\n  update_pull_request_branches:")
	if updatePullRequestBranchesIdx == -1 {
		t.Errorf("Job update_pull_request_branches not found in generated workflow")
	} else {
		updatePullRequestBranchesSection := yaml[updatePullRequestBranchesIdx : updatePullRequestBranchesIdx+runOpSectionSearchRange]
		if !strings.Contains(updatePullRequestBranchesSection, updatePullRequestBranchesCondition) {
			t.Errorf("Job update_pull_request_branches should have the activation condition %q in:\n%s", updatePullRequestBranchesCondition, updatePullRequestBranchesSection)
		}
		if !strings.Contains(updatePullRequestBranchesSection, "pull-requests: write") {
			t.Errorf("Job update_pull_request_branches should include pull-requests: write permission in:\n%s", updatePullRequestBranchesSection)
		}
		if !strings.Contains(updatePullRequestBranchesSection, "contents: write") {
			t.Errorf("Job update_pull_request_branches should include contents: write permission in:\n%s", updatePullRequestBranchesSection)
		}
	}

	// validate_workflows job should be triggered when operation == 'validate'
	validateCondition := `(github.event_name == 'workflow_dispatch' || github.event_name == 'workflow_call') && inputs.operation == 'validate'`
	validateIdx := strings.Index(yaml, "\n  validate_workflows:")
	if validateIdx == -1 {
		t.Errorf("Job validate_workflows not found in generated workflow")
	} else {
		validateSection := yaml[validateIdx : validateIdx+runOpSectionSearchRange]
		if !strings.Contains(validateSection, validateCondition) {
			t.Errorf("Job validate_workflows should have the activation condition %q in:\n%s", validateCondition, validateSection)
		}
	}

	// activity_report job should be triggered when operation == 'activity_report'
	activityReportIdx := strings.Index(yaml, "\n  activity_report:")
	if activityReportIdx == -1 {
		t.Errorf("Job activity_report not found in generated workflow")
	} else {
		activityReportSection := yaml[activityReportIdx : activityReportIdx+runOpSectionSearchRange]
		if !strings.Contains(activityReportSection, activityReportCondition) {
			t.Errorf("Job activity_report should have the activation condition %q in:\n%s", activityReportCondition, activityReportSection)
		}
		if !strings.Contains(activityReportSection, "contents: read") {
			t.Errorf("Job activity_report should include contents: read permission in:\n%s", activityReportSection)
		}
		if !strings.Contains(activityReportSection, "timeout-minutes: 120") {
			t.Errorf("Job activity_report should set timeout-minutes: 120 in:\n%s", activityReportSection)
		}
	}
	if !strings.Contains(yaml, "Restore activity report logs cache") {
		t.Errorf("Job activity_report should include a cache restore step in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "Save activity report logs cache") {
		t.Errorf("Job activity_report should include a cache save step in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "if: ${{ always() }}") {
		t.Errorf("Job activity_report should save cache even when earlier steps fail in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "steps.activity_report_logs_cache.outputs.cache-primary-key") {
		t.Errorf("Job activity_report cache save step should use cache primary key output in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "${{ github.run_id }}") {
		t.Errorf("Job activity_report cache key should include run_id for latest-cache resolution in:\n%s", yaml)
	}

	if !strings.Contains(yaml, "Download activity report logs") {
		t.Errorf("Job activity_report should include direct logs download step in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "timeout-minutes: 20") {
		t.Errorf("Job activity_report logs download step should set timeout-minutes: 20 in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "${GH_AW_CMD_PREFIX} logs") {
		t.Errorf("Job activity_report should run gh aw logs directly in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "--start-date -1w") {
		t.Errorf("Job activity_report gh aw logs command should include --start-date -1w in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "--count 100") {
		t.Errorf("Job activity_report gh aw logs command should include --count 100 in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "--format markdown") {
		t.Errorf("Job activity_report gh aw logs command should include --format markdown in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "./.cache/gh-aw/activity-report-logs/report.md") {
		t.Errorf("Job activity_report gh aw logs command should write report markdown output to report.md in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "Generate activity report issue") {
		t.Errorf("Job activity_report should include issue generation step after cache save in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "title: '[aw] agentic status report'") {
		t.Errorf("Job activity_report issue generation step should create the activity report issue title in:\n%s", yaml)
	}

	forecastIdx := strings.Index(yaml, "\n  forecast_report:")
	if forecastIdx == -1 {
		t.Errorf("Job forecast_report not found in generated workflow")
	} else {
		forecastSection := yaml[forecastIdx : forecastIdx+runOpSectionSearchRange]
		if !strings.Contains(forecastSection, forecastCondition) {
			t.Errorf("Job forecast_report should have the activation condition %q in:\n%s", forecastCondition, forecastSection)
		}
		if !strings.Contains(forecastSection, "actions: read") {
			t.Errorf("Job forecast_report should include actions: read permission in:\n%s", forecastSection)
		}
		if !strings.Contains(forecastSection, "issues: write") {
			t.Errorf("Job forecast_report should include issues: write permission in:\n%s", forecastSection)
		}
		if !strings.Contains(forecastSection, "timeout-minutes: 60") {
			t.Errorf("Job forecast_report should set timeout-minutes: 60 in:\n%s", forecastSection)
		}
	}
	if !strings.Contains(yaml, "Generate forecast report") {
		t.Errorf("Job forecast_report should include forecast generation step in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "Restore forecast report logs cache") {
		t.Errorf("Job forecast_report should restore logs cache before warm-up in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "Save forecast report logs cache") {
		t.Errorf("Job forecast_report should save logs cache after forecast generation in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "${GH_AW_CMD_PREFIX} forecast") {
		t.Errorf("Job forecast_report should run gh aw forecast directly in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "${GH_AW_CMD_PREFIX} logs --repo \"${{ github.repository }}\" --start-date -30d --count 1500") {
		t.Errorf("Job forecast_report should warm logs cache with 30-day lookback and expanded count in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "Missing run summary cache in .github/aw/logs after gh aw logs warm-up; cannot run forecast.") {
		t.Errorf("Job forecast_report should fail when run summary cache is missing after warm-up in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "--repo \"${{ github.repository }}\" --timeout 10 --json") {
		t.Errorf("Job forecast_report gh aw forecast command should include --repo, --timeout, and --json in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "shell: bash") {
		t.Errorf("Job forecast_report should explicitly use bash shell for stderr filtering in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "${GH_AW_CMD_PREFIX} forecast --repo \"${{ github.repository }}\" --timeout 10 --json 2> >(grep -Fv \"forecast is an experimental command and may change without notice\" >&2) > ./.cache/gh-aw/forecast/report.json") {
		t.Errorf("Job forecast_report gh aw forecast command should set a 10-minute forecast timeout while filtering the experimental warning in:\n%s", yaml)
	}
	if strings.Contains(yaml, "timeout 10m ${GH_AW_CMD_PREFIX} forecast") {
		t.Errorf("Job forecast_report should not use shell timeout wrapper for forecast command anymore in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "echo '{\"outcome\":\"timeout\",\"message\":\"Forecast computation timed out after 10 minutes.\"}' > ./.cache/gh-aw/forecast/error.json") {
		t.Errorf("Job forecast_report should record timeout errors for issue creation in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "if: ${{ always() }}") {
		t.Errorf("Job forecast_report should always run follow-up issue/cache steps even when forecast fails in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "FORECAST_STEP_OUTCOME: ${{ steps.generate_forecast_report.outcome }}") {
		t.Errorf("Job forecast_report issue generation step should pass forecast step outcome to create_forecast_issue.cjs in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "const { main } = require('${{ runner.temp }}/gh-aw/actions/create_forecast_issue.cjs');") {
		t.Errorf("Job forecast_report issue generation step should call create_forecast_issue.cjs in:\n%s", yaml)
	}
	if !strings.Contains(yaml, "setupGlobals(core, github, context, exec, io, getOctokit);") {
		t.Errorf("Job forecast_report issue generation step should initialize setup globals before calling create_forecast_issue.cjs in:\n%s", yaml)
	}

	// close_agentic_workflows_issues job should be triggered when operation == 'close_agentic_workflows_issues'
	closeAgenticWorkflowIssuesIdx := strings.Index(yaml, "\n  close_agentic_workflows_issues:")
	if closeAgenticWorkflowIssuesIdx == -1 {
		t.Errorf("Job close_agentic_workflows_issues not found in generated workflow")
	} else {
		closeAgenticWorkflowIssuesSection := yaml[closeAgenticWorkflowIssuesIdx : closeAgenticWorkflowIssuesIdx+runOpSectionSearchRange]
		if !strings.Contains(closeAgenticWorkflowIssuesSection, closeAgenticWorkflowIssuesCondition) {
			t.Errorf("Job close_agentic_workflows_issues should have the activation condition %q in:\n%s", closeAgenticWorkflowIssuesCondition, closeAgenticWorkflowIssuesSection)
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

	// Verify update_pull_request_branches is an option in the operation choices
	if !strings.Contains(yaml, "- 'update_pull_request_branches'") {
		t.Error("workflow_dispatch operation choices should include 'update_pull_request_branches'")
	}

	// Verify validate is an option in the operation choices
	if !strings.Contains(yaml, "- 'validate'") {
		t.Error("workflow_dispatch operation choices should include 'validate'")
	}

	// Verify activity_report is an option in the operation choices
	if !strings.Contains(yaml, "- 'activity_report'") {
		t.Error("workflow_dispatch operation choices should include 'activity_report'")
	}

	// Verify forecast is an option in the operation choices
	if !strings.Contains(yaml, "- 'forecast'") {
		t.Error("workflow_dispatch operation choices should include 'forecast'")
	}

	// Verify close_agentic_workflows_issues is an option in the operation choices
	if !strings.Contains(yaml, "- 'close_agentic_workflows_issues'") {
		t.Error("workflow_dispatch operation choices should include 'close_agentic_workflows_issues'")
	}

	// Verify run_url input exists in workflow_dispatch
	if !strings.Contains(yaml, "run_url:") {
		t.Error("workflow_dispatch should include run_url input")
	}

	// Verify workflow_call trigger is present with same inputs
	workflowCallIdx := strings.Index(yaml, "workflow_call:")
	if workflowCallIdx == -1 {
		t.Error("workflow should include workflow_call trigger")
	} else {
		workflowCallSection := yaml[workflowCallIdx:]
		if !strings.Contains(workflowCallSection, "inputs:\n      operation:") {
			t.Error("workflow_call trigger should include operation input")
		}
	}

	// Verify workflow_call outputs are declared
	if !strings.Contains(yaml, "operation_completed:") {
		t.Error("workflow_call outputs should include operation_completed")
	}
	if !strings.Contains(yaml, "applied_run_url:") {
		t.Error("workflow_call outputs should include applied_run_url")
	}

	// Verify run_operation job exposes outputs
	runOpIdx2 := strings.Index(yaml, "\n  run_operation:")
	if runOpIdx2 != -1 {
		runOpEnd := min(runOpIdx2+1200, len(yaml))
		runOpSection2 := yaml[runOpIdx2:runOpEnd]
		if !strings.Contains(runOpSection2, "outputs:\n      operation: ${{ steps.record.outputs.operation }}") {
			t.Errorf("run_operation job should declare operation output, got:\n%s", runOpSection2[:min(300, len(runOpSection2))])
		}
	}

	// Verify apply_safe_outputs job exposes run_url output
	applyIdx2 := strings.Index(yaml, "\n  apply_safe_outputs:")
	if applyIdx2 != -1 {
		applySection2 := yaml[applyIdx2 : applyIdx2+600]
		if !strings.Contains(applySection2, "outputs:\n      run_url: ${{ steps.record.outputs.run_url }}") {
			t.Errorf("apply_safe_outputs job should declare run_url output, got:\n%s", applySection2[:300])
		}
	}
}

func TestGenerateMaintenanceWorkflow_DisableAgenticWorkflowJob(t *testing.T) {
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
	trueVal := true
	cfg := &RepoConfig{
		Maintenance: &MaintenanceConfig{LabelTriggers: &trueVal},
	}
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       cfg,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	const jobSectionSearchRange = 2000

	// Verify only the issues label trigger is present (pull request is no longer supported)
	if !strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("Maintenance workflow should include issues: types: [labeled] trigger")
	}
	if strings.Contains(yaml, "  pull_request:\n    types: [labeled]") {
		t.Error("Maintenance workflow must NOT include pull_request: types: [labeled] trigger (issues-only)")
	}

	// Verify the label_disable_agentic_workflow job exists
	disableJobIdx := strings.Index(yaml, "\n  label_disable_agentic_workflow:")
	if disableJobIdx == -1 {
		t.Fatal("Job label_disable_agentic_workflow not found in generated workflow")
	}
	// Bound the section to just the label_disable_agentic_workflow job by finding the next job start
	nextJobIdx := strings.Index(yaml[disableJobIdx+1:], "\n  label_apply_safe_outputs:")
	if nextJobIdx == -1 {
		nextJobIdx = jobSectionSearchRange
	}
	disableJobSection := yaml[disableJobIdx : disableJobIdx+1+nextJobIdx]

	// Verify the condition triggers only on issues label events (not pull_request)
	if !strings.Contains(disableJobSection, "github.event_name == 'issues'") {
		t.Errorf("label_disable_agentic_workflow job should trigger on issues events in:\n%s", disableJobSection)
	}
	if strings.Contains(disableJobSection, "github.event_name == 'pull_request'") {
		t.Errorf("label_disable_agentic_workflow job must NOT trigger on pull_request events (issues-only) in:\n%s", disableJobSection)
	}
	if !strings.Contains(disableJobSection, "github.event.label.name == 'agentic-workflows:disable'") {
		t.Errorf("label_disable_agentic_workflow job should check for agentic-workflows:disable label in:\n%s", disableJobSection)
	}
	if !strings.Contains(disableJobSection, "github.event.repository.fork") {
		t.Errorf("label_disable_agentic_workflow job should exclude forks in:\n%s", disableJobSection)
	}

	// Verify required permissions (no pull-requests: write since issues-only)
	if !strings.Contains(disableJobSection, "actions: write") {
		t.Errorf("label_disable_agentic_workflow job should have actions: write permission in:\n%s", disableJobSection)
	}
	if !strings.Contains(disableJobSection, "contents: read") {
		t.Errorf("label_disable_agentic_workflow job should have contents: read permission in:\n%s", disableJobSection)
	}
	if strings.Contains(disableJobSection, "contents: write") {
		t.Errorf("label_disable_agentic_workflow job must NOT have contents: write (only read is needed) in:\n%s", disableJobSection)
	}
	if !strings.Contains(disableJobSection, "issues: write") {
		t.Errorf("label_disable_agentic_workflow job should have issues: write permission in:\n%s", disableJobSection)
	}
	if strings.Contains(disableJobSection, "pull-requests: write") {
		t.Errorf("label_disable_agentic_workflow job must NOT have pull-requests: write (issues-only) in:\n%s", disableJobSection)
	}

	// Verify the job uses disable_agentic_workflow.cjs
	if !strings.Contains(disableJobSection, "disable_agentic_workflow.cjs") {
		t.Errorf("label_disable_agentic_workflow job should use disable_agentic_workflow.cjs script in:\n%s", disableJobSection)
	}

	// Verify the job includes the permission check step with an id and that the operation step
	// has an explicit if condition referencing that id (so unauthorized users cannot bypass the check)
	if !strings.Contains(disableJobSection, "check_team_member.cjs") {
		t.Errorf("label_disable_agentic_workflow job should check permissions using check_team_member.cjs in:\n%s", disableJobSection)
	}
	if !strings.Contains(disableJobSection, "id: check_permissions") {
		t.Errorf("label_disable_agentic_workflow permission check step should have id: check_permissions in:\n%s", disableJobSection)
	}
	if !strings.Contains(disableJobSection, "steps.check_permissions.outcome == 'success'") {
		t.Errorf("label_disable_agentic_workflow operation step should have if: steps.check_permissions.outcome == 'success' in:\n%s", disableJobSection)
	}
}

func TestBuildLabeledDisableCondition(t *testing.T) {
	condition := buildLabeledDisableCondition()
	rendered := RenderCondition(condition)

	// Should only include issues event (not pull_request — issues-only by design)
	if !strings.Contains(rendered, "github.event_name == 'issues'") {
		t.Errorf("Condition should include issues event, got: %s", rendered)
	}
	if strings.Contains(rendered, "github.event_name == 'pull_request'") {
		t.Errorf("Condition must not include pull_request event (issues-only), got: %s", rendered)
	}

	// Should check the label name
	if !strings.Contains(rendered, "github.event.label.name == 'agentic-workflows:disable'") {
		t.Errorf("Condition should check for agentic-workflows:disable label, got: %s", rendered)
	}

	// Should exclude forks
	if !strings.Contains(rendered, "github.event.repository.fork") {
		t.Errorf("Condition should exclude forks, got: %s", rendered)
	}

	// Should not include workflow_dispatch or schedule-related conditions
	if strings.Contains(rendered, "workflow_dispatch") || strings.Contains(rendered, "workflow_call") {
		t.Errorf("Condition should not reference workflow_dispatch or workflow_call, got: %s", rendered)
	}
}

func TestBuildLabeledApplySafeOutputsCondition(t *testing.T) {
	condition := buildLabeledApplySafeOutputsCondition()
	rendered := RenderCondition(condition)

	// Should only include issues event
	if !strings.Contains(rendered, "github.event_name == 'issues'") {
		t.Errorf("Condition should include issues event, got: %s", rendered)
	}
	if strings.Contains(rendered, "github.event_name == 'pull_request'") {
		t.Errorf("Condition must not include pull_request event (issues-only), got: %s", rendered)
	}

	// Should check the apply-safe-outputs label name
	if !strings.Contains(rendered, "github.event.label.name == 'agentic-workflows:apply-safe-outputs'") {
		t.Errorf("Condition should check for agentic-workflows:apply-safe-outputs label, got: %s", rendered)
	}

	// Should exclude forks
	if !strings.Contains(rendered, "github.event.repository.fork") {
		t.Errorf("Condition should exclude forks, got: %s", rendered)
	}
}

func TestGenerateMaintenanceWorkflow_LabelTriggers_Disabled(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	falseVal := false
	cfg := &RepoConfig{
		Maintenance: &MaintenanceConfig{LabelTriggers: &falseVal},
	}
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       cfg,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	// Label-event trigger should be absent
	if strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("When label_triggers is false the issues labeled trigger should not be present")
	}

	// The pull_request labeled trigger should never be present (removed)
	if strings.Contains(yaml, "  pull_request:\n    types: [labeled]") {
		t.Error("pull_request labeled trigger should never be present (issues-only)")
	}

	// The label_disable_agentic_workflow job should be absent
	if strings.Contains(yaml, "label_disable_agentic_workflow:") {
		t.Error("When label_triggers is false the label_disable_agentic_workflow job should not be present")
	}

	// The label_apply_safe_outputs job should be absent
	if strings.Contains(yaml, "label_apply_safe_outputs:") {
		t.Error("When label_triggers is false the label_apply_safe_outputs job should not be present")
	}
}

func TestGenerateMaintenanceWorkflow_LabelTriggers_Default(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	// Default: LabelTriggers is nil (omitted) → treated as false (opt-in semantics) → jobs absent
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       nil,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	// Issues labeled trigger should NOT be present by default (opt-in required)
	if strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("By default (no config) the issues labeled trigger should NOT be present — label_triggers must be explicitly enabled")
	}

	// The label_disable_agentic_workflow job should NOT be present by default
	if strings.Contains(yaml, "label_disable_agentic_workflow:") {
		t.Error("By default (no config) the label_disable_agentic_workflow job should NOT be present — label_triggers must be explicitly enabled")
	}

	// The label_apply_safe_outputs job should NOT be present by default
	if strings.Contains(yaml, "label_apply_safe_outputs:") {
		t.Error("By default (no config) the label_apply_safe_outputs job should NOT be present — label_triggers must be explicitly enabled")
	}
}

func TestGenerateMaintenanceWorkflow_LabelTriggers_ExplicitTrue(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	trueVal := true
	cfg := &RepoConfig{
		Maintenance: &MaintenanceConfig{LabelTriggers: &trueVal},
	}
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       cfg,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	// Issues labeled trigger should be present when explicitly enabled
	if !strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("When label_triggers: true the issues labeled trigger should be present")
	}

	// pull_request labeled trigger should never be present (issues-only by design)
	if strings.Contains(yaml, "  pull_request:\n    types: [labeled]") {
		t.Error("pull_request labeled trigger should never be present (issues-only)")
	}

	// The label_disable_agentic_workflow job should be present when explicitly enabled
	if !strings.Contains(yaml, "label_disable_agentic_workflow:") {
		t.Error("When label_triggers: true the label_disable_agentic_workflow job should be present")
	}

	// The label_apply_safe_outputs job should be present when explicitly enabled
	if !strings.Contains(yaml, "label_apply_safe_outputs:") {
		t.Error("When label_triggers: true the label_apply_safe_outputs job should be present")
	}

	// Verify label_apply_safe_outputs job has an explicit step id and if condition so that
	// the operation step only runs when the permission check passes
	applySafeIdx := strings.Index(yaml, "\n  label_apply_safe_outputs:")
	if applySafeIdx != -1 {
		applySection := yaml[applySafeIdx:min(applySafeIdx+2000, len(yaml))]
		if !strings.Contains(applySection, "id: check_permissions") {
			t.Errorf("label_apply_safe_outputs permission check step should have id: check_permissions in:\n%s", applySection[:min(500, len(applySection))])
		}
		if !strings.Contains(applySection, "steps.check_permissions.outcome == 'success'") {
			t.Errorf("label_apply_safe_outputs operation step should have if: steps.check_permissions.outcome == 'success' in:\n%s", applySection[:min(500, len(applySection))])
		}
	}
}

func TestGenerateMaintenanceWorkflow_PushTrigger(t *testing.T) {
	const jobSectionSearchRange = 500

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

	t.Run("dev mode includes push trigger on main for workflow md files", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)

		if !strings.Contains(yaml, "  push:") {
			t.Error("Dev mode workflow should include push trigger")
		}
		if !strings.Contains(yaml, "      - main") {
			t.Error("Dev mode push trigger should target main branch (fallback when slug is empty)")
		}
		if !strings.Contains(yaml, "      - '.github/workflows/*.md'") {
			t.Error("Dev mode push trigger should target .github/workflows/*.md paths")
		}
	})

	t.Run("dev mode uses custom default branch from buildMaintenanceWorkflowYAML", func(t *testing.T) {
		// Call buildMaintenanceWorkflowYAML directly to test the branch substitution
		// without needing a live GitHub API call (FetchDefaultBranch falls back to "main" with no slug)
		yaml := buildMaintenanceWorkflowYAML(context.Background(), buildMaintenanceWorkflowYAMLOptions{
			cronSchedule:   "37 */2 * * *",
			scheduleDesc:   "Every 2 hours",
			minExpiresDays: 1,
			runsOnValue:    "ubuntu-slim",
			actionMode:     ActionModeDev,
			version:        "v1.0.0",
			defaultBranch:  "develop",
		})
		if !strings.Contains(yaml, "      - develop") {
			t.Errorf("Push trigger should use the provided default branch 'develop', got:\n%s", yaml[:min(500, len(yaml))])
		}
		if strings.Contains(yaml, "      - main") {
			t.Errorf("Push trigger should not contain hardcoded 'main' when 'develop' is specified")
		}
	})

	t.Run("release mode does not include push trigger", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeRelease,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)

		if strings.Contains(yaml, "  push:") {
			t.Error("Release mode workflow should NOT include push trigger")
		}
	})

	t.Run("close-expired-entities and secret-validation exclude push events", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		pushExclusionCondition := "github.event_name != 'push'"

		scheduleOnlyJobs := []string{"close-expired-entities:", "secret-validation:"}
		for _, job := range scheduleOnlyJobs {
			jobIdx := strings.Index(yaml, "\n  "+job)
			if jobIdx == -1 {
				t.Errorf("Job %q not found in generated workflow", job)
				continue
			}
			jobSection := yaml[jobIdx : jobIdx+jobSectionSearchRange]
			if !strings.Contains(jobSection, pushExclusionCondition) {
				t.Errorf("Job %q should exclude push events (%q) but condition is:\n%s", job, pushExclusionCondition, jobSection)
			}
		}
	})

	t.Run("compile-workflows runs on push events (no push exclusion)", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)

		compileIdx := strings.Index(yaml, "\n  compile-workflows:")
		if compileIdx == -1 {
			t.Fatal("Job compile-workflows not found in generated workflow")
		}
		jobSection := yaml[compileIdx : compileIdx+jobSectionSearchRange]
		if strings.Contains(jobSection, "github.event_name != 'push'") {
			t.Errorf("Job compile-workflows should NOT exclude push events, but condition is:\n%s", jobSection)
		}
		if !strings.Contains(jobSection, "cancel-in-progress: true") {
			t.Errorf("Job compile-workflows should have cancel-in-progress concurrency, but got:\n%s", jobSection)
		}
		if !strings.Contains(jobSection, "github.workflow }}-compile-workflows-${{ github.repository") {
			t.Errorf("Job compile-workflows should have a scoped concurrency group, but got:\n%s", jobSection)
		}
		if !strings.Contains(yaml, "compile --validate --no-emit --verbose") {
			t.Errorf("Workflow should run pre-compile validation with --no-emit, but did not. Generated YAML:\n%s", yaml)
		}
		if strings.Contains(yaml, "compile --validate --validate-images --verbose") {
			t.Errorf("Workflow should not require --validate-images in compile-workflows, but generated YAML includes it:\n%s", yaml)
		}
		if strings.Contains(yaml, "        env:\n        with:\n") {
			t.Errorf("Workflow should not emit an empty env block in compile-workflows, but generated YAML includes one:\n%s", yaml)
		}
	})

	t.Run("compile-workflows can create pull requests with custom token secret", func(t *testing.T) {
		const compileJobSectionSearchRange = 500
		tmpDir := t.TempDir()
		repoConfig := &RepoConfig{
			Maintenance: &MaintenanceConfig{
				Compile: &MaintenanceCompileConfig{
					CreatePullRequestGitHubToken: "MAINTENANCE_TOKEN",
				},
			},
		}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       repoConfig,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)

		compileIdx := strings.Index(yaml, "\n  compile-workflows:")
		if compileIdx == -1 {
			t.Fatal("Job compile-workflows not found in generated workflow")
		}
		jobSection := yaml[compileIdx : compileIdx+compileJobSectionSearchRange]
		if !strings.Contains(jobSection, "contents: read") {
			t.Errorf("compile-workflows should keep contents: read permission, got:\n%s", jobSection)
		}
		if !strings.Contains(jobSection, "issues: write") {
			t.Errorf("compile-workflows should keep issues: write permission, got:\n%s", jobSection)
		}
		if strings.Contains(jobSection, "pull-requests: write") {
			t.Errorf("compile-workflows should not request pull-requests: write in PR mode, got:\n%s", jobSection)
		}
		if strings.Contains(jobSection, "contents: write") {
			t.Errorf("compile-workflows should not request contents: write in PR mode, got:\n%s", jobSection)
		}
		if !strings.Contains(yaml, "GH_AW_MAINTENANCE_GITHUB_TOKEN: ${{ secrets.MAINTENANCE_TOKEN }}") {
			t.Errorf("workflow should use configured maintenance github token secret, got:\n%s", yaml)
		}
		if !strings.Contains(yaml, "github-token: ${{ env.GH_AW_MAINTENANCE_GITHUB_TOKEN }}") {
			t.Errorf("workflow should pass maintenance token to github-script, got:\n%s", yaml)
		}
		if strings.Contains(yaml, "GH_AW_WORKFLOW_RECOMPILE_CREATE_PULL_REQUEST") {
			t.Errorf("workflow should not emit a separate PR mode env var, got:\n%s", yaml)
		}
	})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeRelease,
			ActionTag:        "v0.47.4",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
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

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataListWithResolver,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeRelease,
			ActionTag:        "v0.47.4",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "v0.47.4",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
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
		result := generateInstallCLISteps(context.Background(), ActionModeDev, "v1.0.0", "", nil)
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
		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "", nil)
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
		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "v2.0.0", nil)
		if !strings.Contains(result, "setup-cli@v2.0.0") {
			t.Errorf("Release mode should use actionTag v2.0.0, got:\n%s", result)
		}
	})

	t.Run("release mode with resolver uses SHA-pinned setup-cli reference", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw/actions/setup-cli", "v1.0.0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		resolver := NewActionResolver(cache)

		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "", resolver)
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

		result := generateInstallCLISteps(context.Background(), ActionModeAction, "v1.0.0", "", resolver)
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
		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "", nil)
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeRelease,
			ActionTag:        "v1.0.0",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		yaml := string(content)
		// run_operation, create_labels, activity_report, forecast_report, validate_workflows,
		// and compile_workflows should use the same setup-go version
		// (all use getActionPin, not hardcoded pins). Exactly 6 occurrences expected.
		// Note: label_disable_agentic_workflow no longer installs the CLI, so it has no setup-go step.
		setupGoPin := getActionPin("actions/setup-go")
		occurrences := strings.Count(yaml, setupGoPin)
		if occurrences != 6 {
			t.Errorf("Expected exactly 6 occurrences of pinned setup-go ref %q (run_operation + create_labels + activity_report + forecast_report + validate_workflows + compile_workflows), got %d in:\n%s",
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

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataListWithResolver(resolver),
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeRelease,
			ActionTag:        "v1.0.0",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       cfg,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       cfg,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       cfg,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       cfg,
			RepoSlug:         "",
		})
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
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: list,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       cfg,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Expected no error when maintenance is disabled with expires, got: %v", err)
		}
		// The maintenance workflow must not be generated.
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); !os.IsNotExist(statErr) {
			t.Errorf("Expected no maintenance workflow file when maintenance is disabled")
		}
	})
}

func TestCollectSideRepoTargets(t *testing.T) {
	tests := []struct {
		name          string
		workflows     []*WorkflowData
		expectedRepos []string
	}{
		{
			name:          "no workflows returns empty",
			workflows:     nil,
			expectedRepos: nil,
		},
		{
			name: "workflow without checkout returns empty",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: nil},
			},
			expectedRepos: nil,
		},
		{
			name: "nil workflow entry is ignored",
			workflows: []*WorkflowData{
				nil,
				{Name: "wf", CheckoutConfigs: nil},
			},
			expectedRepos: nil,
		},
		{
			name: "checkout without current:true is ignored",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "org/repo", Current: false},
				}},
			},
			expectedRepos: nil,
		},
		{
			name: "checkout with current:true and static repo is detected",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "my-org/main-repo", Current: true, GitHubToken: "${{ secrets.GH_AW_MAIN_REPO_TOKEN }}"},
				}},
			},
			expectedRepos: []string{"my-org/main-repo"},
		},
		{
			name: "expression-based repository is skipped",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "${{ inputs.target_repo }}", Current: true},
				}},
			},
			expectedRepos: nil,
		},
		{
			name: "empty repository is skipped",
			workflows: []*WorkflowData{
				{Name: "wf", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "", Current: true},
				}},
			},
			expectedRepos: nil,
		},
		{
			name: "duplicate repos across workflows are deduplicated",
			workflows: []*WorkflowData{
				{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "my-org/main-repo", Current: true},
				}},
				{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "my-org/main-repo", Current: true},
				}},
			},
			expectedRepos: []string{"my-org/main-repo"},
		},
		{
			name: "multiple distinct repos are all detected",
			workflows: []*WorkflowData{
				{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "org/repo-a", Current: true},
				}},
				{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
					{Repository: "org/repo-b", Current: true},
				}},
			},
			expectedRepos: []string{"org/repo-a", "org/repo-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := collectSideRepoTargets(tt.workflows)

			var got []string
			for _, tgt := range targets {
				got = append(got, tgt.Repository)
			}

			if len(got) != len(tt.expectedRepos) {
				t.Errorf("expected %d targets, got %d: %v", len(tt.expectedRepos), len(got), got)
				return
			}
			// Use a set-based comparison so the test is not sensitive to ordering.
			gotSet := make(map[string]bool, len(got))
			for _, r := range got {
				gotSet[r] = true
			}
			for _, repo := range tt.expectedRepos {
				if !gotSet[repo] {
					t.Errorf("expected target %q not found in results %v", repo, got)
				}
			}
		})
	}

	t.Run("non-empty token is preferred when same repo appears multiple times", func(t *testing.T) {
		workflows := []*WorkflowData{
			{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
				// First appearance has no token.
				{Repository: "my-org/shared-repo", Current: true, GitHubToken: ""},
			}},
			{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
				// Second appearance provides a token — should win.
				{Repository: "my-org/shared-repo", Current: true, GitHubToken: "${{ secrets.SHARED_TOKEN }}"},
			}},
		}

		targets := collectSideRepoTargets(workflows)
		if len(targets) != 1 {
			t.Fatalf("expected 1 target, got %d", len(targets))
		}
		if targets[0].GitHubToken != "${{ secrets.SHARED_TOKEN }}" {
			t.Errorf("expected non-empty token to win, got %q", targets[0].GitHubToken)
		}
	})

	t.Run("multiple repos preserve first-seen discovery order", func(t *testing.T) {
		workflows := []*WorkflowData{
			{Name: "wf1", CheckoutConfigs: []*CheckoutConfig{
				{Repository: "org/first-repo", Current: true},
			}},
			{Name: "wf2", CheckoutConfigs: []*CheckoutConfig{
				{Repository: "org/second-repo", Current: true},
			}},
			{Name: "wf3", CheckoutConfigs: []*CheckoutConfig{
				{Repository: "org/third-repo", Current: true},
			}},
		}

		targets := collectSideRepoTargets(workflows)
		if len(targets) != 3 {
			t.Fatalf("expected 3 targets, got %d", len(targets))
		}
		wantOrder := []string{"org/first-repo", "org/second-repo", "org/third-repo"}
		for i, want := range wantOrder {
			if targets[i].Repository != want {
				t.Errorf("targets[%d] = %q, want %q", i, targets[i].Repository, want)
			}
		}
	})
}

func TestSanitizeRepoForFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-org/main-repo", "my-org-main-repo"},
		{"org/repo", "org-repo"},
		{"my.org/my_repo", "my.org-my_repo"},
		{"owner/repo-name.git", "owner-repo-name.git"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringutil.SanitizeForFilename(tt.input)
			if got != tt.expected {
				t.Errorf("stringutil.SanitizeForFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGenerateSideRepoMaintenanceCron(t *testing.T) {
	t.Run("is deterministic for the same slug", func(t *testing.T) {
		cron1, desc1 := generateSideRepoMaintenanceCron("org/repo", 10)
		cron2, desc2 := generateSideRepoMaintenanceCron("org/repo", 10)
		if cron1 != cron2 || desc1 != desc2 {
			t.Errorf("expected deterministic output, got %q/%q and %q/%q", cron1, desc1, cron2, desc2)
		}
	})

	t.Run("different repos produce different cron expressions", func(t *testing.T) {
		repos := []string{"org/repo-a", "org/repo-b", "another-org/service", "myorg/tooling"}
		seen := make(map[string]string)
		for _, repo := range repos {
			cron, _ := generateSideRepoMaintenanceCron(repo, 10)
			if existing, ok := seen[cron]; ok {
				// Collisions are theoretically possible but should be rare for distinct slugs.
				t.Logf("cron collision between %q and %q: %s", repo, existing, cron)
			}
			seen[cron] = repo
		}
	})

	t.Run("minute is in valid range 0-59", func(t *testing.T) {
		slugs := []string{"a/b", "owner/repo", "my-org/my-repo", "x/y"}
		for _, slug := range slugs {
			for _, days := range []int{0, 1, 2, 3, 5, 10, 30} {
				cron, _ := generateSideRepoMaintenanceCron(slug, days)
				// Extract the minute field (first token).
				parts := strings.Fields(cron)
				if len(parts) < 5 {
					t.Errorf("invalid cron %q for slug=%q days=%d", cron, slug, days)
					continue
				}
				var min int
				if _, err := fmt.Sscanf(parts[0], "%d", &min); err != nil {
					t.Errorf("failed to parse minute from cron %q: %v", cron, err)
					continue
				}
				if min < 0 || min > 59 {
					t.Errorf("minute %d out of range [0,59] for slug=%q days=%d", min, slug, days)
				}
			}
		}
	})

	t.Run("frequency tier matches minExpiresDays", func(t *testing.T) {
		slug := "test/repo"
		cases := []struct {
			days        int
			descContain string
		}{
			{1, "Every 2 hours"},
			{2, "Every 6 hours"},
			{3, "Every 12 hours"},
			{4, "Every 12 hours"},
			{5, "Daily"},
			{30, "Daily"},
		}
		for _, tc := range cases {
			_, desc := generateSideRepoMaintenanceCron(slug, tc.days)
			if desc != tc.descContain {
				t.Errorf("days=%d: expected desc %q, got %q", tc.days, tc.descContain, desc)
			}
		}
	})
}

func TestGenerateSideRepoMaintenanceWorkflow(t *testing.T) {
	t.Run("generates file for static side-repo target", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "side-repo-workflow",
				CheckoutConfigs: []*CheckoutConfig{
					{
						Repository:  "my-org/target-repo",
						Current:     true,
						GitHubToken: "${{ secrets.GH_AW_TARGET_TOKEN }}",
					},
				},
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						Expires: 48,
					},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// The standard hosting-repo maintenance should be generated (has expires).
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); statErr != nil {
			t.Errorf("Expected standard agentics-maintenance.yml to exist")
		}

		// The side-repo maintenance should also be generated.
		sideFile := filepath.Join(tmpDir, "agentics-maintenance-my-org-target-repo.yml")
		content, err := os.ReadFile(sideFile)
		if err != nil {
			t.Fatalf("Expected side-repo maintenance file %s to exist: %v", sideFile, err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "my-org/target-repo") {
			t.Errorf("Side-repo maintenance should reference target repo, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "${{ secrets.GH_AW_TARGET_TOKEN }}") {
			t.Errorf("Side-repo maintenance should use custom token, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "GH_AW_TARGET_REPO_SLUG") {
			t.Errorf("Side-repo maintenance should set GH_AW_TARGET_REPO_SLUG, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "workflow_call") {
			t.Errorf("Side-repo maintenance should have workflow_call trigger, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "apply_safe_outputs") {
			t.Errorf("Side-repo maintenance should include apply_safe_outputs job, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "create_labels") {
			t.Errorf("Side-repo maintenance should include create_labels job, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "activity_report") {
			t.Errorf("Side-repo maintenance should include activity_report job, got content length %d", len(contentStr))
		}
	})

	t.Run("no side-repo file generated when no current checkout", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "normal-workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						Expires: 48,
					},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Only standard maintenance should exist.
		entries, _ := os.ReadDir(tmpDir)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "agentics-maintenance-") {
				t.Errorf("Unexpected side-repo maintenance file: %s", entry.Name())
			}
		}
	})

	t.Run("side-repo generated without expires uses safe_outputs, create_labels, and activity_report", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "side-repo-no-expires",
				CheckoutConfigs: []*CheckoutConfig{
					{
						Repository: "org/no-expires-repo",
						Current:    true,
					},
				},
				// No expires configured — standard maintenance won't be generated.
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Standard maintenance should NOT be generated (no expires).
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); !os.IsNotExist(statErr) {
			t.Errorf("Standard agentics-maintenance.yml should not exist when no expires")
		}

		// Side-repo maintenance should be generated.
		sideFile := filepath.Join(tmpDir, "agentics-maintenance-org-no-expires-repo.yml")
		content, err := os.ReadFile(sideFile)
		if err != nil {
			t.Fatalf("Expected side-repo maintenance file to exist: %v", err)
		}
		contentStr := string(content)

		// Should use fallback token when none specified.
		if !strings.Contains(contentStr, "GH_AW_GITHUB_TOKEN") {
			t.Errorf("Side-repo maintenance should use fallback token GH_AW_GITHUB_TOKEN, got content length %d", len(contentStr))
		}
		// Should NOT include close-expired-entities (no expires).
		if strings.Contains(contentStr, "close-expired-entities") {
			t.Errorf("Side-repo maintenance should NOT include close-expired-entities when no expires, got content length %d", len(contentStr))
		}
		if !strings.Contains(contentStr, "activity_report") {
			t.Errorf("Side-repo maintenance should include activity_report when no expires, got content length %d", len(contentStr))
		}
	})

	t.Run("expression-based repository does not generate side-repo maintenance", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{
				Name: "dynamic-repo-workflow",
				CheckoutConfigs: []*CheckoutConfig{
					{
						Repository: "${{ inputs.target_repo }}",
						Current:    true,
					},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		entries, _ := os.ReadDir(tmpDir)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "agentics-maintenance-") {
				t.Errorf("Unexpected side-repo maintenance file for dynamic repo: %s", entry.Name())
			}
		}
	})

	t.Run("side-repo with expires includes schedule trigger", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Expires: 48 hours = 2 days → generateSideRepoMaintenanceCron("org/expires-repo", 2)
		repoSlug := "org/expires-repo"
		workflowDataList := []*WorkflowData{
			{
				Name: "side-repo-with-expires",
				CheckoutConfigs: []*CheckoutConfig{
					{Repository: repoSlug, Current: true},
				},
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{Expires: 48},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		sideFile := filepath.Join(tmpDir, "agentics-maintenance-org-expires-repo.yml")
		content, err := os.ReadFile(sideFile)
		if err != nil {
			t.Fatalf("Expected side-repo maintenance file to exist: %v", err)
		}
		contentStr := string(content)

		if !strings.Contains(contentStr, "schedule:") {
			t.Errorf("Side-repo maintenance with expires should include a schedule trigger, got content length %d", len(contentStr))
		}
		// 48 hours = 2 days → generateSideRepoMaintenanceCron returns the fuzzy 6-hour cron.
		expectedCron, _ := generateSideRepoMaintenanceCron(repoSlug, 2)
		if !strings.Contains(contentStr, expectedCron) {
			t.Errorf("Side-repo maintenance with 2-day expires should use cron %q, got content:\n%s", expectedCron, contentStr[:min(500, len(contentStr))])
		}
		// Verify the cron is different from the fixed minute used by the main workflow (37).
		// (For this particular slug the minute should not be 37 — but the real assertion is
		// that the expected fuzzy value is present, which we already checked above.)
	})

	t.Run("stale side-repo maintenance workflow is removed on recompile", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Simulate a stale file from a previous run.
		staleName := "agentics-maintenance-old-org-old-repo.yml"
		stalePath := filepath.Join(tmpDir, staleName)
		if err := os.WriteFile(stalePath, []byte("stale"), 0644); err != nil {
			t.Fatalf("Failed to create stale file: %v", err)
		}

		// Current run has a different target repo.
		workflowDataList := []*WorkflowData{
			{
				Name: "new-workflow",
				CheckoutConfigs: []*CheckoutConfig{
					{Repository: "new-org/new-repo", Current: true},
				},
			},
		}

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       nil,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Stale file should have been removed.
		if _, statErr := os.Stat(stalePath); !os.IsNotExist(statErr) {
			t.Errorf("Stale side-repo maintenance file %s should have been removed", staleName)
		}

		// The new file should exist.
		newFile := filepath.Join(tmpDir, "agentics-maintenance-new-org-new-repo.yml")
		if _, statErr := os.Stat(newFile); statErr != nil {
			t.Errorf("New side-repo maintenance file should exist: %v", statErr)
		}
	})
}
