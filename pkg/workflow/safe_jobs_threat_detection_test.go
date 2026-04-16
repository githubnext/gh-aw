//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestSafeOutputsJobsEnableThreatDetectionByDefault verifies that when safe-outputs.jobs
// is configured, threat detection is automatically enabled even if not mentioned in frontmatter
func TestSafeOutputsJobsEnableThreatDetectionByDefault(t *testing.T) {
	c := NewCompiler()

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"jobs": map[string]any{
				"my-custom-job": map[string]any{
					"steps": []any{
						map[string]any{
							"run": "echo 'test'",
						},
					},
				},
			},
		},
	}

	safeOutputsConfig := c.extractSafeOutputsConfig(frontmatter)

	if safeOutputsConfig == nil {
		t.Fatal("Expected safe-outputs config to be extracted, got nil")
	}

	// Verify that Jobs are parsed
	if len(safeOutputsConfig.Jobs) != 1 {
		t.Fatalf("Expected 1 job in safe-outputs, got %d", len(safeOutputsConfig.Jobs))
	}

	// Verify that threat detection is enabled by default
	// A non-nil ThreatDetection means it's enabled; nil means disabled
	if safeOutputsConfig.ThreatDetection == nil {
		t.Error("Expected threat detection to be enabled by default when safe-outputs.jobs is configured")
	}
}

// TestSafeOutputsJobsRespectExplicitThreatDetectionFalse verifies that when
// threat-detection is explicitly set to false, it respects that setting
func TestSafeOutputsJobsRespectExplicitThreatDetectionFalse(t *testing.T) {
	c := NewCompiler()

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"threat-detection": false,
			"jobs": map[string]any{
				"my-custom-job": map[string]any{
					"steps": []any{
						map[string]any{
							"run": "echo 'test'",
						},
					},
				},
			},
		},
	}

	safeOutputsConfig := c.extractSafeOutputsConfig(frontmatter)

	if safeOutputsConfig == nil {
		t.Fatal("Expected safe-outputs config to be extracted, got nil")
	}

	// Verify that threat detection respects explicit false
	// When explicitly disabled, ThreatDetection should be nil
	if safeOutputsConfig.ThreatDetection != nil {
		t.Error("Expected threat detection to be disabled (nil) when explicitly set to false")
	}
}

// TestSafeOutputsJobsRespectExplicitThreatDetectionTrue verifies that when
// threat-detection is explicitly set to true, it respects that setting
func TestSafeOutputsJobsRespectExplicitThreatDetectionTrue(t *testing.T) {
	c := NewCompiler()

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"threat-detection": true,
			"jobs": map[string]any{
				"my-custom-job": map[string]any{
					"steps": []any{
						map[string]any{
							"run": "echo 'test'",
						},
					},
				},
			},
		},
	}

	safeOutputsConfig := c.extractSafeOutputsConfig(frontmatter)

	if safeOutputsConfig == nil {
		t.Fatal("Expected safe-outputs config to be extracted, got nil")
	}

	// Verify that threat detection respects explicit true
	// When explicitly enabled, ThreatDetection should be non-nil
	if safeOutputsConfig.ThreatDetection == nil {
		t.Error("Expected threat detection to be enabled (non-nil) when explicitly set to true")
	}
}

// TestSafeOutputsJobsDependOnDetectionJob verifies that custom safe-output jobs
// depend on both the agent job and the detection job when threat detection is enabled
func TestSafeOutputsJobsDependOnDetectionJob(t *testing.T) {
	c := NewCompiler()

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				// Non-nil ThreatDetection means enabled
			},
			Jobs: map[string]*SafeJobConfig{
				"my-custom-job": {
					Steps: []any{
						map[string]any{
							"run": "echo 'test'",
						},
					},
				},
			},
		},
	}

	// Build safe jobs with threat detection enabled
	_, err := c.buildSafeJobs(workflowData, true)
	if err != nil {
		t.Fatalf("Unexpected error building safe jobs: %v", err)
	}

	jobs := c.jobManager.GetAllJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job to be created, got %d", len(jobs))
	}

	var job *Job
	for _, j := range jobs {
		job = j
		break
	}

	// Detection is a separate job, so safe-jobs depend on both "agent" and "detection"
	hasAgentDep := false
	hasDetectionDep := false
	for _, dep := range job.Needs {
		if dep == "agent" {
			hasAgentDep = true
		}
		if dep == "detection" {
			hasDetectionDep = true
		}
	}

	if !hasAgentDep {
		t.Error("Expected job to depend on 'agent' job")
	}

	if !hasDetectionDep {
		t.Error("Expected job to depend on 'detection' job (detection is now a separate job)")
	}
}

