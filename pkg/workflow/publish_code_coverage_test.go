//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestParseUploadCodeCoverageConfig(t *testing.T) {
	c := &Compiler{}

	tests := []struct {
		name     string
		input    map[string]any
		expected *UploadCodeCoverageConfig
	}{
		{
			name: "upload-code-coverage config with custom values",
			input: map[string]any{
				"upload-code-coverage": map[string]any{
					"fail-on-error":               false,
					"wait-for-processing-timeout": 60,
					"github-token":                "${{ secrets.CUSTOM_TOKEN }}",
				},
			},
			expected: &UploadCodeCoverageConfig{
				FailOnError:              boolPtr(false),
				WaitForProcessingTimeout: 60,
				BaseSafeOutputConfig:     BaseSafeOutputConfig{GitHubToken: "${{ secrets.CUSTOM_TOKEN }}", Max: strPtr("1")},
			},
		},
		{
			name: "upload-code-coverage config with defaults (empty map)",
			input: map[string]any{
				"upload-code-coverage": map[string]any{},
			},
			expected: &UploadCodeCoverageConfig{
				FailOnError:              boolPtr(true),
				WaitForProcessingTimeout: defaultCodeCoverageWaitForProcessingTimeout,
				BaseSafeOutputConfig:     BaseSafeOutputConfig{Max: strPtr("1")},
			},
		},
		{
			name: "upload-code-coverage config null value uses defaults",
			input: map[string]any{
				"upload-code-coverage": nil,
			},
			expected: &UploadCodeCoverageConfig{
				FailOnError:              boolPtr(true),
				WaitForProcessingTimeout: defaultCodeCoverageWaitForProcessingTimeout,
				BaseSafeOutputConfig:     BaseSafeOutputConfig{Max: strPtr("1")},
			},
		},
		{
			name: "upload-code-coverage explicitly disabled",
			input: map[string]any{
				"upload-code-coverage": false,
			},
			expected: nil,
		},
		{
			name:     "no upload-code-coverage config",
			input:    map[string]any{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.parseUploadCodeCoverageConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected %+v, got nil", tt.expected)
			}

			if result.FailOnError == nil || tt.expected.FailOnError == nil || *result.FailOnError != *tt.expected.FailOnError {
				t.Errorf("FailOnError = %v, want %v", result.FailOnError, tt.expected.FailOnError)
			}
			if result.WaitForProcessingTimeout != tt.expected.WaitForProcessingTimeout {
				t.Errorf("WaitForProcessingTimeout = %d, want %d", result.WaitForProcessingTimeout, tt.expected.WaitForProcessingTimeout)
			}
			if result.GitHubToken != tt.expected.GitHubToken {
				t.Errorf("GitHubToken = %q, want %q", result.GitHubToken, tt.expected.GitHubToken)
			}
			gotMax := ""
			if result.Max != nil {
				gotMax = *result.Max
			}
			wantMax := ""
			if tt.expected.Max != nil {
				wantMax = *tt.expected.Max
			}
			if gotMax != wantMax {
				t.Errorf("Max = %q, want %q", gotMax, wantMax)
			}
		})
	}
}

func TestBuildUploadCodeCoverageJob(t *testing.T) {
	c := NewCompiler()
	data := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			UploadCodeCoverage: &UploadCodeCoverageConfig{
				FailOnError:              boolPtr(true),
				WaitForProcessingTimeout: 160,
			},
		},
	}

	job, err := c.buildUploadCodeCoverageJob(data, "agent")
	if err != nil {
		t.Fatalf("Failed to build upload_code_coverage job: %v", err)
	}
	if job == nil {
		t.Fatal("Expected non-nil job")
	}

	var stepsStrSb strings.Builder
	for _, step := range job.Steps {
		stepsStrSb.WriteString(step)
	}
	stepsStr := stepsStrSb.String()

	if !strings.Contains(stepsStr, "actions/upload-code-coverage") {
		t.Error("Expected step to reference actions/upload-code-coverage")
	}
	if !strings.Contains(stepsStr, "language: ${{ needs.safe_outputs.outputs.upload_code_coverage_language }}") {
		t.Error("Expected language input to be wired from safe_outputs job outputs")
	}
	if !strings.Contains(stepsStr, "label: ${{ needs.safe_outputs.outputs.upload_code_coverage_label }}") {
		t.Error("Expected label input to be wired from safe_outputs job outputs")
	}
	if !strings.Contains(stepsStr, "fail-on-error: true") {
		t.Error("Expected fail-on-error: true to be rendered")
	}
	if !strings.Contains(stepsStr, "wait-for-processing-timeout: 160") {
		t.Error("Expected wait-for-processing-timeout: 160 to be rendered")
	}
	if !strings.Contains(stepsStr, "Download upload-code-coverage staging") {
		t.Error("Expected download step for staging artifact")
	}

	if job.If != "needs.safe_outputs.outputs.upload_code_coverage_file != ''" {
		t.Errorf("Unexpected job condition: %s", job.If)
	}

	foundContents := false
	foundCodeQuality := false
	for _, line := range strings.Split(job.Permissions, "\n") {
		if strings.Contains(line, "contents:") && strings.Contains(line, "read") {
			foundContents = true
		}
		if strings.Contains(line, "code-quality:") && strings.Contains(line, "write") {
			foundCodeQuality = true
		}
	}
	if !foundContents {
		t.Errorf("Expected contents: read permission, got: %s", job.Permissions)
	}
	if !foundCodeQuality {
		t.Errorf("Expected code-quality: write permission, got: %s", job.Permissions)
	}

	if len(job.Needs) != 2 || job.Needs[0] != "agent" || job.Needs[1] != "safe_outputs" {
		t.Errorf("Expected job.Needs = [agent, safe_outputs], got %v", job.Needs)
	}
}

func TestBuildUploadCodeCoverageJobMissingConfig(t *testing.T) {
	c := NewCompiler()
	data := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	_, err := c.buildUploadCodeCoverageJob(data, "agent")
	if err == nil {
		t.Fatal("Expected error when upload-code-coverage configuration is missing")
	}
}

func TestGenerateSafeOutputsCodeCoverageStagingUpload(t *testing.T) {
	c := NewCompiler()
	data := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			UploadCodeCoverage: &UploadCodeCoverageConfig{
				FailOnError: boolPtr(true),
			},
		},
	}

	var builder strings.Builder
	generateSafeOutputsCodeCoverageStagingUpload(&builder, data, c.getActionPin)
	out := builder.String()

	if !strings.Contains(out, SafeOutputsUploadCodeCoverageStagingArtifactName) {
		t.Errorf("Expected staging artifact name %q in output, got: %s", SafeOutputsUploadCodeCoverageStagingArtifactName, out)
	}
	if !strings.Contains(out, "actions/upload-artifact") {
		t.Error("Expected staging upload step to use actions/upload-artifact")
	}
}

func TestGenerateSafeOutputsCodeCoverageStagingUploadNoConfig(t *testing.T) {
	data := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	var builder strings.Builder
	generateSafeOutputsCodeCoverageStagingUpload(&builder, data, func(string) string { return "" })
	if builder.Len() != 0 {
		t.Errorf("Expected no output when upload-code-coverage is not configured, got: %s", builder.String())
	}
}
