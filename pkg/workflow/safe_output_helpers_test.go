package workflow

import (
	"strings"
	"testing"
)

// TestExtractSafeOutputToken verifies the helper function extracts tokens correctly
func TestExtractSafeOutputToken(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		accessor     func(*SafeOutputsConfig) string
		expected     string
	}{
		{
			name: "extract token from CreateIssues config",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							GitHubToken: "${{ secrets.CUSTOM_TOKEN }}",
						},
					},
				},
			},
			accessor: func(so *SafeOutputsConfig) string {
				if so.CreateIssues != nil {
					return so.CreateIssues.GitHubToken
				}
				return ""
			},
			expected: "${{ secrets.CUSTOM_TOKEN }}",
		},
		{
			name: "extract token from AddComments config",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					AddComments: &AddCommentsConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							GitHubToken: "${{ secrets.COMMENT_TOKEN }}",
						},
					},
				},
			},
			accessor: func(so *SafeOutputsConfig) string {
				if so.AddComments != nil {
					return so.AddComments.GitHubToken
				}
				return ""
			},
			expected: "${{ secrets.COMMENT_TOKEN }}",
		},
		{
			name: "nil SafeOutputs returns empty string",
			workflowData: &WorkflowData{
				SafeOutputs: nil,
			},
			accessor: func(so *SafeOutputsConfig) string {
				if so.CreateIssues != nil {
					return so.CreateIssues.GitHubToken
				}
				return ""
			},
			expected: "",
		},
		{
			name: "nil config in accessor returns empty string",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: nil,
				},
			},
			accessor: func(so *SafeOutputsConfig) string {
				if so.CreateIssues != nil {
					return so.CreateIssues.GitHubToken
				}
				return ""
			},
			expected: "",
		},
		{
			name: "empty token string in config",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							GitHubToken: "",
						},
					},
				},
			},
			accessor: func(so *SafeOutputsConfig) string {
				if so.CreateIssues != nil {
					return so.CreateIssues.GitHubToken
				}
				return ""
			},
			expected: "",
		},
		{
			name: "extract token from CreateDiscussions config",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateDiscussions: &CreateDiscussionsConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							GitHubToken: "${{ secrets.DISCUSSION_TOKEN }}",
						},
					},
				},
			},
			accessor: func(so *SafeOutputsConfig) string {
				if so.CreateDiscussions != nil {
					return so.CreateDiscussions.GitHubToken
				}
				return ""
			},
			expected: "${{ secrets.DISCUSSION_TOKEN }}",
		},
		{
			name: "extract token from CreateAgentTasks config",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateAgentTasks: &CreateAgentTaskConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							GitHubToken: "${{ secrets.AGENT_TASK_TOKEN }}",
						},
					},
				},
			},
			accessor: func(so *SafeOutputsConfig) string {
				if so.CreateAgentTasks != nil {
					return so.CreateAgentTasks.GitHubToken
				}
				return ""
			},
			expected: "${{ secrets.AGENT_TASK_TOKEN }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSafeOutputToken(tt.workflowData, tt.accessor)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

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
				"- name: Setup agent output environment variable",
				"- name: Test Step",
				"id: test_step",
				"uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd",
				"env:",
				"GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}",
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
					"          GH_AW_ISSUE_TITLE_PREFIX: \"[bot] \"\n",
					"          GH_AW_ISSUE_LABELS: \"automation,ai\"\n",
				},
				Script: "const issue = true;",
				Token:  "",
			},
			expectedInSteps: []string{
				"- name: Download agent output artifact",
				"- name: Setup agent output environment variable",
				"- name: Create Issue",
				"id: create_issue",
				"uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd",
				"GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}",
				"GH_AW_ISSUE_TITLE_PREFIX: \"[bot] \"",
				"GH_AW_ISSUE_LABELS: \"automation,ai\"",
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
				"- name: Setup agent output environment variable",
				"- name: Process Output",
				"id: process",
				"GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}",
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
			if !strings.Contains(stepsStr, "uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd") {
				t.Error("Expected step to use actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd")
			}
			if !strings.Contains(stepsStr, "env:") {
				t.Error("Expected step to have 'env:' section")
			}
			if !strings.Contains(stepsStr, "with:") {
				t.Error("Expected step to have 'with:' section")
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

	// Verify GH_AW_AGENT_OUTPUT comes first (after env: line)
	agentOutputIdx := strings.Index(stepsStr, "GH_AW_AGENT_OUTPUT")
	customVarIdx := strings.Index(stepsStr, "CUSTOM_VAR")
	safeOutputVarIdx := strings.Index(stepsStr, "SAFE_OUTPUT_VAR")

	if agentOutputIdx == -1 {
		t.Error("GH_AW_AGENT_OUTPUT not found in output")
	}
	if customVarIdx == -1 {
		t.Error("CUSTOM_VAR not found in output")
	}
	if safeOutputVarIdx == -1 {
		t.Error("SAFE_OUTPUT_VAR not found in output")
	}

	// Verify order: GH_AW_AGENT_OUTPUT -> custom vars -> safe-outputs.env vars
	if agentOutputIdx > customVarIdx {
		t.Error("GH_AW_AGENT_OUTPUT should come before custom vars")
	}
	if customVarIdx > safeOutputVarIdx {
		t.Error("Custom vars should come before safe-outputs.env vars")
	}
}

// TestApplySafeOutputEnvToMap verifies the helper function for map[string]string env variables
func TestApplySafeOutputEnvToMap(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     map[string]string
	}{
		{
			name: "nil SafeOutputs",
			workflowData: &WorkflowData{
				SafeOutputs: nil,
			},
			expected: map[string]string{},
		},
		{
			name: "basic safe outputs",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{},
			},
			expected: map[string]string{
				"GH_AW_SAFE_OUTPUTS":        "${{ env.GH_AW_SAFE_OUTPUTS }}",
				"GH_AW_SAFE_OUTPUTS_CONFIG": "\"{}\"",
			},
		},
		{
			name: "safe outputs with staged flag",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Staged: true,
				},
			},
			expected: map[string]string{
				"GH_AW_SAFE_OUTPUTS":        "${{ env.GH_AW_SAFE_OUTPUTS }}",
				"GH_AW_SAFE_OUTPUTS_CONFIG": "\"{}\"",
				"GH_AW_SAFE_OUTPUTS_STAGED": "true",
			},
		},
		{
			name: "trial mode",
			workflowData: &WorkflowData{
				TrialMode:        true,
				TrialLogicalRepo: "owner/repo",
				SafeOutputs:      &SafeOutputsConfig{},
			},
			expected: map[string]string{
				"GH_AW_SAFE_OUTPUTS":        "${{ env.GH_AW_SAFE_OUTPUTS }}",
				"GH_AW_SAFE_OUTPUTS_CONFIG": "\"{}\"",
				"GH_AW_SAFE_OUTPUTS_STAGED": "true",
				"GH_AW_TARGET_REPO_SLUG":    "owner/repo",
			},
		},
		{
			name: "upload assets config",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					UploadAssets: &UploadAssetsConfig{
						BranchName:  "gh-aw-assets",
						MaxSizeKB:   10240,
						AllowedExts: []string{".png", ".jpg", ".jpeg"},
					},
				},
			},
			expected: map[string]string{
				"GH_AW_SAFE_OUTPUTS":        "${{ env.GH_AW_SAFE_OUTPUTS }}",
				"GH_AW_SAFE_OUTPUTS_CONFIG": "\"{\\\"upload_asset\\\":{}}\"",
				"GH_AW_ASSETS_BRANCH":       "\"gh-aw-assets\"",
				"GH_AW_ASSETS_MAX_SIZE_KB":  "10240",
				"GH_AW_ASSETS_ALLOWED_EXTS": "\".png,.jpg,.jpeg\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := make(map[string]string)
			applySafeOutputEnvToMap(env, tt.workflowData)

			if len(env) != len(tt.expected) {
				t.Errorf("Expected %d env vars, got %d", len(tt.expected), len(env))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := env[key]; !exists {
					t.Errorf("Expected env var %q not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Env var %q: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestApplySafeOutputEnvToSlice verifies the helper function for YAML string slices
func TestApplySafeOutputEnvToSlice(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     []string
	}{
		{
			name: "nil SafeOutputs",
			workflowData: &WorkflowData{
				SafeOutputs: nil,
			},
			expected: []string{},
		},
		{
			name: "basic safe outputs",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{},
			},
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}",
			},
		},
		{
			name: "safe outputs with staged flag",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Staged: true,
				},
			},
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}",
				"          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "trial mode",
			workflowData: &WorkflowData{
				TrialMode:        true,
				TrialLogicalRepo: "owner/repo",
				SafeOutputs:      &SafeOutputsConfig{},
			},
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}",
				"          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
				"          GH_AW_TARGET_REPO_SLUG: \"owner/repo\"",
			},
		},
		{
			name: "upload assets config",
			workflowData: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					UploadAssets: &UploadAssetsConfig{
						BranchName:  "gh-aw-assets",
						MaxSizeKB:   10240,
						AllowedExts: []string{".png", ".jpg", ".jpeg"},
					},
				},
			},
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}",
				"          GH_AW_ASSETS_BRANCH: \"gh-aw-assets\"",
				"          GH_AW_ASSETS_MAX_SIZE_KB: 10240",
				"          GH_AW_ASSETS_ALLOWED_EXTS: \".png,.jpg,.jpeg\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stepLines []string
			applySafeOutputEnvToSlice(&stepLines, tt.workflowData)

			if len(stepLines) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(stepLines))
			}

			for i, expectedLine := range tt.expected {
				if i >= len(stepLines) {
					t.Errorf("Missing expected line %d: %q", i, expectedLine)
					continue
				}
				if stepLines[i] != expectedLine {
					t.Errorf("Line %d: expected %q, got %q", i, expectedLine, stepLines[i])
				}
			}
		})
	}
}

