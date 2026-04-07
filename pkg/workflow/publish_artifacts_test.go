//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUploadArtifactConfig(t *testing.T) {
	c := &Compiler{}

	tests := []struct {
		name     string
		input    map[string]any
		expected *UploadArtifactConfig
		isNil    bool
	}{
		{
			name:  "no upload-artifact key",
			input: map[string]any{},
			isNil: true,
		},
		{
			name:  "upload-artifact explicitly false",
			input: map[string]any{"upload-artifact": false},
			isNil: true,
		},
		{
			name:  "upload-artifact true uses defaults",
			input: map[string]any{"upload-artifact": true},
			expected: &UploadArtifactConfig{
				MaxUploads:           defaultArtifactMaxUploads,
				DefaultRetentionDays: defaultArtifactRetentionDays,
				MaxRetentionDays:     defaultArtifactMaxRetentionDays,
				MaxSizeBytes:         defaultArtifactMaxSizeBytes,
			},
		},
		{
			name: "upload-artifact with custom values",
			input: map[string]any{
				"upload-artifact": map[string]any{
					"max-uploads":            3,
					"default-retention-days": 14,
					"max-retention-days":     60,
					"max-size-bytes":         52428800,
					"allowed-paths":          []any{"dist/**", "reports/**"},
					"github-token":           "${{ secrets.MY_TOKEN }}",
				},
			},
			expected: &UploadArtifactConfig{
				MaxUploads:           3,
				DefaultRetentionDays: 14,
				MaxRetentionDays:     60,
				MaxSizeBytes:         52428800,
				AllowedPaths:         []string{"dist/**", "reports/**"},
				BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubToken: "${{ secrets.MY_TOKEN }}"},
			},
		},
		{
			name: "upload-artifact with filters",
			input: map[string]any{
				"upload-artifact": map[string]any{
					"filters": map[string]any{
						"include": []any{"reports/**/*.json"},
						"exclude": []any{"**/*.env", "**/*.pem"},
					},
				},
			},
			expected: &UploadArtifactConfig{
				MaxUploads:           defaultArtifactMaxUploads,
				DefaultRetentionDays: defaultArtifactRetentionDays,
				MaxRetentionDays:     defaultArtifactMaxRetentionDays,
				MaxSizeBytes:         defaultArtifactMaxSizeBytes,
				Filters: &ArtifactFiltersConfig{
					Include: []string{"reports/**/*.json"},
					Exclude: []string{"**/*.env", "**/*.pem"},
				},
			},
		},
		{
			name: "upload-artifact with defaults and allow",
			input: map[string]any{
				"upload-artifact": map[string]any{
					"defaults": map[string]any{
						"skip-archive": false,
						"if-no-files":  "ignore",
					},
					"allow": map[string]any{
						"skip-archive": true,
					},
				},
			},
			expected: &UploadArtifactConfig{
				MaxUploads:           defaultArtifactMaxUploads,
				DefaultRetentionDays: defaultArtifactRetentionDays,
				MaxRetentionDays:     defaultArtifactMaxRetentionDays,
				MaxSizeBytes:         defaultArtifactMaxSizeBytes,
				Defaults: &ArtifactDefaultsConfig{
					SkipArchive: false,
					IfNoFiles:   "ignore",
				},
				Allow: &ArtifactAllowConfig{
					SkipArchive: true,
				},
			},
		},
		{
			name: "upload-artifact with max field",
			input: map[string]any{
				"upload-artifact": map[string]any{
					"max": 5,
				},
			},
			expected: &UploadArtifactConfig{
				MaxUploads:           defaultArtifactMaxUploads,
				DefaultRetentionDays: defaultArtifactRetentionDays,
				MaxRetentionDays:     defaultArtifactMaxRetentionDays,
				MaxSizeBytes:         defaultArtifactMaxSizeBytes,
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.parseUploadArtifactConfig(tt.input)

			if tt.isNil {
				assert.Nil(t, result, "expected nil result")
				return
			}

			require.NotNil(t, result, "expected non-nil result")
			assert.Equal(t, tt.expected.MaxUploads, result.MaxUploads, "MaxUploads mismatch")
			assert.Equal(t, tt.expected.DefaultRetentionDays, result.DefaultRetentionDays, "DefaultRetentionDays mismatch")
			assert.Equal(t, tt.expected.MaxRetentionDays, result.MaxRetentionDays, "MaxRetentionDays mismatch")
			assert.Equal(t, tt.expected.MaxSizeBytes, result.MaxSizeBytes, "MaxSizeBytes mismatch")
			assert.Equal(t, tt.expected.AllowedPaths, result.AllowedPaths, "AllowedPaths mismatch")
			assert.Equal(t, tt.expected.GitHubToken, result.GitHubToken, "GitHubToken mismatch")

			if tt.expected.Max == nil {
				assert.Nil(t, result.Max, "Max should be nil")
			} else {
				require.NotNil(t, result.Max, "Max should not be nil")
				assert.Equal(t, *tt.expected.Max, *result.Max, "Max value mismatch")
			}

			if tt.expected.Filters == nil {
				assert.Nil(t, result.Filters, "Filters should be nil")
			} else {
				require.NotNil(t, result.Filters, "Filters should not be nil")
				assert.Equal(t, tt.expected.Filters.Include, result.Filters.Include, "Filters.Include mismatch")
				assert.Equal(t, tt.expected.Filters.Exclude, result.Filters.Exclude, "Filters.Exclude mismatch")
			}

			if tt.expected.Defaults == nil {
				assert.Nil(t, result.Defaults, "Defaults should be nil")
			} else {
				require.NotNil(t, result.Defaults, "Defaults should not be nil")
				assert.Equal(t, tt.expected.Defaults.SkipArchive, result.Defaults.SkipArchive, "Defaults.SkipArchive mismatch")
				assert.Equal(t, tt.expected.Defaults.IfNoFiles, result.Defaults.IfNoFiles, "Defaults.IfNoFiles mismatch")
			}

			if tt.expected.Allow == nil {
				assert.Nil(t, result.Allow, "Allow should be nil")
			} else {
				require.NotNil(t, result.Allow, "Allow should not be nil")
				assert.Equal(t, tt.expected.Allow.SkipArchive, result.Allow.SkipArchive, "Allow.SkipArchive mismatch")
			}
		})
	}
}

