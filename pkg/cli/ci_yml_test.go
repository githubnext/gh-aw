package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestCIWorkflowConcurrency validates that the ci.yml workflow has proper concurrency configuration
func TestCIWorkflowConcurrency(t *testing.T) {
	// Read the ci.yml file
	ciPath := filepath.Join("..", "..", ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("Failed to read ci.yml: %v", err)
	}

	// Parse YAML
	var workflow map[string]any
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Failed to parse ci.yml: %v", err)
	}

	// Get jobs
	jobs, ok := workflow["jobs"].(map[string]any)
	if !ok {
		t.Fatal("No jobs found in ci.yml")
	}

	// Expected concurrency patterns
	expectedPatterns := map[string]struct {
		hasMatrix       bool
		expectedPattern string
	}{
		"test": {
			hasMatrix:       false,
			expectedPattern: "${{ github.workflow }}-${{ github.ref }}-${{ github.job }}",
		},
		"build": {
			hasMatrix:       true,
			expectedPattern: "${{ github.workflow }}-${{ github.ref }}-${{ github.job }}-${{ matrix.os }}",
		},
		"js": {
			hasMatrix:       false,
			expectedPattern: "${{ github.workflow }}-${{ github.ref }}-${{ github.job }}",
		},
		"lint": {
			hasMatrix:       false,
			expectedPattern: "${{ github.workflow }}-${{ github.ref }}-${{ github.job }}",
		},
	}

	// Validate each job
	for jobName, expected := range expectedPatterns {
		t.Run(jobName, func(t *testing.T) {
			job, ok := jobs[jobName].(map[string]any)
			if !ok {
				t.Fatalf("Job %s not found or invalid", jobName)
			}

			// Check for concurrency configuration
			concurrency, ok := job["concurrency"].(map[string]any)
			if !ok {
				t.Fatalf("Job %s has no concurrency configuration", jobName)
			}

			// Check group
			group, ok := concurrency["group"].(string)
			if !ok {
				t.Fatalf("Job %s concurrency has no group string", jobName)
			}

			// Validate the group pattern
			if group != expected.expectedPattern {
				t.Errorf("Job %s has incorrect concurrency group.\nExpected: %s\nGot: %s",
					jobName, expected.expectedPattern, group)
			}

			// Validate that cancel-in-progress is true
			cancelInProgress, ok := concurrency["cancel-in-progress"].(bool)
			if !ok || !cancelInProgress {
				t.Errorf("Job %s should have cancel-in-progress: true", jobName)
			}

			// For matrix jobs, verify matrix configuration exists
			if expected.hasMatrix {
				strategy, ok := job["strategy"].(map[string]any)
				if !ok {
					t.Fatalf("Job %s should have strategy configuration", jobName)
				}

				matrix, ok := strategy["matrix"].(map[string]any)
				if !ok {
					t.Fatalf("Job %s should have matrix configuration", jobName)
				}

				// Verify os matrix exists
				osMatrix, ok := matrix["os"].([]any)
				if !ok || len(osMatrix) == 0 {
					t.Fatalf("Job %s should have os matrix with values", jobName)
				}

				// Verify that the concurrency group includes matrix.os
				if !strings.Contains(group, "${{ matrix.os }}") {
					t.Errorf("Job %s concurrency group should include ${{ matrix.os }}", jobName)
				}
			}

			// Verify best practices: using github.job instead of hardcoded job name
			if !strings.Contains(group, "${{ github.job }}") {
				t.Errorf("Job %s should use ${{ github.job }} in concurrency group", jobName)
			}

			// Verify no redundant prefixes
			if strings.HasPrefix(group, "ci-") {
				t.Errorf("Job %s has redundant 'ci-' prefix in concurrency group", jobName)
			}

			// Verify no unnecessary event_name variable
			if strings.Contains(group, "${{ github.event_name }}") {
				t.Errorf("Job %s should not use ${{ github.event_name }} in concurrency group (adds unnecessary variability)", jobName)
			}
		})
	}
}

// TestCIWorkflowMatrixConcurrencyUniqueness validates that matrix jobs have unique concurrency groups
func TestCIWorkflowMatrixConcurrencyUniqueness(t *testing.T) {
	// Read the ci.yml file
	ciPath := filepath.Join("..", "..", ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("Failed to read ci.yml: %v", err)
	}

	// Parse YAML
	var workflow map[string]any
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Failed to parse ci.yml: %v", err)
	}

	// Get jobs
	jobs, ok := workflow["jobs"].(map[string]any)
	if !ok {
		t.Fatal("No jobs found in ci.yml")
	}

	// Check the build job specifically
	buildJob, ok := jobs["build"].(map[string]any)
	if !ok {
		t.Fatal("Build job not found")
	}

	// Get strategy and matrix
	strategy, ok := buildJob["strategy"].(map[string]any)
	if !ok {
		t.Fatal("Build job has no strategy")
	}

	matrix, ok := strategy["matrix"].(map[string]any)
	if !ok {
		t.Fatal("Build job has no matrix")
	}

	osMatrix, ok := matrix["os"].([]any)
	if !ok {
		t.Fatal("Build job has no os matrix")
	}

	// Get concurrency configuration
	concurrency, ok := buildJob["concurrency"].(map[string]any)
	if !ok {
		t.Fatal("Build job has no concurrency configuration")
	}

	group, ok := concurrency["group"].(string)
	if !ok {
		t.Fatal("Build job concurrency has no group")
	}

	// Simulate concurrency groups for each matrix value
	// This validates that each matrix instance would get a unique group
	groups := make(map[string]bool)
	for _, os := range osMatrix {
		osStr, ok := os.(string)
		if !ok {
			t.Fatalf("Matrix os value is not a string: %v", os)
		}

		// Simulate the concurrency group for this matrix instance
		// Replace the template with actual values
		simulatedGroup := strings.ReplaceAll(group, "${{ github.workflow }}", "CI")
		simulatedGroup = strings.ReplaceAll(simulatedGroup, "${{ github.ref }}", "refs/heads/main")
		simulatedGroup = strings.ReplaceAll(simulatedGroup, "${{ github.job }}", "build")
		simulatedGroup = strings.ReplaceAll(simulatedGroup, "${{ matrix.os }}", osStr)

		// Check if this group is unique
		if groups[simulatedGroup] {
			t.Errorf("Duplicate concurrency group detected: %s", simulatedGroup)
		}
		groups[simulatedGroup] = true

		t.Logf("Matrix os=%s would get concurrency group: %s", osStr, simulatedGroup)
	}

	// Verify we have the expected number of unique groups
	expectedCount := len(osMatrix)
	if len(groups) != expectedCount {
		t.Errorf("Expected %d unique concurrency groups, got %d", expectedCount, len(groups))
	}
}
