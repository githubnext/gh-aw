package cli

import "math"

// Statistics describe the returned failure sample, not fleet-wide incidence.
// Wilson bounds assume independent binomial observations; retries can violate that assumption.
type FailureClusterStatistics struct {
	FailedPopulation int     `json:"failed_population"`
	Share            float64 `json:"share"`
	WilsonLower95    float64 `json:"wilson_lower_95"`
	WilsonUpper95    float64 `json:"wilson_upper_95"`
}

func annotateFailureClusterStatistics(runs []RunData, clusters []FailureCluster) {
	populations := map[bool]map[string]struct{}{
		false: {}, true: {},
	}
	for _, run := range runs {
		if isClusterFailureConclusion(run.Conclusion) && !isObsoletePRRun(run) {
			populations[run.IntentionalFailure][failureRunIdentityFor(run).key()] = struct{}{}
		}
	}
	for i := range clusters {
		cluster := &clusters[i]
		n := len(populations[cluster.IntentionalFailure])
		lower, upper := failureShareWilson95(cluster.Count, n)
		cluster.Statistics = FailureClusterStatistics{
			FailedPopulation: n, Share: float64(cluster.Count) / float64(n),
			WilsonLower95: lower, WilsonUpper95: upper,
		}
	}
}

func failureShareWilson95(matches, population int) (float64, float64) {
	if population == 0 {
		return 0, 0
	}
	const z = 1.959963984540054
	n := float64(population)
	p := float64(matches) / n
	denominator := 1 + z*z/n
	center := (p + z*z/(2*n)) / denominator
	half := z * math.Sqrt(p*(1-p)/n+z*z/(4*n*n)) / denominator
	return math.Max(0, center-half), math.Min(1, center+half)
}
