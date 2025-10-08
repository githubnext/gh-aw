package workflow

import (
	"testing"
)

// TestSetupSafeOutputsEnv tests the SetupSafeOutputsEnv helper function
func TestSetupSafeOutputsEnv(t *testing.T) {
	t.Run("nil SafeOutputs does nothing", func(t *testing.T) {
		env := make(map[string]string)
		workflowData := &WorkflowData{
			SafeOutputs: nil,
		}

		SetupSafeOutputsEnv(env, workflowData)

		if len(env) != 0 {
			t.Errorf("Expected empty env map, got %d entries", len(env))
		}
	})

	t.Run("basic SafeOutputs without staging", func(t *testing.T) {
		env := make(map[string]string)
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				Staged: false,
			},
			TrialMode: false,
		}

		SetupSafeOutputsEnv(env, workflowData)

		if env["GITHUB_AW_SAFE_OUTPUTS"] != "${{ env.GITHUB_AW_SAFE_OUTPUTS }}" {
			t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS to be set")
		}

		if _, exists := env["GITHUB_AW_SAFE_OUTPUTS_STAGED"]; exists {
			t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED to not be set")
		}
	})

	t.Run("SafeOutputs with staging enabled", func(t *testing.T) {
		env := make(map[string]string)
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				Staged: true,
			},
			TrialMode: false,
		}

		SetupSafeOutputsEnv(env, workflowData)

		if env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] != "true" {
			t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED to be 'true'")
		}
	})

	t.Run("SafeOutputs with TrialMode", func(t *testing.T) {
		env := make(map[string]string)
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				Staged: false,
			},
			TrialMode:        true,
			TrialTargetRepo:  "owner/repo",
		}

		SetupSafeOutputsEnv(env, workflowData)

		if env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] != "true" {
			t.Errorf("Expected GITHUB_AW_SAFE_OUTPUTS_STAGED to be 'true' in trial mode")
		}

		if env["GITHUB_AW_TARGET_REPO"] != "owner/repo" {
			t.Errorf("Expected GITHUB_AW_TARGET_REPO to be set to 'owner/repo'")
		}
	})

	t.Run("SafeOutputs with UploadAssets", func(t *testing.T) {
		env := make(map[string]string)
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{
					BranchName:   "test-branch",
					MaxSizeKB:    100,
					AllowedExts:  []string{".txt", ".md"},
				},
			},
		}

		SetupSafeOutputsEnv(env, workflowData)

		if env["GITHUB_AW_ASSETS_BRANCH"] != "test-branch" {
			t.Errorf("Expected GITHUB_AW_ASSETS_BRANCH to be 'test-branch'")
		}

		if env["GITHUB_AW_ASSETS_MAX_SIZE_KB"] != "100" {
			t.Errorf("Expected GITHUB_AW_ASSETS_MAX_SIZE_KB to be '100'")
		}

		if env["GITHUB_AW_ASSETS_ALLOWED_EXTS"] != ".txt,.md" {
			t.Errorf("Expected GITHUB_AW_ASSETS_ALLOWED_EXTS to be '.txt,.md', got '%s'", env["GITHUB_AW_ASSETS_ALLOWED_EXTS"])
		}
	})
}

// TestAddCustomEngineEnv tests the AddCustomEngineEnv helper function
func TestAddCustomEngineEnv(t *testing.T) {
	t.Run("nil EngineConfig does nothing", func(t *testing.T) {
		env := make(map[string]string)
		AddCustomEngineEnv(env, nil)

		if len(env) != 0 {
			t.Errorf("Expected empty env map, got %d entries", len(env))
		}
	})

	t.Run("EngineConfig with no Env does nothing", func(t *testing.T) {
		env := make(map[string]string)
		engineConfig := &EngineConfig{
			Env: nil,
		}

		AddCustomEngineEnv(env, engineConfig)

		if len(env) != 0 {
			t.Errorf("Expected empty env map, got %d entries", len(env))
		}
	})

	t.Run("EngineConfig with custom env vars", func(t *testing.T) {
		env := make(map[string]string)
		engineConfig := &EngineConfig{
			Env: map[string]string{
				"CUSTOM_VAR_1": "value1",
				"CUSTOM_VAR_2": "value2",
			},
		}

		AddCustomEngineEnv(env, engineConfig)

		if env["CUSTOM_VAR_1"] != "value1" {
			t.Errorf("Expected CUSTOM_VAR_1 to be 'value1'")
		}

		if env["CUSTOM_VAR_2"] != "value2" {
			t.Errorf("Expected CUSTOM_VAR_2 to be 'value2'")
		}
	})
}

