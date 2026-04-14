//go:build !integration

package envutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_GetIntFromEnv validates the documented behavior of
// GetIntFromEnv as described in the package README.md.
//
// Specification:
// - Returns defaultValue when the environment variable is not set.
// - Returns defaultValue and emits a warning when the value cannot be parsed as an integer.
// - Returns defaultValue and emits a warning when the value is outside [minValue, maxValue].
// - Logs the accepted value at debug level when log is non-nil.
// - Pass nil for log to suppress debug output.

// TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenNotSet validates that
// defaultValue is returned when the environment variable is not set.
func TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenNotSet(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_UNSET"
	os.Unsetenv(envVar)
	defer os.Unsetenv(envVar)

	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 5, result,
		"GetIntFromEnv should return defaultValue when env var is not set")
}

// TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenNotInteger validates that
// defaultValue is returned when the value cannot be parsed as an integer.
func TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenNotInteger(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_NAN"
	os.Setenv(envVar, "not-a-number")
	defer os.Unsetenv(envVar)

	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 5, result,
		"GetIntFromEnv should return defaultValue when value cannot be parsed as integer")
}

// TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenBelowMin validates that
// defaultValue is returned when the value is below minValue.
func TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenBelowMin(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_BELOW_MIN"
	os.Setenv(envVar, "0")
	defer os.Unsetenv(envVar)

	// minValue is 1, so 0 is outside [1, 20]
	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 5, result,
		"GetIntFromEnv should return defaultValue when value is below minValue")
}

// TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenAboveMax validates that
// defaultValue is returned when the value is above maxValue.
func TestSpec_PublicAPI_GetIntFromEnv_ReturnsDefault_WhenAboveMax(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_ABOVE_MAX"
	os.Setenv(envVar, "21")
	defer os.Unsetenv(envVar)

	// maxValue is 20, so 21 is outside [1, 20]
	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 5, result,
		"GetIntFromEnv should return defaultValue when value is above maxValue")
}

// TestSpec_PublicAPI_GetIntFromEnv_ReturnsValue_WhenInRange validates that
// the parsed value is returned when it is within [minValue, maxValue].
func TestSpec_PublicAPI_GetIntFromEnv_ReturnsValue_WhenInRange(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_IN_RANGE"
	os.Setenv(envVar, "10")
	defer os.Unsetenv(envVar)

	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 10, result,
		"GetIntFromEnv should return the env var value when within [minValue, maxValue]")
}

// TestSpec_PublicAPI_GetIntFromEnv_AcceptsInclusiveMinBoundary validates that
// minValue itself is accepted (inclusive lower bound).
func TestSpec_PublicAPI_GetIntFromEnv_AcceptsInclusiveMinBoundary(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_MIN_BOUNDARY"
	os.Setenv(envVar, "1")
	defer os.Unsetenv(envVar)

	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 1, result,
		"GetIntFromEnv should accept value equal to minValue (inclusive lower bound)")
}

// TestSpec_PublicAPI_GetIntFromEnv_AcceptsInclusiveMaxBoundary validates that
// maxValue itself is accepted (inclusive upper bound).
func TestSpec_PublicAPI_GetIntFromEnv_AcceptsInclusiveMaxBoundary(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_MAX_BOUNDARY"
	os.Setenv(envVar, "20")
	defer os.Unsetenv(envVar)

	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 20, result,
		"GetIntFromEnv should accept value equal to maxValue (inclusive upper bound)")
}

// TestSpec_PublicAPI_GetIntFromEnv_AcceptsNilLogger validates that passing nil
// for the log parameter suppresses debug output without panicking.
//
// Specification: "Pass nil for log to suppress debug output."
func TestSpec_PublicAPI_GetIntFromEnv_AcceptsNilLogger(t *testing.T) {
	const envVar = "GH_AW_SPEC_TEST_NIL_LOGGER"
	os.Setenv(envVar, "10")
	defer os.Unsetenv(envVar)

	assert.NotPanics(t, func() {
		GetIntFromEnv(envVar, 5, 1, 20, nil)
	}, "GetIntFromEnv should not panic when nil logger is passed")
}

// TestSpec_PublicAPI_GetIntFromEnv_DocExample validates the usage example
// from the package README.md.
//
// Specification example:
//
//	concurrency := envutil.GetIntFromEnv("GH_AW_MAX_CONCURRENT_DOWNLOADS", 5, 1, 20, log)
func TestSpec_PublicAPI_GetIntFromEnv_DocExample(t *testing.T) {
	const envVar = "GH_AW_MAX_CONCURRENT_DOWNLOADS"
	saved := os.Getenv(envVar)
	defer func() {
		if saved != "" {
			os.Setenv(envVar, saved)
		} else {
			os.Unsetenv(envVar)
		}
	}()

	// When not set, should return default of 5
	os.Unsetenv(envVar)
	result := GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 5, result,
		"documented example: default 5 when env var not set")

	// When set to valid value, should return it
	os.Setenv(envVar, "8")
	result = GetIntFromEnv(envVar, 5, 1, 20, nil)
	assert.Equal(t, 8, result,
		"documented example: returns valid value within [1, 20]")
}
