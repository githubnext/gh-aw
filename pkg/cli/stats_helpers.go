package cli

import (
	"math"
	"slices"
)

// safePercent returns percentage of part/total, returning 0 when total is 0.
func safePercent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

// medianFloat returns the median of a float slice.
func medianFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := slices.Clone(vals)
	slices.Sort(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// meanStdDevInt computes the arithmetic mean and population standard deviation
// of the int slice xs (assumed non-empty).
//
// The mean is returned as an int (truncated toward zero after integer division),
// which is used for the milli-AIC intermediate representation.
// The standard deviation uses the full floating-point mean to avoid accumulating
// rounding error in the variance calculation.
func meanStdDevInt(xs []int) (mean int, stddev float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum int
	for _, x := range xs {
		sum += x
	}
	mean = sum / len(xs)
	// Use the exact float mean for stddev to avoid bias from integer truncation.
	fmean := float64(sum) / float64(len(xs))
	for _, x := range xs {
		d := float64(x) - fmean
		stddev += d * d
	}
	stddev = math.Sqrt(stddev / float64(len(xs)))
	return
}

// percentileInt returns the p-th percentile of an already-sorted int slice
// using the nearest-rank method.  p must be in [1, 100].
func percentileInt(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100*float64(len(sorted)))) - 1
	idx = max(idx, 0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
