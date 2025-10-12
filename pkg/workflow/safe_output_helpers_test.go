package workflow

import (
	"strings"
	"testing"
)

// TestBuildGitHubScriptStep verifies the common helper function produces correct GitHub Script steps
func TestBuildGitHubScriptStep(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name            string
		workflowData    *WorkflowData
		config          GitHubScriptStepConfig
		expectedInSteps []string
	}{
		{
			name: "basic script step with minimal config",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
			},
			config: GitHubScriptStepConfig{
				StepName:    "Test Step",
				StepID:      "test_step",
				MainJobName: "main_job",
				Script:      "console.log('test');",
				Token:       "",
			},
			expectedInSteps: []string{
				"- name: Download agent output artifact",
				"uses: actions/download-artifact@v5",
				"name: ${{ needs.main_job.outputs.output-artifact }}",
				"- name: Test Step",
				"id: test_step",
				"uses: actions/github-script@v8",
				"with:",
				"script: |",
				"console.log('test');",
			},
		},
		{
			name: "script step with custom env vars",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
			},
			config: GitHubScriptStepConfig{
				StepName:    "Create Issue",
				StepID:      "create_issue",
				MainJobName: "agent",
				CustomEnvVars: []string{
					"          GITHUB_AW_ISSUE_TITLE_PREFIX: \"[bot] \"\n",
					"          GITHUB_AW_ISSUE_LABELS: \"automation,ai\"\n",
				},
				Script: "const issue = true;",
				Token:  "",
			},
			expectedInSteps: []string{
				"- name: Download agent output artifact",
				"uses: actions/download-artifact@v5",
				"name: ${{ needs.agent.outputs.output-artifact }}",
				"- name: Create Issue",
				"id: create_issue",
				"uses: actions/github-script@v8",
				"env:",
				"GITHUB_AW_ISSUE_TITLE_PREFIX: \"[bot] \"",
				"GITHUB_AW_ISSUE_LABELS: \"automation,ai\"",
				"const issue = true;",
			},
		},
		{
			name: "script step with safe-outputs.env variables",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					Env: map[string]string{
						"CUSTOM_VAR_1": "value1",
						"CUSTOM_VAR_2": "value2",
					},
				},
			},
			config: GitHubScriptStepConfig{
				StepName:    "Process Output",
				StepID:      "process",
				MainJobName: "main",
				Script:      "const x = 1;",
				Token:       "",
			},
			expectedInSteps: []string{
				"- name: Download agent output artifact",
				"uses: actions/download-artifact@v5",
				"name: ${{ needs.main.outputs.output-artifact }}",
				"- name: Process Output",
				"id: process",
				"env:",
				"CUSTOM_VAR_1: value1",
				"CUSTOM_VAR_2: value2",
			},
		},
		{
			name: "script step with custom token",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
			},
			config: GitHubScriptStepConfig{
				StepName:    "Secure Action",
				StepID:      "secure",
				MainJobName: "main",
				Script:      "const secure = true;",
				Token:       "${{ secrets.CUSTOM_TOKEN }}",
			},
			expectedInSteps: []string{
				"- name: Download agent output artifact",
				"uses: actions/download-artifact@v5",
				"name: ${{ needs.main.outputs.output-artifact }}",
				"- name: Secure Action",
				"id: secure",
				"with:",
				"github-token: ${{ secrets.CUSTOM_TOKEN }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildGitHubScriptStep(tt.workflowData, tt.config)

			// Convert steps slice to a single string for easier searching
			stepsStr := strings.Join(steps, "")

			// Verify expected strings are present in the output
			for _, expected := range tt.expectedInSteps {
				if !strings.Contains(stepsStr, expected) {
					t.Errorf("Expected step to contain %q, but it was not found.\nGenerated steps:\n%s", expected, stepsStr)
				}
			}

			// Verify basic structure is present
			if !strings.Contains(stepsStr, "- name:") {
				t.Error("Expected step to have '- name:' field")
			}
			if !strings.Contains(stepsStr, "id:") {
				t.Error("Expected step to have 'id:' field")
			}
			if !strings.Contains(stepsStr, "uses: actions/github-script@v8") {
				t.Error("Expected step to use actions/github-script@v8")
			}
			if !strings.Contains(stepsStr, "with:") {
				t.Error("Expected step to have 'with:' section")
			}
			// Note: env: section is now optional based on config
			if !strings.Contains(stepsStr, "uses: actions/download-artifact@v5") {
				t.Error("Expected step to download agent output artifact")
			}
			if !strings.Contains(stepsStr, "script: |") {
				t.Error("Expected step to have 'script: |' section")
			}
		})
	}
}

// TestBuildGitHubScriptStepMaintainsOrder verifies that environment variables appear in expected order
func TestBuildGitHubScriptStepMaintainsOrder(t *testing.T) {
	compiler := &Compiler{}
	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Env: map[string]string{
				"SAFE_OUTPUT_VAR": "value",
			},
		},
	}

	config := GitHubScriptStepConfig{
		StepName:    "Test Step",
		StepID:      "test",
		MainJobName: "main",
		CustomEnvVars: []string{
			"          CUSTOM_VAR: custom_value\n",
		},
		Script: "const test = 1;",
		Token:  "",
	}

	steps := compiler.buildGitHubScriptStep(workflowData, config)
	stepsStr := strings.Join(steps, "")

	// Verify artifact download step is present
	if !strings.Contains(stepsStr, "Download agent output artifact") {
		t.Error("Download agent output artifact step not found in output")
	}

	// Verify environment variables are in order (custom vars before safe-outputs.env vars)
	customVarIdx := strings.Index(stepsStr, "CUSTOM_VAR")
	safeOutputVarIdx := strings.Index(stepsStr, "SAFE_OUTPUT_VAR")

	if customVarIdx == -1 {
		t.Error("CUSTOM_VAR not found in output")
	}
	if safeOutputVarIdx == -1 {
		t.Error("SAFE_OUTPUT_VAR not found in output")
	}

	// Verify order: custom vars -> safe-outputs.env vars
	if customVarIdx > safeOutputVarIdx {
		t.Error("Custom vars should come before safe-outputs.env vars")
	}
}
