//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
)

func TestExtractExperimentVariantStubs_NoExperiments(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{}
	stubs := extractExperimentVariantStubs(cfg)
	assert.Empty(t, stubs)
}

func TestExtractExperimentVariantStubs_NilExperimentConfig(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"exp1": nil,
		},
	}
	stubs := extractExperimentVariantStubs(cfg)
	assert.Empty(t, stubs)
}

func TestExtractExperimentVariantStubs_SingleExperiment(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"prompt_style": {Variants: []string{"concise", "verbose"}},
		},
	}
	stubs := extractExperimentVariantStubs(cfg)
	assert.Len(t, stubs, 2)
	assert.Equal(t, "prompt_style", stubs[0].ExperimentName)
	assert.Equal(t, "concise", stubs[0].Variant)
	assert.Equal(t, "prompt_style", stubs[1].ExperimentName)
	assert.Equal(t, "verbose", stubs[1].Variant)
}

func TestExtractExperimentVariantStubs_SortsByExperimentNameThenVariant(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"zeta":  {Variants: []string{"b", "a"}},
			"alpha": {Variants: []string{"y", "x"}},
		},
	}
	stubs := extractExperimentVariantStubs(cfg)
	assert.Len(t, stubs, 4)
	// alpha comes before zeta
	assert.Equal(t, "alpha", stubs[0].ExperimentName)
	assert.Equal(t, "x", stubs[0].Variant)
	assert.Equal(t, "alpha", stubs[1].ExperimentName)
	assert.Equal(t, "y", stubs[1].Variant)
	assert.Equal(t, "zeta", stubs[2].ExperimentName)
	assert.Equal(t, "a", stubs[2].Variant)
	assert.Equal(t, "zeta", stubs[3].ExperimentName)
	assert.Equal(t, "b", stubs[3].Variant)
}

func TestExtractExperimentVariantStubs_MultipleExperimentsMixedNil(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"exp1": {Variants: []string{"v1"}},
			"exp2": nil,
			"exp3": {Variants: []string{}},
		},
	}
	stubs := extractExperimentVariantStubs(cfg)
	assert.Len(t, stubs, 1)
	assert.Equal(t, "exp1", stubs[0].ExperimentName)
	assert.Equal(t, "v1", stubs[0].Variant)
}

func TestExtractExperimentVariantStubs_EmptyVariants(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"exp1": {Variants: []string{}},
		},
	}
	assert.Empty(t, extractExperimentVariantStubs(cfg))
}

func TestExtractExperimentVariantStubs_RunCountAndFractionDefaultZero(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"exp1": {Variants: []string{"a"}},
		},
	}
	stubs := extractExperimentVariantStubs(cfg)
	assert.Len(t, stubs, 1)
	assert.Equal(t, 0, stubs[0].RunCount)
	assert.InDelta(t, 0.0, stubs[0].Fraction, 0)
}

func TestExtractExperimentVariantStubs_Pure(t *testing.T) {
	t.Parallel()
	cfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"exp1": {Variants: []string{"b", "a"}},
		},
	}
	wantCfg := &workflow.FrontmatterConfig{
		ExperimentConfigs: map[string]*workflow.ExperimentConfig{
			"exp1": {Variants: []string{"b", "a"}},
		},
	}

	first := extractExperimentVariantStubs(cfg)
	second := extractExperimentVariantStubs(cfg)

	assert.Equal(t, first, second)
	assert.Equal(t, wantCfg, cfg)
}
