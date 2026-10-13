//go:build !integration

package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputsNumericTargetCompilation(t *testing.T) {
	outputs := []string{
		"comment-memory",
		"add-comment",
		"push-to-pull-request-branch",
		"create-check-run",
		"create-pull-request-review-comment",
		"submit-pull-request-review",
		"reply-to-pull-request-review-comment",
	}
	targets := []struct {
		name     string
		yaml     string
		expected string
	}{
		{name: "integer", yaml: "123", expected: "123"},
		{name: "quoted integer", yaml: `"123"`, expected: "123"},
		{name: "hash integer", yaml: `"#123"`, expected: "123"},
		{name: "triggering", yaml: "triggering", expected: "triggering"},
		{name: "wildcard", yaml: `"*"`, expected: "*"},
		{name: "fractional", yaml: "123.5"},
		{name: "boolean", yaml: "true"},
	}

	for _, output := range outputs {
		t.Run(output, func(t *testing.T) {
			for _, target := range targets {
				t.Run(target.name, func(t *testing.T) {
					dir := testutil.TempDir(t, "numeric-target-compile")
					section := "safe-outputs"
					extraConfig := ""
					if output == "comment-memory" {
						section = "tools"
						extraConfig = "safe-outputs:\n  add-comment:\n    max: 1\n"
					}
					source := fmt.Sprintf(`---
on: pull_request
engine: copilot
permissions:
  contents: read
strict: false
%s:
  %s:
    target: %s
%s
---
# Target compilation
`, section, output, target.yaml, extraConfig)
					path := filepath.Join(dir, "workflow.md")
					require.NoError(t, os.WriteFile(path, []byte(source), 0644))

					err := NewCompiler().CompileWorkflow(path)
					if target.expected == "" {
						require.Error(t, err, "Invalid target type must not silently default to triggering")
						require.ErrorContains(t, err, "target")
						return
					}
					require.NoError(t, err)
					content, err := os.ReadFile(stringutil.MarkdownToLockFile(path))
					require.NoError(t, err)

					var compiled struct {
						Jobs map[string]struct {
							Steps []struct {
								Env map[string]any `yaml:"env"`
							} `yaml:"steps"`
						} `yaml:"jobs"`
					}
					require.NoError(t, yaml.Unmarshal(content, &compiled))
					job, ok := compiled.Jobs["safe_outputs"]
					require.True(t, ok, "Compiled workflow must contain the safe_outputs job")
					found := false
					for _, step := range job.Steps {
						raw, ok := step.Env["GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG"].(string)
						if !ok {
							continue
						}
						var config map[string]any
						require.NoError(t, json.Unmarshal([]byte(raw), &config))
						handler, ok := config[strings.ReplaceAll(output, "-", "_")].(map[string]any)
						require.True(t, ok, "Compiled workflow must enable the requested handler")
						require.Equal(t, target.expected, handler["target"], "Compiled target must be preserved as a string")
						found = true
					}
					require.True(t, found, "Compiled workflow must include the runtime handler configuration")
				})
			}
		})
	}
}

func TestNormalizeTargetString(t *testing.T) {
	for _, target := range []string{"123", "#123"} {
		require.Equal(t, "123", normalizeTargetString(target))
	}
	for _, target := range []string{"", "triggering", "*", "${{ inputs.pr_number }}"} {
		require.Equal(t, target, normalizeTargetString(target))
	}
	for _, target := range []string{"#0", "#0123", "#123.5", "#triggering", "##123", "#123oops"} {
		require.Equal(t, target, normalizeTargetString(target))
		require.Error(t, validateTargetValue("push-to-pull-request-branch", normalizeTargetString(target)))
	}
}
