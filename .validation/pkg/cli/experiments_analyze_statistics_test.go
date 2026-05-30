//go:build !integration

package cli

import (
"encoding/json"
"math"
"testing"

"github.com/github/gh-aw/pkg/workflow"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

type chiSquarePValueRangeCase struct {
name    string
chi2    float64
df      int
wantMin float64
wantMax float64
}

type expectedProportionsCase struct {
name    string
names   []string
cfg     *workflow.ExperimentConfig
want    []float64
wantNil bool
}

// TestChiSquarePValue verifies the Wilson-Hilferty approximation against known values.
func TestChiSquarePValue(t *testing.T) {
for _, tt := range chiSquarePValueCases() {
t.Run(tt.name, func(t *testing.T) {
got := chiSquarePValue(tt.chi2, tt.df)
assert.True(t, got >= tt.wantMin && got <= tt.wantMax,
"chiSquarePValue(%.3f, %d) = %.6f, want [%.4f, %.4f]",
tt.chi2, tt.df, got, tt.wantMin, tt.wantMax)
})
}
}

func chiSquarePValueCases() []chiSquarePValueRangeCase {
return []chiSquarePValueRangeCase{
{name: "chi2=3.841 df=1 critical value for alpha=0.05", chi2: 3.841, df: 1, wantMin: 0.04, wantMax: 0.06},
{name: "chi2=0 df=1 returns 1.0", chi2: 0, df: 1, wantMin: 1.0, wantMax: 1.0},
{name: "very large chi2 df=2 returns near-zero p", chi2: 50.0, df: 2, wantMin: 0.0, wantMax: 0.0001},
{name: "chi2=5.991 df=2 critical value for alpha=0.05", chi2: 5.991, df: 2, wantMin: 0.04, wantMax: 0.06},
{name: "negative chi2 returns 1.0", chi2: -1.0, df: 1, wantMin: 1.0, wantMax: 1.0},
{name: "df=0 returns 1.0", chi2: 3.0, df: 0, wantMin: 1.0, wantMax: 1.0},
{name: "chi2=0.5 df=1 returns large p (balanced)", chi2: 0.5, df: 1, wantMin: 0.4, wantMax: 0.6},
}
}

// TestExpectedProportions verifies equal and weighted proportion computations.
func TestExpectedProportions(t *testing.T) {
for _, tt := range expectedProportionsCases() {
t.Run(tt.name, func(t *testing.T) {
got := expectedProportions(tt.names, tt.cfg)
if tt.wantNil {
assert.Nil(t, got, "nil input should return nil")
return
}
assertExpectedProportions(t, got, tt.want)
})
}
}

func expectedProportionsCases() []expectedProportionsCase {
return []expectedProportionsCase{
{name: "equal proportions when no config", names: []string{"A", "B", "C"}, want: []float64{1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0}},
{name: "equal proportions when config has no weights", names: []string{"concise", "detailed"}, cfg: &workflow.ExperimentConfig{Variants: []string{"concise", "detailed"}}, want: []float64{0.5, 0.5}},
{name: "weighted proportions from config", names: []string{"concise", "detailed"}, cfg: &workflow.ExperimentConfig{Variants: []string{"concise", "detailed"}, Weight: []int{70, 30}}, want: []float64{0.70, 0.30}},
{name: "equal proportions when weight length mismatches variants", names: []string{"A", "B", "C"}, cfg: &workflow.ExperimentConfig{Variants: []string{"A", "B", "C"}, Weight: []int{50, 50}}, want: []float64{1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0}},
{name: "equal proportions when all weights are zero", names: []string{"A", "B"}, cfg: &workflow.ExperimentConfig{Variants: []string{"A", "B"}, Weight: []int{0, 0}}, want: []float64{0.5, 0.5}},
{name: "empty variant list returns nil", wantNil: true},
{name: "sorted variant names matched correctly to weights", names: []string{"alpha", "beta", "gamma"}, cfg: &workflow.ExperimentConfig{Variants: []string{"gamma", "alpha", "beta"}, Weight: []int{10, 60, 30}}, want: []float64{0.60, 0.30, 0.10}},
}
}

func assertExpectedProportions(t *testing.T, got, want []float64) {
t.Helper()
require.Len(t, got, len(want), "should return one proportion per variant")
total := 0.0
for i, expected := range want {
assert.InDelta(t, expected, got[i], 0.0001, "unexpected proportion at index %d", i)
total += got[i]
}
assert.InDelta(t, 1.0, total, 0.0001, "proportions should sum to 1.0")
}

func TestComputeExperimentAnalysisBalancedBelowMinSamples(t *testing.T) {
exp := newExperimentVariantStats("style", map[string]int{"concise": 5, "detailed": 5})
a := computeExperimentAnalysis(exp, nil)

assert.Equal(t, "style", a.ExperimentName, "experiment name")
assert.Equal(t, defaultMinSamples, a.MinSamples, "default min_samples")
assert.Equal(t, 10, a.TotalRuns, "total runs")
assert.Len(t, a.Variants, 2, "two variants")
assert.Equal(t, "EXTEND", a.Recommendation, "below min_samples → EXTEND")
assert.True(t, a.IsBalanced, "perfectly balanced → is_balanced")
assert.Equal(t, 1, a.DegreesOfFreedom, "df=1 for two variants")
assert.Greater(t, a.PValue, balanceSignificanceThreshold, "perfect balance p > 0.05")
assert.InDelta(t, 0.0, a.BonferroniAlpha, 1e-10, "no Bonferroni for K=2")
}

func TestComputeExperimentAnalysisUsesMinSamplesFromConfig(t *testing.T) {
exp := newExperimentVariantStats("tone", map[string]int{"formal": 8, "casual": 8})
cfg := &workflow.ExperimentConfig{Variants: []string{"formal", "casual"}, MinSamples: 5}
a := computeExperimentAnalysis(exp, cfg)

assert.Equal(t, 5, a.MinSamples, "min_samples from config")
assert.Equal(t, "READY_FOR_ANALYSIS", a.Recommendation, "count >= min_samples → READY")
for _, v := range a.Variants {
assert.False(t, v.BelowMinSamples, "no variant should be below min_samples=5")
}
}

func TestComputeExperimentAnalysisUsesHypothesisAndAnalysisTypeFromConfig(t *testing.T) {
exp := newExperimentVariantStats("prompt", map[string]int{"short": 20, "long": 20})
cfg := &workflow.ExperimentConfig{
Variants:     []string{"short", "long"},
Hypothesis:   "H0: no change. H1: short reduces tokens by 15%.",
AnalysisType: "t_test",
MinSamples:   20,
}
a := computeExperimentAnalysis(exp, cfg)

assert.Equal(t, "H0: no change. H1: short reduces tokens by 15%.", a.Hypothesis, "hypothesis from config")
assert.Equal(t, "t_test", a.AnalysisType, "analysis_type from config")
assert.Equal(t, "READY_FOR_ANALYSIS", a.Recommendation, "count >= min_samples → READY")
}

func TestComputeExperimentAnalysisPropagatesGuardrails(t *testing.T) {
exp := newExperimentVariantStats("quality", map[string]int{"A": 3, "B": 2})
cfg := &workflow.ExperimentConfig{
Variants: []string{"A", "B"},
GuardrailMetrics: []workflow.GuardrailMetric{
{Name: "success_rate", Threshold: ">=0.95"},
{Name: "empty_output_rate", Threshold: "==0"},
},
}
a := computeExperimentAnalysis(exp, cfg)

require.Len(t, a.Guardrails, 2, "should have two guardrails")
assert.Equal(t, "success_rate", a.Guardrails[0].Name, "first guardrail name")
assert.Equal(t, ">=0.95", a.Guardrails[0].Threshold, "first guardrail threshold")
assert.Equal(t, "empty_output_rate", a.Guardrails[1].Name, "second guardrail name")
}

func TestComputeExperimentAnalysisBonferroniAlpha(t *testing.T) {
t.Run("Bonferroni alpha for K=3 variants (§11.3)", func(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("multi", map[string]int{"A": 5, "B": 5, "C": 5}), nil)
assert.Len(t, a.Variants, 3, "three variants")
assert.InDelta(t, 0.025, a.BonferroniAlpha, 0.0001, "Bonferroni alpha for K=3")
assert.Equal(t, 2, a.DegreesOfFreedom, "df=2 for K=3")
})

t.Run("Bonferroni alpha for K=4 variants", func(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("four", map[string]int{"A": 5, "B": 5, "C": 5, "D": 5}), nil)
assert.InDelta(t, 0.05/3.0, a.BonferroniAlpha, 0.0001, "Bonferroni alpha for K=4")
})
}