// TestSafeOutputsJobsDoNotDependOnDetectionWhenDisabled verifies that custom safe-output jobs
// do NOT depend on the detection job when threat detection is disabled
func TestSafeOutputsJobsDoNotDependOnDetectionWhenDisabled(t *testing.T) {
	c := NewCompiler()

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: nil, // nil means disabled
			Jobs: map[string]*SafeJobConfig{
				"my-custom-job": {
					Steps: []any{
						map[string]any{
							"run": "echo 'test'",
						},
					},
				},
			},
		},
	}

	// Build safe jobs with threat detection disabled
	_, err := c.buildSafeJobs(workflowData, false)
	if err != nil {
		t.Fatalf("Unexpected error building safe jobs: %v", err)
	}

	jobs := c.jobManager.GetAllJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job to be created, got %d", len(jobs))
	}

	var job *Job
	for _, j := range jobs {
		job = j
		break
	}

	// Verify the job depends on 'agent' but NOT 'detection'
	hasAgentDep := false
	hasDetectionDep := false
	for _, dep := range job.Needs {
		if dep == "agent" {
			hasAgentDep = true
		}
		if dep == "detection" {
			hasDetectionDep = true
		}
	}

	if !hasAgentDep {
		t.Error("Expected job to depend on 'agent' job")
	}

	if hasDetectionDep {
		t.Error("Expected job NOT to depend on 'detection' job when threat detection is disabled")
	}
}

// TestHasSafeOutputsEnabledWithJobs verifies that HasSafeOutputsEnabled returns true
// when only safe-outputs.jobs is configured (no other safe-outputs)
func TestHasSafeOutputsEnabledWithJobs(t *testing.T) {
	config := &SafeOutputsConfig{
		Jobs: map[string]*SafeJobConfig{
			"my-job": {},
		},
	}

	if !HasSafeOutputsEnabled(config) {
		t.Error("Expected HasSafeOutputsEnabled to return true when safe-outputs.jobs is configured")
	}
}

// TestHasSafeOutputsEnabledWithoutJobs verifies that HasSafeOutputsEnabled returns false
// when safe-outputs exists but has no enabled features
func TestHasSafeOutputsEnabledWithoutJobs(t *testing.T) {
	config := &SafeOutputsConfig{
		Jobs: map[string]*SafeJobConfig{},
	}

	if HasSafeOutputsEnabled(config) {
		t.Error("Expected HasSafeOutputsEnabled to return false when safe-outputs.jobs is empty")
	}
}

// TestSafeJobsWithThreatDetectionConfigObject verifies that threat detection
// configuration object is properly handled
func TestSafeJobsWithThreatDetectionConfigObject(t *testing.T) {
	c := NewCompiler()

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"threat-detection": map[string]any{
				"enabled": true,
				"prompt":  "Additional security instructions",
			},
			"jobs": map[string]any{
				"my-custom-job": map[string]any{
					"steps": []any{
						map[string]any{
							"run": "echo 'test'",
						},
					},
				},
			},
		},
	}

	safeOutputsConfig := c.extractSafeOutputsConfig(frontmatter)

	if safeOutputsConfig == nil {
		t.Fatal("Expected safe-outputs config to be extracted, got nil")
	}

	// Verify that threat detection is enabled
	// Non-nil ThreatDetection means enabled
	if safeOutputsConfig.ThreatDetection == nil {
		t.Error("Expected threat detection to be enabled (non-nil)")
	}

	// Verify custom prompt is preserved
	if safeOutputsConfig.ThreatDetection.Prompt != "Additional security instructions" {
		t.Errorf("Expected custom prompt to be preserved, got %q", safeOutputsConfig.ThreatDetection.Prompt)
	}
}

