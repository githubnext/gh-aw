//go:build !integration

package stats

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_EmptyBehavior validates that an empty StatVar returns
// documented zero values as described in the package README.md.
//
// Specification:
//   - Min: "Returns the minimum observed value (or 0 if empty)"
//   - Max: "Returns the maximum observed value (or 0 if empty)"
//   - Mean: "Returns the arithmetic mean (or 0 if empty)"
func TestSpec_PublicAPI_EmptyBehavior(t *testing.T) {
	var sv StatVar

	assert.Equal(t, 0, sv.Count(), "Count should return 0 for an empty StatVar")
	assert.Equal(t, 0.0, sv.Sum(), "Sum should return 0 for an empty StatVar")
	assert.Equal(t, 0.0, sv.Min(), "Min should return 0 when no observations have been added")
	assert.Equal(t, 0.0, sv.Max(), "Max should return 0 when no observations have been added")
	assert.Equal(t, 0.0, sv.Mean(), "Mean should return 0 when no observations have been added")
	assert.Equal(t, 0.0, sv.Median(), "Median should return 0 when no observations have been added")
}

// TestSpec_PublicAPI_Add validates that each Add call records exactly one
// observation as described in the package README.md.
//
// Specification: "Add: Adds one observation"
func TestSpec_PublicAPI_Add(t *testing.T) {
	var sv StatVar

	sv.Add(10.0)
	assert.Equal(t, 1, sv.Count(), "Count should be 1 after one Add call")

	sv.Add(20.0)
	assert.Equal(t, 2, sv.Count(), "Count should be 2 after two Add calls")

	sv.Add(30.0)
	assert.Equal(t, 3, sv.Count(), "Count should be 3 after three Add calls")
}

// TestSpec_PublicAPI_Sum validates that Sum returns the arithmetic sum of all
// observations as described in the package README.md.
//
// Specification: "Sum: Returns the arithmetic sum"
func TestSpec_PublicAPI_Sum(t *testing.T) {
	var sv StatVar
	sv.Add(1.0)
	sv.Add(2.0)
	sv.Add(3.0)

	assert.Equal(t, 6.0, sv.Sum(),
		"Sum should equal the arithmetic sum of all added observations")
}

// TestSpec_PublicAPI_MinMax validates that Min and Max track the extreme
// observed values as described in the package README.md.
//
// Specification:
//   - Min: "Returns the minimum observed value"
//   - Max: "Returns the maximum observed value"
func TestSpec_PublicAPI_MinMax(t *testing.T) {
	var sv StatVar
	sv.Add(5.0)
	sv.Add(1.0)
	sv.Add(9.0)
	sv.Add(3.0)

	assert.Equal(t, 1.0, sv.Min(),
		"Min should return the smallest value ever passed to Add")
	assert.Equal(t, 9.0, sv.Max(),
		"Max should return the largest value ever passed to Add")
}

// TestSpec_PublicAPI_Mean validates that Mean returns the arithmetic mean
// as described in the package README.md.
//
// Specification: "Mean: Returns the arithmetic mean"
func TestSpec_PublicAPI_Mean(t *testing.T) {
	var sv StatVar
	sv.Add(2.0)
	sv.Add(4.0)
	sv.Add(6.0)

	assert.InDelta(t, 4.0, sv.Mean(), 1e-9,
		"Mean should equal (sum of observations) / count")
}

// TestSpec_PublicAPI_Variance_PopulationFormula validates that Variance uses
// the population formula with N as the denominator as described in the README.md.
//
// Specification: "Variance: Returns population variance (N denominator)"
//
// Example: values [2,4,4,4,5,5,7,9] (N=8, mean=5) → population variance = 32/8 = 4
func TestSpec_PublicAPI_Variance_PopulationFormula(t *testing.T) {
	var sv StatVar
	for _, v := range []float64{2, 4, 4, 4, 5, 5, 7, 9} {
		sv.Add(v)
	}

	assert.InDelta(t, 4.0, sv.Variance(), 1e-9,
		"Variance should use N (population) denominator: sum_sq_dev / N")
}

// TestSpec_PublicAPI_SampleVariance_SampleFormula validates that SampleVariance
// uses the sample formula with N-1 as the denominator as described in the README.md.
//
// Specification: "SampleVariance: Returns sample variance (N-1 denominator)"
//
// Example: values [2,4,4,4,5,5,7,9] (N=8, mean=5, sum_sq_dev=32) → sample variance = 32/7
func TestSpec_PublicAPI_SampleVariance_SampleFormula(t *testing.T) {
	var sv StatVar
	for _, v := range []float64{2, 4, 4, 4, 5, 5, 7, 9} {
		sv.Add(v)
	}

	expected := 32.0 / 7.0
	assert.InDelta(t, expected, sv.SampleVariance(), 1e-9,
		"SampleVariance should use N-1 (sample) denominator: sum_sq_dev / (N-1)")
}

// TestSpec_PublicAPI_StdDev validates that StdDev and SampleStdDev equal the
// square root of their respective variances as described in the README.md.
//
// Specification:
//   - StdDev: "Returns population standard deviation"
//   - SampleStdDev: "Returns sample standard deviation"
func TestSpec_PublicAPI_StdDev(t *testing.T) {
	var sv StatVar
	for _, v := range []float64{2, 4, 4, 4, 5, 5, 7, 9} {
		sv.Add(v)
	}

	assert.InDelta(t, math.Sqrt(sv.Variance()), sv.StdDev(), 1e-9,
		"StdDev should equal sqrt(Variance)")
	assert.InDelta(t, math.Sqrt(sv.SampleVariance()), sv.SampleStdDev(), 1e-9,
		"SampleStdDev should equal sqrt(SampleVariance)")
}

// TestSpec_PublicAPI_Median_OddCount validates that Median returns the middle
// value for an odd-count set as described in the README.md.
//
// Specification: "Median: Returns the exact median (middle value...)"
func TestSpec_PublicAPI_Median_OddCount(t *testing.T) {
	var sv StatVar
	sv.Add(3.0)
	sv.Add(1.0)
	sv.Add(2.0)

	assert.Equal(t, 2.0, sv.Median(),
		"Median with odd count should return the middle value in sorted order")
}

// TestSpec_PublicAPI_Median_EvenCount validates that Median returns the midpoint
// of the two middle values for an even-count set as described in the README.md.
//
// Specification: "Median: Returns the exact median (...or midpoint of two middle values)"
func TestSpec_PublicAPI_Median_EvenCount(t *testing.T) {
	var sv StatVar
	sv.Add(4.0)
	sv.Add(1.0)
	sv.Add(3.0)
	sv.Add(2.0)

	assert.Equal(t, 2.5, sv.Median(),
		"Median with even count should return the average of the two middle sorted values")
}
