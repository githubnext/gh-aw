//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestParseContentRedactionConfig validates parsing of the content-redaction configuration
// from the safe-outputs frontmatter.
func TestParseContentRedactionConfig(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		outputMap      map[string]any
		expectedNil    bool
		expectedAgents []string
		expectedModel  string
		expectedOnFail string
		expectedScope  []string
		expectExprExpr bool
	}{
		{
			name:        "missing content-redaction should return nil",
			outputMap:   map[string]any{},
			expectedNil: true,
		},
		{
			name: "boolean false should return nil",
			outputMap: map[string]any{
				"content-redaction": false,
			},
			expectedNil: true,
		},
		{
			name: "boolean true (no agent) should return nil",
			outputMap: map[string]any{
				"content-redaction": true,
			},
			expectedNil: true,
		},
		{
			name: "single inline string",
			outputMap: map[string]any{
				"content-redaction": "Do not disclose security vulnerabilities",
			},
			expectedNil:    false,
			expectedAgents: []string{"Do not disclose security vulnerabilities"},
		},
		{
			name: "array shorthand",
			outputMap: map[string]any{
				"content-redaction": []any{
					"https://corp.example.com/policy.md",
					".github/policies/inclusive-language.md",
					"Never mention internal codenames",
				},
			},
			expectedNil: false,
			expectedAgents: []string{
				"https://corp.example.com/policy.md",
				".github/policies/inclusive-language.md",
				"Never mention internal codenames",
			},
		},
		{
			name: "array shorthand with empty array returns nil",
			outputMap: map[string]any{
				"content-redaction": []any{},
			},
			expectedNil: true,
		},
		{
			name: "object form with all fields",
			outputMap: map[string]any{
				"content-redaction": map[string]any{
					"agent":             "https://corp.example.com/policy.md",
					"model":             "gpt-4o-mini",
					"on-failure":        "warn",
					"scope":             []any{"add-comment", "create-issue"},
					"continue-on-error": true,
				},
			},
			expectedNil:    false,
			expectedAgents: []string{"https://corp.example.com/policy.md"},
			expectedModel:  "gpt-4o-mini",
			expectedOnFail: "warn",
			expectedScope:  []string{"add-comment", "create-issue"},
		},
		{
			name: "object form with agent array",
			outputMap: map[string]any{
				"content-redaction": map[string]any{
					"agent": []any{
						"https://corp.example.com/policy.md",
						".github/policies/redaction.md",
					},
				},
			},
			expectedNil: false,
			expectedAgents: []string{
				"https://corp.example.com/policy.md",
				".github/policies/redaction.md",
			},
		},
		{
			name: "object form with enabled: false returns nil",
			outputMap: map[string]any{
				"content-redaction": map[string]any{
					"enabled": false,
					"agent":   "Some policy",
				},
			},
			expectedNil: true,
		},
		{
			name: "object form without agent returns nil",
			outputMap: map[string]any{
				"content-redaction": map[string]any{
					"model": "gpt-4o-mini",
				},
			},
			expectedNil: true,
		},
		{
			name: "runtime expression string enables conditional redaction",
			outputMap: map[string]any{
				"content-redaction": "${{ inputs.enable-content-redaction }}",
			},
			expectedNil:    false,
			expectExprExpr: true,
		},
		{
			name: "object form with expression-controlled enabled",
			outputMap: map[string]any{
				"content-redaction": map[string]any{
					"enabled": "${{ inputs.enable-redaction }}",
					"agent":   "Do not disclose CVE IDs",
				},
			},
			expectedNil:    false,
			expectExprExpr: true,
			expectedAgents: []string{"Do not disclose CVE IDs"},
		},
		{
			name: "unknown on-failure value is ignored (defaults to block)",
			outputMap: map[string]any{
				"content-redaction": map[string]any{
					"agent":      "Policy",
					"on-failure": "invalid-value",
				},
			},
			expectedNil:    false,
			expectedAgents: []string{"Policy"},
			expectedOnFail: "", // unknown value, field is not set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := compiler.parseContentRedactionConfig(tt.outputMap)

			if tt.expectedNil {
				if cr != nil {
					t.Errorf("expected nil ContentRedactionConfig, got %+v", cr)
				}
				return
			}

			if cr == nil {
				t.Fatal("expected non-nil ContentRedactionConfig, got nil")
			}

			if tt.expectExprExpr {
				if cr.EnabledExpr == nil {
					t.Error("expected EnabledExpr to be set")
				}
			}

			if len(tt.expectedAgents) > 0 {
				if len(cr.Agent) != len(tt.expectedAgents) {
					t.Errorf("expected %d agent entries, got %d: %v", len(tt.expectedAgents), len(cr.Agent), cr.Agent)
				} else {
					for i, want := range tt.expectedAgents {
						if cr.Agent[i] != want {
							t.Errorf("agent[%d]: expected %q, got %q", i, want, cr.Agent[i])
						}
					}
				}
			}

			if tt.expectedModel != "" && cr.Model != tt.expectedModel {
				t.Errorf("model: expected %q, got %q", tt.expectedModel, cr.Model)
			}

			if tt.expectedOnFail != "" && cr.OnFailure != tt.expectedOnFail {
				t.Errorf("on-failure: expected %q, got %q", tt.expectedOnFail, cr.OnFailure)
			}

			if len(tt.expectedScope) > 0 {
				if len(cr.Scope) != len(tt.expectedScope) {
					t.Errorf("scope: expected %v, got %v", tt.expectedScope, cr.Scope)
				}
			}
		})
	}
}