func TestComputeExperimentAnalysisBalanceScenarios(t *testing.T) {
t.Run("unbalanced distribution detected", func(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("skewed", map[string]int{"A": 19, "B": 1}), nil)
assert.False(t, a.IsBalanced, "extreme imbalance should not be balanced")
assert.Less(t, a.PValue, balanceSignificanceThreshold, "p < 0.05 for extreme imbalance")
})

t.Run("empty experiment returns EXTEND with zero total", func(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("empty", map[string]int{"A": 0, "B": 0}), nil)
assert.Equal(t, "EXTEND", a.Recommendation, "zero runs → EXTEND")
assert.True(t, a.IsBalanced, "insufficient data → default to balanced")
assert.InDelta(t, 0.0, a.ChiSquare, 1e-10, "no chi-square for zero total")
})
}

func TestComputeExperimentAnalysisVariantsSorted(t *testing.T) {
exp := newExperimentVariantStats("order", map[string]int{"z_last": 3, "a_first": 7})
a := computeExperimentAnalysis(exp, nil)

require.Len(t, a.Variants, 2, "two variants")
assert.Equal(t, "a_first", a.Variants[0].Name, "first variant alphabetically")
assert.Equal(t, "z_last", a.Variants[1].Name, "second variant alphabetically")
}

