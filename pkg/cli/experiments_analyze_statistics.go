package cli

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var experimentsStatsLog = logger.New("cli:experiments_statistics")

// defaultMinSamples is the minimum samples per variant before analysis is reliable (§11.4 / R-STAT-007).
const defaultMinSamples = 20

// balanceSignificanceThreshold is the p-value threshold for the chi-square balance test.
const balanceSignificanceThreshold = 0.05

// ExperimentAnalysis holds statistical analysis results for one named A/B experiment.
// The analysis is computed from state.json counts and optional experiment configuration.
type ExperimentAnalysis struct {
	// ExperimentName is the name of the A/B experiment (key in state.counts).
	ExperimentName string `json:"experiment_name"`

	// Hypothesis is the null/alternative hypothesis text (from experiment config).
	Hypothesis string `json:"hypothesis,omitempty"`

	// AnalysisType is the statistical test declared in the experiment config
	// (t_test, mann_whitney, proportion_test, bayesian_ab).
	AnalysisType string `json:"analysis_type,omitempty"`

	// Metric is the primary metric string declared in the experiment config
	// (e.g. "effective_tokens" or "evals.builds").
	Metric string `json:"metric,omitempty"`

	// MetricQuestion is the resolved eval question text when Metric references a declared
	// eval question ID (e.g. "evals.builds" resolves to "Does the generated code compile?").
	// Empty when Metric is absent or does not reference an eval.
	MetricQuestion string `json:"metric_question,omitempty"`

	// MinSamples is the minimum runs per variant required before analysis is reliable.
	// Defaults to 20 when not declared in the experiment config (R-STAT-007).
	MinSamples int `json:"min_samples"`

	// TotalRuns is the total number of observed runs across all variants.
	TotalRuns int `json:"total_runs"`

	// Variants holds per-variant statistics in alphabetical order.
	Variants []VariantAnalysis `json:"variants"`

	// Balance test (chi-square goodness-of-fit against expected allocation, §11.1).
	ChiSquare        float64 `json:"chi_square"`
	DegreesOfFreedom int     `json:"degrees_of_freedom"`
	PValue           float64 `json:"p_value"`
	IsBalanced       bool    `json:"is_balanced"`

	// BonferroniAlpha is the Bonferroni-corrected significance threshold for experiments
	// with K ≥ 3 variants (§11.3: α_adjusted = 0.05 / (K − 1)).
	// Zero when fewer than 3 variants are declared.
	BonferroniAlpha float64 `json:"bonferroni_alpha,omitempty"`

	// Guardrails lists the declared metric thresholds.
	// Pass/fail evaluation requires per-run outcome data not stored in state.json (R-STAT-009).
	Guardrails []GuardrailStatus `json:"guardrails,omitempty"`

	// Recommendation is the analysis recommendation: EXTEND or READY_FOR_ANALYSIS.
	// EXTEND is issued when any variant is below min_samples (R-STAT-007).
	Recommendation string `json:"recommendation"`

	// Rationale is a one-sentence explanation of the recommendation.
	Rationale string `json:"rationale"`
}

// VariantAnalysis holds per-variant statistics for one experiment.
type VariantAnalysis struct {
	// Name is the variant identifier (e.g., "concise", "detailed").
	Name string `json:"name"`

	// Count is the number of times this variant was selected (from state.counts).
	Count int `json:"count"`

	// ObservedPct is the observed percentage share of total runs (0–100).
	ObservedPct float64 `json:"observed_pct"`

	// ExpectedPct is the expected percentage share based on declared weights or equal split (0–100).
	ExpectedPct float64 `json:"expected_pct"`

	// MinSamples is the minimum required count for this variant.
	MinSamples int `json:"min_samples"`

	// BelowMinSamples is true when Count < MinSamples.
	BelowMinSamples bool `json:"below_min_samples"`
}

// GuardrailStatus represents a declared guardrail metric threshold (R-STAT-009).
// The Threshold field records the declared expression (e.g. ">=0.95").
// Pass/fail evaluation is not performed here as it requires outcome metric data.
type GuardrailStatus struct {
	Name      string `json:"name"`
	Threshold string `json:"threshold"`
}

