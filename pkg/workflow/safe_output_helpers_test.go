package workflow

import (
	"os"
	"path/filepath"
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
				"- name: Test Step",
				"id: test_step",
				"uses: actions/github-script@v8",
				"env:",
				"GITHUB_AW_AGENT_OUTPUT: ${{ needs.main_job.outputs.output }}",
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
				"- name: Create Issue",
				"id: create_issue",
				"uses: actions/github-script@v8",
				"GITHUB_AW_AGENT_OUTPUT: ${{ needs.agent.outputs.output }}",
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
				"- name: Process Output",
				"id: process",
				"GITHUB_AW_AGENT_OUTPUT: ${{ needs.main.outputs.output }}",
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
			if !strings.Contains(stepsStr, "uses: actions/github-script@v8") {
				t.Error("Expected step to use actions/github-script@v8")
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

	// Verify GITHUB_AW_AGENT_OUTPUT comes first (after env: line)
	agentOutputIdx := strings.Index(stepsStr, "GITHUB_AW_AGENT_OUTPUT")
	customVarIdx := strings.Index(stepsStr, "CUSTOM_VAR")
	safeOutputVarIdx := strings.Index(stepsStr, "SAFE_OUTPUT_VAR")

	if agentOutputIdx == -1 {
		t.Error("GITHUB_AW_AGENT_OUTPUT not found in output")
	}
	if customVarIdx == -1 {
		t.Error("CUSTOM_VAR not found in output")
	}
	if safeOutputVarIdx == -1 {
		t.Error("SAFE_OUTPUT_VAR not found in output")
	}

	// Verify order: GITHUB_AW_AGENT_OUTPUT -> custom vars -> safe-outputs.env vars
	if agentOutputIdx > customVarIdx {
		t.Error("GITHUB_AW_AGENT_OUTPUT should come before custom vars")
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
				"GITHUB_AW_SAFE_OUTPUTS":        "${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
				"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "\"{}\"",
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
				"GITHUB_AW_SAFE_OUTPUTS":        "${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
				"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "\"{}\"",
				"GITHUB_AW_SAFE_OUTPUTS_STAGED": "true",
			},
		},
		{
			name: "trial mode",
			workflowData: &WorkflowData{
				TrialMode:       true,
				TrialTargetRepo: "owner/repo",
				SafeOutputs:     &SafeOutputsConfig{},
			},
			expected: map[string]string{
				"GITHUB_AW_SAFE_OUTPUTS":        "${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
				"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "\"{}\"",
				"GITHUB_AW_SAFE_OUTPUTS_STAGED": "true",
				"GITHUB_AW_TARGET_REPO_SLUG":    "owner/repo",
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
				"GITHUB_AW_SAFE_OUTPUTS":        "${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
				"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "\"{\\\"upload-asset\\\":{}}\"",
				"GITHUB_AW_ASSETS_BRANCH":       "\"gh-aw-assets\"",
				"GITHUB_AW_ASSETS_MAX_SIZE_KB":  "10240",
				"GITHUB_AW_ASSETS_ALLOWED_EXTS": "\".png,.jpg,.jpeg\"",
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
				"          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
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
				"          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
				"          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "trial mode",
			workflowData: &WorkflowData{
				TrialMode:       true,
				TrialTargetRepo: "owner/repo",
				SafeOutputs:     &SafeOutputsConfig{},
			},
			expected: []string{
				"          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
				"          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"",
				"          GITHUB_AW_TARGET_REPO_SLUG: \"owner/repo\"",
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
				"          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}",
				"          GITHUB_AW_ASSETS_BRANCH: \"gh-aw-assets\"",
				"          GITHUB_AW_ASSETS_MAX_SIZE_KB: 10240",
				"          GITHUB_AW_ASSETS_ALLOWED_EXTS: \".png,.jpg,.jpeg\"",
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
				"          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
			},
		},
		{
			name:      "trial mode only",
			trialMode: true,
			expected: []string{
				"          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
			},
		},
		{
			name:                 "trial mode with trial repo slug",
			trialMode:            true,
			trialLogicalRepoSlug: "owner/trial-repo",
			expected: []string{
				"          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
				"          GITHUB_AW_TARGET_REPO_SLUG: \"owner/trial-repo\"\n",
			},
		},
		{
			name:           "target repo slug only",
			targetRepoSlug: "owner/target-repo",
			expected: []string{
				"          GITHUB_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n",
			},
		},
		{
			name:                 "target repo slug overrides trial repo slug",
			trialMode:            true,
			trialLogicalRepoSlug: "owner/trial-repo",
			targetRepoSlug:       "owner/target-repo",
			expected: []string{
				"          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
				"          GITHUB_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n",
			},
		},
		{
			name:                 "all flags",
			trialMode:            true,
			trialLogicalRepoSlug: "owner/trial-repo",
			staged:               true,
			targetRepoSlug:       "owner/target-repo",
			expected: []string{
				"          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"\n",
				"          GITHUB_AW_TARGET_REPO_SLUG: \"owner/target-repo\"\n",
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
		TrialMode:       true,
		TrialTargetRepo: "owner/trial-repo",
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
		"GITHUB_AW_SAFE_OUTPUTS",
		"GITHUB_AW_SAFE_OUTPUTS_STAGED",
		"GITHUB_AW_TARGET_REPO_SLUG",
		"GITHUB_AW_ASSETS_BRANCH",
		"GITHUB_AW_ASSETS_MAX_SIZE_KB",
		"GITHUB_AW_ASSETS_ALLOWED_EXTS",
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

// TestFilterPermissionErrorPatterns verifies permission-related error pattern filtering
func TestFilterPermissionErrorPatterns(t *testing.T) {
	allPatterns := []ErrorPattern{
		{
			Pattern:     `(?i)permission denied`,
			Description: "Permission denied error",
		},
		{
			Pattern:     `(?i)unauthorized access`,
			Description: "Unauthorized access attempt",
		},
		{
			Pattern:     `(?i)access forbidden`,
			Description: "Forbidden resource access",
		},
		{
			Pattern:     `(?i)authentication failed`,
			Description: "Authentication failure",
		},
		{
			Pattern:     `(?i)invalid token`,
			Description: "Invalid authentication token",
		},
		{
			Pattern:     `(?i)syntax error`,
			Description: "Code syntax error",
		},
		{
			Pattern:     `(?i)runtime error`,
			Description: "Runtime exception",
		},
	}

	permissionPatterns := FilterPermissionErrorPatterns(allPatterns)

	// Should filter to only permission-related patterns (5 out of 7)
	expectedCount := 5
	if len(permissionPatterns) != expectedCount {
		t.Errorf("Expected %d permission patterns, got %d", expectedCount, len(permissionPatterns))
	}

	// Verify all returned patterns are permission-related
	for _, pattern := range permissionPatterns {
		desc := strings.ToLower(pattern.Description)
		isPermissionRelated := strings.Contains(desc, "permission") ||
			strings.Contains(desc, "unauthorized") ||
			strings.Contains(desc, "forbidden") ||
			strings.Contains(desc, "access") ||
			strings.Contains(desc, "authentication") ||
			strings.Contains(desc, "token")

		if !isPermissionRelated {
			t.Errorf("Non-permission pattern returned: %s", pattern.Description)
		}
	}
}

// TestCreateMissingToolEntry verifies missing-tool entry creation
func TestCreateMissingToolEntry(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	safeOutputsFile := filepath.Join(tempDir, "safe-outputs.ndjson")

	// Set environment variable
	os.Setenv("GITHUB_AW_SAFE_OUTPUTS", safeOutputsFile)
	defer os.Unsetenv("GITHUB_AW_SAFE_OUTPUTS")

	// Create a missing-tool entry
	err := CreateMissingToolEntry("test-tool", "access denied", true)
	if err != nil {
		t.Errorf("CreateMissingToolEntry failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(safeOutputsFile); os.IsNotExist(err) {
		t.Error("Safe outputs file was not created")
		return
	}

	// Read and verify content
	content, err := os.ReadFile(safeOutputsFile)
	if err != nil {
		t.Errorf("Failed to read safe outputs file: %v", err)
		return
	}

	contentStr := string(content)

	// Verify expected fields are present
	expectedStrings := []string{
		`"type":"missing-tool"`,
		`"tool":"test-tool"`,
		`"reason":"Permission denied: access denied"`,
		`"alternatives":"Check repository permissions and access controls"`,
		`"timestamp":`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected content to contain %s, got: %s", expected, contentStr)
		}
	}
}

// TestCreateMissingToolEntry_NoEnvVar verifies graceful handling when env var is not set
func TestCreateMissingToolEntry_NoEnvVar(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("GITHUB_AW_SAFE_OUTPUTS")

	// Should not error when env var is not set
	err := CreateMissingToolEntry("test-tool", "access denied", false)
	if err != nil {
		t.Errorf("CreateMissingToolEntry should not error when env var not set, got: %v", err)
	}
}

// TestScanLogForPermissionErrors verifies log scanning for permission errors
func TestScanLogForPermissionErrors(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	safeOutputsFile := filepath.Join(tempDir, "safe-outputs.ndjson")

	// Set environment variable
	os.Setenv("GITHUB_AW_SAFE_OUTPUTS", safeOutputsFile)
	defer os.Unsetenv("GITHUB_AW_SAFE_OUTPUTS")

	logContent := `
[INFO] Starting operation
[ERROR] permission denied for resource
[INFO] Continuing...
[ERROR] unauthorized access attempt
[INFO] Complete
`

	patterns := []ErrorPattern{
		{
			Pattern:     `(?i)permission denied`,
			Description: "Permission denied error",
		},
		{
			Pattern:     `(?i)unauthorized`,
			Description: "Unauthorized access",
		},
	}

	// Scan with no custom extractor (use default tool name)
	ScanLogForPermissionErrors(logContent, patterns, nil, "default-tool", false)

	// Read and verify entries were created
	content, err := os.ReadFile(safeOutputsFile)
	if err != nil {
		t.Errorf("Failed to read safe outputs file: %v", err)
		return
	}

	contentStr := string(content)

	// Should have 2 entries (one for each error pattern match)
	lines := strings.Split(strings.TrimSpace(contentStr), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(lines))
	}

	// Both should use the default tool name
	if !strings.Contains(contentStr, `"tool":"default-tool"`) {
		t.Error("Expected default tool name in entries")
	}
}

// TestScanLogForPermissionErrors_WithExtractor verifies custom tool name extraction
func TestScanLogForPermissionErrors_WithExtractor(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	safeOutputsFile := filepath.Join(tempDir, "safe-outputs.ndjson")

	// Set environment variable
	os.Setenv("GITHUB_AW_SAFE_OUTPUTS", safeOutputsFile)
	defer os.Unsetenv("GITHUB_AW_SAFE_OUTPUTS")

	logContent := `
[INFO] tool github-api(list_issues)
[ERROR] permission denied
`

	patterns := []ErrorPattern{
		{
			Pattern:     `(?i)permission denied`,
			Description: "Permission denied error",
		},
	}

	// Custom extractor that looks for "tool X(" pattern
	extractor := func(lines []string, errorLineIndex int, defaultTool string) string {
		for i := errorLineIndex - 1; i >= 0 && i >= errorLineIndex-5; i-- {
			if strings.Contains(lines[i], "tool ") {
				// Extract tool name from "tool toolname("
				parts := strings.Split(lines[i], "tool ")
				if len(parts) > 1 {
					toolPart := strings.Split(parts[1], "(")
					if len(toolPart) > 0 {
						return strings.TrimSpace(toolPart[0])
					}
				}
			}
		}
		return defaultTool
	}

	ScanLogForPermissionErrors(logContent, patterns, extractor, "default-tool", false)

	// Read and verify custom tool name was extracted
	content, err := os.ReadFile(safeOutputsFile)
	if err != nil {
		t.Errorf("Failed to read safe outputs file: %v", err)
		return
	}

	contentStr := string(content)

	// Should have extracted "github-api" as the tool name
	if !strings.Contains(contentStr, `"tool":"github-api"`) {
		t.Errorf("Expected custom tool name 'github-api' in entry, got: %s", contentStr)
	}
}
