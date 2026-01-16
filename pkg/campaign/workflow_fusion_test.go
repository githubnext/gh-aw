package campaign

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuseWorkflowForCampaign(t *testing.T) {
	tests := []struct {
		name                   string
		workflowContent        string
		campaignID             string
		expectWorkflowDispatch bool
		expectCampaignMetadata bool
		expectError            bool
	}{
		{
			name: "add workflow_dispatch to workflow without it",
			workflowContent: `---
name: Security Scanner
description: Scan for vulnerabilities
on: issues
---
# Security Scanner
Scan repositories`,
			campaignID:             "security-q1-2025",
			expectWorkflowDispatch: true,
			expectCampaignMetadata: true,
		},
		{
			name: "preserve existing workflow_dispatch",
			workflowContent: `---
name: Dependency Updater
on:
  workflow_dispatch:
  schedule:
    - cron: "0 0 * * *"
---
# Updater`,
			campaignID:             "deps-update",
			expectWorkflowDispatch: true,
			expectCampaignMetadata: true,
		},
		{
			name: "handle string format trigger",
			workflowContent: `---
name: Test Workflow
on: workflow_dispatch
---
# Test`,
			campaignID:             "test-campaign",
			expectWorkflowDispatch: true,
			expectCampaignMetadata: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
			require.NoError(t, os.MkdirAll(workflowsDir, 0755))

			// Create original workflow
			workflowID := "test-workflow"
			originalPath := filepath.Join(workflowsDir, workflowID+".md")
			require.NoError(t, os.WriteFile(originalPath, []byte(tt.workflowContent), 0644))

			// Fuse workflow
			result, err := FuseWorkflowForCampaign(tmpDir, workflowID, tt.campaignID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify result
			assert.Equal(t, workflowID, result.OriginalWorkflowID)
			assert.Equal(t, workflowID+"-worker", result.CampaignWorkflowID)

			// Verify file was created
			assert.FileExists(t, result.OutputPath)

			// Read fused workflow
			fusedContent, err := os.ReadFile(result.OutputPath)
			require.NoError(t, err)

			fusedStr := string(fusedContent)

			// Verify workflow_dispatch exists
			if tt.expectWorkflowDispatch {
				assert.Contains(t, fusedStr, "workflow_dispatch", "Expected workflow_dispatch in fused workflow")
			}

			// Verify campaign metadata
			if tt.expectCampaignMetadata {
				assert.Contains(t, fusedStr, "campaign-worker: true", "Expected campaign-worker metadata")
				assert.Contains(t, fusedStr, "campaign-id: "+tt.campaignID, "Expected campaign-id metadata")
				assert.Contains(t, fusedStr, "source-workflow: "+workflowID, "Expected source-workflow metadata")
			}
		})
	}
}

func TestCheckWorkflowDispatch(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    bool
	}{
		{
			name: "has workflow_dispatch in map format",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_dispatch": nil,
				},
			},
			expected: true,
		},
		{
			name: "has workflow_dispatch in string format",
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
			},
			expected: true,
		},
		{
			name: "no workflow_dispatch",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": nil,
				},
			},
			expected: false,
		},
		{
			name:        "no on field",
			frontmatter: map[string]any{},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkWorkflowDispatch(tt.frontmatter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddWorkflowDispatch(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		verify      func(t *testing.T, result map[string]any)
	}{
		{
			name:        "add to empty frontmatter",
			frontmatter: map[string]any{},
			verify: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "workflow_dispatch", result["on"])
			},
		},
		{
			name: "add to existing map format",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": nil,
				},
			},
			verify: func(t *testing.T, result map[string]any) {
				onMap, ok := result["on"].(map[string]any)
				require.True(t, ok)
				_, hasDispatch := onMap["workflow_dispatch"]
				assert.True(t, hasDispatch)
			},
		},
		{
			name: "add to existing string format",
			frontmatter: map[string]any{
				"on": "issues",
			},
			verify: func(t *testing.T, result map[string]any) {
				onStr, ok := result["on"].(string)
				require.True(t, ok)
				assert.Contains(t, onStr, "workflow_dispatch")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addWorkflowDispatch(tt.frontmatter)
			tt.verify(t, result)
		})
	}
}

func TestFuseMultipleWorkflows(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Create multiple workflows
	workflows := map[string]string{
		"workflow1": `---
name: Workflow 1
on: issues
---
# W1`,
		"workflow2": `---
name: Workflow 2
on: pull_request
---
# W2`,
	}

	for id, content := range workflows {
		path := filepath.Join(workflowsDir, id+".md")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	// Fuse multiple workflows
	workflowIDs := []string{"workflow1", "workflow2"}
	results, err := FuseMultipleWorkflows(tmpDir, workflowIDs, "test-campaign")
	require.NoError(t, err)

	// Verify results
	assert.Len(t, results, 2)

	for _, result := range results {
		assert.True(t, strings.HasSuffix(result.CampaignWorkflowID, "-worker"))
		assert.FileExists(t, result.OutputPath)
	}
}
