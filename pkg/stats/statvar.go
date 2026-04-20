// Package stats provides numerical statistics utilities for metric collection.
package stats

import (
	"math"
	"sort"
)

// StatVar accumulates a stream of float64 observations and computes descriptive
// statistics: count, sum, min, max, mean, variance, standard deviation, and
// median.  Mean and variance are maintained incrementally using Welford's online
// algorithm, which avoids catastrophic cancellation when values are large or
// tightly clustered.  All observed values are also stored so that an exact
// median can be returned on demand.
//
// Memory usage: all observations are stored to enable exact median computation.
// This is suitable for the expected scale of agentic workflow metrics (typically
// tens to a few hundred observations per session).  For very large streams
// (thousands of observations), consider a streaming approximation instead.
type StatVar struct {
	count  int
	mean   float64 // running mean (Welford)
	m2     float64 // running sum of squared deviations from the mean (Welford)
	sum    float64
	min    float64
	max    float64
	values []float64 // stored for median computation
}

// Add records a new observation.  Finite values are recommended; NaN and ±Inf
// will propagate through sum, mean, and variance, but Min and Max may not update
// correctly for NaN inputs because IEEE 754 comparisons with NaN always return
// false.  In practice, observations come from time.Duration, token counts, and
// cost values, which are always finite.
func (s *StatVar) Add(v float64) {
	if s.count == 0 {
		s.min = v
		s.max = v
	} else {
		if v < s.min {
			s.min = v
		}
		if v > s.max {
			s.max = v
		}
	}
	s.count++
	s.sum += v
	s.values = append(s.values, v)
	// Welford's one-pass algorithm: numerically stable incremental mean and M2.
	delta := v - s.mean
	s.mean += delta / float64(s.count)
	delta2 := v - s.mean
	s.m2 += delta * delta2
}

// Count returns the number of observations added so far.
func (s *StatVar) Count() int { return s.count }

// Min returns the smallest observed value, or 0 if no observations have been added.
func (s *StatVar) Min() float64 {
	if s.count == 0 {
		return 0
	}
	return s.min
}

// Max returns the largest observed value, or 0 if no observations have been added.
func (s *StatVar) Max() float64 {
	if s.count == 0 {
		return 0
	}
	return s.max
}

// Mean returns the arithmetic mean of all observations, or 0 if none.
func (s *StatVar) Mean() float64 {
	if s.count == 0 {
		return 0
	}
	return s.mean
}

// SampleVariance returns the sample variance with Bessel's correction (divides
// by N−1), giving an unbiased estimator when the observations are a random
// sample from a larger population.  Returns 0 when fewer than two observations
// are present.
func (s *StatVar) SampleVariance() float64 {
	if s.count < 2 {
		return 0
	}
	return s.m2 / float64(s.count-1)
}

// SampleStdDev returns the sample standard deviation with Bessel's correction
// (sqrt of SampleVariance).
func (s *StatVar) SampleStdDev() float64 {
	return math.Sqrt(s.SampleVariance())
}

// Median returns the median of all observations.
// For an even number of observations the two middle values are averaged.
// Returns 0 if no observations have been added.
// The internal values slice is copied and sorted; the receiver is not modified.
func (s *StatVar) Median() float64 {
	if s.count == 0 {
		return 0
	}
	sorted := make([]float64, len(s.values))
	copy(sorted, s.values)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
