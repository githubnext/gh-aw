package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaximumPatchSizeEnvironmentVariable(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "patch-size-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name                 string
		frontmatterContent   string
		expectedEnvValue     string
		shouldContainPushJob bool
		shouldContainPRJob   bool
	}{
		{
			name: "default patch size (no config)",
			frontmatterContent: `---
on: push
safe-outputs:
  push-to-pull-request-branch: null
  create-pull-request: null
---

# Test Workflow

This workflow tests default patch size configuration.`,
			expectedEnvValue:     "GH_AW_MAX_PATCH_SIZE: 1024",
			shouldContainPushJob: true,
			shouldContainPRJob:   true,
		},
		{
			name: "custom patch size 512 KB",
			frontmatterContent: `---
on: push
safe-outputs:
  max-patch-size: 512
  push-to-pull-request-branch: null
  create-pull-request: null
---

# Test Workflow

This workflow tests custom 512KB patch size configuration.`,
			expectedEnvValue:     "GH_AW_MAX_PATCH_SIZE: 512",
			shouldContainPushJob: true,
			shouldContainPRJob:   true,
		},
		{
			name: "custom patch size 2MB",
			frontmatterContent: `---
on: push
safe-outputs:
  max-patch-size: 2048
  create-pull-request: null
---

# Test Workflow

This workflow tests custom 2MB patch size configuration.`,
			expectedEnvValue:     "GH_AW_MAX_PATCH_SIZE: 2048",
			shouldContainPushJob: false,
			shouldContainPRJob:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create markdown file
			mdFile := filepath.Join(tmpDir, tt.name+".md")
			err := os.WriteFile(mdFile, []byte(tt.frontmatterContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Compile workflow
			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(mdFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Determine expected lock file name
			lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"

			// Read lock file content
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockContentStr := string(lockContent)

			// Check that the environment variable is set correctly in push job
			if tt.shouldContainPushJob {
				if !strings.Contains(lockContentStr, "push_to_pull_request_branch:") {
					t.Errorf("Expected push_to_pull_request_branch job to be generated")
				}
				if !strings.Contains(lockContentStr, tt.expectedEnvValue) {
					t.Errorf("Expected '%s' to be found in push job, got:\n%s", tt.expectedEnvValue, lockContentStr)
				}
			}

			// Check that the environment variable is set correctly in create PR job
			if tt.shouldContainPRJob {
				if !strings.Contains(lockContentStr, "create_pull_request:") {
					t.Errorf("Expected create_pull_request job to be generated")
				}
				if !strings.Contains(lockContentStr, tt.expectedEnvValue) {
					t.Errorf("Expected '%s' to be found in create PR job, got:\n%s", tt.expectedEnvValue, lockContentStr)
				}
			}

			// Cleanup
			if err := os.Remove(lockFile); err != nil {
				t.Logf("Warning: Failed to remove lock file: %v", err)
			}
		})
	}
}

func TestPatchSizeWithInvalidValues(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "patch-size-invalid-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name               string
		frontmatterContent string
		expectedEnvValue   string
	}{
		{
			name: "very small patch size should work",
			frontmatterContent: `---
on: push
safe-outputs:
  max-patch-size: 1
  push-to-pull-request-branch: null
---

# Test Workflow

This workflow tests very small patch size configuration.`,
			expectedEnvValue: "GH_AW_MAX_PATCH_SIZE: 1",
		},
		{
			name: "large valid patch size should work",
			frontmatterContent: `---
on: push
safe-outputs:
  max-patch-size: 10240
  create-pull-request: null
---

# Test Workflow

This workflow tests large valid patch size configuration.`,
			expectedEnvValue: "GH_AW_MAX_PATCH_SIZE: 10240",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create markdown file
			mdFile := filepath.Join(tmpDir, tt.name+".md")
			err := os.WriteFile(mdFile, []byte(tt.frontmatterContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Compile workflow
			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(mdFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Determine expected lock file name
			lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"

			// Read lock file content
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockContentStr := string(lockContent)

			// Check that the environment variable falls back to default
			if !strings.Contains(lockContentStr, tt.expectedEnvValue) {
				t.Errorf("Expected '%s' to be found in workflow, got:\n%s", tt.expectedEnvValue, lockContentStr)
			}

			// Cleanup
			if err := os.Remove(lockFile); err != nil {
				t.Logf("Warning: Failed to remove lock file: %v", err)
			}
		})
	}
}
