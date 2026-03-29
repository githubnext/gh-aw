//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateSafeJobNeeds_NoSafeOutputs verifies that validation is a no-op
// when there are no safe-outputs or no custom jobs.
func TestValidateSafeJobNeeds_NoSafeOutputs(t *testing.T) {
	data := &WorkflowData{}
	require.NoError(t, validateSafeJobNeeds(data), "no safe-outputs config: should pass")

	data.SafeOutputs = &SafeOutputsConfig{}
	require.NoError(t, validateSafeJobNeeds(data), "nil jobs map: should pass")

	data.SafeOutputs.Jobs = map[string]*SafeJobConfig{}
	require.NoError(t, validateSafeJobNeeds(data), "zero-length jobs map: should pass")
}

// TestValidateSafeJobNeeds_ValidTargets verifies that valid targets are accepted.
func TestValidateSafeJobNeeds_ValidTargets(t *testing.T) {
	tests := []struct {
		name        string
		data        *WorkflowData
		wantErr     bool
		errContains string
	}{
		{
			name: "needs safe_outputs – valid when builtin type is configured",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{}, // creates the consolidated safe_outputs job
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"safe_outputs"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name:        "needs safe_outputs – invalid when only custom jobs are configured",
			wantErr:     true,
			errContains: "unknown needs target",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					// No builtin types, no scripts, no actions, no user steps.
					// Only custom jobs → safe_outputs job will NOT be compiled.
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"safe_outputs"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name: "needs agent – always valid",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"agent"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name: "needs detection – valid when threat detection enabled",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					// Non-nil ThreatDetection means detection is enabled
					ThreatDetection: &ThreatDetectionConfig{},
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"detection"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name: "needs upload_assets – valid when upload-asset is configured",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					UploadAssets: &UploadAssetsConfig{},
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"upload_assets"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name: "needs unlock – valid when lock-for-agent is enabled",
			data: &WorkflowData{
				LockForAgent: true,
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"unlock"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name: "needs another custom safe-job – valid",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"first_job": {
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
						"second_job": {
							Needs: []string{"first_job"},
							Steps: []any{map[string]any{"run": "echo there"}},
						},
					},
				},
			},
		},
		{
			name: "needs another custom safe-job with dashes in source – valid, normalized",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"first-job": {
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
						"second-job": {
							Needs: []string{"first-job"}, // dash form; should be accepted and normalized
							Steps: []any{map[string]any{"run": "echo there"}},
						},
					},
				},
			},
		},
		{
			name:        "needs unknown job – should fail",
			wantErr:     true,
			errContains: "unknown needs target",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"nonexistent_job"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name:        "needs conclusion – should fail (not a valid target)",
			wantErr:     true,
			errContains: "unknown needs target",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"conclusion"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name:        "needs activation – should fail (not a valid target)",
			wantErr:     true,
			errContains: "unknown needs target",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"activation"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name:        "needs detection – invalid when threat detection disabled",
			wantErr:     true,
			errContains: "unknown needs target",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					// ThreatDetection nil means explicitly disabled
					ThreatDetection: nil,
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"detection"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name:        "needs upload_assets – invalid when upload-asset not configured",
			wantErr:     true,
			errContains: "unknown needs target",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"upload_assets"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name:        "needs unlock – invalid when lock-for-agent disabled",
			wantErr:     true,
			errContains: "unknown needs target",
			data: &WorkflowData{
				LockForAgent: false,
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"unlock"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
		{
			name:        "self-dependency – should fail",
			wantErr:     true,
			errContains: "cannot depend on itself",
			data: &WorkflowData{
				SafeOutputs: &SafeOutputsConfig{
					Jobs: map[string]*SafeJobConfig{
						"packaging": {
							Needs: []string{"packaging"},
							Steps: []any{map[string]any{"run": "echo hi"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeJobNeeds(tt.data)
			if tt.wantErr {
				require.Error(t, err, "expected validation error")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"error should contain expected substring")
				}
			} else {
				assert.NoError(t, err, "expected no validation error")
			}
		})
	}
}

// TestValidateSafeJobNeeds_NeedsNormalization verifies that dash-form needs values are
// rewritten to the canonical underscore form so the compiled YAML references the correct job ID.
func TestValidateSafeJobNeeds_NeedsNormalization(t *testing.T) {
	jobCfg := &SafeJobConfig{
		Needs: []string{"safe-outputs", "first-job"}, // dash forms
		Steps: []any{map[string]any{"run": "echo hi"}},
	}
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
			Jobs: map[string]*SafeJobConfig{
				"first-job":  {Steps: []any{map[string]any{"run": "echo hi"}}},
				"second-job": jobCfg,
			},
		},
	}

	require.NoError(t, validateSafeJobNeeds(data), "should pass validation")

	// After validation the needs entries must be in underscore form
	assert.Equal(t, []string{"safe_outputs", "first_job"}, jobCfg.Needs,
		"needs entries should be normalized to underscore form")
}

// TestDetectSafeJobCycles tests cycle detection between custom safe-jobs.
func TestDetectSafeJobCycles(t *testing.T) {
	t.Run("no cycles – linear chain", func(t *testing.T) {
		jobs := map[string]*SafeJobConfig{
			"job_a": {Needs: []string{"job_b"}},
			"job_b": {},
		}
		require.NoError(t, detectSafeJobCycles(jobs))
	})

	t.Run("no cycles – diamond", func(t *testing.T) {
		jobs := map[string]*SafeJobConfig{
			"job_a": {Needs: []string{"job_b", "job_c"}},
			"job_b": {Needs: []string{"job_d"}},
			"job_c": {Needs: []string{"job_d"}},
			"job_d": {},
		}
		require.NoError(t, detectSafeJobCycles(jobs))
	})

	t.Run("direct cycle A→B→A", func(t *testing.T) {
		jobs := map[string]*SafeJobConfig{
			"job_a": {Needs: []string{"job_b"}},
			"job_b": {Needs: []string{"job_a"}},
		}
		err := detectSafeJobCycles(jobs)
		require.Error(t, err, "expected cycle error")
		assert.Contains(t, err.Error(), "cycle detected", "error should mention cycle")
	})

	t.Run("three-node cycle", func(t *testing.T) {
		jobs := map[string]*SafeJobConfig{
			"job_a": {Needs: []string{"job_b"}},
			"job_b": {Needs: []string{"job_c"}},
			"job_c": {Needs: []string{"job_a"}},
		}
		err := detectSafeJobCycles(jobs)
		require.Error(t, err, "expected cycle error")
		assert.Contains(t, err.Error(), "cycle detected", "error should mention cycle")
	})

	t.Run("empty jobs – no error", func(t *testing.T) {
		require.NoError(t, detectSafeJobCycles(nil))
		require.NoError(t, detectSafeJobCycles(map[string]*SafeJobConfig{}))
	})
}

// TestComputeValidSafeJobNeeds verifies the set of valid job IDs for different configurations.
func TestComputeValidSafeJobNeeds(t *testing.T) {
	t.Run("base – no safe-outputs", func(t *testing.T) {
		data := &WorkflowData{}
		valid := computeValidSafeJobNeeds(data)
		assert.True(t, valid["agent"], "agent should always be valid")
		assert.False(t, valid["detection"], "detection not valid without safe-outputs")
		assert.False(t, valid["safe_outputs"], "safe_outputs not valid without safe-outputs")
	})

	t.Run("only custom jobs configured – safe_outputs absent", func(t *testing.T) {
		// When no builtin types / scripts / actions are configured, the consolidated
		// safe_outputs job is never emitted, so it must not be a valid target.
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				Jobs: map[string]*SafeJobConfig{
					"my-job": {},
				},
			},
		}
		valid := computeValidSafeJobNeeds(data)
		assert.False(t, valid["safe_outputs"], "safe_outputs should not be valid when only custom jobs present")
	})

	t.Run("builtin type configured – safe_outputs present", func(t *testing.T) {
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
				CreateIssues:    &CreateIssuesConfig{},
			},
		}
		valid := computeValidSafeJobNeeds(data)
		assert.True(t, valid["agent"])
		assert.True(t, valid["safe_outputs"])
		assert.True(t, valid["detection"], "detection enabled when ThreatDetection is non-nil")
		assert.False(t, valid["upload_assets"])
		assert.False(t, valid["unlock"])
	})

	t.Run("with upload-asset configured", func(t *testing.T) {
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{UploadAssets: &UploadAssetsConfig{}},
		}
		valid := computeValidSafeJobNeeds(data)
		assert.True(t, valid["upload_assets"])
	})

	t.Run("with lock-for-agent enabled", func(t *testing.T) {
		data := &WorkflowData{
			LockForAgent: true,
			SafeOutputs:  &SafeOutputsConfig{},
		}
		valid := computeValidSafeJobNeeds(data)
		assert.True(t, valid["unlock"])
	})

	t.Run("custom safe-job names are included", func(t *testing.T) {
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				Jobs: map[string]*SafeJobConfig{
					"my-packager": {},
					"notify_team": {},
				},
			},
		}
		valid := computeValidSafeJobNeeds(data)
		assert.True(t, valid["my_packager"], "dash-to-underscore normalized name should be valid")
		assert.True(t, valid["notify_team"])
	})
}

