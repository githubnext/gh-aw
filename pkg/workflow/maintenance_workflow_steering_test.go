//go:build !integration

package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestMaintenanceWorkflowSteeringGeneration(t *testing.T) {
	for _, tt := range []struct {
		name            string
		safeOutputs     *SafeOutputsConfig
		config          *RepoConfig
		wantMaintenance bool
		wantCleanup     bool
	}{
		{
			name: "steering only", safeOutputs: &SafeOutputsConfig{Steer: true},
			wantMaintenance: true, wantCleanup: true,
		},
		{
			name: "steering job disabled", safeOutputs: &SafeOutputsConfig{Steer: true},
			config: &RepoConfig{Maintenance: &MaintenanceConfig{
				DisabledJobs: []string{"cleanup-stale-steering-issues"},
			}},
			wantMaintenance: true,
		},
		{
			name: "maintenance disabled", safeOutputs: &SafeOutputsConfig{Steer: true},
			config: &RepoConfig{MaintenanceDisabled: true},
		},
		{name: "no steering", safeOutputs: &SafeOutputsConfig{}},
		{name: "staged steering", safeOutputs: &SafeOutputsConfig{Steer: true, Staged: templatableBoolPtr("true")}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			data := &WorkflowData{Name: "test", WorkflowID: "test", SafeOutputs: tt.safeOutputs}
			hasExpires, _, _ := scanWorkflowsForExpires([]*WorkflowData{data}, tt.config)
			require.False(t, hasExpires, "steering must trigger generation without expiring outputs")
			require.NoError(t, GenerateMaintenanceWorkflow(context.Background(), GenerateMaintenanceWorkflowOptions{
				WorkflowDataList: []*WorkflowData{nil, data},
				WorkflowDir:      directory,
				Version:          "v1.0.0",
				ActionMode:       ActionModeRelease,
				RepoConfig:       tt.config,
			}))
			content, err := os.ReadFile(filepath.Join(directory, "agentics-maintenance.yml"))
			if !tt.wantMaintenance {
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			require.Contains(t, string(content), `cron: "37 0 * * *"`)
			require.Contains(t, string(content), "for steering issue cleanup")
			var document struct {
				Jobs map[string]any `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal(content, &document))
			_, hasCleanup := document.Jobs["cleanup-stale-steering-issues"]
			require.Equal(t, tt.wantCleanup, hasCleanup)
		})
	}
}

func TestMaintenanceWorkflowOptionalSteeringCleanup(t *testing.T) {
	opts := buildMaintenanceWorkflowYAMLOptions{
		cronSchedule: "37 0 * * *", scheduleDesc: "Daily", runsOnValue: "ubuntu-slim",
		actionMode: ActionModeRelease, version: "v1.0.0", minExpiresDays: 7,
	}
	for _, tt := range []struct {
		name        string
		hasSteering bool
		disabled    []string
		wantCleanup bool
	}{
		{name: "no steering"},
		{name: "steering enabled", hasSteering: true, wantCleanup: true},
		{name: "steering disabled", hasSteering: true, disabled: []string{"CLEANUP_STALE_STEERING_ISSUES"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts.hasSteering = tt.hasSteering
			opts.maintenanceConfig = &MaintenanceConfig{DisabledJobs: tt.disabled}
			generated, err := buildMaintenanceWorkflowYAML(context.Background(), opts)
			require.NoError(t, err)
			var document struct {
				Jobs map[string]any `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(generated), &document))
			_, hasCleanup := document.Jobs["cleanup-stale-steering-issues"]
			require.Equal(t, tt.wantCleanup, hasCleanup)
			require.Contains(t, document.Jobs, "close-expired-issues")
			require.Contains(t, document.Jobs, "run_operation")
		})
	}
}

func TestMaintenanceSteeringCleanupCheckoutModes(t *testing.T) {
	for _, mode := range []ActionMode{ActionModeRelease, ActionModeAction, ActionModeDev, ActionModeScript} {
		t.Run(string(mode), func(t *testing.T) {
			opts := buildMaintenanceWorkflowYAMLOptions{
				actionMode: mode, runsOnValue: "ubuntu-slim", version: "v1.0.0",
			}
			setupRef := ResolveSetupActionReference(context.Background(), mode, opts.version, "", nil)
			generated := buildMaintenanceCleanupStaleSteeringIssuesJob(opts, setupRef)
			var jobs map[string]struct {
				If          string            `yaml:"if"`
				Permissions map[string]string `yaml:"permissions"`
				Steps       []struct {
					Name string `yaml:"name"`
					Uses string `yaml:"uses"`
					With struct {
						PersistCredentials bool `yaml:"persist-credentials"`
					} `yaml:"with"`
				} `yaml:"steps"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(generated), &jobs))
			job := jobs["cleanup-stale-steering-issues"]
			require.Contains(t, job.If, "github.event_name == 'schedule'")
			require.Contains(t, job.If, "github.event.repository.fork")
			permissions := map[string]string{"actions": "read", "issues": "write"}
			if mode == ActionModeDev || mode == ActionModeScript {
				permissions["contents"] = "read"
				require.Equal(t, "Checkout actions folder", job.Steps[0].Name)
				require.False(t, job.Steps[0].With.PersistCredentials)
				require.Equal(t, setupRef, job.Steps[1].Uses)
			} else {
				require.Len(t, job.Steps, 2)
				require.Equal(t, "Setup Scripts", job.Steps[0].Name)
				require.Equal(t, setupRef, job.Steps[0].Uses)
				require.NotContains(t, generated, "actions/checkout")
			}
			require.Equal(t, permissions, job.Permissions)
			opts.maintenanceConfig = &MaintenanceConfig{DisabledJobs: []string{"cleanup-stale-steering-issues"}}
			require.Empty(t, buildMaintenanceCleanupStaleSteeringIssuesJob(opts, setupRef))
		})
	}
}

func TestLoadRepoConfigDisabledSteeringCleanup(t *testing.T) {
	for _, jobID := range []string{"cleanup-stale-steering-issues", "CLEANUP_STALE_STEERING_ISSUES"} {
		t.Run(jobID, func(t *testing.T) {
			directory := t.TempDir()
			writeAWJSON(t, directory, `{"maintenance":{"disabled_jobs":["`+jobID+`"]}}`)
			config, err := LoadRepoConfig(directory)
			require.NoError(t, err)
			require.True(t, config.Maintenance.IsJobDisabled("cleanup-stale-steering-issues"))
		})
	}
}
