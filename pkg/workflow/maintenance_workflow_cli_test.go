//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateInstallCLISteps(t *testing.T) {
	t.Run("dev mode generates Setup Go and Build gh-aw steps", func(t *testing.T) {
		result := generateInstallCLISteps(context.Background(), ActionModeDev, "v1.0.0", "", nil)
		if !strings.Contains(result, "Setup Go") {
			t.Errorf("Dev mode should include Setup Go step, got:\n%s", result)
		}
		if !strings.Contains(result, "make build") {
			t.Errorf("Dev mode should include make build step, got:\n%s", result)
		}
		if strings.Contains(result, "setup-cli") {
			t.Errorf("Dev mode should NOT use setup-cli action, got:\n%s", result)
		}
	})

	t.Run("release mode generates setup-cli action step", func(t *testing.T) {
		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "", nil)
		if !strings.Contains(result, "github/gh-aw-actions/setup-cli@v1.0.0") {
			t.Errorf("Release mode should use setup-cli action with version, got:\n%s", result)
		}
		if !strings.Contains(result, "version: v1.0.0") {
			t.Errorf("Release mode should pass version to setup-cli, got:\n%s", result)
		}
		if strings.Contains(result, "make build") {
			t.Errorf("Release mode should NOT build from source, got:\n%s", result)
		}
	})

	t.Run("release mode uses actionTag over version", func(t *testing.T) {
		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "v2.0.0", nil)
		if !strings.Contains(result, "setup-cli@v2.0.0") {
			t.Errorf("Release mode should use actionTag v2.0.0, got:\n%s", result)
		}
	})

	t.Run("release mode with resolver uses SHA-pinned setup-cli reference", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw-actions/setup-cli", "v1.0.0", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		resolver := NewActionResolver(cache)

		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "", resolver)
		expectedRef := "github/gh-aw-actions/setup-cli@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa # v1.0.0"
		if !strings.Contains(result, expectedRef) {
			t.Errorf("Release mode with resolver should use SHA-pinned setup-cli reference %q, got:\n%s", expectedRef, result)
		}
		// Must not contain the bare mutable tag
		if strings.Contains(result, "setup-cli@v1.0.0") {
			t.Errorf("Release mode with resolver must not use mutable tag setup-cli@v1.0.0, got:\n%s", result)
		}
	})

	t.Run("action mode with resolver uses SHA-pinned setup-cli reference", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw-actions/setup-cli", "v1.0.0", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		resolver := NewActionResolver(cache)

		result := generateInstallCLISteps(context.Background(), ActionModeAction, "v1.0.0", "", resolver)
		expectedRef := "github/gh-aw-actions/setup-cli@bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb # v1.0.0"
		if !strings.Contains(result, expectedRef) {
			t.Errorf("Action mode with resolver should use SHA-pinned setup-cli reference %q, got:\n%s", expectedRef, result)
		}
		// Must not contain the bare mutable tag
		if strings.Contains(result, "setup-cli@v1.0.0") {
			t.Errorf("Action mode with resolver must not use mutable tag setup-cli@v1.0.0, got:\n%s", result)
		}
	})

	t.Run("release mode without resolver falls back to tag reference", func(t *testing.T) {
		result := generateInstallCLISteps(context.Background(), ActionModeRelease, "v1.0.0", "", nil)
		if !strings.Contains(result, "github/gh-aw-actions/setup-cli@v1.0.0") {
			t.Errorf("Release mode without resolver should fall back to tag reference, got:\n%s", result)
		}
	})
}

func TestGetCLICmdPrefix(t *testing.T) {
	if getCLICmdPrefix(ActionModeDev) != "./gh-aw" {
		t.Errorf("Dev mode should use ./gh-aw prefix")
	}
	if getCLICmdPrefix(ActionModeRelease) != "gh aw" {
		t.Errorf("Release mode should use 'gh aw' prefix")
	}
}

func TestGenerateMaintenanceWorkflow_RunOperationCLICodegen(t *testing.T) {
	workflowDataList := []*WorkflowData{
		{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					Expires: 48,
				},
			},
		},
	}

	t.Run("dev mode run_operation uses build from source", func(t *testing.T) {
		tmpDir := t.TempDir()
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
		if !strings.Contains(yaml, "make build") {
			t.Errorf("Dev mode run_operation should build from source, got:\n%s", yaml)
		}
		if !strings.Contains(yaml, "GH_AW_CMD_PREFIX: ./gh-aw") {
			t.Errorf("Dev mode run_operation should use ./gh-aw prefix, got:\n%s", yaml)
		}
	})

	t.Run("release mode run_operation uses setup-cli action not gh extension install", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataList,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeRelease,
			ActionTag:        "v1.0.0",
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
		if strings.Contains(yaml, "gh extension install") {
			t.Errorf("Release mode should NOT use gh extension install, got:\n%s", yaml)
		}
		if !strings.Contains(yaml, "github/gh-aw-actions/setup-cli@v1.0.0") {
			t.Errorf("Release mode run_operation should use setup-cli action, got:\n%s", yaml)
		}
		if !strings.Contains(yaml, "GH_AW_CMD_PREFIX: gh aw") {
			t.Errorf("Release mode run_operation should use 'gh aw' prefix, got:\n%s", yaml)
		}
	})

	t.Run("dev mode compile_workflows uses same codegen as run_operation", func(t *testing.T) {
		tmpDir := t.TempDir()
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
		// run_operation, create_labels, activity_report, forecast_report, validate_workflows,
		// and compile_workflows should use the same setup-go version
		// (all use getActionPin, not hardcoded pins). Exactly 6 occurrences expected.
		// Note: label_disable_agentic_workflow no longer installs the CLI, so it has no setup-go step.
		setupGoPin := getActionPin("actions/setup-go")
		occurrences := strings.Count(yaml, setupGoPin)
		if occurrences != 6 {
			t.Errorf("Expected exactly 6 occurrences of pinned setup-go ref %q (run_operation + create_labels + activity_report + forecast_report + validate_workflows + compile_workflows), got %d in:\n%s",
				setupGoPin, occurrences, yaml)
		}
	})
}

