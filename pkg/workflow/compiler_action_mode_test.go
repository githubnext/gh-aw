package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestActionModeDetection tests the DetectActionMode function
func TestActionModeDetection(t *testing.T) {
	tests := []struct {
		name          string
		githubRef     string
		githubEvent   string
		envOverride   string
		expectedMode  ActionMode
		description   string
	}{
		{
			name:         "main branch",
			githubRef:    "refs/heads/main",
			githubEvent:  "push",
			expectedMode: ActionModeRelease,
			description:  "Main branch should use release mode",
		},
		{
			name:         "release tag",
			githubRef:    "refs/tags/v1.0.0",
			githubEvent:  "push",
			expectedMode: ActionModeRelease,
			description:  "Release tags should use release mode",
		},
		{
			name:         "release event",
			githubRef:    "refs/heads/main",
			githubEvent:  "release",
			expectedMode: ActionModeRelease,
			description:  "Release events should use release mode",
		},
		{
			name:         "pull request",
			githubRef:    "refs/pull/123/merge",
			githubEvent:  "pull_request",
			expectedMode: ActionModeDev,
			description:  "Pull requests should use dev mode",
		},
		{
			name:         "feature branch",
			githubRef:    "refs/heads/feature/test",
			githubEvent:  "push",
			expectedMode: ActionModeDev,
			description:  "Feature branches should use dev mode",
		},
		{
			name:         "local development",
			githubRef:    "",
			githubEvent:  "",
			expectedMode: ActionModeDev,
			description:  "Local development (no GITHUB_REF) should use dev mode",
		},
		{
			name:         "env override to inline",
			githubRef:    "refs/heads/main",
			githubEvent:  "push",
			envOverride:  "inline",
			expectedMode: ActionModeInline,
			description:  "Environment variable should override detection",
		},
		{
			name:         "env override to dev",
			githubRef:    "refs/heads/main",
			githubEvent:  "push",
			envOverride:  "dev",
			expectedMode: ActionModeDev,
			description:  "Environment variable should override to dev mode",
		},
		{
			name:         "env override to release",
			githubRef:    "refs/heads/feature/test",
			githubEvent:  "push",
			envOverride:  "release",
			expectedMode: ActionModeRelease,
			description:  "Environment variable should override to release mode",
		},
		{
			name:         "invalid env override",
			githubRef:    "refs/heads/main",
			githubEvent:  "push",
			envOverride:  "invalid",
			expectedMode: ActionModeRelease,
			description:  "Invalid environment variable should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origRef := os.Getenv("GITHUB_REF")
			origEvent := os.Getenv("GITHUB_EVENT_NAME")
			origMode := os.Getenv("GH_AW_ACTION_MODE")
			defer func() {
				os.Setenv("GITHUB_REF", origRef)
				os.Setenv("GITHUB_EVENT_NAME", origEvent)
				os.Setenv("GH_AW_ACTION_MODE", origMode)
			}()

			// Set test environment
			if tt.githubRef != "" {
				os.Setenv("GITHUB_REF", tt.githubRef)
			} else {
				os.Unsetenv("GITHUB_REF")
			}

			if tt.githubEvent != "" {
				os.Setenv("GITHUB_EVENT_NAME", tt.githubEvent)
			} else {
				os.Unsetenv("GITHUB_EVENT_NAME")
			}

			if tt.envOverride != "" {
				os.Setenv("GH_AW_ACTION_MODE", tt.envOverride)
			} else {
				os.Unsetenv("GH_AW_ACTION_MODE")
			}

			// Test detection
			mode := DetectActionMode()
			if mode != tt.expectedMode {
				t.Errorf("%s: expected mode %s, got %s", tt.description, tt.expectedMode, mode)
			}
		})
	}
}

// TestActionModeReleaseValidation tests that release mode is valid
func TestActionModeReleaseValidation(t *testing.T) {
	if !ActionModeRelease.IsValid() {
		t.Error("ActionModeRelease should be valid")
	}

	if ActionModeRelease.String() != "release" {
		t.Errorf("Expected string 'release', got %q", ActionModeRelease.String())
	}
}

// TestConvertToRemoteActionRef tests conversion of local paths to remote references
func TestConvertToRemoteActionRef(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Mock getCurrentCommitSHA to return a known value
	mockSHA := "abc123def456abc123def456abc123def456abc1"

	// Save original GITHUB_SHA
	origSHA := os.Getenv("GITHUB_SHA")
	defer os.Setenv("GITHUB_SHA", origSHA)

	// Set mock SHA
	os.Setenv("GITHUB_SHA", mockSHA)

	tests := []struct {
		name        string
		localPath   string
		expectedRef string
		description string
	}{
		{
			name:        "local path with ./ prefix",
			localPath:   "./actions/create-issue",
			expectedRef: "githubnext/gh-aw/actions/create-issue@" + mockSHA,
			description: "Should strip ./ and add repo prefix with SHA",
		},
		{
			name:        "local path without ./ prefix",
			localPath:   "actions/create-issue",
			expectedRef: "githubnext/gh-aw/actions/create-issue@" + mockSHA,
			description: "Should add repo prefix with SHA",
		},
		{
			name:        "nested action path",
			localPath:   "./actions/nested/action",
			expectedRef: "githubnext/gh-aw/actions/nested/action@" + mockSHA,
			description: "Should handle nested paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := compiler.convertToRemoteActionRef(tt.localPath)
			if ref != tt.expectedRef {
				t.Errorf("%s: expected %q, got %q", tt.description, tt.expectedRef, ref)
			}
		})
	}
}