func TestComputeExperimentAnalysisWeightedProportionsUsedForBalance(t *testing.T) {
exp := newExperimentVariantStats("weighted", map[string]int{"control": 70, "variant": 30})
cfg := &workflow.ExperimentConfig{Variants: []string{"control", "variant"}, Weight: []int{70, 30}}
a := computeExperimentAnalysis(exp, cfg)

assert.True(t, a.IsBalanced, "70/30 split with 70/30 weights should be balanced")
require.Len(t, a.Variants, 2, "two variants")
assert.InDelta(t, 70.0, a.Variants[0].ExpectedPct, 0.1, "control expected 70%")
assert.InDelta(t, 30.0, a.Variants[1].ExpectedPct, 0.1, "variant expected 30%")
}

func newExperimentVariantStats(name string, variants map[string]int) ExperimentVariantStats {
total := 0
for _, count := range variants {
total += count
}
return ExperimentVariantStats{Name: name, Variants: variants, Total: total}
}

// TestComputeExperimentAnalyses tests the bulk analysis function.
func TestComputeExperimentAnalyses(t *testing.T) {
t.Run("empty experiments returns nil", func(t *testing.T) {
result := computeExperimentAnalyses(nil, nil)
assert.Nil(t, result, "nil experiments should return nil")
})

t.Run("multiple experiments analysed independently", func(t *testing.T) {
experiments := []ExperimentVariantStats{
{Name: "exp1", Variants: map[string]int{"A": 5, "B": 5}, Total: 10},
{Name: "exp2", Variants: map[string]int{"X": 3, "Y": 7}, Total: 10},
}
analyses := computeExperimentAnalyses(experiments, nil)
require.Len(t, analyses, 2, "should produce one analysis per experiment")
assert.Equal(t, "exp1", analyses[0].ExperimentName, "first analysis name")
assert.Equal(t, "exp2", analyses[1].ExperimentName, "second analysis name")
})

t.Run("config map applied per experiment", func(t *testing.T) {
experiments := []ExperimentVariantStats{{Name: "alpha", Variants: map[string]int{"on": 25, "off": 25}, Total: 50}}
configs := map[string]*workflow.ExperimentConfig{
"alpha": {Variants: []string{"on", "off"}, Hypothesis: "test hypothesis", AnalysisType: "proportion_test", MinSamples: 20},
}
analyses := computeExperimentAnalyses(experiments, configs)
require.Len(t, analyses, 1, "one analysis")
assert.Equal(t, "test hypothesis", analyses[0].Hypothesis, "hypothesis from config")
assert.Equal(t, "proportion_test", analyses[0].AnalysisType, "analysis type from config")
assert.Equal(t, "READY_FOR_ANALYSIS", analyses[0].Recommendation, "above min_samples")
})
}

