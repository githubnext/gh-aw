//go:build !integration

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
			name: "parse basic agent-session config",
			outputMap: map[string]any{
				"create-agent-session": map[string]any{},
			},
			wantConfig: true,
			wantBase:   "",
			wantRepo:   "",
		},
		{
			name: "parse agent-session config with base branch",
			outputMap: map[string]any{
				"create-agent-session": map[string]any{
					"base": "develop",
				},
			},
			wantConfig: true,
			wantBase:   "develop",
			wantRepo:   "",
		},
		{
			name: "parse agent-session config with target-repo",
			outputMap: map[string]any{
				"create-agent-session": map[string]any{
					"target-repo": "owner/repo",
				},
			},
			wantConfig: true,
			wantBase:   "",
			wantRepo:   "owner/repo",
		},
		{
			name: "parse agent-session config with all fields",
			outputMap: map[string]any{
				"create-agent-session": map[string]any{
					"base":        "main",
					"target-repo": "owner/repo",
					"max":         1,
				},
			},
			wantConfig: true,
			wantBase:   "main",
			wantRepo:   "owner/repo",
		},
		{
			name:       "no agent-session config",
			outputMap:  map[string]any{},
			wantConfig: false,
		},
		{
			name: "reject wildcard target-repo",
			outputMap: map[string]any{
				"create-agent-session": map[string]any{
					"target-repo": "*",
				},
			},
			wantConfig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			config := compiler.parseAgentSessionConfig(tt.outputMap)

			if (config != nil) != tt.wantConfig {
				t.Errorf("parseAgentSessionConfig() returned config = %v, want config existence = %v", config != nil, tt.wantConfig)
				return
			}

			if config != nil {
				if config.Base != tt.wantBase {
					t.Errorf("parseAgentSessionConfig().Base = %v, want %v", config.Base, tt.wantBase)
				}
				if config.TargetRepoSlug != tt.wantRepo {
					t.Errorf("parseAgentSessionConfig().TargetRepoSlug = %v, want %v", config.TargetRepoSlug, tt.wantRepo)
				}
				if templatableIntValue(config.Max) != 1 {
					t.Errorf("parseAgentSessionConfig().Max = %v, want 1", config.Max)
				}
			}
		})
	}
}

func TestExtractSafeOutputsConfigWithAgentTask(t *testing.T) {
	compiler := NewCompiler()
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-agent-session": map[string]any{
				"base": "develop",
			},
		},
	}

	config := compiler.extractSafeOutputsConfig(frontmatter)

	if config == nil {
		t.Fatal("extractSafeOutputsConfig() returned nil")
	}

	if config.CreateAgentSessions == nil {
		t.Fatal("extractSafeOutputsConfig().CreateAgentSessions is nil")
	}

	if config.CreateAgentSessions.Base != "develop" {
		t.Errorf("extractSafeOutputsConfig().CreateAgentSessions.Base = %v, want 'develop'", config.CreateAgentSessions.Base)
	}
}

func TestHasSafeOutputsEnabledWithAgentTask(t *testing.T) {
	config := &SafeOutputsConfig{
		CreateAgentSessions: &CreateAgentSessionConfig{},
	}

	if !HasSafeOutputsEnabled(config) {
		t.Error("HasSafeOutputsEnabled() = false, want true when CreateAgentSessions is set")
	}

	emptyConfig := &SafeOutputsConfig{}
	if HasSafeOutputsEnabled(emptyConfig) {
		t.Error("HasSafeOutputsEnabled() = true, want false when no safe outputs are configured")
	}
}