// TestResolveActionReference tests the resolveActionReference function
func TestResolveActionReference(t *testing.T) {
	// Save original GITHUB_SHA
	origSHA := os.Getenv("GITHUB_SHA")
	defer os.Setenv("GITHUB_SHA", origSHA)

	// Set mock SHA
	mockSHA := "abc123def456abc123def456abc123def456abc1"
	os.Setenv("GITHUB_SHA", mockSHA)

	tests := []struct {
		name         string
		actionMode   ActionMode
		localPath    string
		expectedRef  string
		shouldBeEmpty bool
		description  string
	}{
		{
			name:        "dev mode",
			actionMode:  ActionModeDev,
			localPath:   "./actions/create-issue",
			expectedRef: "./actions/create-issue",
			description: "Dev mode should return local path",
		},
		{
			name:        "release mode",
			actionMode:  ActionModeRelease,
			localPath:   "./actions/create-issue",
			expectedRef: "githubnext/gh-aw/actions/create-issue@" + mockSHA,
			description: "Release mode should return SHA-pinned remote reference",
		},
		{
			name:          "inline mode",
			actionMode:    ActionModeInline,
			localPath:     "./actions/create-issue",
			shouldBeEmpty: true,
			description:   "Inline mode should return empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "1.0.0")
			compiler.SetActionMode(tt.actionMode)

			data := &WorkflowData{}
			ref := compiler.resolveActionReference(tt.localPath, data)

			if tt.shouldBeEmpty {
				if ref != "" {
					t.Errorf("%s: expected empty string, got %q", tt.description, ref)
				}
			} else {
				if ref != tt.expectedRef {
					t.Errorf("%s: expected %q, got %q", tt.description, tt.expectedRef, ref)
				}
			}
		})
	}
}

// TestReleaseModeCompilation tests workflow compilation in release mode
func TestReleaseModeCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Save original environment
	origSHA := os.Getenv("GITHUB_SHA")
	defer os.Setenv("GITHUB_SHA", origSHA)

	// Set mock SHA for testing
	mockSHA := "abc123def456abc123def456abc123def456abc1"
	os.Setenv("GITHUB_SHA", mockSHA)

	// Create a test workflow file
	workflowContent := `---
name: Test Release Mode
on: issues
safe-outputs:
  create-issue:
    max: 1
---

Test workflow with release mode.
`

	workflowPath := tempDir + "/test-workflow.md"
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Register a test script with an action path using RegisterWithAction
	// This simulates what would happen once we have actual actions in the actions/ directory
	testScript := `
const { core } = require('@actions/core');
core.info('Creating issue');
`
	// Save the original registration to restore later
	origScript := DefaultScriptRegistry.Get("create_issue")
	
	// Register with action path for this test
	DefaultScriptRegistry.RegisterWithAction(
		"create_issue",
		testScript,
		RuntimeModeGitHubScript,
		"./actions/create-issue",
	)
	
	// Restore original registration after test
	defer func() {
		if origScript != "" {
			DefaultScriptRegistry.RegisterWithMode("create_issue", origScript, RuntimeModeGitHubScript)
		}
	}()

	// Compile with release action mode
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeRelease)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Log lines containing "uses:" for debugging
	t.Logf("Checking lock file for action references...")
	lines := strings.Split(lockStr, "\n")
	createIssueStepFound := false
	for i, line := range lines {
		if strings.Contains(line, "uses:") {
			t.Logf("Line %d: %s", i, line)
		}
		// Check if this is the create_issue step
		if strings.Contains(line, "uses: githubnext/gh-aw/actions/create-issue@") {
			createIssueStepFound = true
		}
	}

	// Verify it uses SHA-pinned remote action reference
	expectedRef := "uses: githubnext/gh-aw/actions/create-issue@" + mockSHA
	if !createIssueStepFound {
		t.Errorf("Expected SHA-pinned remote reference %q not found in lock file", expectedRef)
	}

	// Verify it does NOT contain local path reference
	if strings.Contains(lockStr, "uses: ./actions/create-issue") {
		t.Error("Lock file should not contain local action path in release mode")
	}

	// Verify the create_issue step specifically does NOT use actions/github-script
	// (other steps may still use it, which is fine)
	inCreateIssueJob := false
	for i, line := range lines {
		// Detect when we enter the create_issue job
		if strings.Contains(line, "create_issue:") && !strings.Contains(line, "steps.create_issue") {
			inCreateIssueJob = true
		}
		// Detect when we exit to the next job
		if inCreateIssueJob && i > 0 && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && line != "" {
			// We've exited the create_issue job (indentation changed)
			if !strings.Contains(line, "steps:") && !strings.Contains(line, "permissions:") {
				inCreateIssueJob = false
			}
		}
		// Check if we're using actions/github-script in the create_issue job
		if inCreateIssueJob && strings.Contains(line, "uses: actions/github-script@") {
			t.Error("create_issue job should not use actions/github-script@ in release mode")
			break
		}
	}

	// Verify it has the token input
	if !strings.Contains(lockStr, "token:") {
		t.Error("Expected 'token:' input not found for custom action")
	}
}