// TestSafeJobsIntegrationWithWorkflowCompilation is an integration test that verifies
// the entire workflow compilation process with safe-output jobs and threat detection
func TestSafeJobsIntegrationWithWorkflowCompilation(t *testing.T) {
	c := NewCompiler()

	markdown := `---
on: issues
safe-outputs:
  jobs:
    my-custom-job:
      steps:
        - run: echo "test"
---

# Test Workflow
Test workflow content
`

	// Create temporary test file
	tmpDir := testutil.TempDir(t, "test-*")
	testFile := tmpDir + "/test-safe-jobs.md"
	if err := os.WriteFile(testFile, []byte(markdown), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compile the workflow
	err := c.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := tmpDir + "/test-safe-jobs.lock.yml"
	workflow, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowStr := string(workflow)

	// Verify detection is a separate job (not inline in agent job)
	if !strings.Contains(workflowStr, "  detection:") {
		t.Error("Expected compiled workflow to contain separate 'detection:' job")
	}

	// Verify detection steps exist in detection job (not agent job)
	detectionSection := extractJobSection(workflowStr, "detection")
	if detectionSection == "" {
		t.Error("Expected compiled workflow to contain 'detection:' job")
	}
	if !strings.Contains(detectionSection, "detection_guard") {
		t.Error("Expected detection job to contain detection_guard step")
	}

	// Verify custom safe job is created
	if !strings.Contains(workflowStr, "my_custom_job:") {
		t.Error("Expected compiled workflow to contain 'my_custom_job:' job")
	}

	// Verify custom job depends on detection job (threat detection enabled by default)
	if !strings.Contains(workflowStr, "- detection") {
		t.Error("Expected custom safe job to depend on detection job")
	}
}

// TestIsThreatDetectionExplicitlyDisabledInConfigs verifies the helper function
// that checks whether any imported safe-outputs config explicitly disables detection.
func TestIsThreatDetectionExplicitlyDisabledInConfigs(t *testing.T) {
	tests := []struct {
		name     string
		configs  []string
		expected bool
	}{
		{
			name:     "empty configs",
			configs:  []string{},
			expected: false,
		},
		{
			name:     "empty JSON objects",
			configs:  []string{"{}", ""},
			expected: false,
		},
		{
			name:     "config without threat-detection key",
			configs:  []string{`{"create-issue": {"max": 1}}`},
			expected: false,
		},
		{
			name:     "config with threat-detection false",
			configs:  []string{`{"create-issue": {"max": 1}, "threat-detection": false}`},
			expected: true,
		},
		{
			name:     "config with threat-detection true",
			configs:  []string{`{"create-issue": {"max": 1}, "threat-detection": true}`},
			expected: false,
		},
		{
			name:     "config with threat-detection as object",
			configs:  []string{`{"create-issue": {"max": 1}, "threat-detection": {"prompt": "check for injection"}}`},
			expected: false,
		},
		{
			name:     "config with threat-detection object and enabled: false",
			configs:  []string{`{"create-issue": {"max": 1}, "threat-detection": {"enabled": false}}`},
			expected: true,
		},
		{
			name:     "config with threat-detection object and enabled: true",
			configs:  []string{`{"create-issue": {"max": 1}, "threat-detection": {"enabled": true}}`},
			expected: false,
		},
		{
			name:     "multiple configs, one has false",
			configs:  []string{`{"create-issue": {"max": 1}}`, `{"create-discussion": {"max": 1}, "threat-detection": false}`},
			expected: true,
		},
		{
			name:     "multiple configs, none disabled",
			configs:  []string{`{"create-issue": {"max": 1}}`, `{"create-discussion": {"max": 1}}`},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isThreatDetectionExplicitlyDisabledInConfigs(tt.configs)
			if result != tt.expected {
				t.Errorf("isThreatDetectionExplicitlyDisabledInConfigs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDefaultThreatDetectionAppliedWhenSafeOutputsFromImportsOnly verifies that when
// safe-outputs configuration comes entirely from imports (no safe-outputs: in main frontmatter),
// threat detection is enabled by default — ensuring the detection gate is wired for
// MCP-driven safe-output writes in native-card-style workflows.
func TestDefaultThreatDetectionAppliedWhenSafeOutputsFromImportsOnly(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Shared workflow provides safe-outputs with no threat-detection key (default should apply)
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    max: 1
    labels: [test]
---

# Shared Safe Outputs
`
	sharedFile := filepath.Join(workflowsDir, "shared-safe-outputs.md")
	if err := os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Main workflow has NO safe-outputs: section — safe-outputs comes entirely from import
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-safe-outputs.md
---

# Native-Card-Style Workflow

This workflow uses safe-outputs via MCP tool calls (no explicit safe-outputs: frontmatter).
`
	mainFile := filepath.Join(workflowsDir, "native-card.md")
	if err := os.WriteFile(mainFile, []byte(mainWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	if workflowData.SafeOutputs == nil {
		t.Fatal("Expected SafeOutputs to be non-nil after importing shared safe-outputs config")
	}

	// Core assertion: threat detection must be enabled by default when safe-outputs
	// comes entirely from imports and no config explicitly disabled it.
	if workflowData.SafeOutputs.ThreatDetection == nil {
		t.Error("Expected ThreatDetection to be enabled (non-nil) by default when safe-outputs comes from imports only")
	}
}

// TestDefaultThreatDetectionNotAppliedWhenImportedConfigExplicitlyDisables verifies that
// when an imported config explicitly sets threat-detection: false, the default is NOT applied.
func TestDefaultThreatDetectionNotAppliedWhenImportedConfigExplicitlyDisables(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Shared workflow explicitly disables threat detection
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    max: 1
  threat-detection: false
---

# Shared Safe Outputs (detection disabled)
`
	sharedFile := filepath.Join(workflowsDir, "shared-no-detection.md")
	if err := os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Main workflow has NO safe-outputs: section
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-no-detection.md
---

# Workflow with detection explicitly disabled in import
`
	mainFile := filepath.Join(workflowsDir, "no-detection.md")
	if err := os.WriteFile(mainFile, []byte(mainWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	if workflowData.SafeOutputs == nil {
		t.Fatal("Expected SafeOutputs to be non-nil after importing shared safe-outputs config")
	}

	// When imported config explicitly disabled detection, the default should NOT be applied.
	if workflowData.SafeOutputs.ThreatDetection != nil {
		t.Error("Expected ThreatDetection to be nil (disabled) when imported config explicitly sets threat-detection: false")
	}
}

// TestImportedSafeOutputsCompiledWithDetectionJob verifies that the compiled workflow
// for a native-card-style workflow (safe-outputs from imports only) contains a detection job
// and that safe_outputs depends on both agent and detection.
func TestImportedSafeOutputsCompiledWithDetectionJob(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := testutil.TempDir(t, "native-card-*")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Shared workflow provides safe-outputs with no threat-detection key
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    max: 1
---

# Shared Safe Outputs
`
	sharedFile := filepath.Join(workflowsDir, "shared.md")
	if err := os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Main workflow has NO safe-outputs: section
	mainWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
imports:
  - ./shared.md
---

# Native-Card-Style Workflow

Test that safe_outputs depends on detection when safe-outputs comes from imports.
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	if err := os.WriteFile(mainFile, []byte(mainWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Compile the workflow
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled lock file
	lockFile := filepath.Join(workflowsDir, "main.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	workflowStr := string(content)

	// Verify that a detection job was generated
	if !strings.Contains(workflowStr, "  detection:") {
		t.Error("Expected compiled workflow to contain 'detection:' job when safe-outputs comes from imports")
	}

	// Verify that safe_outputs depends on both agent and detection
	safeOutputsSection := extractJobSection(workflowStr, "safe_outputs")
	if safeOutputsSection == "" {
		t.Fatal("Expected compiled workflow to contain 'safe_outputs:' job")
	}
	if !strings.Contains(safeOutputsSection, "- detection") {
		t.Error("Expected safe_outputs job to depend on 'detection' job when safe-outputs comes from imports")
	}
	if !strings.Contains(safeOutputsSection, "detection.result == 'success'") {
		t.Error("Expected safe_outputs job to gate on detection.result == 'success'")
	}
}

// TestDefaultThreatDetectionNotAppliedWhenImportedConfigObjectFormDisables verifies that
// when an imported config disables detection via the object form (threat-detection: { enabled: false }),
// the default is NOT applied — mirroring parseThreatDetectionConfig's object-form support.
func TestDefaultThreatDetectionNotAppliedWhenImportedConfigObjectFormDisables(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Shared workflow disables threat detection using the object form
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    max: 1
  threat-detection:
    enabled: false
---

# Shared Safe Outputs (detection disabled via object form)
`
	sharedFile := filepath.Join(workflowsDir, "shared-no-detection-obj.md")
	if err := os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Main workflow has NO safe-outputs: section
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-no-detection-obj.md
---

# Workflow with detection disabled via object form in import
`
	mainFile := filepath.Join(workflowsDir, "no-detection-obj.md")
	if err := os.WriteFile(mainFile, []byte(mainWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	if workflowData.SafeOutputs == nil {
		t.Fatal("Expected SafeOutputs to be non-nil after importing shared safe-outputs config")
	}

	// When imported config uses { enabled: false }, the default should NOT be applied.
	if workflowData.SafeOutputs.ThreatDetection != nil {
		t.Error("Expected ThreatDetection to be nil (disabled) when imported config uses threat-detection: { enabled: false }")
	}
}
