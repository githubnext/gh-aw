//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgenticEngineSecretPrefilter(t *testing.T) {
	for _, tc := range []struct {
		content string
		want    bool
	}{
		{"permissions: read", false},
		{"pipeline: build", false},
		{"copy: file", false},
		{"optional: true", false},
		{"pin: v1", false},
		{"run: pi --print 'review'", false},
		{"ANTHROPIC_API_KEY", false},
		{"uses: anthropics/claude-code-action@v1", false},
		{"env:\n  TOKEN: ${{ secrets.ANTHROPIC_API_KEY }}", true},
		{"env:\n  TOKEN: ${{ secrets['OPENAI_API_KEY'] }}", true},
		{"env:\n  TOKEN: ${{ secrets.CUSTOM_TOKEN }}", false},
	} {
		t.Run(tc.content, func(t *testing.T) {
			assert.Equal(t, tc.want, referencesAgenticEngineSecret(tc.content, agenticEngineSecretNames(NewEngineRegistry())))
		})
	}
}

func TestValidateAgenticEngineSteps(t *testing.T) {
	for _, job := range []string{"agent", "detection", "evals", "safe_outputs", "custom", "activation", "conclusion", "detection_custom"} {
		for _, strict := range []bool{true, false} {
			t.Run(job+map[bool]string{true: "/strict", false: "/non-strict"}[strict], func(t *testing.T) {
				c := NewCompiler()
				workflow := map[string]any{"jobs": map[string]any{
					job: map[string]any{"steps": []any{
						map[string]any{"env": map[string]any{"TOKEN": "${{ secrets.ANTHROPIC_API_KEY }}"}},
						map[string]any{"env": map[string]any{"TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}"}},
					}},
				}}
				err := c.validateAgenticEngineSteps(workflow, strict, "test.md")
				exempt := job == "agent" || job == "detection" || job == "evals"
				if exempt {
					require.NoError(t, err)
					assert.Zero(t, c.warningCount)
				} else if strict {
					require.ErrorContains(t, err, "strict mode: agentic engine")
					assert.Contains(t, err.Error(), `job "`+job+`", step 1`)
					assert.Contains(t, err.Error(), "'engine:'")
					assert.Zero(t, c.warningCount)
				} else {
					require.NoError(t, err)
					assert.Equal(t, 2, c.warningCount)
				}
			})
		}
	}
}

func TestAgenticEngineSecretReferences(t *testing.T) {
	secretNames := agenticEngineSecretNames(NewEngineRegistry())
	assert.Contains(t, secretNames, "COPILOT_GITHUB_TOKEN")
	assert.Contains(t, secretNames, "ANTHROPIC_API_KEY")
	assert.Contains(t, secretNames, "OPENAI_API_KEY")
	assert.Contains(t, secretNames, "GEMINI_API_KEY")

	for _, tc := range []struct {
		name string
		job  map[string]any
		want bool
	}{
		{
			name: "step environment",
			job: map[string]any{"steps": []any{
				map[string]any{"env": map[string]any{"TOKEN": "${{ secrets.ANTHROPIC_API_KEY }}"}},
			}},
			want: true,
		},
		{
			name: "job environment",
			job:  map[string]any{"env": map[string]any{"TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}"}},
			want: true,
		},
		{
			name: "combined references",
			job: map[string]any{"steps": []any{
				map[string]any{"env": map[string]any{"TOKEN": "${{ secrets.CUSTOM_TOKEN || secrets.OPENAI_API_KEY }}"}},
			}},
			want: true,
		},
		{
			name: "bracket reference",
			job: map[string]any{"steps": []any{
				map[string]any{"env": map[string]any{"TOKEN": `${{ secrets['GEMINI_API_KEY'] }}`}},
			}},
			want: true,
		},
		{
			name: "unrelated secret",
			job: map[string]any{"steps": []any{
				map[string]any{"env": map[string]any{"TOKEN": "${{ secrets.CUSTOM_TOKEN }}"}},
			}},
			want: false,
		},
		{
			name: "text outside expression",
			job: map[string]any{"steps": []any{
				map[string]any{"run": "echo secrets.ANTHROPIC_API_KEY"},
			}},
			want: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, jobReferencesAgenticEngineSecret(tc.job, secretNames) ||
				referencesAgenticEngineSecret(tc.job["steps"], secretNames))
		})
	}
	catalogFetch := map[string]any{
		"env": map[string]any{"ANTHROPIC_API_KEY": "${{ secrets.ANTHROPIC_API_KEY }}"},
		"run": "curl https://api.anthropic.com/v1/models",
	}
	assert.True(t, referencesAgenticEngineSecret(catalogFetch, secretNames))
}

