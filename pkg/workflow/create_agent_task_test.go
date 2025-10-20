package workflow

import (
	"testing"
)

func TestParseAgentTaskConfig(t *testing.T) {
	tests := []struct {
		name       string
		outputMap  map[string]any
		wantConfig bool
		wantBase   string
		wantRepo   string
	}{
		{
			name: "parse basic agent-task config",
			outputMap: map[string]any{
				"create-agent-task": map[string]any{},
			},
			wantConfig: true,
			wantBase:   "",
			wantRepo:   "",
		},
		{
			name: "parse agent-task config with base branch",
			outputMap: map[string]any{
				"create-agent-task": map[string]any{
					"base": "develop",
				},
			},
			wantConfig: true,
			wantBase:   "develop",
			wantRepo:   "",
		},
		{
			name: "parse agent-task config with target-repo",
			outputMap: map[string]any{
				"create-agent-task": map[string]any{
					"target-repo": "owner/repo",
				},
			},
			wantConfig: true,
			wantBase:   "",
			wantRepo:   "owner/repo",
		},
		{
			name: "parse agent-task config with all fields",
			outputMap: map[string]any{
				"create-agent-task": map[string]any{
					"base":        "main",
					"target-repo": "owner/repo",
					"max":         1,
					"min":         0,
				},
			},
			wantConfig: true,
			wantBase:   "main",
			wantRepo:   "owner/repo",
		},
		{
			name:       "no agent-task config",
			outputMap:  map[string]any{},
			wantConfig: false,
		},
		{
			name: "reject wildcard target-repo",
			outputMap: map[string]any{
				"create-agent-task": map[string]any{
					"target-repo": "*",
				},
			},
			wantConfig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			config := compiler.parseAgentTaskConfig(tt.outputMap)

			if (config != nil) != tt.wantConfig {
				t.Errorf("parseAgentTaskConfig() returned config = %v, want config existence = %v", config != nil, tt.wantConfig)
				return
			}

			if config != nil {
				if config.Base != tt.wantBase {
					t.Errorf("parseAgentTaskConfig().Base = %v, want %v", config.Base, tt.wantBase)
				}
				if config.TargetRepoSlug != tt.wantRepo {
					t.Errorf("parseAgentTaskConfig().TargetRepoSlug = %v, want %v", config.TargetRepoSlug, tt.wantRepo)
				}
				if config.Max != 1 {
					t.Errorf("parseAgentTaskConfig().Max = %v, want 1", config.Max)
				}
			}
		})
	}
}

func TestBuildCreateOutputAgentTaskJob(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateAgentTasks: &CreateAgentTaskConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Base:           "main",
				TargetRepoSlug: "owner/repo",
			},
		},
	}

	job, err := compiler.buildCreateOutputAgentTaskJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("buildCreateOutputAgentTaskJob() error = %v", err)
	}

	if job == nil {
		t.Fatal("buildCreateOutputAgentTaskJob() returned nil job")
	}

	if job.Name != "create_agent_task" {
		t.Errorf("buildCreateOutputAgentTaskJob().Name = %v, want 'create_agent_task'", job.Name)
	}

	if job.TimeoutMinutes != 10 {
		t.Errorf("buildCreateOutputAgentTaskJob().TimeoutMinutes = %v, want 10", job.TimeoutMinutes)
	}

	if len(job.Outputs) != 2 {
		t.Errorf("buildCreateOutputAgentTaskJob().Outputs length = %v, want 2", len(job.Outputs))
	}

	if _, ok := job.Outputs["task_number"]; !ok {
		t.Error("buildCreateOutputAgentTaskJob().Outputs missing 'task_number'")
	}

	if _, ok := job.Outputs["task_url"]; !ok {
		t.Error("buildCreateOutputAgentTaskJob().Outputs missing 'task_url'")
	}

	if len(job.Steps) == 0 {
		t.Error("buildCreateOutputAgentTaskJob().Steps is empty")
	}

	if len(job.Needs) != 1 || job.Needs[0] != "main_job" {
		t.Errorf("buildCreateOutputAgentTaskJob().Needs = %v, want ['main_job']", job.Needs)
	}
}

func TestExtractSafeOutputsConfigWithAgentTask(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-agent-task": map[string]any{
				"base": "develop",
			},
		},
	}

	config := compiler.extractSafeOutputsConfig(frontmatter)

	if config == nil {
		t.Fatal("extractSafeOutputsConfig() returned nil")
	}

	if config.CreateAgentTasks == nil {
		t.Fatal("extractSafeOutputsConfig().CreateAgentTasks is nil")
	}

	if config.CreateAgentTasks.Base != "develop" {
		t.Errorf("extractSafeOutputsConfig().CreateAgentTasks.Base = %v, want 'develop'", config.CreateAgentTasks.Base)
	}
}

func TestHasSafeOutputsEnabledWithAgentTask(t *testing.T) {
	config := &SafeOutputsConfig{
		CreateAgentTasks: &CreateAgentTaskConfig{},
	}

	if !HasSafeOutputsEnabled(config) {
		t.Error("HasSafeOutputsEnabled() = false, want true when CreateAgentTasks is set")
	}

	emptyConfig := &SafeOutputsConfig{}
	if HasSafeOutputsEnabled(emptyConfig) {
		t.Error("HasSafeOutputsEnabled() = true, want false when no safe outputs are configured")
	}
}