// TestConsolidatedSafeOutputsJobWillExist verifies the helper correctly predicts
// whether the safe_outputs consolidated job will be emitted.
func TestConsolidatedSafeOutputsJobWillExist(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		assert.False(t, consolidatedSafeOutputsJobWillExist(nil))
	})

	t.Run("only custom jobs – no consolidated job", func(t *testing.T) {
		cfg := &SafeOutputsConfig{
			Jobs: map[string]*SafeJobConfig{"my-job": {}},
		}
		assert.False(t, consolidatedSafeOutputsJobWillExist(cfg))
	})

	t.Run("custom scripts – consolidated job exists", func(t *testing.T) {
		cfg := &SafeOutputsConfig{
			Scripts: map[string]*SafeScriptConfig{"my-script": {}},
		}
		assert.True(t, consolidatedSafeOutputsJobWillExist(cfg))
	})

	t.Run("user-provided steps – consolidated job exists", func(t *testing.T) {
		cfg := &SafeOutputsConfig{
			Steps: []any{map[string]any{"run": "echo hi"}},
		}
		assert.True(t, consolidatedSafeOutputsJobWillExist(cfg))
	})

	t.Run("builtin type (create-issue) – consolidated job exists", func(t *testing.T) {
		cfg := &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		}
		assert.True(t, consolidatedSafeOutputsJobWillExist(cfg))
	})

	t.Run("builtin type + custom jobs – consolidated job exists", func(t *testing.T) {
		cfg := &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
			Jobs:         map[string]*SafeJobConfig{"my-job": {}},
		}
		assert.True(t, consolidatedSafeOutputsJobWillExist(cfg))
	})
}
