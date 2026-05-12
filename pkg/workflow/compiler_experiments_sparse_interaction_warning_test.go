//go:build !integration

package workflow

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateToolConfigurationSparseInteractionWarning(t *testing.T) {
	tests := []struct {
		name          string
		workflowData  *WorkflowData
		expectWarning bool
	}{
		{
			name: "warns for multiple experiments with weighted traffic",
			workflowData: &WorkflowData{
				Experiments: map[string][]string{
					"prompt_style": {"concise", "verbose"},
					"emoji":        {"none", "heavy"},
				},
				ExperimentConfigs: map[string]*ExperimentConfig{
					"prompt_style": {
						Variants: []string{"concise", "verbose"},
						Weight:   []int{80, 20},
					},
					"emoji": {
						Variants: []string{"none", "heavy"},
					},
				},
			},
			expectWarning: true,
		},
		{
			name: "does not warn for single experiment even when weighted",
			workflowData: &WorkflowData{
				Experiments: map[string][]string{
					"prompt_style": {"concise", "verbose"},
				},
				ExperimentConfigs: map[string]*ExperimentConfig{
					"prompt_style": {
						Variants: []string{"concise", "verbose"},
						Weight:   []int{70, 30},
					},
				},
			},
			expectWarning: false,
		},
		{
			name: "does not warn for multiple experiments without weighted traffic",
			workflowData: &WorkflowData{
				Experiments: map[string][]string{
					"prompt_style": {"concise", "verbose"},
					"emoji":        {"none", "heavy"},
				},
				ExperimentConfigs: map[string]*ExperimentConfig{
					"prompt_style": {
						Variants: []string{"concise", "verbose"},
					},
					"emoji": {
						Variants: []string{"none", "heavy"},
					},
				},
			},
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.SetStrictMode(false)

			oldStderr := os.Stderr
			r, w, err := os.Pipe()
			require.NoError(t, err, "should create stderr capture pipe")
			os.Stderr = w

			validateErr := compiler.validateToolConfiguration(tt.workflowData, "test.md", &Permissions{})

			require.NoError(t, w.Close(), "should close stderr capture writer")
			os.Stderr = oldStderr

			var buf bytes.Buffer
			_, copyErr := io.Copy(&buf, r)
			require.NoError(t, copyErr, "should copy stderr output")
			output := buf.String()

			require.NoError(t, validateErr, "validation should succeed for this test input")

			expectedMessage := "potential sparse interaction cells detected"
			if tt.expectWarning {
				assert.Contains(t, output, expectedMessage, "should emit sparse interaction warning")
			} else {
				assert.NotContains(t, output, expectedMessage, "should not emit sparse interaction warning")
			}
		})
	}
}

func TestHasWeightedTrafficExperiment(t *testing.T) {
	assert.False(t, hasWeightedTrafficExperiment(nil), "nil configs should not be weighted")
	assert.False(t, hasWeightedTrafficExperiment(map[string]*ExperimentConfig{}), "empty configs should not be weighted")
	assert.False(t, hasWeightedTrafficExperiment(map[string]*ExperimentConfig{
		"a": {Variants: []string{"x", "y"}},
	}), "missing weight should not be weighted")
	assert.False(t, hasWeightedTrafficExperiment(map[string]*ExperimentConfig{
		"a": {Variants: []string{"x", "y"}, Weight: []int{100}},
	}), "mismatched weight length should not be treated as weighted")
	assert.True(t, hasWeightedTrafficExperiment(map[string]*ExperimentConfig{
		"a": {Variants: []string{"x", "y"}, Weight: []int{70, 30}},
	}), "matching weight length should be treated as weighted")
}