// TestIsContentRedactionEnabled validates the IsContentRedactionEnabled helper.
func TestIsContentRedactionEnabled(t *testing.T) {
	tests := []struct {
		name     string
		so       *SafeOutputsConfig
		expected bool
	}{
		{
			name:     "nil safe outputs",
			so:       nil,
			expected: false,
		},
		{
			name:     "no content redaction configured",
			so:       &SafeOutputsConfig{},
			expected: false,
		},
		{
			name: "content redaction with agent",
			so: &SafeOutputsConfig{
				ContentRedaction: &ContentRedactionConfig{
					Agent: []string{"policy text"},
				},
			},
			expected: true,
		},
		{
			name: "content redaction without agent (not enabled)",
			so: &SafeOutputsConfig{
				ContentRedaction: &ContentRedactionConfig{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsContentRedactionEnabled(tt.so)
			if got != tt.expected {
				t.Errorf("IsContentRedactionEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestContentRedactionJobOutputs verifies that the content_redaction job exposes
// the correct outputs (redaction_success, redaction_conclusion).
func TestContentRedactionJobOutputs(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
			AddComments:     &AddCommentsConfig{},
			ContentRedaction: &ContentRedactionConfig{
				Agent: []string{"Do not disclose security vulnerabilities"},
			},
		},
	}

	job, err := compiler.buildContentRedactionJob(data, true)
	if err != nil {
		t.Fatalf("buildContentRedactionJob() error = %v", err)
	}
	if job == nil {
		t.Fatal("expected non-nil content_redaction job, got nil")
	}

	if _, ok := job.Outputs["redaction_success"]; !ok {
		t.Error("content_redaction job missing redaction_success output")
	}
	if _, ok := job.Outputs["redaction_conclusion"]; !ok {
		t.Error("content_redaction job missing redaction_conclusion output")
	}

	if job.Name != string(constants.ContentRedactionJobName) {
		t.Errorf("job name = %q, want %q", job.Name, string(constants.ContentRedactionJobName))
	}
}

// TestContentRedactionJobDependsOnDetection verifies that the content_redaction job
// depends on the detection job when threat detection is enabled.
func TestContentRedactionJobDependsOnDetection(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
			AddComments:     &AddCommentsConfig{},
			ContentRedaction: &ContentRedactionConfig{
				Agent: []string{"Policy"},
			},
		},
	}

	// With detection enabled.
	job, err := compiler.buildContentRedactionJob(data, true)
	if err != nil {
		t.Fatalf("buildContentRedactionJob() error = %v", err)
	}

	found := slices.Contains(job.Needs, string(constants.DetectionJobName))
	if !found {
		t.Errorf("content_redaction job needs %v, expected %q to be present", job.Needs, string(constants.DetectionJobName))
	}

	// Without detection enabled.
	jobNoDetection, err := compiler.buildContentRedactionJob(data, false)
	if err != nil {
		t.Fatalf("buildContentRedactionJob() error = %v", err)
	}

	if slices.Contains(jobNoDetection.Needs, string(constants.DetectionJobName)) {
		t.Errorf("content_redaction job should not depend on detection when detection is disabled, needs: %v", jobNoDetection.Needs)
	}
}

// TestContentRedactionJobCondition verifies that the content_redaction job condition
// always includes always() so it runs even after other job failures.
func TestContentRedactionJobCondition(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
			AddComments:     &AddCommentsConfig{},
			ContentRedaction: &ContentRedactionConfig{
				Agent: []string{"Policy"},
			},
		},
	}

	job, err := compiler.buildContentRedactionJob(data, true)
	if err != nil {
		t.Fatalf("buildContentRedactionJob() error = %v", err)
	}

	if !strings.Contains(job.If, "always()") {
		t.Errorf("content_redaction job condition should include always(), got: %q", job.If)
	}
	if !strings.Contains(job.If, string(constants.AgentJobName)) {
		t.Errorf("content_redaction job condition should reference agent job, got: %q", job.If)
	}
}

// TestContentRedactionLoadPoliciesStep validates the policy loading step for various source types.
func TestContentRedactionLoadPoliciesStep(t *testing.T) {
	tests := []struct {
		name       string
		cr         *ContentRedactionConfig
		wantCurl   bool
		wantFileOp bool
		wantPrintf bool
	}{
		{
			name: "URL policy uses curl",
			cr: &ContentRedactionConfig{
				Agent: []string{"https://corp.example.com/policy.md"},
			},
			wantCurl: true,
		},
		{
			name: "repo-relative path uses file check",
			cr: &ContentRedactionConfig{
				Agent: []string{".github/policies/redaction.md"},
			},
			wantFileOp: true,
		},
		{
			name: "inline policy uses printf",
			cr: &ContentRedactionConfig{
				Agent: []string{"Never disclose internal codenames"},
			},
			wantPrintf: true,
		},
		{
			name: "./relative path uses file check",
			cr: &ContentRedactionConfig{
				Agent: []string{"./policies/redaction.md"},
			},
			wantFileOp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := buildLoadRedactionPoliciesStep(tt.cr)
			combined := strings.Join(steps, "")

			if tt.wantCurl && !strings.Contains(combined, "curl") {
				t.Errorf("expected curl for URL policy; got:\n%s", combined)
			}
			if tt.wantFileOp && !strings.Contains(combined, "if [ -f") {
				t.Errorf("expected file existence check for path policy; got:\n%s", combined)
			}
			if tt.wantPrintf && !strings.Contains(combined, "printf") {
				t.Errorf("expected printf for inline policy; got:\n%s", combined)
			}
		})
	}
}

