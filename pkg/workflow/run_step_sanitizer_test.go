//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeRunStepExpressions(t *testing.T) {
	tests := []struct {
		name            string
		step            map[string]any
		expectChanged   bool
		expectRunHas    []string
		expectRunNotHas []string
		expectEnvKeys   []string
		expectEnvVals   map[string]string
		expectWarnings  int
	}{
		{
			name: "no expressions - not changed",
			step: map[string]any{
				"name": "Safe step",
				"run":  "echo hello",
			},
			expectChanged: false,
		},
		{
			name: "single expression extracted",
			step: map[string]any{
				"name": "Print title",
				"run":  `echo "${{ github.event.issue.title }}"`,
			},
			expectChanged:   true,
			expectRunHas:    []string{"$GH_AW_GITHUB_EVENT_ISSUE_TITLE"},
			expectRunNotHas: []string{"${{"},
			expectEnvKeys:   []string{"GH_AW_GITHUB_EVENT_ISSUE_TITLE"},
			expectEnvVals: map[string]string{
				"GH_AW_GITHUB_EVENT_ISSUE_TITLE": "${{ github.event.issue.title }}",
			},
			expectWarnings: 1,
		},
		{
			name: "multiple distinct expressions extracted",
			step: map[string]any{
				"name": "Multi expr",
				"run":  "echo \"${{ github.event.issue.title }}\" && echo \"${{ github.event.issue.number }}\"",
			},
			expectChanged:   true,
			expectRunHas:    []string{"$GH_AW_GITHUB_EVENT_ISSUE_TITLE", "$GH_AW_GITHUB_EVENT_ISSUE_NUMBER"},
			expectRunNotHas: []string{"${{"},
			expectEnvKeys:   []string{"GH_AW_GITHUB_EVENT_ISSUE_TITLE", "GH_AW_GITHUB_EVENT_ISSUE_NUMBER"},
			expectWarnings:  2,
		},
		{
			name: "duplicate expression appears only once in env",
			step: map[string]any{
				"run": `echo "${{ github.event.issue.title }}" && echo "${{ github.event.issue.title }}"`,
			},
			expectChanged:   true,
			expectRunNotHas: []string{"${{"},
			expectEnvKeys:   []string{"GH_AW_GITHUB_EVENT_ISSUE_TITLE"},
			expectWarnings:  1,
		},
		{
			name: "existing env keys preserved",
			step: map[string]any{
				"name": "With env",
				"run":  `echo "${{ github.event.issue.title }}"`,
				"env": map[string]any{
					"EXISTING": "value",
				},
			},
			expectChanged:   true,
			expectRunNotHas: []string{"${{"},
			expectEnvKeys:   []string{"EXISTING", "GH_AW_GITHUB_EVENT_ISSUE_TITLE"},
			expectEnvVals: map[string]string{
				"EXISTING":                       "value",
				"GH_AW_GITHUB_EVENT_ISSUE_TITLE": "${{ github.event.issue.title }}",
			},
			expectWarnings: 1,
		},
		{
			name: "expression in heredoc not extracted from scan but still replaced in output",
			step: map[string]any{
				"name": "Heredoc step",
				"run": `cat > /tmp/out.txt << 'EOF'
${{ github.event.issue.title }}
EOF`,
			},
			// The run: script only has ${{ }} inside a heredoc, so scanContent
			// won't detect it and nothing will be extracted.
			expectChanged: false,
		},
		{
			name: "steps outputs expression extracted",
			step: map[string]any{
				"run": `bash script.sh "${{ steps.build.outputs.artifact }}"`,
			},
			expectChanged:   true,
			expectRunNotHas: []string{"${{"},
			expectEnvKeys:   []string{"GH_AW_STEPS_BUILD_OUTPUTS_ARTIFACT"},
			expectWarnings:  1,
		},
		{
			name: "inputs expression extracted",
			step: map[string]any{
				"run": `echo "${{ inputs.my_param }}"`,
			},
			expectChanged:   true,
			expectRunNotHas: []string{"${{"},
			expectEnvKeys:   []string{"GH_AW_INPUTS_MY_PARAM"},
			expectWarnings:  1,
		},
		{
			name: "safe context expressions (non-user-controlled) are still extracted",
			step: map[string]any{
				"run": `echo "${{ github.sha }}"`,
			},
			// github.sha is safe but we extract all expressions from run: for
			// consistency and defence-in-depth.
			expectChanged:   true,
			expectRunNotHas: []string{"${{"},
			expectEnvKeys:   []string{"GH_AW_GITHUB_SHA"},
			expectWarnings:  1,
		},
		{
			name: "no run field - not changed",
			step: map[string]any{
				"uses": "actions/checkout@v4",
			},
			expectChanged: false,
		},
		{
			name: "non-string run field - not changed",
			step: map[string]any{
				"run": 42,
			},
			expectChanged: false,
		},
		{
			name: "warning includes step name",
			step: map[string]any{
				"name": "My Step",
				"run":  `echo "${{ github.event.issue.title }}"`,
			},
			expectChanged:  true,
			expectWarnings: 1,
		},
		{
			name: "warning omits name when step has no name",
			step: map[string]any{
				"run": `echo "${{ github.event.issue.title }}"`,
			},
			expectChanged:  true,
			expectWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, warnings, changed := sanitizeRunStepExpressions(tt.step)

			assert.Equal(t, tt.expectChanged, changed, "changed flag mismatch")

			if !tt.expectChanged {
				// When unchanged the original map should be returned as-is.
				assert.Equal(t, tt.step, result, "unchanged step should equal input")
				assert.Empty(t, warnings, "no warnings expected for unchanged step")
				return
			}

			// Verify run: field changes.
			runVal, ok := result["run"].(string)
			require.True(t, ok, "run field should be a string")

			for _, want := range tt.expectRunHas {
				assert.Contains(t, runVal, want, "run field should contain %q", want)
			}
			for _, notWant := range tt.expectRunNotHas {
				assert.NotContains(t, runVal, notWant, "run field should not contain %q", notWant)
			}

			// Verify env: block.
			if len(tt.expectEnvKeys) > 0 || len(tt.expectEnvVals) > 0 {
				envMap, ok := result["env"].(map[string]any)
				require.True(t, ok, "env field should be a map")

				for _, key := range tt.expectEnvKeys {
					assert.Contains(t, envMap, key, "env should contain key %q", key)
				}
				for key, val := range tt.expectEnvVals {
					assert.Equal(t, val, envMap[key], "env[%q] value mismatch", key)
				}
			}

			// Verify warning count.
			assert.Len(t, warnings, tt.expectWarnings, "warning count mismatch")

			// Verify that warnings contain the injection-related text.
			for _, w := range warnings {
				assert.Contains(t, w, "shell injection", "warning should mention shell injection")
			}

			// Verify that the step name appears in warnings when present.
			if name, hasName := tt.step["name"].(string); hasName && len(warnings) > 0 {
				assert.Contains(t, warnings[0], name, "warning should mention step name")
			}
		})
	}
}

