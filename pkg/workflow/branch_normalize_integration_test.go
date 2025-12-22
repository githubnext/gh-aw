package workflow

import (
	"strings"
	"testing"
)

func TestBranchNormalizationInlinedInMainJob(t *testing.T) {
	// Test that normalization logic is inlined in upload_assets.cjs when upload-assets is configured
	compiler := NewCompiler(false, "", "")

	// Create test workflow data with upload-assets configured
	data := &WorkflowData{
		Name:        "Test Workflow",
		On:          "on:\n  push:\n",
		Permissions: "permissions:\n  contents: read\n",
		SafeOutputs: &SafeOutputsConfig{
			UploadAssets: &UploadAssetsConfig{
				BranchName:  "assets/${{ github.workflow }}",
				MaxSizeKB:   10240,
				AllowedExts: []string{".png", ".jpg"},
			},
		},
		AI:              "copilot",
		EngineConfig:    &EngineConfig{ID: "copilot"},
		MarkdownContent: "Test content",
	}

	// Build the main job
	job, err := compiler.buildMainJob(data, false)
	if err != nil {
		t.Fatalf("Failed to build main job: %v", err)
	}

	// Check that the job has steps
	if len(job.Steps) == 0 {
		t.Fatal("Expected job to have steps")
	}

	// Convert steps to string
	stepsContent := strings.Join(job.Steps, "\n")

	// Verify that the separate normalization step is NOT present
	if strings.Contains(stepsContent, "Normalize GH_AW_ASSETS_BRANCH") {
		t.Error("Expected separate normalization step to NOT be present (should be inlined)")
	}
}

func TestBranchNormalizationStepNotAddedWhenNoUploadAssets(t *testing.T) {
	// Test that no normalization-related content appears when upload-assets is not configured
	compiler := NewCompiler(false, "", "")

	// Create test workflow data WITHOUT upload-assets
	data := &WorkflowData{
		Name:        "Test Workflow",
		On:          "on:\n  push:\n",
		Permissions: "permissions:\n  contents: read\n",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
		AI:              "copilot",
		EngineConfig:    &EngineConfig{ID: "copilot"},
		MarkdownContent: "Test content",
	}

	// Build the main job
	job, err := compiler.buildMainJob(data, false)
	if err != nil {
		t.Fatalf("Failed to build main job: %v", err)
	}

	// Check that the job has steps
	if len(job.Steps) == 0 {
		t.Fatal("Expected job to have steps")
	}

	// Convert steps to string
	stepsContent := strings.Join(job.Steps, "\n")

	// Verify that the normalization step is NOT present
	if strings.Contains(stepsContent, "Normalize GH_AW_ASSETS_BRANCH") {
		t.Error("Expected normalization step to NOT be present when upload-assets is not configured")
	}
}

func TestUploadAssetsJobHasInlinedNormalization(t *testing.T) {
	// Test that upload_assets job has inlined normalization logic
	compiler := NewCompiler(false, "", "")

	// Create test workflow data with upload-assets configured
	data := &WorkflowData{
		Name:        "Test Workflow",
		On:          "on:\n  push:\n",
		Permissions: "permissions:\n  contents: read\n",
		SafeOutputs: &SafeOutputsConfig{
			UploadAssets: &UploadAssetsConfig{
				BranchName:  "assets/${{ github.workflow }}",
				MaxSizeKB:   10240,
				AllowedExts: []string{".png", ".jpg"},
			},
		},
		AI:              "copilot",
		EngineConfig:    &EngineConfig{ID: "copilot"},
		MarkdownContent: "Test content",
	}

	// Build the upload_assets job
	job, err := compiler.buildUploadAssetsJob(data, "agent", false)
	if err != nil {
		t.Fatalf("Failed to build upload_assets job: %v", err)
	}

	// Check that the job has steps
	if len(job.Steps) == 0 {
		t.Fatal("Expected job to have steps")
	}

	// Convert steps to string
	stepsContent := strings.Join(job.Steps, "\n")

	// Verify that the separate normalization step is NOT present (it's now inlined in the script)
	if strings.Contains(stepsContent, "- name: Normalize GH_AW_ASSETS_BRANCH") {
		t.Error("Expected separate normalization step to NOT be present (should be inlined in upload_assets.cjs)")
	}

	// Verify that the upload assets step exists (which contains the inlined normalization)
	if !strings.Contains(stepsContent, "Upload Assets to Orphaned Branch") {
		t.Error("Expected upload assets step to be present")
	}

	// Verify the script contains the normalizeBranchName function (inlined)
	if !strings.Contains(stepsContent, "normalizeBranchName") {
		t.Error("Expected upload_assets.cjs to contain inlined normalizeBranchName function")
	}
}