func TestHasSafeOutputsEnabledWithUploadArtifact(t *testing.T) {
	t.Run("UploadArtifact is detected as enabled", func(t *testing.T) {
		config := &SafeOutputsConfig{
			UploadArtifact: &UploadArtifactConfig{},
		}
		assert.True(t, HasSafeOutputsEnabled(config), "UploadArtifact should be detected as enabled")
	})

	t.Run("nil SafeOutputsConfig returns false", func(t *testing.T) {
		assert.False(t, HasSafeOutputsEnabled(nil), "nil config should return false")
	})

	t.Run("empty SafeOutputsConfig returns false", func(t *testing.T) {
		assert.False(t, HasSafeOutputsEnabled(&SafeOutputsConfig{}), "empty config should return false")
	})
}

func TestComputeEnabledToolNamesIncludesUploadArtifact(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			UploadArtifact: &UploadArtifactConfig{},
		},
	}
	tools := computeEnabledToolNames(data)
	assert.True(t, tools["upload_artifact"], "upload_artifact should be in enabled tools")
}

func TestBuildUploadArtifactJobBasicStructure(t *testing.T) {
	c := NewCompiler()
	data := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			UploadArtifact: &UploadArtifactConfig{
				MaxUploads:           2,
				DefaultRetentionDays: 7,
				MaxRetentionDays:     30,
				MaxSizeBytes:         defaultArtifactMaxSizeBytes,
				AllowedPaths:         []string{"dist/**", "reports/**"},
			},
		},
	}

	job, err := c.buildUploadArtifactJob(data, "agent", false)
	require.NoError(t, err, "buildUploadArtifactJob should not return error")
	require.NotNil(t, job, "job should not be nil")

	assert.Equal(t, "upload_artifact", job.Name, "job name should be upload_artifact")

	// Convert steps to string for inspection.
	var stepsStr strings.Builder
	for _, step := range job.Steps {
		stepsStr.WriteString(step)
	}
	s := stepsStr.String()

	assert.Contains(t, s, "Download agent output artifact", "should have agent output download step")
	assert.Contains(t, s, "Download upload-artifact staging", "should have staging artifact download step")
	assert.Contains(t, s, "GH_AW_ARTIFACT_MAX_UPLOADS", "should have max uploads env var")
	assert.Contains(t, s, "GH_AW_ARTIFACT_DEFAULT_RETENTION_DAYS", "should have default retention env var")
	assert.Contains(t, s, "GH_AW_ARTIFACT_MAX_RETENTION_DAYS", "should have max retention env var")
	assert.Contains(t, s, "GH_AW_ARTIFACT_MAX_SIZE_BYTES", "should have max size bytes env var")
	assert.Contains(t, s, "GH_AW_ARTIFACT_ALLOWED_PATHS", "should have allowed paths env var")

	// Should have upload steps for each slot (MaxUploads = 2).
	assert.Contains(t, s, "Upload artifact slot 0", "should have upload step for slot 0")
	assert.Contains(t, s, "Upload artifact slot 1", "should have upload step for slot 1")
	assert.NotContains(t, s, "Upload artifact slot 2", "should NOT have upload step for slot 2")
}