func TestValidateAgenticEngineSecrets(t *testing.T) {
	for _, tc := range []struct {
		name      string
		job       map[string]any
		strict    bool
		wantError bool
		warnings  int
	}{
		{
			name: "strict job-level credential",
			job: map[string]any{
				"env":   map[string]any{"TOKEN": "${{ secrets.ANTHROPIC_API_KEY }}"},
				"steps": []any{map[string]any{"run": "echo ready"}},
			},
			strict:    true,
			wantError: true,
		},
		{
			name: "non-strict step credential",
			job: map[string]any{"steps": []any{
				map[string]any{
					"env": map[string]any{"TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}"},
					"run": "echo ready",
				},
			}},
			warnings: 1,
		},
		{
			name: "non-strict job credential reported once",
			job: map[string]any{
				"env": map[string]any{"TOKEN": "${{ secrets.OPENAI_API_KEY }}"},
				"steps": []any{
					map[string]any{"env": map[string]any{"TOKEN": "${{ secrets.OPENAI_API_KEY }}"}},
					map[string]any{"run": "copilot -p test"},
				},
			},
			warnings: 1,
		},
		{
			name: "reusable job delegates execution",
			job: map[string]any{
				"uses":    "./.github/workflows/review.yml",
				"secrets": map[string]any{"TOKEN": "${{ secrets.OPENAI_API_KEY }}"},
			},
			strict: true,
		},
		{
			name: "unrelated secret",
			job: map[string]any{"steps": []any{
				map[string]any{
					"env": map[string]any{"TOKEN": "${{ secrets.CUSTOM_TOKEN }}"},
					"run": "echo ready",
				},
			}},
			strict: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCompiler()
			workflow := map[string]any{"jobs": map[string]any{"custom": tc.job}}
			err := c.validateAgenticEngineSteps(workflow, tc.strict, "test.md")
			if tc.wantError {
				require.ErrorContains(t, err, "agentic engine credential")
				require.ErrorContains(t, err, `job "custom"`)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.warnings, c.warningCount)
		})
	}
}

func TestActivationEngineSecretValidationExemption(t *testing.T) {
	c := NewCompiler()
	workflow := map[string]any{"jobs": map[string]any{"activation": map[string]any{"steps": []any{
		map[string]any{
			"env": map[string]any{"TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}"},
			"run": "bash validate_multi_secret.sh",
		},
		map[string]any{
			"env": map[string]any{"TOKEN": "${{ secrets.ANTHROPIC_API_KEY }}"},
			"run": "bash check_oauth_tokens.sh",
		},
	}}}}
	require.NoError(t, c.validateAgenticEngineSteps(workflow, true, "test.md"))
	assert.Zero(t, c.warningCount)
}

func TestAgenticEngineStepsAtCompileTime(t *testing.T) {
	for _, tc := range []struct {
		name      string
		config    string
		strict    string
		cliStrict bool
		wantError bool
	}{
		{"commands without engine secrets", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - run: npm install -g @github/copilot && copilot>out.log\n", "", false, false},
		{"action without engine secrets", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: anthropics/claude-code-action@0123456789abcdef0123456789abcdef01234567\n", "", false, false},
		{"custom job non-strict", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    env:\n      TOKEN: ${{ secrets.ANTHROPIC_API_KEY }}\n    steps:\n      - run: echo ready\n", "strict: false\n", false, false},
		{"CLI overrides non-strict", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    env:\n      TOKEN: ${{ secrets.ANTHROPIC_API_KEY }}\n    steps:\n      - run: echo ready\n", "strict: false\n", true, true},
		{"safe output steps", "safe-outputs:\n  create-issue:\n  steps:\n    - env:\n        TOKEN: ${{ secrets.OPENAI_API_KEY }}\n      run: echo ready\n", "", false, true},
		{"safe output custom job", "safe-outputs:\n  jobs:\n    review:\n      runs-on: ubuntu-latest\n      steps:\n        - env:\n            TOKEN: ${{ secrets.GEMINI_API_KEY }}\n          run: echo ready\n", "", false, true},
		{"activation augmentation", "jobs:\n  activation:\n    steps:\n      - env:\n          TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}\n        run: echo ready\n", "", false, true},
		{"job-level engine credential", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    env:\n      TOKEN: ${{ secrets.ANTHROPIC_API_KEY }}\n    steps:\n      - run: echo ready\n", "", false, true},
		{"step-level engine credential", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - env:\n          TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}\n        run: echo ready\n", "", false, true},
		{"arithmetic data", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo $((pi + 1))\n", "", false, false},
		{"heredoc data after command", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - run: |\n          echo ready; cat <<EOF\n          copilot -p test\n          EOF\n", "", false, false},
		{"read heredoc data", "jobs:\n  custom:\n    runs-on: ubuntu-latest\n    steps:\n      - run: |\n          read -r -d \"\" prompt <<EOF\n          copilot -p test\n          EOF\n", "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "engine-steps.md")
			content := "---\non: workflow_dispatch\npermissions:\n  contents: read\nnetwork:\n  allowed: [defaults]\n" +
				tc.strict + tc.config + "---\n\nTest engine isolation.\n"
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			c := NewCompiler()
			c.SetSkipValidation(true)
			c.SetStrictMode(tc.cliStrict)
			c.approve = true
			err := c.CompileWorkflow(path)
			if tc.wantError {
				require.ErrorContains(t, err, "strict mode: agentic engine")
				_, statErr := os.Stat(strings.TrimSuffix(path, ".md") + ".lock.yml")
				assert.True(t, os.IsNotExist(statErr), "rejected workflows must not emit a lock file")
			} else {
				require.NoError(t, err)
				if tc.strict == "strict: false\n" {
					assert.Positive(t, c.warningCount)
				} else {
					assert.Zero(t, c.warningCount)
				}
				_, statErr := os.Stat(strings.TrimSuffix(path, ".md") + ".lock.yml")
				require.NoError(t, statErr)
			}
		})
	}
}