// TestExperimentAnalysisJSONOutput verifies that ExperimentAnalysis serialises correctly.
func TestExperimentAnalysisJSONOutput(t *testing.T) {
exp := newExperimentVariantStats("style", map[string]int{"concise": 12, "detailed": 8})
cfg := &workflow.ExperimentConfig{
Variants:     []string{"concise", "detailed"},
Hypothesis:   "H0: no change. H1: concise is better.",
AnalysisType: "t_test",
MinSamples:   30,
GuardrailMetrics: []workflow.GuardrailMetric{
{Name: "success_rate", Threshold: ">=0.95"},
},
}

a := computeExperimentAnalysis(exp, cfg)
result := marshalExperimentAnalysis(t, a)

assert.Equal(t, "style", result["experiment_name"], "experiment_name field")
assert.Equal(t, "H0: no change. H1: concise is better.", result["hypothesis"], "hypothesis field")
assert.Equal(t, "t_test", result["analysis_type"], "analysis_type field")
assert.EqualValues(t, 30, result["min_samples"], "min_samples field")
assert.Equal(t, "EXTEND", result["recommendation"], "recommendation field (below min_samples)")
assertJSONArrayLen(t, result, "variants", 2)
assertJSONArrayLen(t, result, "guardrails", 1)
}

func marshalExperimentAnalysis(t *testing.T, analysis ExperimentAnalysis) map[string]any {
t.Helper()
jsonBytes, err := json.MarshalIndent(analysis, "", "  ")
require.NoError(t, err, "should marshal analysis to JSON")

var result map[string]any
require.NoError(t, json.Unmarshal(jsonBytes, &result), "should unmarshal JSON")
return result
}

func assertJSONArrayLen(t *testing.T, data map[string]any, field string, want int) {
t.Helper()
values, ok := data[field].([]any)
require.True(t, ok, "%s should be an array", field)
require.Len(t, values, want, "unexpected %s length", field)
}

// TestExperimentAnalysisBonferroniAbsent verifies BonferroniAlpha is omitted for K < 3.
func TestExperimentAnalysisBonferroniAbsent(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("binary", map[string]int{"yes": 10, "no": 10}), nil)
assert.InDelta(t, 0.0, a.BonferroniAlpha, 1e-10, "BonferroniAlpha should be zero for K=2")

result := marshalExperimentAnalysis(t, a)
_, present := result["bonferroni_alpha"]
assert.False(t, present, "bonferroni_alpha should be omitted from JSON for K=2")
}

// TestMinSamplesDefaultApplied verifies the default min_samples value is 20.
func TestMinSamplesDefaultApplied(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("test", map[string]int{"A": 10, "B": 10}), nil)
assert.Equal(t, defaultMinSamples, a.MinSamples, "default min_samples should be 20")
}

// TestChiSquarePValueMonotonicity verifies that larger chi2 values produce smaller p-values.
func TestChiSquarePValueMonotonicity(t *testing.T) {
prev := chiSquarePValue(0.1, 1)
for _, chi2 := range []float64{0.5, 1.0, 2.0, 3.841, 6.0, 10.0} {
p := chiSquarePValue(chi2, 1)
assert.Less(t, p, prev, "p-value should decrease as chi2 increases (chi2=%.1f)", chi2)
prev = p
}
}

// TestChiSquarePValueReturnRange verifies p-values are always in [0, 1].
func TestChiSquarePValueReturnRange(t *testing.T) {
inputs := []struct {
chi2 float64
df   int
}{{0, 1}, {0.1, 1}, {1.0, 1}, {3.841, 1}, {100, 1}, {0.1, 2}, {5.991, 2}, {0.1, 10}, {20.0, 5}}
for _, tc := range inputs {
p := chiSquarePValue(tc.chi2, tc.df)
assert.True(t, p >= 0 && p <= 1,
"p-value %.6f out of [0,1] for chi2=%.2f df=%d", p, tc.chi2, tc.df)
}
}