func TestGenerateMaintenanceWorkflow_SetupCLISHAPinning(t *testing.T) {
	setupCLISHA := "cccccccccccccccccccccccccccccccccccccccc"

	workflowDataListWithResolver := func(resolver *ActionResolver) []*WorkflowData {
		return []*WorkflowData{
			{
				Name:              "test-workflow",
				ActionResolver:    resolver,
				ActionPinWarnings: make(map[string]bool),
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						Expires: 48,
					},
				},
			},
		}
	}

	t.Run("release mode with resolver SHA-pins setup-cli in run_operation", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache := NewActionCache(tmpDir)
		cache.Set("github/gh-aw-actions/setup-cli", "v1.0.0", setupCLISHA)
		// Also seed the setup action to keep the test hermetic (GenerateMaintenanceWorkflow
		// calls ResolveSetupActionReference with the same resolver, which would otherwise
		// attempt a real gh api call on a cache miss).
		cache.Set("github/gh-aw/actions/setup", "v1.0.0", "dddddddddddddddddddddddddddddddddddddddd")
		resolver := NewActionResolver(cache)

		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: workflowDataListWithResolver(resolver),
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeRelease,
			ActionTag:        "v1.0.0",
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
		expectedRef := "github/gh-aw-actions/setup-cli@" + setupCLISHA + " # v1.0.0"
		if !strings.Contains(yaml, expectedRef) {
			t.Errorf("Expected SHA-pinned setup-cli reference %q in generated workflow, got:\n%s", expectedRef, yaml)
		}
		// Bare tag must not appear
		if strings.Contains(yaml, "setup-cli@v1.0.0") {
			t.Errorf("Generated workflow must not use mutable tag setup-cli@v1.0.0; got:\n%s", yaml)
		}
	})
}

func TestGenerateMaintenanceWorkflow_RepoConfig(t *testing.T) {
	// makeList returns a fresh workflow data list for each sub-test to avoid
	// shared-state issues between parallel or repeated sub-tests.
	makeList := func() []*WorkflowData {
		return []*WorkflowData{
			{
				Name: "test-workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{Expires: 24},
				},
			},
		}
	}

	t.Run("custom string runs_on is used in all jobs", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &RepoConfig{
			Maintenance: &MaintenanceConfig{RunsOn: RunsOnValue{"my-custom-runner"}},
		}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
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
		if !strings.Contains(yaml, "runs-on: my-custom-runner") {
			t.Errorf("Expected 'runs-on: my-custom-runner' in generated workflow, got:\n%s", yaml)
		}
		// Default runner must not appear
		if strings.Contains(yaml, "runs-on: ubuntu-slim") {
			t.Errorf("Generated workflow must not use default runner 'ubuntu-slim' when overridden; got:\n%s", yaml)
		}
	})

	t.Run("array runs_on is used in all jobs", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &RepoConfig{
			Maintenance: &MaintenanceConfig{RunsOn: RunsOnValue{"self-hosted", "linux"}},
		}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
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
		if !strings.Contains(yaml, `runs-on: ["self-hosted","linux"]`) {
			t.Errorf("Expected array runs-on in generated workflow, got:\n%s", yaml)
		}
	})

	t.Run("maintenance disabled deletes existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a pre-existing maintenance file to be deleted
		maintenanceFile := filepath.Join(tmpDir, "agentics-maintenance.yml")
		if err := os.WriteFile(maintenanceFile, []byte("existing content"), 0o600); err != nil {
			t.Fatalf("Failed to write pre-existing file: %v", err)
		}
		cfg := &RepoConfig{MaintenanceDisabled: true}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
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
		if _, statErr := os.Stat(maintenanceFile); !os.IsNotExist(statErr) {
			t.Errorf("Expected maintenance workflow to be deleted when disabled, but file still exists")
		}
	})

	t.Run("maintenance disabled skips generation even with expires", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &RepoConfig{MaintenanceDisabled: true}
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: makeList(),
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
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); !os.IsNotExist(statErr) {
			t.Errorf("Expected no maintenance workflow to be generated when disabled")
		}
	})

	t.Run("maintenance disabled with expires emits warning (no error)", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Workflow with expires configured – maintenance is disabled in aw.json.
		list := []*WorkflowData{
			{
				Name: "my-workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{Expires: 48},
				},
			},
		}
		cfg := &RepoConfig{MaintenanceDisabled: true}
		// The function must succeed (no error), even though a warning is printed.
		err := GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
			WorkflowDataList: list,
			WorkflowDir:      tmpDir,
			Version:          "v1.0.0",
			ActionMode:       ActionModeDev,
			ActionTag:        "",
			RepoConfig:       cfg,
			RepoSlug:         "",
		})
		if err != nil {
			t.Fatalf("Expected no error when maintenance is disabled with expires, got: %v", err)
		}
		// The maintenance workflow must not be generated.
		if _, statErr := os.Stat(filepath.Join(tmpDir, "agentics-maintenance.yml")); !os.IsNotExist(statErr) {
			t.Errorf("Expected no maintenance workflow file when maintenance is disabled")
		}
	})
}
