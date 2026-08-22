//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllCopilotWorkflowsUseOrgBilling(t *testing.T) {
	orgBillingPermission := "permissions:\n  copilot-requests: write"

	t.Run("empty list returns false", func(t *testing.T) {
		result := allCopilotWorkflowsUseOrgBilling([]*WorkflowData{})
		if result {
			t.Error("Expected false for empty list")
		}
	})

	t.Run("nil entries are skipped", func(t *testing.T) {
		result := allCopilotWorkflowsUseOrgBilling([]*WorkflowData{nil, nil})
		if result {
			t.Error("Expected false when all entries are nil")
		}
	})

	t.Run("single copilot workflow with org billing returns true", func(t *testing.T) {
		data := []*WorkflowData{
			{Name: "wf1", Permissions: orgBillingPermission},
		}
		if !allCopilotWorkflowsUseOrgBilling(data) {
			t.Error("Expected true when single Copilot workflow has copilot-requests: write")
		}
	})

	t.Run("single copilot workflow without org billing returns false", func(t *testing.T) {
		data := []*WorkflowData{
			{Name: "wf1"},
		}
		if allCopilotWorkflowsUseOrgBilling(data) {
			t.Error("Expected false when Copilot workflow lacks copilot-requests: write")
		}
	})

	t.Run("all copilot workflows with org billing returns true", func(t *testing.T) {
		data := []*WorkflowData{
			{Name: "wf1", Permissions: orgBillingPermission},
			{Name: "wf2", Permissions: orgBillingPermission},
		}
		if !allCopilotWorkflowsUseOrgBilling(data) {
			t.Error("Expected true when all Copilot workflows have copilot-requests: write")
		}
	})

	t.Run("mixed copilot workflows returns false", func(t *testing.T) {
		data := []*WorkflowData{
			{Name: "wf1", Permissions: orgBillingPermission},
			{Name: "wf2"},
		}
		if allCopilotWorkflowsUseOrgBilling(data) {
			t.Error("Expected false when only some Copilot workflows have copilot-requests: write")
		}
	})

	t.Run("non-copilot engines are ignored", func(t *testing.T) {
		data := []*WorkflowData{
			{Name: "wf1", Permissions: orgBillingPermission},
			{Name: "wf2", EngineConfig: &EngineConfig{ID: "openai"}},
		}
		if !allCopilotWorkflowsUseOrgBilling(data) {
			t.Error("Expected true when non-Copilot workflows are ignored and all Copilot workflows have org billing")
		}
	})

	t.Run("only non-copilot engines returns false", func(t *testing.T) {
		data := []*WorkflowData{
			{Name: "wf1", EngineConfig: &EngineConfig{ID: "openai"}},
		}
		if allCopilotWorkflowsUseOrgBilling(data) {
			t.Error("Expected false when no Copilot workflows found")
		}
	})
}

func TestGenerateMaintenanceWorkflow_CopilotOrgBilling(t *testing.T) {
	orgBillingPermission := "permissions:\n  copilot-requests: write"
	// SafeOutputs with Expires is required for the maintenance workflow to be generated.
	safeOutputsWithExpires := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{Expires: 48},
	}

	t.Run("org billing mode sets GH_AW_COPILOT_ORG_BILLING in secret-validation step", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{Name: "wf1", Permissions: orgBillingPermission, SafeOutputs: safeOutputsWithExpires},
		}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		if !strings.Contains(string(content), `GH_AW_COPILOT_ORG_BILLING: "true"`) {
			t.Errorf("Expected GH_AW_COPILOT_ORG_BILLING to be set in org billing mode, got:\n%s", string(content))
		}
	})

	t.Run("non-org billing mode does not set GH_AW_COPILOT_ORG_BILLING", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{Name: "wf1", SafeOutputs: safeOutputsWithExpires},
		}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		if strings.Contains(string(content), "GH_AW_COPILOT_ORG_BILLING") {
			t.Errorf("Expected GH_AW_COPILOT_ORG_BILLING to be absent in non-org billing mode, got:\n%s", string(content))
		}
	})

	t.Run("mixed billing mode does not set GH_AW_COPILOT_ORG_BILLING", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowDataList := []*WorkflowData{
			{Name: "wf1", Permissions: orgBillingPermission, SafeOutputs: safeOutputsWithExpires},
			{Name: "wf2", SafeOutputs: safeOutputsWithExpires},
		}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tmpDir, "agentics-maintenance.yml"))
		if err != nil {
			t.Fatalf("Expected maintenance workflow to be generated: %v", err)
		}
		if strings.Contains(string(content), "GH_AW_COPILOT_ORG_BILLING") {
			t.Errorf("Expected GH_AW_COPILOT_ORG_BILLING to be absent in mixed billing mode, got:\n%s", string(content))
		}
	})
}
