//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── extractExperimentsFromFrontmatter ─────────────────────────────────────

func TestExtractExperimentsFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		want        map[string][]string
	}{
		{
			name:        "nil frontmatter returns nil",
			frontmatter: map[string]any{},
			want:        nil,
		},
		{
			name: "basic two-variant experiment",
			frontmatter: map[string]any{
				"experiments": map[string]any{
					"feature1": []any{"A", "B"},
				},
			},
			want: map[string][]string{"feature1": {"A", "B"}},
		},
		{
			name: "three variants",
			frontmatter: map[string]any{
				"experiments": map[string]any{
					"style": []any{"concise", "detailed", "structured"},
				},
			},
			want: map[string][]string{"style": {"concise", "detailed", "structured"}},
		},
		{
			name: "skips experiment with fewer than two variants",
			frontmatter: map[string]any{
				"experiments": map[string]any{
					"bad":  []any{"only-one"},
					"good": []any{"A", "B"},
				},
			},
			want: map[string][]string{"good": {"A", "B"}},
		},
		{
			name: "multiple experiments",
			frontmatter: map[string]any{
				"experiments": map[string]any{
					"feat1": []any{"X", "Y"},
					"feat2": []any{"P", "Q", "R"},
				},
			},
			want: map[string][]string{
				"feat1": {"X", "Y"},
				"feat2": {"P", "Q", "R"},
			},
		},
		{
			name: "returns nil when experiments map is empty",
			frontmatter: map[string]any{
				"experiments": map[string]any{},
			},
			want: nil,
		},
		{
			name: "handles native []string slice",
			frontmatter: map[string]any{
				"experiments": map[string]any{
					"feature1": []string{"A", "B"},
				},
			},
			want: map[string][]string{"feature1": {"A", "B"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExperimentsFromFrontmatter(tt.frontmatter)
			assert.Equal(t, tt.want, got, "extracted experiments should match")
		})
	}
}

// ── sortedExperimentNames ─────────────────────────────────────────────────

func TestSortedExperimentNames(t *testing.T) {
	experiments := map[string][]string{
		"z_exp": {"A", "B"},
		"a_exp": {"X", "Y"},
		"m_exp": {"P", "Q"},
	}
	got := sortedExperimentNames(experiments)
	require.Equal(t, []string{"a_exp", "m_exp", "z_exp"}, got, "names should be sorted alphabetically")
}

// ── buildExperimentSpecJSON ───────────────────────────────────────────────

func TestBuildExperimentSpecJSON(t *testing.T) {
	experiments := map[string][]string{
		"feature1": {"A", "B"},
		"style":    {"concise", "detailed"},
	}
	names := []string{"feature1", "style"}
	got := buildExperimentSpecJSON(experiments, names)
	assert.JSONEq(t, `{"feature1":["A","B"],"style":["concise","detailed"]}`, got, "JSON spec should match expected structure")
}

func TestBuildExperimentSpecJSONEscaping(t *testing.T) {
	experiments := map[string][]string{
		`quote"test`: {`val"1`, `val\2`},
	}
	names := []string{`quote"test`}
	got := buildExperimentSpecJSON(experiments, names)
	assert.Contains(t, got, `\"`, "double quotes should be escaped in JSON")
}

// ── generateExperimentSteps ───────────────────────────────────────────────

func TestGenerateExperimentSteps_Empty(t *testing.T) {
	c := &Compiler{}
	data := &WorkflowData{}
	steps := c.generateExperimentSteps(data)
	assert.Empty(t, steps, "no steps should be generated when experiments is nil")
}

func TestGenerateExperimentSteps_Generated(t *testing.T) {
	c := &Compiler{}
	data := &WorkflowData{
		Experiments: map[string][]string{
			"feature1": {"A", "B"},
		},
	}
	steps := c.generateExperimentSteps(data)
	require.NotEmpty(t, steps, "steps should be generated when experiments are declared")

	joined := strings.Join(steps, "")
	assert.Contains(t, joined, "Restore experiment state", "should include cache restore step")
	assert.Contains(t, joined, "Pick experiment variants", "should include pick step")
	assert.Contains(t, joined, "pick_experiment.cjs", "should reference pick_experiment.cjs")
	assert.Contains(t, joined, "Save experiment state", "should include cache save step")
	assert.Contains(t, joined, "Upload experiment artifact", "should include artifact upload step")
	assert.Contains(t, joined, "experiment", "artifact name should include 'experiment'")
}

func TestGenerateExperimentSteps_SpecJSON(t *testing.T) {
	c := &Compiler{}
	data := &WorkflowData{
		Experiments: map[string][]string{
			"style": {"concise", "detailed"},
		},
	}
	steps := c.generateExperimentSteps(data)
	joined := strings.Join(steps, "")
	assert.Contains(t, joined, `{"style":["concise","detailed"]}`, "spec JSON should be embedded in the step")
}