// computeExperimentAnalysis computes the statistical analysis for a single named experiment.
// cfg may be nil when no workflow frontmatter is available, in which case defaults are used.
// evals provides the eval definitions for resolving eval-backed metric references; may be nil.
func computeExperimentAnalysis(exp ExperimentVariantStats, cfg *workflow.ExperimentConfig, evals *workflow.EvalsConfig) ExperimentAnalysis {
	experimentsStatsLog.Printf("Computing analysis for experiment %q: %d variant(s), %d total runs", exp.Name, len(exp.Variants), exp.Total)
	a := ExperimentAnalysis{
		ExperimentName: exp.Name,
		TotalRuns:      exp.Total,
		MinSamples:     defaultMinSamples,
	}

	// Extract metadata from config when available.
	if cfg != nil {
		a.Hypothesis = cfg.Hypothesis
		a.AnalysisType = cfg.AnalysisType
		if cfg.MinSamples > 0 {
			a.MinSamples = cfg.MinSamples
		}
		for _, g := range cfg.GuardrailMetrics {
			a.Guardrails = append(a.Guardrails, GuardrailStatus{
				Name:      g.Name,
				Threshold: g.Threshold,
			})
		}
		// Populate metric and resolve any eval reference to its question text.
		if cfg.Metric != "" {
			a.Metric = cfg.Metric
			evalID, isEval := workflow.ParseExperimentMetricEvalReference(cfg.Metric)
			if isEval && evalID != "" && evals != nil {
				for _, q := range evals.Questions {
					if q.ID == evalID {
						a.MetricQuestion = q.Question
						break
					}
				}
			}
		}
	}

	// Degenerate: fewer than 2 variants cannot be meaningfully analysed.
	if len(exp.Variants) < 2 {
		experimentsStatsLog.Printf("Experiment %q has fewer than 2 variants; skipping analysis", exp.Name)
		a.IsBalanced = true
		a.Recommendation = "EXTEND"
		a.Rationale = "experiment has fewer than 2 variants; cannot perform statistical analysis"
		return a
	}

	// Collect variant names in alphabetical order for deterministic output.
	variantNames := sliceutil.SortedKeys(exp.Variants)

	// Compute expected proportions (weighted or equal split).
	expectedPcts := expectedProportions(variantNames, cfg)

	// Populate per-variant entries.
	for i, name := range variantNames {
		count := exp.Variants[name]
		obsPct := safePercent(count, exp.Total)
		a.Variants = append(a.Variants, VariantAnalysis{
			Name:            name,
			Count:           count,
			ObservedPct:     obsPct,
			ExpectedPct:     expectedPcts[i] * 100,
			MinSamples:      a.MinSamples,
			BelowMinSamples: count < a.MinSamples,
		})
	}

	k := len(variantNames)

	// Chi-square goodness-of-fit balance test.
	if exp.Total > 0 && k >= 2 {
		chi2 := 0.0
		for i, name := range variantNames {
			expected := float64(exp.Total) * expectedPcts[i]
			if expected > 0 {
				diff := float64(exp.Variants[name]) - expected
				chi2 += (diff * diff) / expected
			}
		}
		df := k - 1
		pval := chiSquarePValue(chi2, df)
		a.ChiSquare = chi2
		a.DegreesOfFreedom = df
		a.PValue = pval
		a.IsBalanced = pval >= balanceSignificanceThreshold
		experimentsStatsLog.Printf("Chi-square balance test for %q: χ²=%.3f df=%d p=%.3f balanced=%v", exp.Name, chi2, df, pval, a.IsBalanced)
	} else {
		a.IsBalanced = true // insufficient data to assess balance
	}

	// Bonferroni correction for K ≥ 3 variants (§11.3).
	if k >= 3 {
		a.BonferroniAlpha = 0.05 / float64(k-1)
	}

	// Recommendation (R-STAT-007).
	belowCount := 0
	minObserved := math.MaxInt
	for _, v := range a.Variants {
		if v.BelowMinSamples {
			belowCount++
		}
		if v.Count < minObserved {
			minObserved = v.Count
		}
	}
	if belowCount > 0 {
		a.Recommendation = "EXTEND"
		a.Rationale = fmt.Sprintf("%d of %d variant(s) below min_samples threshold (min observed: %d / %d)",
			belowCount, k, minObserved, a.MinSamples)
		experimentsStatsLog.Printf("Experiment %q recommendation: EXTEND (%d/%d variants below min_samples=%d)", exp.Name, belowCount, k, a.MinSamples)
	} else {
		a.Recommendation = "READY_FOR_ANALYSIS"
		a.Rationale = fmt.Sprintf("all %d variants have reached min_samples (%d); proceed with outcome metric analysis",
			k, a.MinSamples)
		experimentsStatsLog.Printf("Experiment %q recommendation: READY_FOR_ANALYSIS (all %d variants above min_samples=%d)", exp.Name, k, a.MinSamples)
	}

	return a
}

// expectedProportions returns the expected fraction (0.0–1.0) for each variant, in the
// same order as sortedVariantNames. When cfg provides valid per-variant weights that cover
// every name in sortedVariantNames, uses weighted proportions; otherwise returns an equal split.
func expectedProportions(sortedVariantNames []string, cfg *workflow.ExperimentConfig) []float64 {
	k := len(sortedVariantNames)
	if k == 0 {
		return nil
	}

	// Use weights from config when they are well-formed and cover the observed variants.
	if cfg != nil && len(cfg.Weight) == len(cfg.Variants) && len(cfg.Weight) > 0 {
		nameToWeight := make(map[string]float64, len(cfg.Variants))
		totalWeight := 0.0
		for i, name := range cfg.Variants {
			w := float64(cfg.Weight[i])
			nameToWeight[name] = w
			totalWeight += w
		}
		if totalWeight > 0 {
			// Only use weights when every observed variant name has a declared weight.
			result := make([]float64, k)
			allFound := true
			for i, name := range sortedVariantNames {
				w, ok := nameToWeight[name]
				if !ok {
					allFound = false
					break
				}
				result[i] = w / totalWeight
			}
			if allFound {
				return result
			}
		}
	}

	// Default: equal proportions.
	result := make([]float64, k)
	for i := range result {
		result[i] = 1.0 / float64(k)
	}
	return result
}