// TestBuildSafeOutputJobEnvVars verifies the helper function for safe-output job env vars
func TestBuildSafeOutputJobEnvVars(t *testing.T) {
	tests := []struct {
		name                 string
		trialMode            bool
		trialLogicalRepoSlug string
		staged               bool
		targetRepoSlug       string
		expected             []string
	}{
		{
			name:     "no flags",
			expected: []string{},
		},
		{
			name:   "staged only",
			staged: true,
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
			},
		},
		{
			name:      "trial mode only",
			trialMode: true,
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
			},
		},
		{
			name:                 "trial mode with trial repo slug",
			trialMode:            true,
			trialLogicalRepoSlug: "owner/trial-repo",
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
				"          GH_AW_TARGET_REPO_SLUG: \"owner/trial-repo\"\n",
			},
		},
		{
			name:           "target repo slug only",
			targetRepoSlug: "owner/target-repo",
			expected: []string{
				"          GH_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n",
			},
		},
		{
			name:                 "target repo slug overrides trial repo slug",
			trialMode:            true,
			trialLogicalRepoSlug: "owner/trial-repo",
			targetRepoSlug:       "owner/target-repo",
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
				"          GH_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n",
			},
		},
		{
			name:                 "all flags",
			trialMode:            true,
			trialLogicalRepoSlug: "owner/trial-repo",
			staged:               true,
			targetRepoSlug:       "owner/target-repo",
			expected: []string{
				"          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
				"          GH_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSafeOutputJobEnvVars(
				tt.trialMode,
				tt.trialLogicalRepoSlug,
				tt.staged,
				tt.targetRepoSlug,
			)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d env vars, got %d", len(tt.expected), len(result))
			}

			for i, expectedVar := range tt.expected {
				if i >= len(result) {
					t.Errorf("Missing expected env var %d: %q", i, expectedVar)
					continue
				}
				if result[i] != expectedVar {
					t.Errorf("Env var %d: expected %q, got %q", i, expectedVar, result[i])
				}
			}
		})
	}
}