// TestDevModeCompilation tests workflow compilation in dev mode
func TestDevModeCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test workflow file
	workflowContent := `---
name: Test Dev Mode
on: issues
safe-outputs:
  create-issue:
    max: 1
---

Test workflow with dev mode.
`

	workflowPath := tempDir + "/test-workflow.md"
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Register a test script with an action path using RegisterWithAction
	testScript := `
const { core } = require('@actions/core');
core.info('Creating issue');
`
	// Save the original registration to restore later
	origScript := DefaultScriptRegistry.Get("create_issue")
	
	// Register with action path for this test
	DefaultScriptRegistry.RegisterWithAction(
		"create_issue",
		testScript,
		RuntimeModeGitHubScript,
		"./actions/create-issue",
	)
	
	// Restore original registration after test
	defer func() {
		if origScript != "" {
			DefaultScriptRegistry.RegisterWithMode("create_issue", origScript, RuntimeModeGitHubScript)
		}
	}()

	// Compile with dev action mode
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeDev)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify it uses local action path
	if !strings.Contains(lockStr, "uses: ./actions/create-issue") {
		t.Error("Expected local action reference './actions/create-issue' not found in lock file")
		
		// Log lines containing "uses:" for debugging
		t.Logf("Lock file 'uses:' lines:")
		lines := strings.Split(lockStr, "\n")
		for i, line := range lines {
			if strings.Contains(line, "uses:") {
				t.Logf("Line %d: %s", i, line)
			}
		}
	}

	// Verify it does NOT contain remote SHA-pinned reference
	if strings.Contains(lockStr, "uses: githubnext/gh-aw/actions/create-issue@") {
		t.Error("Lock file should not contain remote SHA-pinned reference in dev mode")
	}

	// Verify the create_issue step specifically does NOT use actions/github-script
	// (other steps may still use it, which is fine)
	lines := strings.Split(lockStr, "\n")
	inCreateIssueJob := false
	for i, line := range lines {
		// Detect when we enter the create_issue job
		if strings.Contains(line, "create_issue:") && !strings.Contains(line, "steps.create_issue") {
			inCreateIssueJob = true
		}
		// Detect when we exit to the next job
		if inCreateIssueJob && i > 0 && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && line != "" {
			// We've exited the create_issue job (indentation changed)
			if !strings.Contains(line, "steps:") && !strings.Contains(line, "permissions:") {
				inCreateIssueJob = false
			}
		}
		// Check if we're using actions/github-script in the create_issue job
		if inCreateIssueJob && strings.Contains(line, "uses: actions/github-script@") {
			t.Error("create_issue job should not use actions/github-script@ in dev mode")
			break
		}
	}
}

// TestGetCurrentCommitSHA tests the getCurrentCommitSHA function
func TestGetCurrentCommitSHA(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Save original GITHUB_SHA
	origSHA := os.Getenv("GITHUB_SHA")
	defer os.Setenv("GITHUB_SHA", origSHA)

	t.Run("GITHUB_SHA environment variable", func(t *testing.T) {
		mockSHA := "1234567890abcdef1234567890abcdef12345678"
		os.Setenv("GITHUB_SHA", mockSHA)

		sha := compiler.getCurrentCommitSHA()
		if sha != mockSHA {
			t.Errorf("Expected SHA %q from GITHUB_SHA, got %q", mockSHA, sha)
		}
	})

	t.Run("git rev-parse fallback", func(t *testing.T) {
		os.Unsetenv("GITHUB_SHA")

		sha := compiler.getCurrentCommitSHA()
		// The SHA should be a valid 40-character hex string
		if len(sha) != 40 {
			t.Errorf("Expected 40-character SHA from git rev-parse, got %d characters: %q", len(sha), sha)
		}

		// Verify it's a valid hex string
		for _, c := range sha {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("SHA contains invalid character %q: %s", c, sha)
			}
		}
	})
}
