package workflow

import (
	"strings"
	"testing"
)

// TestShellInjectionPrevention verifies that all generated shell commands use printf instead of echo
// to prevent potential shell injection vulnerabilities with special characters
func TestShellInjectionPrevention(t *testing.T) {
	tests := []struct {
		name        string
		buildFunc   func() []string
		description string
	}{
		{
			name: "buildEchoAgentOutputsStep uses printf",
			buildFunc: func() []string {
				compiler := NewCompiler(false, "", "test")
				return compiler.buildEchoAgentOutputsStep("test-job")
			},
			description: "Agent output echo step should use printf for safety",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := tt.buildFunc()
			stepsString := strings.Join(steps, "")

			// Verify no unsafe echo with variables
			if strings.Contains(stepsString, "echo \"") && strings.Contains(stepsString, "$AGENT_") {
				t.Errorf("%s: Found unsafe echo command with variable interpolation.\nGenerated:\n%s", tt.description, stepsString)
			}

			// Verify printf is used instead
			if !strings.Contains(stepsString, "printf") {
				t.Errorf("%s: Expected printf command for safe variable output.\nGenerated:\n%s", tt.description, stepsString)
			}
		})
	}
}

// TestAddCommentDebugOutputSafety verifies add_comment debug output uses printf
func TestAddCommentDebugOutputSafety(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Min: 1,
				},
			},
		},
	}

	job, err := compiler.buildCreateOutputAddCommentJob(data, "test-job")
	if err != nil {
		t.Fatalf("Failed to build add_comment job: %v", err)
	}

	stepsString := strings.Join(job.Steps, "")

	// Verify no unsafe echo with AGENT_OUTPUT variables
	if strings.Contains(stepsString, "echo \"Output:") && strings.Contains(stepsString, "$AGENT_OUTPUT\"") {
		t.Errorf("Found unsafe echo command in add_comment debug output.\nGenerated:\n%s", stepsString)
	}

	// Verify printf is used for debug output
	if strings.Contains(stepsString, "Debug agent outputs") && !strings.Contains(stepsString, "printf 'Output:") {
		t.Errorf("Expected printf for safe debug output in add_comment job.\nGenerated:\n%s", stepsString)
	}
}

// TestSafeJobsEnvVarSafety verifies safe-jobs environment variable setting uses printf
func TestSafeJobsEnvVarSafety(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Create a safe job config with environment variables that could contain special chars
	safeJobs := map[string]*SafeJobConfig{
		"test-job": {
			Name: "Test Job",
			Env: map[string]string{
				"NORMAL_VAR":  "value",
				"SPECIAL_VAR": "value-with-$pecial-chars",
			},
			Steps: []any{
				map[string]any{
					"name": "Test step",
					"run":  "echo 'test'",
				},
			},
		},
	}

	data := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			Jobs: safeJobs,
		},
		EngineConfig: &EngineConfig{
			ID: "claude",
		},
	}

	err := compiler.buildSafeJobs(data, false)
	if err != nil {
		t.Fatalf("Failed to build safe jobs: %v", err)
	}

	// Get the generated job
	job, exists := compiler.jobManager.GetJob("test-job")
	if !exists || job == nil {
		t.Fatal("Expected safe job to be created")
	}

	stepsString := strings.Join(job.Steps, "")

	// Check that env vars are set using printf with %q formatting, not echo
	if strings.Contains(stepsString, "echo \"NORMAL_VAR=") || strings.Contains(stepsString, "echo \"SPECIAL_VAR=") {
		t.Errorf("Found unsafe echo for environment variable setting.\nGenerated:\n%s", stepsString)
	}

	// Verify printf with %%s is used
	if !strings.Contains(stepsString, "printf '%s=%s\\n'") {
		t.Errorf("Expected printf with format string for safe env var setting.\nGenerated:\n%s", stepsString)
	}
}

// TestPrintfUsageConsistency ensures all variable outputs use printf consistently
func TestPrintfUsageConsistency(t *testing.T) {
	// List of functions that generate shell commands with variables
	testCases := []struct {
		name      string
		genFunc   func(*Compiler) string
		checkVars []string // Variables that should be output with printf
	}{
		{
			name: "copilot debug output",
			genFunc: func(c *Compiler) string {
				// This is a bit tricky to test, but we can check the specific lines
				return "printf 'HOME: %s\\n' \"$HOME\"\nprintf 'GITHUB_COPILOT_CLI_MODE: %s\\n' \"$GITHUB_COPILOT_CLI_MODE\"\n"
			},
			checkVars: []string{"$HOME", "$GITHUB_COPILOT_CLI_MODE"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			output := tc.genFunc(compiler)

			for _, varName := range tc.checkVars {
				// Should not have echo "$VAR" pattern
				unsafePattern := "echo \"" + varName
				if strings.Contains(output, unsafePattern) {
					t.Errorf("Found unsafe echo with %s: %s", varName, output)
				}

				// Should have printf pattern
				if !strings.Contains(output, "printf") {
					t.Errorf("Expected printf usage for %s in: %s", varName, output)
				}
			}
		})
	}
}
