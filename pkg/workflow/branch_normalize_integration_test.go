package workflow

import (
	"strings"
	"testing"
)

func TestBranchNormalizationStepAdded(t *testing.T) {
	// Test that the normalization step is added to the agent job when upload-assets is configured
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

	// Verify that the normalization step is present
	if !strings.Contains(stepsContent, "Normalize GITHUB_AW_ASSETS_BRANCH") {
		t.Error("Expected normalization step to be present in agent job")
	}

	// Verify it uses github-script
	if !strings.Contains(stepsContent, "uses: actions/github-script@v8") {
		t.Error("Expected normalization step to use actions/github-script@v8")
	}

	// Verify the script contains normalization logic
	if !strings.Contains(stepsContent, "core.exportVariable") {
		t.Error("Expected normalization script to use core.exportVariable")
	}
}

func TestBranchNormalizationStepNotAddedWhenNoUploadAssets(t *testing.T) {
	// Test that the normalization step is NOT added when upload-assets is not configured
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
	if strings.Contains(stepsContent, "Normalize GITHUB_AW_ASSETS_BRANCH") {
		t.Error("Expected normalization step to NOT be present when upload-assets is not configured")
	}
}

func TestUploadAssetsJobHasNormalizationStep(t *testing.T) {
	// Test that the normalization step is added to the upload_assets job
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
	job, err := compiler.buildUploadAssetsJob(data, "agent")
	if err != nil {
		t.Fatalf("Failed to build upload_assets job: %v", err)
	}

	// Check that the job has steps
	if len(job.Steps) == 0 {
		t.Fatal("Expected job to have steps")
	}

	// Convert steps to string
	stepsContent := strings.Join(job.Steps, "\n")

	// Verify that the normalization step is present
	if !strings.Contains(stepsContent, "Normalize GITHUB_AW_ASSETS_BRANCH") {
		t.Error("Expected normalization step to be present in upload_assets job")
	}

	// Verify it uses github-script
	if !strings.Contains(stepsContent, "uses: actions/github-script@v8") {
		t.Error("Expected normalization step to use actions/github-script@v8")
	}

	// Verify the script contains normalization logic
	if !strings.Contains(stepsContent, "core.exportVariable") {
		t.Error("Expected normalization script to use core.exportVariable")
	}
}
