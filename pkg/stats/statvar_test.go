//go:build !integration

package stats

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatVar_Empty(t *testing.T) {
	var s StatVar
	assert.Equal(t, 0, s.Count(), "empty: count should be 0")
	assert.InDelta(t, 0.0, s.Min(), 1e-9, "empty: min should be 0")
	assert.InDelta(t, 0.0, s.Max(), 1e-9, "empty: max should be 0")
	assert.InDelta(t, 0.0, s.Mean(), 1e-9, "empty: mean should be 0")
	assert.InDelta(t, 0.0, s.SampleVariance(), 1e-9, "empty: sample variance should be 0")
	assert.InDelta(t, 0.0, s.SampleStdDev(), 1e-9, "empty: sample stddev should be 0")
	assert.InDelta(t, 0.0, s.Median(), 1e-9, "empty: median should be 0")
}

func TestStatVar_SingleObservation(t *testing.T) {
	var s StatVar
	s.Add(42.0)

	assert.Equal(t, 1, s.Count(), "single: count should be 1")
	assert.InDelta(t, 42.0, s.Min(), 1e-9, "single: min should be 42")
	assert.InDelta(t, 42.0, s.Max(), 1e-9, "single: max should be 42")
	assert.InDelta(t, 42.0, s.Mean(), 1e-9, "single: mean should be 42")
	assert.InDelta(t, 0.0, s.SampleVariance(), 1e-9, "single: sample variance should be 0 (needs >=2)")
	assert.InDelta(t, 42.0, s.Median(), 1e-9, "single: median should be 42")
}

func TestStatVar_TwoObservations(t *testing.T) {
	var s StatVar
	s.Add(10.0)
	s.Add(20.0)

	assert.Equal(t, 2, s.Count(), "two: count should be 2")
	assert.InDelta(t, 10.0, s.Min(), 1e-9, "two: min should be 10")
	assert.InDelta(t, 20.0, s.Max(), 1e-9, "two: max should be 20")
	assert.InDelta(t, 15.0, s.Mean(), 1e-9, "two: mean should be 15")
	// Sample variance: 50 / (2-1) = 50
	assert.InDelta(t, 50.0, s.SampleVariance(), 1e-9, "two: sample variance should be 50")
	assert.InDelta(t, math.Sqrt(50), s.SampleStdDev(), 1e-9, "two: sample stddev should be sqrt(50)")
	// Median of [10, 20] → (10+20)/2 = 15
	assert.InDelta(t, 15.0, s.Median(), 1e-9, "two: median should be 15")
}

func TestStatVar_OddCount(t *testing.T) {
	var s StatVar
	// [1, 3, 5] — odd number of values
	for _, v := range []float64{5, 1, 3} {
		s.Add(v)
	}

	assert.Equal(t, 3, s.Count(), "odd: count should be 3")
	assert.InDelta(t, 1.0, s.Min(), 1e-9, "odd: min should be 1")
	assert.InDelta(t, 5.0, s.Max(), 1e-9, "odd: max should be 5")
	assert.InDelta(t, 3.0, s.Mean(), 1e-9, "odd: mean should be 3")
	assert.InDelta(t, 3.0, s.Median(), 1e-9, "odd: median of [1,3,5] should be 3")
}

func TestStatVar_EvenCount(t *testing.T) {
	var s StatVar
	// [2, 4, 6, 8] — even number of values
	for _, v := range []float64{8, 2, 6, 4} {
		s.Add(v)
	}

	assert.Equal(t, 4, s.Count(), "even: count should be 4")
	assert.InDelta(t, 2.0, s.Min(), 1e-9, "even: min should be 2")
	assert.InDelta(t, 8.0, s.Max(), 1e-9, "even: max should be 8")
	assert.InDelta(t, 5.0, s.Mean(), 1e-9, "even: mean should be 5")
	// Median of [2,4,6,8] → (4+6)/2 = 5
	assert.InDelta(t, 5.0, s.Median(), 1e-9, "even: median of [2,4,6,8] should be 5")
}

func TestStatVar_KnownVariance(t *testing.T) {
	// Dataset: [2, 4, 4, 4, 5, 5, 7, 9]
	// N=8, mean=5, population variance=4, population stddev=2
	// Sample variance = 32/7, sample stddev = sqrt(32/7)
	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	var s StatVar
	for _, v := range values {
		s.Add(v)
	}

	require.Equal(t, 8, s.Count(), "known: count should be 8")
	assert.InDelta(t, 5.0, s.Mean(), 1e-9, "known: mean should be 5")
	assert.InDelta(t, 32.0/7.0, s.SampleVariance(), 1e-9, "known: sample variance should be 32/7")
	assert.InDelta(t, math.Sqrt(32.0/7.0), s.SampleStdDev(), 1e-9, "known: sample stddev should be sqrt(32/7)")
	// Median of [2,4,4,4,5,5,7,9] → (4+5)/2 = 4.5
	assert.InDelta(t, 4.5, s.Median(), 1e-9, "known: median should be 4.5")
}

func TestStatVar_NumericalStability(t *testing.T) {
	// Test with large values close together (catastrophic cancellation case).
	// Mean = 1e9 + 0.5, stddev should be ~0.5
	var s StatVar
	for range 1000 {
		v := 1e9 + float64(s.Count()%2) // alternates between 1e9 and 1e9+1
		s.Add(v)
	}
	// 500 observations at 1e9 and 500 at 1e9+1 → mean = 1e9 + 0.5
	assert.InDelta(t, 1e9+0.5, s.Mean(), 1e-6, "stability: mean should be ~1e9+0.5")
}

func TestStatVar_MedianDoesNotMutateState(t *testing.T) {
	var s StatVar
	s.Add(3)
	s.Add(1)
	s.Add(2)

	median1 := s.Median()
	median2 := s.Median()
	assert.InDelta(t, median1, median2, 1e-9, "repeated Median calls must return the same result")
	assert.InDelta(t, 2.0, s.Median(), 1e-9, "median of [3,1,2] should be 2")
	// Ensure the internal slice is not permanently sorted (add a value and re-check)
	s.Add(10)
	assert.Equal(t, 4, s.Count(), "count should grow after post-median Add")
	// Median of [1,2,3,10] → (2+3)/2 = 2.5
	assert.InDelta(t, 2.5, s.Median(), 1e-9, "median after additional Add should be 2.5")
}

func TestStatVar_AllIdentical(t *testing.T) {
	var s StatVar
	for range 5 {
		s.Add(7.0)
	}
	assert.InDelta(t, 7.0, s.Mean(), 1e-9, "identical: mean should be 7")
	assert.InDelta(t, 7.0, s.Median(), 1e-9, "identical: median should be 7")
}