// TestObservedPctSumsToHundred verifies that observed percentages sum to ~100%.
func TestObservedPctSumsToHundred(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("pct_test", map[string]int{"A": 7, "B": 13}), nil)
total := 0.0
for _, v := range a.Variants {
total += v.ObservedPct
}
assert.InDelta(t, 100.0, total, 0.01, "observed percentages should sum to 100")
}

// TestExpectedPctSumsToHundred verifies expected percentages sum to ~100%.
func TestExpectedPctSumsToHundred(t *testing.T) {
tests := []struct {
name     string
variants map[string]int
cfg      *workflow.ExperimentConfig
}{
{name: "equal split", variants: map[string]int{"A": 10, "B": 10}},
{name: "three equal", variants: map[string]int{"A": 5, "B": 5, "C": 5}},
{name: "weighted 60/40", variants: map[string]int{"A": 12, "B": 8}, cfg: &workflow.ExperimentConfig{Variants: []string{"A", "B"}, Weight: []int{60, 40}}},
}
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("e", tt.variants), tt.cfg)
sum := 0.0
for _, v := range a.Variants {
sum += v.ExpectedPct
}
assert.InDelta(t, 100.0, sum, 0.01, "expected percentages should sum to 100")
})
}
}

// TestReadyForAnalysisAllAboveMinSamples verifies the recommendation when all counts >= min_samples.
func TestReadyForAnalysisAllAboveMinSamples(t *testing.T) {
exp := newExperimentVariantStats("ready", map[string]int{"X": 25, "Y": 30})
cfg := &workflow.ExperimentConfig{Variants: []string{"X", "Y"}, MinSamples: 20}
a := computeExperimentAnalysis(exp, cfg)
assert.Equal(t, "READY_FOR_ANALYSIS", a.Recommendation, "all variants above min_samples → READY")
assert.Contains(t, a.Rationale, "min_samples", "rationale should mention min_samples")

for _, v := range a.Variants {
assert.False(t, v.BelowMinSamples, "no variant should be below min_samples")
}
}

// TestPartiallyBelowMinSamples verifies EXTEND when only some variants are below threshold.
func TestPartiallyBelowMinSamples(t *testing.T) {
exp := newExperimentVariantStats("partial", map[string]int{"above": 25, "below": 5})
cfg := &workflow.ExperimentConfig{Variants: []string{"above", "below"}, MinSamples: 20}
a := computeExperimentAnalysis(exp, cfg)
assert.Equal(t, "EXTEND", a.Recommendation, "one variant below threshold → EXTEND")
assert.Contains(t, a.Rationale, "1 of 2", "rationale should count variants below threshold")

require.Len(t, a.Variants, 2, "should have two variants")
assert.Equal(t, "above", a.Variants[0].Name, "first variant alphabetically")
assert.Equal(t, "below", a.Variants[1].Name, "second variant alphabetically")
assert.False(t, a.Variants[0].BelowMinSamples, "'above' (count=25) should not be flagged")
assert.True(t, a.Variants[1].BelowMinSamples, "'below' (count=5) should be flagged")
}

// TestChiSquarePerfectBalance verifies that chi² = 0 for a perfectly balanced sample.
func TestChiSquarePerfectBalance(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("perfect", map[string]int{"A": 10, "B": 10, "C": 10}), nil)
assert.InDelta(t, 0.0, a.ChiSquare, 1e-10, "chi² should be 0 for perfectly balanced sample")
assert.InDelta(t, 1.0, a.PValue, 1e-10, "p should be 1.0 for chi²=0")
assert.True(t, a.IsBalanced, "perfectly balanced → is_balanced")
}

// TestFindWorkflowFileForExperiment verifies that the function returns "" in isolation
// (no .github/workflows directory in the test working directory).
func TestFindWorkflowFileForExperiment(t *testing.T) {
result := findWorkflowFileForExperiment("nonexistent_experiment")
assert.Empty(t, result, "should return empty string when no matching file found")
}