// TestEnginesUseSameHelperLogic ensures all engines produce consistent env vars
func TestEnginesUseSameHelperLogic(t *testing.T) {
	workflowData := &WorkflowData{
		TrialMode:        true,
		TrialLogicalRepo: "owner/trial-repo",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			UploadAssets: &UploadAssetsConfig{
				BranchName:  "gh-aw-assets",
				MaxSizeKB:   10240,
				AllowedExts: []string{".png", ".jpg"},
			},
		},
	}

	// Test map-based helper (used by copilot, codex, and custom)
	envMap := make(map[string]string)
	applySafeOutputEnvToMap(envMap, workflowData)

	// Test slice-based helper (used by claude)
	var stepLines []string
	applySafeOutputEnvToSlice(&stepLines, workflowData)

	// Verify both approaches produce the same env vars
	expectedKeys := []string{
		"GH_AW_SAFE_OUTPUTS",
		"GH_AW_SAFE_OUTPUTS_STAGED",
		"GH_AW_TARGET_REPO_SLUG",
		"GH_AW_ASSETS_BRANCH",
		"GH_AW_ASSETS_MAX_SIZE_KB",
		"GH_AW_ASSETS_ALLOWED_EXTS",
	}

	// Check map
	for _, key := range expectedKeys {
		if _, exists := envMap[key]; !exists {
			t.Errorf("envMap missing expected key: %s", key)
		}
	}

	// Check slice (should contain all keys)
	sliceContent := strings.Join(stepLines, "\n")
	for _, key := range expectedKeys {
		if !strings.Contains(sliceContent, key) {
			t.Errorf("stepLines missing expected key: %s", key)
		}
	}
}

