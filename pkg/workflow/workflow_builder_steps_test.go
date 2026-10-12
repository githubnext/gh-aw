//go:build !integration

package workflow

import (
	"fmt"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessAndMergePreStepSections(t *testing.T) {
	for _, section := range []string{"pre-steps", "pre-agent-steps"} {
		t.Run(section, func(t *testing.T) {
			process := func(frontmatter map[string]any, data *WorkflowData, imported string) error {
				compiler := NewCompiler()
				imports := &parser.ImportsResult{
					MergedPreSteps:      "- run: echo wrong import\n",
					MergedPreAgentSteps: "- run: echo wrong import\n",
				}
				if section == "pre-steps" {
					imports.MergedPreSteps = imported
					return compiler.processAndMergePreSteps(frontmatter, data, imports)
				}
				imports.MergedPreAgentSteps = imported
				return compiler.processAndMergePreAgentSteps(frontmatter, data, imports)
			}
			output := func(data *WorkflowData) string {
				if section == "pre-steps" {
					assert.Equal(t, "other section", data.PreAgentSteps)
					return data.PreSteps
				}
				assert.Equal(t, "other section", data.PreSteps)
				return data.PreAgentSteps
			}
			newData := func() *WorkflowData {
				data := &WorkflowData{PreSteps: "other section", PreAgentSteps: "other section"}
				if section == "pre-steps" {
					data.PreSteps = "unchanged"
				} else {
					data.PreAgentSteps = "unchanged"
				}
				return data
			}

			t.Run("merge and pin both sources", func(t *testing.T) {
				data := newData()
				latest, ok := getLatestActionPinByRepo("actions/checkout")
				require.True(t, ok)
				action := "actions/checkout@" + latest.Version
				frontmatter := map[string]any{
					section: []any{
						map[string]any{
							"id": "main", "uses": action,
							"if":                "${{ success() }}",
							"with":              map[string]any{"fetch-depth": 0},
							"env":               map[string]any{"MODE": "main"},
							"continue-on-error": true, "timeout-minutes": 3,
						},
						map[string]any{"id": "script", "run": "echo first\necho second\n", "shell": "bash"},
					},
				}
				require.NoError(t, process(frontmatter, data, "- id: imported\n  uses: "+action+"\n"))
				result := output(data)
				pin, err := getActionPinWithData("actions/checkout", latest.Version, data)
				require.NoError(t, err)
				require.Regexp(t, `^actions/checkout@[0-9a-f]{40} # `, pin)
				assert.Contains(t, result, "uses: "+pin+"\n", "version comment must remain unquoted")

				var wrapper map[string][]map[string]any
				require.NoError(t, yaml.Unmarshal([]byte(result), &wrapper))
				require.Len(t, wrapper, 1)
				steps := wrapper[section]
				require.Len(t, steps, 3)
				assert.Equal(t, "imported", steps[0]["id"])
				assert.Equal(t, "main", steps[1]["id"])
				assert.Equal(t, "script", steps[2]["id"])
				assert.Equal(t, pin[:len("actions/checkout@")+40], steps[0]["uses"])
				assert.Equal(t, steps[0]["uses"], steps[1]["uses"])
				assert.Equal(t, "${{ success() }}", steps[1]["if"])
				assert.Equal(t, map[string]any{"fetch-depth": uint64(0)}, steps[1]["with"])
				assert.Equal(t, map[string]any{"MODE": "main"}, steps[1]["env"])
				assert.Equal(t, true, steps[1]["continue-on-error"])
				assert.EqualValues(t, 3, steps[1]["timeout-minutes"])
				assert.Equal(t, "echo first\necho second\n", steps[2]["run"])
				assert.Equal(t, "bash", steps[2]["shell"])
			})

			for _, tc := range []struct {
				name     string
				main     any
				imported string
				want     string
			}{
				{name: "absent"},
				{name: "empty lists", main: []any{}, imported: "[]"},
				{name: "non-list main", main: "not a list"},
				{name: "import only", imported: "- run: echo imported\n", want: "echo imported"},
				{name: "main only", main: []any{map[string]any{"run": "echo main"}}, want: "echo main"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					data := newData()
					frontmatter := map[string]any{}
					if tc.main != nil {
						frontmatter[section] = tc.main
					}
					require.NoError(t, process(frontmatter, data, tc.imported))
					if tc.want == "" {
						assert.Equal(t, "unchanged", output(data))
					} else {
						assert.Equal(t, fmt.Sprintf("%s:\n- run: %s\n", section, tc.want), output(data))
					}
				})
			}

			for _, tc := range []struct {
				name     string
				main     []any
				imported string
				prefix   string
			}{
				{name: "malformed import", imported: "invalid: [yaml", prefix: "imported " + section + " YAML is not recognized"},
				{name: "non-list import", imported: "run: echo test", prefix: "imported " + section + " YAML is not recognized"},
				{name: "invalid imported step", imported: "- not a step", prefix: "imported " + section + " could not be converted to typed steps"},
				{name: "invalid main step", main: []any{"not a step"}, prefix: section + " could not be converted to typed steps"},
				{name: "import pin error", imported: "- uses: gh-aw-test/unversioned-action", prefix: "imported " + section + ": unversioned action"},
				{name: "main pin error", main: []any{map[string]any{"uses": "gh-aw-test/unversioned-action"}}, prefix: section + ": unversioned action"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					data := newData()
					err := process(map[string]any{section: tc.main}, data, tc.imported)
					require.ErrorContains(t, err, tc.prefix)
					assert.Equal(t, "unchanged", output(data))
				})
			}
		})
	}
}
