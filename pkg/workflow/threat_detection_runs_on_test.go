package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreatDetectionUsesRunsOnFromSafeOutputs(t *testing.T) {
	tests := []struct {
		name           string
		frontmatter    string
		expectedRunsOn string
	}{
		{
			name: "default runs-on when not specified",
			frontmatter: `---
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn: "runs-on: ubuntu-slim",
		},
		{
			name: "custom runs-on from safe-outputs",
			frontmatter: `---
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
  runs-on: self-hosted
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn: "runs-on: self-hosted",
		},
		{
			name: "windows runner from safe-outputs",
			frontmatter: `---
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
  runs-on: windows-latest
---

# Test Workflow

This is a test workflow.`,
			expectedRunsOn: "runs-on: windows-latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tmpDir, err := os.MkdirTemp("", "threat-detection-runs-on-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test.md")
			err = os.WriteFile(testFile, []byte(tt.frontmatter), 0644)
			if err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the compiled lock file
			lockFile := filepath.Join(tmpDir, "test.lock.yml")
			yamlContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			yamlStr := string(yamlContent)

			// Find the detection job in the YAML
			if !strings.Contains(yamlStr, "detection:") {
				t.Fatal("Expected compiled YAML to contain detection job, but it didn't")
			}

			// Extract the detection job section
			detectionIndex := strings.Index(yamlStr, "detection:")
			if detectionIndex == -1 {
				t.Fatal("Detection job not found in compiled YAML")
			}

			// Get a substring starting from the detection job to the next job or end
			detectionSection := yamlStr[detectionIndex:]

			// Find the next job (starts with a non-indented line or end of file)
			lines := strings.Split(detectionSection, "\n")
			var detectionJobYAML string
			for i, line := range lines {
				if i > 0 && len(line) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
					// Found the start of the next job
					detectionJobYAML = strings.Join(lines[:i], "\n")
					break
				}
			}
			if detectionJobYAML == "" {
				detectionJobYAML = detectionSection
			}

			// Check that the detection job uses the expected runs-on
			if !strings.Contains(detectionJobYAML, tt.expectedRunsOn) {
				t.Errorf("Expected detection job to contain %q, but it didn't.\nDetection job section:\n%s", tt.expectedRunsOn, detectionJobYAML)
			}

			// Ensure it doesn't use the old hardcoded value when we expect a different one
			if tt.expectedRunsOn != "runs-on: ubuntu-latest" && strings.Contains(detectionJobYAML, "runs-on: ubuntu-latest") {
				t.Errorf("Detection job still uses hardcoded 'runs-on: ubuntu-latest' instead of %q.\nDetection job section:\n%s", tt.expectedRunsOn, detectionJobYAML)
			}
		})
	}
}

func TestThreatDetectionJobDefaultRunsOn(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test the default runs-on when safe-outputs config has no runs-on specified
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	job, err := compiler.buildThreatDetectionJob(data, "agent")
	if err != nil {
		t.Fatalf("Failed to build detection job: %v", err)
	}

	expectedRunsOn := "runs-on: ubuntu-slim"
	if job.RunsOn != expectedRunsOn {
		t.Errorf("Expected default runs-on to be %q, got %q", expectedRunsOn, job.RunsOn)
	}
}

func TestThreatDetectionJobCustomRunsOn(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test custom runs-on configuration
	tests := []struct {
		name           string
		runsOn         string
		expectedRunsOn string
	}{
		{
			name:           "self-hosted runner",
			runsOn:         "self-hosted",
			expectedRunsOn: "runs-on: self-hosted",
		},
		{
			name:           "windows runner",
			runsOn:         "windows-latest",
			expectedRunsOn: "runs-on: windows-latest",
		},
		{
			name:           "macos runner",
			runsOn:         "macos-latest",
			expectedRunsOn: "runs-on: macos-latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					RunsOn:          tt.runsOn,
					ThreatDetection: &ThreatDetectionConfig{},
				},
			}

			job, err := compiler.buildThreatDetectionJob(data, "agent")
			if err != nil {
				t.Fatalf("Failed to build detection job: %v", err)
			}

			if job.RunsOn != tt.expectedRunsOn {
				t.Errorf("Expected runs-on to be %q, got %q", tt.expectedRunsOn, job.RunsOn)
			}
		})
	}
}
