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
			err := GenerateMaintenanceWorkflow(tt.workflowDataList, tmpDir, "v1.0.0", ActionModeDev, false)

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

func TestGenerateMaintenanceWorkflow_ActionModeJobs(t *testing.T) {
	tests := []struct {
		name                      string
		actionMode                ActionMode
		expectCompileWorkflowsJob bool
		expectZizmorScanJob       bool
	}{
		{
			name:                      "dev mode - should include compile-workflows and zizmor-scan jobs",
			actionMode:                ActionModeDev,
			expectCompileWorkflowsJob: true,
			expectZizmorScanJob:       true,
		},
		{
			name:                      "release mode - should NOT include compile-workflows and zizmor-scan jobs",
			actionMode:                ActionModeRelease,
			expectCompileWorkflowsJob: false,
			expectZizmorScanJob:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create a workflow with expires field to trigger maintenance workflow generation
			workflowDataList := []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateIssues: &CreateIssuesConfig{
							Expires: 168, // 7 days
						},
					},
				},
			}

			// Generate maintenance workflow
			err := GenerateMaintenanceWorkflow(workflowDataList, tmpDir, "v1.0.0", tt.actionMode, false)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Read the generated workflow file
			maintenanceFile := filepath.Join(tmpDir, "agentics-maintenance.yml")
			content, err := os.ReadFile(maintenanceFile)
			if err != nil {
				t.Fatalf("Failed to read maintenance workflow file: %v", err)
			}

			contentStr := string(content)

			// Check for compile-workflows job
			hasCompileWorkflowsJob := strings.Contains(contentStr, "compile-workflows:")
			if tt.expectCompileWorkflowsJob && !hasCompileWorkflowsJob {
				t.Errorf("Expected compile-workflows job in %s mode but it was not found", tt.actionMode)
			}
			if !tt.expectCompileWorkflowsJob && hasCompileWorkflowsJob {
				t.Errorf("Did NOT expect compile-workflows job in %s mode but it was found", tt.actionMode)
			}

			// Check for zizmor-scan job
			hasZizmor := strings.Contains(contentStr, "zizmor-scan:")
			if tt.expectZizmorScanJob && !hasZizmor {
				t.Errorf("Expected zizmor-scan job in %s mode but it was not found", tt.actionMode)
			}
			if !tt.expectZizmorScanJob && hasZizmor {
				t.Errorf("Did NOT expect zizmor-scan job in %s mode but it was found", tt.actionMode)
			}

			// Verify that close-expired-discussions and close-expired-issues jobs are always present
			if !strings.Contains(contentStr, "close-expired-discussions:") {
				t.Errorf("Expected close-expired-discussions job but it was not found")
			}
			if !strings.Contains(contentStr, "close-expired-issues:") {
				t.Errorf("Expected close-expired-issues job but it was not found")
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
			err := GenerateMaintenanceWorkflow(tt.workflowDataList, tmpDir, "v1.0.0", ActionModeDev, false)
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