// TestMatchWorkflowFilenameByExperiment verifies that the helper correctly resolves
// hyphenated workflow filenames from sanitized experiment branch names.
func TestMatchWorkflowFilenameByExperiment(t *testing.T) {
files := []string{"ci-coach.md", "daily-issues-report.md", "smoke-copilot.md", "agent-performance-analyzer.md", "deep-report.md", "noHyphen.md"}
tests := []struct {
experimentName string
want           string
}{
{"cicoach", "ci-coach"},
{"dailyissuesreport", "daily-issues-report"},
{"smokecopilot", "smoke-copilot"},
{"agentperformanceanalyzer", "agent-performance-analyzer"},
{"deepreport", "deep-report"},
{"nohyphen", "noHyphen"},
{"notfound", ""},
}
for _, tt := range tests {
t.Run(tt.experimentName, func(t *testing.T) {
got := matchWorkflowFilenameByExperiment(files, tt.experimentName)
assert.Equal(t, tt.want, got)
})
}
}

// TestMatchWorkflowFilenameByExperimentAmbiguous verifies that the first match is returned
// and a warning is logged when multiple files share the same sanitized name.
func TestMatchWorkflowFilenameByExperimentAmbiguous(t *testing.T) {
got := matchWorkflowFilenameByExperiment([]string{"ci-coach.md", "cicoach.md"}, "cicoach")
assert.Equal(t, "ci-coach", got, "should return the first match")
}

// TestWorkflowFileCandidates verifies the fallback candidate list for remote lookups.
// The real resolution is handled by findRemoteWorkflowFilenameForExperiment; this list
// is only used as a last-resort fallback when the directory listing is unavailable.
func TestWorkflowFileCandidates(t *testing.T) {
for _, tt := range []struct{ experimentName, wantContains string }{{"myworkflow", "myworkflow"}, {"dailyreport", "dailyreport"}, {"test123", "test123"}} {
t.Run(tt.experimentName, func(t *testing.T) {
candidates := workflowFileCandidates(tt.experimentName)
assert.NotEmpty(t, candidates, "should return at least one candidate")
assert.Contains(t, candidates, tt.wantContains, "should contain the experiment name itself")
})
}
}

// TestAnalysisWithNilConfig verifies analysis runs cleanly without a config.
func TestAnalysisWithNilConfig(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("no_config", map[string]int{"on": 8, "off": 12}), nil)
assert.Equal(t, "no_config", a.ExperimentName, "experiment name preserved")
assert.Empty(t, a.Hypothesis, "no hypothesis without config")
assert.Empty(t, a.AnalysisType, "no analysis type without config")
assert.Equal(t, defaultMinSamples, a.MinSamples, "default min_samples")
assert.Empty(t, a.Guardrails, "no guardrails without config")
assert.Equal(t, "EXTEND", a.Recommendation, "below default min_samples → EXTEND")
assert.False(t, math.IsNaN(a.PValue), "p-value should not be NaN")
assert.True(t, a.PValue >= 0 && a.PValue <= 1, "p-value should be in [0,1]")
}

// TestComputeExperimentAnalysisDegenerateVariants tests degenerate experiments with < 2 variants.
func TestComputeExperimentAnalysisDegenerateVariants(t *testing.T) {
t.Run("zero variants returns EXTEND", func(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("no_variants", map[string]int{}), nil)
assert.Equal(t, "EXTEND", a.Recommendation, "zero variants → EXTEND")
assert.True(t, a.IsBalanced, "degenerate case defaults to balanced")
assert.Empty(t, a.Variants, "no variant entries")
assert.Contains(t, a.Rationale, "fewer than 2 variants", "rationale mentions variant count")
})

t.Run("single variant returns EXTEND", func(t *testing.T) {
a := computeExperimentAnalysis(newExperimentVariantStats("one_variant", map[string]int{"only": 10}), nil)
assert.Equal(t, "EXTEND", a.Recommendation, "single variant → EXTEND")
assert.True(t, a.IsBalanced, "degenerate case defaults to balanced")
assert.Empty(t, a.Variants, "no variant entries emitted for degenerate case")
assert.Contains(t, a.Rationale, "fewer than 2 variants", "rationale is clear")
})
}