// chiSquarePValue computes the right-tail p-value P(X ≥ chi2) where X ~ Chi²(df).
// Uses the Wilson-Hilferty normal approximation via math.Erfc.
// Accuracy: good for df ≥ 1 and chi2 in roughly the range [0, 100]; the approximation
// degrades for very large chi2 values (>100) or very small df (df=1 with extreme chi2),
// but is adequate for the balance-testing use case (variants rarely exceed 10, and
// chi2 values outside the critical region are truncated by the significance gate).
// Returns 1.0 for degenerate inputs (chi2 ≤ 0 or df ≤ 0).
func chiSquarePValue(chi2 float64, df int) float64 {
	if chi2 <= 0 || df <= 0 {
		return 1.0
	}

	dfF := float64(df)
	// Wilson-Hilferty approximation: transform chi²/df to a standard normal deviate.
	h := 2.0 / (9.0 * dfF)
	x := math.Pow(chi2/dfF, 1.0/3.0)
	z := (x - (1.0 - h)) / math.Sqrt(h)

	// Right-tail: P(Z > z) = erfc(z / √2) / 2
	return math.Erfc(z/math.Sqrt2) / 2.0
}

// printExperimentAnalyses renders the statistical analyses to stderr.
func printExperimentAnalyses(analyses []ExperimentAnalysis) {
	if len(analyses) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "\nStatistical Analysis:")
	for _, a := range analyses {
		printOneExperimentAnalysis(a)
	}
}

// printOneExperimentAnalysis renders a single experiment analysis to stderr.
func printOneExperimentAnalysis(a ExperimentAnalysis) {
	fmt.Fprintf(os.Stderr, "\n  [%s]\n", a.ExperimentName)

	if a.Hypothesis != "" {
		fmt.Fprintf(os.Stderr, "  Hypothesis : %s\n", a.Hypothesis)
	}
	if a.AnalysisType != "" {
		fmt.Fprintf(os.Stderr, "  Test type  : %s\n", a.AnalysisType)
	}
	if a.Metric != "" {
		if a.MetricQuestion != "" {
			fmt.Fprintf(os.Stderr, "  Metric     : %s — %s\n", a.Metric, a.MetricQuestion)
		} else {
			fmt.Fprintf(os.Stderr, "  Metric     : %s\n", a.Metric)
		}
	}
	fmt.Fprintf(os.Stderr, "  Min samples: %d per variant\n", a.MinSamples)

	// Per-variant progress table.
	fmt.Fprintf(os.Stderr, "\n  %-20s %6s  %7s  %7s  %s\n", "Variant", "Count", "Obs%", "Exp%", "Progress")
	fmt.Fprintf(os.Stderr, "  %s\n", strings.Repeat("─", 62))
	for _, v := range a.Variants {
		progressStr := fmt.Sprintf("%d/%d", v.Count, v.MinSamples)
		if v.BelowMinSamples {
			progressStr += " ⚠"
		}
		fmt.Fprintf(os.Stderr, "  %-20s %6d  %6.1f%%  %6.1f%%  %s\n",
			v.Name, v.Count, v.ObservedPct, v.ExpectedPct, progressStr)
	}

	// Balance test.
	fmt.Fprintln(os.Stderr)
	if a.TotalRuns > 0 {
		balancedStr := "balanced ✓"
		if !a.IsBalanced {
			balancedStr = "unbalanced ✗"
		}
		fmt.Fprintf(os.Stderr, "  Balance    : χ² = %.3f  df = %d  p = %.3f  (%s)\n",
			a.ChiSquare, a.DegreesOfFreedom, a.PValue, balancedStr)
	} else {
		fmt.Fprintln(os.Stderr, "  Balance    : no data")
	}

	// Bonferroni correction.
	if a.BonferroniAlpha > 0 {
		fmt.Fprintf(os.Stderr, "  Bonferroni : α_adjusted = %.4f (for %d variants, K-1 comparisons)\n",
			a.BonferroniAlpha, len(a.Variants))
	}

	// Guardrails.
	if len(a.Guardrails) > 0 {
		parts := make([]string, 0, len(a.Guardrails))
		for _, g := range a.Guardrails {
			parts = append(parts, fmt.Sprintf("%s %s", g.Name, g.Threshold))
		}
		fmt.Fprintf(os.Stderr, "  Guardrails : %s\n", strings.Join(parts, "  •  "))
		fmt.Fprintln(os.Stderr, "               (pass/fail evaluation requires per-run outcome metric data)")
	}

	// Recommendation.
	fmt.Fprintln(os.Stderr)
	switch a.Recommendation {
	case "EXTEND":
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("  EXTEND — "+a.Rationale))
	case "READY_FOR_ANALYSIS":
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("  READY FOR ANALYSIS — "+a.Rationale))
	default:
		fmt.Fprintf(os.Stderr, "  %s — %s\n", a.Recommendation, a.Rationale)
	}
}