// TestAddMaxTurnsEnv tests the AddMaxTurnsEnv helper function
func TestAddMaxTurnsEnv(t *testing.T) {
	t.Run("nil EngineConfig does nothing", func(t *testing.T) {
		env := make(map[string]string)
		AddMaxTurnsEnv(env, nil)

		if len(env) != 0 {
			t.Errorf("Expected empty env map, got %d entries", len(env))
		}
	})

	t.Run("EngineConfig without MaxTurns does nothing", func(t *testing.T) {
		env := make(map[string]string)
		engineConfig := &EngineConfig{
			MaxTurns: "",
		}

		AddMaxTurnsEnv(env, engineConfig)

		if len(env) != 0 {
			t.Errorf("Expected empty env map, got %d entries", len(env))
		}
	})

	t.Run("EngineConfig with MaxTurns", func(t *testing.T) {
		env := make(map[string]string)
		engineConfig := &EngineConfig{
			MaxTurns: "10",
		}

		AddMaxTurnsEnv(env, engineConfig)

		if env["GITHUB_AW_MAX_TURNS"] != "10" {
			t.Errorf("Expected GITHUB_AW_MAX_TURNS to be '10'")
		}
	})
}

// TestSetupSafeOutputsEnvAny tests the SetupSafeOutputsEnvAny helper function (for map[string]any)
func TestSetupSafeOutputsEnvAny(t *testing.T) {
	t.Run("basic SafeOutputs with UploadAssets", func(t *testing.T) {
		env := make(map[string]any)
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				UploadAssets: &UploadAssetsConfig{
					BranchName:   "test-branch",
					MaxSizeKB:    100,
					AllowedExts:  []string{".txt", ".md"},
				},
			},
		}

		SetupSafeOutputsEnvAny(env, workflowData)

		if env["GITHUB_AW_ASSETS_BRANCH"] != "test-branch" {
			t.Errorf("Expected GITHUB_AW_ASSETS_BRANCH to be 'test-branch'")
		}

		if env["GITHUB_AW_ASSETS_MAX_SIZE_KB"] != "100" {
			t.Errorf("Expected GITHUB_AW_ASSETS_MAX_SIZE_KB to be '100'")
		}

		if env["GITHUB_AW_ASSETS_ALLOWED_EXTS"] != ".txt,.md" {
			t.Errorf("Expected GITHUB_AW_ASSETS_ALLOWED_EXTS to be '.txt,.md'")
		}
	})
}

// TestAddCustomEngineEnvAny tests the AddCustomEngineEnvAny helper function (for map[string]any)
func TestAddCustomEngineEnvAny(t *testing.T) {
	t.Run("EngineConfig with custom env vars", func(t *testing.T) {
		env := make(map[string]any)
		engineConfig := &EngineConfig{
			Env: map[string]string{
				"CUSTOM_VAR_1": "value1",
				"CUSTOM_VAR_2": "value2",
			},
		}

		AddCustomEngineEnvAny(env, engineConfig)

		if env["CUSTOM_VAR_1"] != "value1" {
			t.Errorf("Expected CUSTOM_VAR_1 to be 'value1'")
		}

		if env["CUSTOM_VAR_2"] != "value2" {
			t.Errorf("Expected CUSTOM_VAR_2 to be 'value2'")
		}
	})
}

// TestAddMaxTurnsEnvAny tests the AddMaxTurnsEnvAny helper function (for map[string]any)
func TestAddMaxTurnsEnvAny(t *testing.T) {
	t.Run("EngineConfig with MaxTurns", func(t *testing.T) {
		env := make(map[string]any)
		engineConfig := &EngineConfig{
			MaxTurns: "10",
		}

		AddMaxTurnsEnvAny(env, engineConfig)

		if env["GITHUB_AW_MAX_TURNS"] != "10" {
			t.Errorf("Expected GITHUB_AW_MAX_TURNS to be '10'")
		}
	})
}

// MockStepConverter implements StepConverter interface for testing
type MockStepConverter struct{}

func (m *MockStepConverter) convertStepToYAML(stepMap map[string]any) (string, error) {
	// Simple mock implementation that just returns a fixed YAML string
	return "      - name: Test Step\n        run: echo test\n", nil
}

// TestProcessCustomEngineSteps tests the ProcessCustomEngineSteps helper function
func TestProcessCustomEngineSteps(t *testing.T) {
	converter := &MockStepConverter{}

	t.Run("nil EngineConfig returns empty steps", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: nil,
		}

		steps := ProcessCustomEngineSteps(workflowData, converter)

		if len(steps) != 0 {
			t.Errorf("Expected 0 steps, got %d", len(steps))
		}
	})

	t.Run("EngineConfig with no Steps returns empty steps", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				Steps: []map[string]any{},
			},
		}

		steps := ProcessCustomEngineSteps(workflowData, converter)

		if len(steps) != 0 {
			t.Errorf("Expected 0 steps, got %d", len(steps))
		}
	})

	t.Run("EngineConfig with custom steps", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				Steps: []map[string]any{
					{"name": "Step 1", "run": "echo step1"},
					{"name": "Step 2", "run": "echo step2"},
				},
			},
		}

		steps := ProcessCustomEngineSteps(workflowData, converter)

		if len(steps) != 2 {
			t.Errorf("Expected 2 steps, got %d", len(steps))
		}

		// Each step should contain the YAML string as a single element
		if len(steps[0]) != 1 {
			t.Errorf("Expected step to have 1 element (YAML string), got %d", len(steps[0]))
		}
	})
}
