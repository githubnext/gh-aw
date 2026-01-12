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

func TestGenerateMaintenanceWorkflow_MaintenanceFlag(t *testing.T) {
	tests := []struct {
		name                      string
		workflowDataList          []*WorkflowData
		expectWorkflowGenerated   bool
		expectError               bool
		expectedWarningSubstrings []string
	}{
		{
			name: "maintenance: true (default) with expires - should generate workflow",
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
			name: "maintenance: false with expires - should NOT generate workflow and show warning",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{
							Expires: 168, // 7 days
						},
						Maintenance: boolPtr(false),
					},
				},
			},
			expectWorkflowGenerated:   false,
			expectError:               false,
			expectedWarningSubstrings: []string{"test-workflow", "expires", "maintenance: false"},
		},
		{
			name: "maintenance: false with expires in issues - should show warning",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow-issues",
					SafeOutputs: &SafeOutputsConfig{
						CreateIssues: &CreateIssuesConfig{
							Expires: 48, // 2 days
						},
						Maintenance: boolPtr(false),
					},
				},
			},
			expectWorkflowGenerated:   false,
			expectError:               false,
			expectedWarningSubstrings: []string{"test-workflow-issues", "expires", "maintenance: false"},
		},
		{
			name: "maintenance: false WITHOUT expires - should NOT generate and NO warning",
			workflowDataList: []*WorkflowData{
				{
					Name: "test-workflow",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{},
						Maintenance:       boolPtr(false),
					},
				},
			},
			expectWorkflowGenerated: false,
			expectError:             false,
		},
		{
			name: "no expires field - should NOT generate workflow",
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
			name: "multiple workflows - one with maintenance:false, one with expires - should show warning for one",
			workflowDataList: []*WorkflowData{
				{
					Name: "workflow-with-expires-disabled",
					SafeOutputs: &SafeOutputsConfig{
						CreateDiscussions: &CreateDiscussionsConfig{
							Expires: 168,
						},
						Maintenance: boolPtr(false),
					},
				},
				{
					Name: "workflow-without-expires",
					SafeOutputs: &SafeOutputsConfig{
						CreateIssues: &CreateIssuesConfig{},
					},
				},
			},
			expectWorkflowGenerated:   false,
			expectError:               false,
			expectedWarningSubstrings: []string{"workflow-with-expires-disabled"},
		},
		{
			name: "maintenance: false with both discussions and issues expires - should show warning once per workflow",
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
						Maintenance: boolPtr(false),
					},
				},
			},
			expectWorkflowGenerated:   false,
			expectError:               false,
			expectedWarningSubstrings: []string{"multi-expires-workflow", "expires", "maintenance: false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the workflow
			tmpDir := t.TempDir()

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Call GenerateMaintenanceWorkflow
			err := GenerateMaintenanceWorkflow(tt.workflowDataList, tmpDir, "v1.0.0", ActionModeDev, false)

			// Restore stderr and read captured output
			w.Close()
			os.Stderr = oldStderr
			var buf [4096]byte
			n, _ := r.Read(buf[:])
			stderrOutput := string(buf[:n])

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

			// Check for expected warning substrings
			for _, expectedSubstr := range tt.expectedWarningSubstrings {
				if !strings.Contains(stderrOutput, expectedSubstr) {
					t.Errorf("Expected warning to contain %q but got: %s", expectedSubstr, stderrOutput)
				}
			}
		})
	}
}
