package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestTrialModeCompilation(t *testing.T) {
	// Create a test markdown workflow file with safe outputs
	workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
safe-outputs:
  create-pull-request: {}
  create-issue: {}
---

# Test Workflow

This is a test workflow for trial mode compilation.

## Instructions

- Test with safe outputs
- Test checkout token handling
`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "trial-mode-test-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content to file
	if _, err := tmpFile.WriteString(workflowContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test normal mode compilation (should include safe outputs)
	t.Run("Normal Mode", func(t *testing.T) {
		compiler := NewCompiler(false, "", "test")
		compiler.SetTrialMode(false)     // Normal mode
		compiler.SetSkipValidation(true) // Skip validation for test

		// Parse the workflow file to get WorkflowData
		workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to parse workflow file in normal mode: %v", err)
		}

		// Generate YAML content
		lockContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to generate YAML in normal mode: %v", err)
		}

		// In normal mode, safe output jobs should be included
		if !strings.Contains(lockContent, "create_pull_request:") {
			t.Error("Expected create_pull_request job in normal mode")
		}
		if !strings.Contains(lockContent, "create_issue:") {
			t.Error("Expected create_issue job in normal mode")
		}

		// Checkout should not include github-token in normal mode
		// Check specifically that the checkout step doesn't have a token parameter
		lines := strings.Split(lockContent, "\n")
		for i, line := range lines {
			if strings.Contains(line, "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8") {
				// Check the next few lines for "with:" and "token:"
				for j := i + 1; j < len(lines) && j < i+10; j++ {
					if strings.TrimSpace(lines[j]) == "with:" {
						// Found "with:" section, check for token
						for k := j + 1; k < len(lines) && k < j+5; k++ {
							if strings.Contains(lines[k], "token:") {
								t.Error("Did not expect github-token in checkout step in normal mode")
								break
							}
							// If we hit another step or section, stop checking
							if strings.HasPrefix(strings.TrimSpace(lines[k]), "- name:") {
								break
							}
						}
						break
					}
					// If we hit another step, stop checking
					if strings.HasPrefix(strings.TrimSpace(lines[j]), "- name:") {
						break
					}
				}
				break
			}
		}
	})

	// Test trial mode compilation (should suppress safe outputs and add token)
	t.Run("Trial Mode", func(t *testing.T) {
		compiler := NewCompiler(false, "", "test")
		compiler.SetTrialMode(true)      // Trial mode
		compiler.SetSkipValidation(true) // Skip validation for test

		// Parse the workflow file to get WorkflowData
		workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to parse workflow file in trial mode: %v", err)
		}

		// Generate YAML content
		lockContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to generate YAML in trial mode: %v", err)
		}

		// In trial mode, safe output jobs should be suppressed
		if !strings.Contains(lockContent, "create_pull_request:") {
			t.Error("Expected create_pull_request job in trial mode")
		}
		if !strings.Contains(lockContent, "create_issue:") {
			t.Error("Expected create_issue job in trial mode")
		}

		// Checkout in agent job should include github-token in trial mode
		// Extract the agent job section first
		agentJobStart := strings.Index(lockContent, "agent:")
		if agentJobStart == -1 {
			t.Error("Expected agent job in trial mode")
			return
		}

		// Find the end of the agent job (next job or end of file)
		agentJobEnd := len(lockContent)
		nextJobStart := strings.Index(lockContent[agentJobStart+6:], "\n  ")
		if nextJobStart != -1 {
			searchPos := agentJobStart + 6 + nextJobStart
			for idx := searchPos; idx < len(lockContent); idx++ {
				if lockContent[idx] == '\n' {
					lineStart := idx + 1
					if lineStart < len(lockContent) && lineStart+2 < len(lockContent) {
						if lockContent[lineStart:lineStart+2] == "  " && lockContent[lineStart+2] != ' ' {
							colonIdx := strings.Index(lockContent[lineStart:], ":")
							if colonIdx > 0 && colonIdx < 50 {
								agentJobEnd = idx
								break
							}
						}
					}
				}
			}
		}

		agentJobContent := lockContent[agentJobStart:agentJobEnd]

		// Check specifically that the checkout step in agent job has the token parameter
		lines := strings.Split(agentJobContent, "\n")
		foundCheckoutToken := false
		for i, line := range lines {
			if strings.Contains(line, "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8") {
				// Check the next few lines for "with:" and "token:"
				for j := i + 1; j < len(lines) && j < i+10; j++ {
					if strings.TrimSpace(lines[j]) == "with:" {
						// Found "with:" section, check for token
						for k := j + 1; k < len(lines) && k < j+5; k++ {
							if strings.Contains(lines[k], "token:") && strings.Contains(lines[k], "${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}") {
								foundCheckoutToken = true
								break
							}
							// If we hit another step or section, stop checking
							if strings.HasPrefix(strings.TrimSpace(lines[k]), "- name:") {
								break
							}
						}
						break
					}
					// If we hit another step, stop checking
					if strings.HasPrefix(strings.TrimSpace(lines[j]), "- name:") {
						break
					}
				}
				break
			}
		}
		if !foundCheckoutToken {
			t.Error("Expected github-token in checkout step in trial mode")
		}

		// Should still include the main workflow job
		if !strings.Contains(lockContent, "jobs:") {
			t.Error("Expected jobs section to be present in trial mode")
		}
	})
}

func TestTrialModeWithDifferentSafeOutputs(t *testing.T) {
	// Test different combinations of safe outputs
	testCases := []struct {
		name          string
		safeOutputs   string
		shouldContain []string
	}{
		{
			name:          "CreatePullRequest only",
			safeOutputs:   "create-pull-request",
			shouldContain: []string{"create_pull_request:"},
		},
		{
			name:          "CreateIssue only",
			safeOutputs:   "create-issue",
			shouldContain: []string{"create_issue:"},
		},
		{
			name:          "Both safe outputs",
			safeOutputs:   "create-pull-request, create-issue",
			shouldContain: []string{"create_pull_request:", "create_issue:"},
		},
		{
			name:          "No safe outputs",
			safeOutputs:   "",
			shouldContain: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create workflow content with specific safe outputs
			workflowContent := `---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
`
			if tc.safeOutputs != "" {
				// Convert comma-separated string to YAML object format
				safeOutputsList := strings.Split(tc.safeOutputs, ",")
				workflowContent += "safe-outputs:\n"
				for _, output := range safeOutputsList {
					workflowContent += "  " + strings.TrimSpace(output) + ": {}\n"
				}
			}
			workflowContent += `---

# Test Workflow

This is a test workflow for trial mode compilation.

## Instructions

- Test with different safe outputs configurations
`

			// Create temporary file
			tmpFile, err := os.CreateTemp("", "trial-mode-safe-outputs-"+strings.ReplaceAll(tc.name, " ", "_")+"-*.md")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content to file
			if _, err := tmpFile.WriteString(workflowContent); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			compiler := NewCompiler(false, "", "test")
			compiler.SetTrialMode(true)      // Trial mode
			compiler.SetSkipValidation(true) // Skip validation for test

			// Parse the workflow file to get WorkflowData
			workflowData, err := compiler.ParseWorkflowFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to parse workflow file: %v", err)
			}

			// Generate YAML content
			lockContent, err := compiler.generateYAML(workflowData, tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to generate YAML: %v", err)
			}

			// Check that specified jobs are present
			for _, presentJob := range tc.shouldContain {
				if !strings.Contains(lockContent, presentJob) {
					t.Errorf("Expected job %s to be suppressed in trial mode", presentJob)
				}
			}

			// Check that the main workflow jobs section is included
			if !strings.Contains(lockContent, "jobs:") {
				t.Error("Expected jobs section to be present in trial mode")
			}

			// In trial mode, checkout should always include github-token
			if strings.Contains(lockContent, "uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8") {
				if !strings.Contains(lockContent, "token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}") {
					t.Error("Expected github-token in checkout step in trial mode")
				}
			}
		})
	}
}

func TestTrialModeSetterAndGetter(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test default value
	if compiler.trialMode {
		t.Error("Expected trialMode to be false by default")
	}

	// Test setting trial mode to true
	compiler.SetTrialMode(true)
	if !compiler.trialMode {
		t.Error("Expected trialMode to be true after setting")
	}

	// Test setting trial mode to false
	compiler.SetTrialMode(false)
	if compiler.trialMode {
		t.Error("Expected trialMode to be false after setting to false")
	}
}
