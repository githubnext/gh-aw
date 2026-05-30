//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSafeOutputsStepsShellExpansion(t *testing.T) {
	t.Run("nil config passes", func(t *testing.T) {
		err := validateSafeOutputsStepsShellExpansion(nil)
		assert.NoError(t, err, "nil config should pass validation")
	})

	t.Run("empty steps passes", func(t *testing.T) {
		config := &SafeOutputsConfig{}
		err := validateSafeOutputsStepsShellExpansion(config)
		assert.NoError(t, err, "empty steps should pass validation")
	})

	t.Run("step without run field passes", func(t *testing.T) {
		config := &SafeOutputsConfig{
			Steps: []any{
				map[string]any{
					"name": "Setup step",
					"uses": "actions/checkout@v4",
				},
			},
		}
		err := validateSafeOutputsStepsShellExpansion(config)
		assert.NoError(t, err, "step without run field should pass")
	})

	t.Run("simple run script without expansion passes", func(t *testing.T) {
		config := &SafeOutputsConfig{
			Steps: []any{
				map[string]any{
					"name": "Print info",
					"run":  "echo 'hello world'\ncat /tmp/data.json",
				},
			},
		}
		err := validateSafeOutputsStepsShellExpansion(config)
		assert.NoError(t, err, "simple run script should pass")
	})

	t.Run("simple variable reference passes", func(t *testing.T) {
		config := &SafeOutputsConfig{
			Steps: []any{
				map[string]any{
					"name": "Use variable",
					"run":  "echo $MY_ENV_VAR",
				},
			},
		}
		err := validateSafeOutputsStepsShellExpansion(config)
		assert.NoError(t, err, "simple $VAR should pass")
	})

	t.Run("braced variable reference passes", func(t *testing.T) {
		config := &SafeOutputsConfig{
			Steps: []any{
				map[string]any{
					"name": "Use braced variable",
					"run":  "echo ${MY_ENV_VAR}",
				},
			},
		}
		err := validateSafeOutputsStepsShellExpansion(config)
		assert.NoError(t, err, "${VAR} without operators should pass")
	})

	t.Run("GitHub Actions expression passes", func(t *testing.T) {
		config := &SafeOutputsConfig{
			Steps: []any{
				map[string]any{
					"name": "Use expression",
					"run":  "echo ${{ github.repository }}",
					"env": map[string]any{
						"REPO": "${{ github.repository }}",
					},
				},
			},
		}
		err := validateSafeOutputsStepsShellExpansion(config)
		assert.NoError(t, err, "GitHub Actions ${{ }} expression should pass")
	})

	t.Run("non-map step entry is skipped", func(t *testing.T) {
		config := &SafeOutputsConfig{
			Steps: []any{"not-a-map"},
		}
		err := validateSafeOutputsStepsShellExpansion(config)
		assert.NoError(t, err, "non-map step should be skipped without error")
	})
}

func TestValidateSafeOutputsStepsShellExpansion_DangerousPatterns(t *testing.T) {
	tests := []struct {
		name           string
		runScript      string
		wantErrContain string
	}{
		{
			name:           "command substitution $(...)",
			runScript:      `URL=$(cat /tmp/url.txt)`,
			wantErrContain: "command substitution",
		},
		{
			name:           "command substitution with space",
			runScript:      "RESULT=$(gh api /repos)\necho $RESULT",
			wantErrContain: "command substitution",
		},
		{
			name:           "nested command substitution in assignment",
			runScript:      `SENT_DIST_URL="$(gh aw upload-asset chart.png)"`,
			wantErrContain: "command substitution",
		},
		{
			name:           "backtick command substitution",
			runScript:      "RESULT=`cat /tmp/file.txt`",
			wantErrContain: "backtick command substitution",
		},
		{
			name:           "parameter transformation @P",
			runScript:      `echo ${MY_VAR@P}`,
			wantErrContain: "parameter transformation",
		},
		{
			name:           "parameter transformation @U",
			runScript:      `UPPER=${NAME@U}`,
			wantErrContain: "parameter transformation",
		},
		{
			name:           "indirect expansion ${!var}",
			runScript:      `echo ${!VARNAME}`,
			wantErrContain: "indirect expansion",
		},
		{
			name:           "indirect expansion in assignment",
			runScript:      `VALUE="${!KEY}"`,
			wantErrContain: "indirect expansion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SafeOutputsConfig{
				Steps: []any{
					map[string]any{
						"name": "Step with dangerous expansion",
						"run":  tt.runScript,
					},
				},
			}
			err := validateSafeOutputsStepsShellExpansion(config)
			require.Error(t, err, "dangerous pattern should be rejected: %s", tt.name)
			assert.Contains(t, err.Error(), tt.wantErrContain,
				"error message should describe the pattern type")
			assert.Contains(t, err.Error(), "safe-outputs.steps[0]",
				"error message should include the step index")
		})
	}
}

func TestValidateRunScriptForShellExpansion(t *testing.T) {
	t.Run("empty script passes", func(t *testing.T) {
		err := validateRunScriptForShellExpansion(0, "")
		assert.NoError(t, err, "empty script should pass")
	})

	t.Run("arithmetic expansion $(( )) passes", func(t *testing.T) {
		err := validateRunScriptForShellExpansion(0, "echo $((1+1))")
		assert.NoError(t, err, "arithmetic expansion $(( )) should be allowed")
	})

	t.Run("error includes step index", func(t *testing.T) {
		err := validateRunScriptForShellExpansion(3, "$(echo bad)")
		require.Error(t, err, "command substitution should be rejected")
		assert.Contains(t, err.Error(), "safe-outputs.steps[3]",
			"error should include the step index")
	})

	t.Run("error includes offending snippet", func(t *testing.T) {
		err := validateRunScriptForShellExpansion(0, `URL=$(cat /tmp/url.txt)`)
		require.Error(t, err, "should reject command substitution")
		// The snippet includes at least the $( opener
		assert.Contains(t, err.Error(), "$(", "error should include the offending snippet")
	})

	t.Run("error includes remediation guidance", func(t *testing.T) {
		err := validateRunScriptForShellExpansion(0, "$(echo hi)")
		require.Error(t, err, "should reject command substitution")
		assert.Contains(t, err.Error(), "/tmp/gh-aw/agent/",
			"error should include remediation guidance about writing to a file")
	})

	t.Run("long snippet is truncated", func(t *testing.T) {
		longScript := "$(echo " + strings.Repeat("a", 100) + ")"
		err := validateRunScriptForShellExpansion(0, longScript)
		require.Error(t, err, "should reject command substitution")
		// Snippet in error should not exceed 60+3="..." = 63 characters
		assert.NotContains(t, err.Error(), strings.Repeat("a", 61),
			"snippet in error message should be truncated")
	})
}