// TestBuildAgentOutputDownloadSteps verifies the agent output download steps
// include directory creation to handle cases where artifact doesn't exist
func TestBuildAgentOutputDownloadSteps(t *testing.T) {
	steps := buildAgentOutputDownloadSteps()
	stepsStr := strings.Join(steps, "")

	// Verify expected steps are present
	expectedComponents := []string{
		"- name: Download agent output artifact",
		"continue-on-error: true",
		"uses: actions/download-artifact@634f93cb2916e3fdff6788551b99b062d0335ce0",
		"name: agent_output.json",
		"path: /tmp/gh-aw/safeoutputs/",
		"- name: Setup agent output environment variable",
		"mkdir -p /tmp/gh-aw/safeoutputs/",
		"find /tmp/gh-aw/safeoutputs/ -type f -print",
		"GH_AW_AGENT_OUTPUT=/tmp/gh-aw/safeoutputs/agent_output.json",
	}

	for _, expected := range expectedComponents {
		if !strings.Contains(stepsStr, expected) {
			t.Errorf("Expected step to contain %q, but it was not found.\nGenerated steps:\n%s", expected, stepsStr)
		}
	}

	// Verify mkdir comes before find to ensure directory exists
	mkdirIdx := strings.Index(stepsStr, "mkdir -p /tmp/gh-aw/safeoutputs/")
	findIdx := strings.Index(stepsStr, "find /tmp/gh-aw/safeoutputs/")

	if mkdirIdx == -1 {
		t.Fatal("mkdir command not found in steps")
	}
	if findIdx == -1 {
		t.Fatal("find command not found in steps")
	}
	if mkdirIdx > findIdx {
		t.Error("mkdir should come before find to ensure directory exists")
	}
}
