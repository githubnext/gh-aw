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
		{
			name: "skips experiment with invalid name (hyphen)",
			frontmatter: map[string]any{
				"experiments": map[string]any{
					"my-flag": []any{"A", "B"},
					"valid":   []any{"X", "Y"},
				},
			},
			want: map[string][]string{"valid": {"X", "Y"}},
		},
		{
			name: "skips experiment with name starting with digit",
			frontmatter: map[string]any{
				"experiments": map[string]any{
					"1invalid": []any{"A", "B"},
					"valid":    []any{"X", "Y"},
				},
			},
			want: map[string][]string{"valid": {"X", "Y"}},
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

func TestGenerateExperimentSteps_SingleQuoteEscaping(t *testing.T) {
	c := &Compiler{}
	data := &WorkflowData{
		Experiments: map[string][]string{
			"variant": {"Bob's choice", "Alice's choice"},
		},
	}
	steps := c.generateExperimentSteps(data)
	joined := strings.Join(steps, "")
	// Single quotes in JSON string values must be doubled for YAML single-quoted scalar.
	assert.Contains(t, joined, "Bob''s", "single quotes in variant values must be escaped as '' in YAML")
	assert.Contains(t, joined, "Alice''s", "single quotes in variant values must be escaped as '' in YAML")
}

func TestExperimentExpressionMappings(t *testing.T) {
	experiments := map[string][]string{
		"caveman": {"yes", "no"},
		"style":   {"concise", "detailed"},
	}
	mappings := ExperimentExpressionMappings(experiments)
	require.Len(t, mappings, 2, "one mapping per experiment")

	// Build a lookup by EnvVar for easier assertions
	byEnvVar := make(map[string]*ExpressionMapping, len(mappings))
	for _, m := range mappings {
		byEnvVar[m.EnvVar] = m
	}

	m := byEnvVar["GH_AW_EXPERIMENTS_CAVEMAN"]
	require.NotNil(t, m, "mapping for GH_AW_EXPERIMENTS_CAVEMAN should exist")
	assert.Equal(t, "steps.pick-experiment.outputs.caveman", m.Content, "content should be the step output expression")
	assert.Equal(t, "${{ experiments.caveman }}", m.Original, "original should be the experiments expression")

	m2 := byEnvVar["GH_AW_EXPERIMENTS_STYLE"]
	require.NotNil(t, m2, "mapping for GH_AW_EXPERIMENTS_STYLE should exist")
	assert.Equal(t, "steps.pick-experiment.outputs.style", m2.Content, "content should be the step output expression")
}

// ── buildExperimentArtifactDownloadSteps ──────────────────────────────────

func TestBuildExperimentArtifactDownloadStep_Empty(t *testing.T) {
	steps := buildExperimentArtifactDownloadSteps("prefix-", nil)
	assert.Empty(t, steps, "no steps when experiments is nil")

	steps = buildExperimentArtifactDownloadSteps("prefix-", map[string][]string{})
	assert.Empty(t, steps, "no steps when experiments is empty")
}

func TestBuildExperimentArtifactDownloadStep_Generated(t *testing.T) {
	experiments := map[string][]string{"caveman": {"yes", "no"}}
	steps := buildExperimentArtifactDownloadSteps("${{ needs.activation.outputs.artifact_prefix }}", experiments)
	require.NotEmpty(t, steps, "steps should be generated when experiments are declared")
	joined := strings.Join(steps, "")
	assert.Contains(t, joined, "Download experiment artifact", "should include download step name")
	assert.Contains(t, joined, "experiment", "should reference experiment artifact")
	assert.Contains(t, joined, experimentsCacheDir, "should download to experiments cache dir")
	assert.Contains(t, joined, "actions/download-artifact", "should use download-artifact action")
}

func TestBuildExperimentArtifactDownloadStep_NoPrefix(t *testing.T) {
	// Non-workflow_call workflows use empty prefix
	experiments := map[string][]string{"style": {"A", "B"}}
	steps := buildExperimentArtifactDownloadSteps("", experiments)
	require.NotEmpty(t, steps, "steps should be generated")
	joined := strings.Join(steps, "")
	// Artifact name should be just the base name (no prefix)
	assert.Contains(t, joined, "          name: experiment\n", "artifact name should be unqualified for non-workflow_call")
}