// TestSanitizeRunStepExpressionsOriginalNotMutated verifies that sanitizeRunStepExpressions
// does not modify the input step map.
func TestSanitizeRunStepExpressionsOriginalNotMutated(t *testing.T) {
	original := map[string]any{
		"name": "My step",
		"run":  `echo "${{ github.event.issue.title }}"`,
	}
	originalRun := original["run"].(string)

	_, _, changed := sanitizeRunStepExpressions(original)
	require.True(t, changed, "expected change")

	assert.Equal(t, originalRun, original["run"], "input run field must not be mutated")
	_, hasEnv := original["env"]
	assert.False(t, hasEnv, "input map must not gain an env field")
}

// TestSanitizeRunStepExpressionsEnvVarNameGeneration checks env var name generation for
// known expression patterns.
func TestSanitizeRunStepExpressionsEnvVarNameGeneration(t *testing.T) {
	cases := []struct {
		expression string
		wantVarRef string // expected $VAR in the run: script
		wantEnvKey string
	}{
		{"${{ github.event.issue.title }}", "$GH_AW_GITHUB_EVENT_ISSUE_TITLE", "GH_AW_GITHUB_EVENT_ISSUE_TITLE"},
		{"${{ github.event.pull_request.number }}", "$GH_AW_GITHUB_EVENT_PULL_REQUEST_NUMBER", "GH_AW_GITHUB_EVENT_PULL_REQUEST_NUMBER"},
		{"${{ inputs.my_value }}", "$GH_AW_INPUTS_MY_VALUE", "GH_AW_INPUTS_MY_VALUE"},
		{"${{ steps.setup.outputs.path }}", "$GH_AW_STEPS_SETUP_OUTPUTS_PATH", "GH_AW_STEPS_SETUP_OUTPUTS_PATH"},
	}

	for _, c := range cases {
		t.Run(c.expression, func(t *testing.T) {
			step := map[string]any{"run": `echo "` + c.expression + `"`}
			result, _, changed := sanitizeRunStepExpressions(step)

			require.True(t, changed, "should have changed")
			assert.Contains(t, result["run"].(string), c.wantVarRef, "run field should reference env var")

			envMap, ok := result["env"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, c.expression, envMap[c.wantEnvKey], "env var value should be original expression")
		})
	}
}

// TestSanitizeRunStepExpressions_MultilineRun verifies multiline run scripts are handled.
func TestSanitizeRunStepExpressions_MultilineRun(t *testing.T) {
	step := map[string]any{
		"name": "Multi-line",
		"run": strings.Join([]string{
			"echo \"Title: ${{ github.event.issue.title }}\"",
			"echo \"Body: ${{ github.event.issue.body }}\"",
			"echo done",
		}, "\n"),
	}

	result, warnings, changed := sanitizeRunStepExpressions(step)

	require.True(t, changed)
	runVal := result["run"].(string)
	assert.NotContains(t, runVal, "${{", "run field should have no inline expressions")
	assert.Contains(t, runVal, "$GH_AW_GITHUB_EVENT_ISSUE_TITLE")
	assert.Contains(t, runVal, "$GH_AW_GITHUB_EVENT_ISSUE_BODY")
	assert.Len(t, warnings, 2, "one warning per unique expression")

	envMap := result["env"].(map[string]any)
	assert.Equal(t, "${{ github.event.issue.title }}", envMap["GH_AW_GITHUB_EVENT_ISSUE_TITLE"])
	assert.Equal(t, "${{ github.event.issue.body }}", envMap["GH_AW_GITHUB_EVENT_ISSUE_BODY"])
}
