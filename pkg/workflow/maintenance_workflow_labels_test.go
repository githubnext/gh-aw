//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"
)

func TestGenerateMaintenanceWorkflow_LabelTriggers_Disabled(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	falseVal := false
	cfg := &RepoConfig{
		Maintenance: &MaintenanceConfig{LabelTriggers: &falseVal},
	}
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       cfg,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	// Label-event trigger should be absent
	if strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("When label_triggers is false the issues labeled trigger should not be present")
	}

	// The pull_request labeled trigger should never be present (removed)
	if strings.Contains(yaml, "  pull_request:\n    types: [labeled]") {
		t.Error("pull_request labeled trigger should never be present (issues-only)")
	}

	// The label_disable_agentic_workflow job should be absent
	if strings.Contains(yaml, "label_disable_agentic_workflow:") {
		t.Error("When label_triggers is false the label_disable_agentic_workflow job should not be present")
	}

	// The label_apply_safe_outputs job should be absent
	if strings.Contains(yaml, "label_apply_safe_outputs:") {
		t.Error("When label_triggers is false the label_apply_safe_outputs job should not be present")
	}
}

func TestGenerateMaintenanceWorkflow_LabelTriggers_Default(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	// Default: LabelTriggers is nil (omitted) → treated as false (opt-in semantics) → jobs absent
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       nil,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	// Issues labeled trigger should NOT be present by default (opt-in required)
	if strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("By default (no config) the issues labeled trigger should NOT be present — label_triggers must be explicitly enabled")
	}

	// The label_disable_agentic_workflow job should NOT be present by default
	if strings.Contains(yaml, "label_disable_agentic_workflow:") {
		t.Error("By default (no config) the label_disable_agentic_workflow job should NOT be present — label_triggers must be explicitly enabled")
	}

	// The label_apply_safe_outputs job should NOT be present by default
	if strings.Contains(yaml, "label_apply_safe_outputs:") {
		t.Error("By default (no config) the label_apply_safe_outputs job should NOT be present — label_triggers must be explicitly enabled")
	}
}

func TestGenerateMaintenanceWorkflow_LabelTriggers_ExplicitTrue(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	trueVal := true
	cfg := &RepoConfig{
		Maintenance: &MaintenanceConfig{LabelTriggers: &trueVal},
	}
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       cfg,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	// Issues labeled trigger should be present when explicitly enabled
	if !strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("When label_triggers: true the issues labeled trigger should be present")
	}

	// pull_request labeled trigger should never be present (issues-only by design)
	if strings.Contains(yaml, "  pull_request:\n    types: [labeled]") {
		t.Error("pull_request labeled trigger should never be present (issues-only)")
	}

	// The label_disable_agentic_workflow job should be present when explicitly enabled
	if !strings.Contains(yaml, "label_disable_agentic_workflow:") {
		t.Error("When label_triggers: true the label_disable_agentic_workflow job should be present")
	}

	// The label_apply_safe_outputs job should be present when explicitly enabled
	if !strings.Contains(yaml, "label_apply_safe_outputs:") {
		t.Error("When label_triggers: true the label_apply_safe_outputs job should be present")
	}

	// Verify label_apply_safe_outputs job has an explicit step id and if condition so that
	// the operation step only runs when the permission check passes
	applySafeIdx := strings.Index(yaml, "\n  label_apply_safe_outputs:")
	if applySafeIdx != -1 {
		applySection := yaml[applySafeIdx:min(applySafeIdx+2000, len(yaml))]
		if !strings.Contains(applySection, "id: check_permissions") {
			t.Errorf("label_apply_safe_outputs permission check step should have id: check_permissions in:\n%s", applySection[:min(500, len(applySection))])
		}
		if !strings.Contains(applySection, "steps.check_permissions.outcome == 'success'") {
			t.Errorf("label_apply_safe_outputs operation step should have if: steps.check_permissions.outcome == 'success' in:\n%s", applySection[:min(500, len(applySection))])
		}
	}
}

func TestGenerateMaintenanceWorkflow_DisabledJobs(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	trueVal := true
	cfg := &RepoConfig{
		Maintenance: &MaintenanceConfig{
			LabelTriggers: &trueVal,
			DisabledJobs: []string{
				"close-expired-entities",
				"apply_safe_outputs",
				"label_disable_agentic_workflow",
				"label_apply_safe_outputs",
			},
		},
	}
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       cfg,
		RepoSlug:         "",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	if err != nil {
		t.Fatalf("Expected maintenance workflow to be generated: %v", err)
	}
	yaml := string(content)

	if strings.Contains(yaml, "close-expired-discussions:") {
		t.Error("close-expired-discussions job should be omitted when close-expired-entities is disabled in aw.json")
	}
	if strings.Contains(yaml, "close-expired-issues:") {
		t.Error("close-expired-issues job should be omitted when close-expired-entities is disabled in aw.json")
	}
	if strings.Contains(yaml, "close-expired-pull-requests:") {
		t.Error("close-expired-pull-requests job should be omitted when close-expired-entities is disabled in aw.json")
	}
	if strings.Contains(yaml, "apply_safe_outputs:") {
		t.Error("apply_safe_outputs job should be omitted when disabled in aw.json")
	}
	if strings.Contains(yaml, "label_disable_agentic_workflow:") {
		t.Error("label_disable_agentic_workflow job should be omitted when disabled in aw.json")
	}
	if strings.Contains(yaml, "label_apply_safe_outputs:") {
		t.Error("label_apply_safe_outputs job should be omitted when disabled in aw.json")
	}
	if strings.Contains(yaml, "  issues:\n    types: [labeled]") {
		t.Error("issues:labeled trigger should be omitted when all label-triggered jobs are disabled")
	}

	var workflowDoc maintenanceWorkflowDocument
	require.NoError(t, yamlv3.Unmarshal(content, &workflowDoc), "generated maintenance workflow should be valid YAML")
	require.Equal(
		t,
		"${{ inputs.run_url }}",
		workflowDoc.On.WorkflowCall.Outputs.AppliedRunURL.Value,
		"workflow_call applied_run_url should fall back to inputs.run_url",
	)
	require.Contains(t, yaml, "workflow_call falls back to inputs.run_url when apply_safe_outputs is disabled; other triggers leave this empty", "generated output description should document the fallback scope")
}

func TestGenerateMaintenanceWorkflow_DisabledJobs_PartialLabelTrigger(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Expires: 48},
			},
		},
	}

	tmpDir := t.TempDir()
	trueVal := true
	cfg := &RepoConfig{
		Maintenance: &MaintenanceConfig{
			LabelTriggers: &trueVal,
			DisabledJobs:  []string{"label_disable_agentic_workflow"},
		},
	}
	err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
		WorkflowDataList: workflowDataList,
		WorkflowDir:      tmpDir,
		Version:          "v1.0.0",
		ActionMode:       ActionModeDev,
		ActionTag:        "",
		RepoConfig:       cfg,
		RepoSlug:         "",
	})
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
	require.NoError(t, err)
	yaml := string(content)

	require.Contains(t, yaml, "  issues:\n    types: [labeled]", "issues:labeled trigger should remain when any label-triggered job is still enabled")
	require.NotContains(t, yaml, "label_disable_agentic_workflow:", "disabled label job should be omitted")
	require.Contains(t, yaml, "label_apply_safe_outputs:", "remaining label-triggered job should still be emitted")
}