// TestContentRedactionNilConfigReturnsNilJob verifies that no job is created when
// content redaction is not configured.
func TestContentRedactionNilConfigReturnsNilJob(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		Name: "test-workflow",
		AI:   "copilot",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
			AddComments:     &AddCommentsConfig{},
		},
	}

	job, err := compiler.buildContentRedactionJob(data, true)
	if err != nil {
		t.Fatalf("buildContentRedactionJob() error = %v", err)
	}
	if job != nil {
		t.Errorf("expected nil job when content redaction not configured, got: %+v", job)
	}
}

// TestContentRedactionCompiledWorkflowHasJob verifies that a compiled workflow with
// content-redaction configured includes the content_redaction job in the generated YAML.
func TestContentRedactionCompiledWorkflowHasJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "content-redaction-test-*")
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	frontmatter := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  add-comment:
  content-redaction:
    agent: "Do not disclose security vulnerabilities in public comments"
---

# Test Workflow

Test workflow with content redaction.
`

	if err := os.WriteFile(workflowPath, []byte(frontmatter), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow() error = %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// The content_redaction job must be present.
	contentRedactionSection := extractJobSection(yaml, string(constants.ContentRedactionJobName))
	if contentRedactionSection == "" {
		t.Fatalf("content_redaction job not found in compiled YAML:\n%s", yaml[:min(len(yaml), 3000)])
	}

	// The job must expose redaction_success and redaction_conclusion outputs.
	if !strings.Contains(contentRedactionSection, "redaction_success:") {
		t.Errorf("content_redaction job missing redaction_success output; section:\n%s", contentRedactionSection)
	}
	if !strings.Contains(contentRedactionSection, "redaction_conclusion:") {
		t.Errorf("content_redaction job missing redaction_conclusion output; section:\n%s", contentRedactionSection)
	}
}

// TestContentRedactionSafeOutputsDependsOnJob verifies that the safe_outputs job depends on
// content_redaction when content redaction is configured.
func TestContentRedactionSafeOutputsDependsOnJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "content-redaction-test-*")
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	frontmatter := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  add-comment:
  content-redaction:
    agent: "Policy text"
---

# Test

Test.
`

	if err := os.WriteFile(workflowPath, []byte(frontmatter), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow() error = %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	yamlBytes, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled YAML: %v", err)
	}
	yaml := string(yamlBytes)

	// The safe_outputs job must depend on content_redaction.
	safeOutputsSection := extractJobSection(yaml, "safe_outputs")
	if safeOutputsSection == "" {
		t.Fatal("safe_outputs job not found in compiled YAML")
	}
	if !strings.Contains(safeOutputsSection, string(constants.ContentRedactionJobName)) {
		t.Errorf("safe_outputs job should depend on content_redaction; needs section:\n%s", safeOutputsSection[:min(len(safeOutputsSection), 500)])
	}
}

// TestContentRedactionArrayShorthandCompiles verifies that the array shorthand form compiles.
func TestContentRedactionArrayShorthandCompiles(t *testing.T) {
	tmpDir := testutil.TempDir(t, "content-redaction-test-*")
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	frontmatter := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  add-comment:
  content-redaction:
    - "https://corp.example.com/content-policy.md"
    - "Never mention internal project codenames"
---

# Test

Test.
`

	if err := os.WriteFile(workflowPath, []byte(frontmatter), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow() with array shorthand error = %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("Lock file not created: %v", err)
	}
}