func TestBuildUploadArtifactJobRequiresConfig(t *testing.T) {
	c := NewCompiler()

	t.Run("nil SafeOutputs returns error", func(t *testing.T) {
		data := &WorkflowData{Name: "Test", SafeOutputs: nil}
		_, err := c.buildUploadArtifactJob(data, "agent", false)
		assert.Error(t, err, "should return error when SafeOutputs is nil")
	})

	t.Run("nil UploadArtifact returns error", func(t *testing.T) {
		data := &WorkflowData{
			Name:        "Test",
			SafeOutputs: &SafeOutputsConfig{UploadArtifact: nil},
		}
		_, err := c.buildUploadArtifactJob(data, "agent", false)
		assert.Error(t, err, "should return error when UploadArtifact is nil")
	})
}

func TestGenerateSafeOutputsArtifactStagingUpload(t *testing.T) {
	t.Run("generates step when UploadArtifact is configured", func(t *testing.T) {
		var b strings.Builder
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				UploadArtifact: &UploadArtifactConfig{},
			},
		}
		generateSafeOutputsArtifactStagingUpload(&b, data)
		result := b.String()
		assert.Contains(t, result, "safe-outputs-upload-artifacts", "should reference staging artifact name")
		assert.Contains(t, result, artifactStagingDirExpr, "should reference staging directory")
		assert.Contains(t, result, "if: always()", "should have always() condition")
	})

	t.Run("generates nothing when UploadArtifact is nil", func(t *testing.T) {
		var b strings.Builder
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{UploadArtifact: nil},
		}
		generateSafeOutputsArtifactStagingUpload(&b, data)
		assert.Empty(t, b.String(), "should generate nothing when UploadArtifact is nil")
	})

	t.Run("generates nothing when SafeOutputs is nil", func(t *testing.T) {
		var b strings.Builder
		data := &WorkflowData{SafeOutputs: nil}
		generateSafeOutputsArtifactStagingUpload(&b, data)
		assert.Empty(t, b.String(), "should generate nothing when SafeOutputs is nil")
	})
}

func TestMarshalStringSliceJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"empty slice", []string{}, "[]"},
		{"single value", []string{"dist/**"}, `["dist/**"]`},
		{"multiple values", []string{"dist/**", "reports/**/*.json"}, `["dist/**","reports/**/*.json"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := marshalStringSliceJSON(tt.input)
			assert.Equal(t, tt.expected, result, "JSON output mismatch")
		})
	}
}
