// Package envutil provides utilities for reading and validating environment variables.
package envutil

import (
	"fmt"
	"os"
	"strconv"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

// warnf is an internal helper shared by GetIntFromEnv and GetBoolFromEnv. It
// formats an environment parsing warning and routes it through the provided
// logger when available, or to stderr using the standard console warning
// format otherwise. Callers pass a format string and args directly, avoiding
// a separate fmt.Sprintf call at each call site. Formatting is deferred to the
// selected output path so the message is only built once.
func warnf(debugLog *logger.Logger, format string, args ...any) {
	if debugLog != nil {
		debugLog.Printf("WARNING: "+format, args...)
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf(format, args...)))
}

// debugf is an internal helper shared by GetIntFromEnv, GetBoolFromEnv, and
// GetStringFromEnv. It routes a formatted debug message through the provided
// logger, doing nothing when debugLog is nil. This centralizes the repeated
// "if debugLog != nil { debugLog.Printf(...) }" guard used by each getter.
func debugf(debugLog *logger.Logger, format string, args ...any) {
	if debugLog != nil {
		debugLog.Printf(format, args...)
	}
}

// GetIntFromEnv is a generic helper that reads an integer value from an environment variable,
// validates it against min/max bounds, and returns a default value if invalid.
// This follows the configuration helper pattern from pkg/workflow/config_helpers.go.
//
// Parameters:
//   - envVar: The environment variable name (e.g., "GH_AW_MAX_CONCURRENT_DOWNLOADS")
//   - defaultValue: The default value to return if env var is not set or invalid
//   - minValue: Minimum allowed value (inclusive)
//   - maxValue: Maximum allowed value (inclusive)
//   - debugLog: Optional logger for debug output
//
// Returns the parsed integer value, or defaultValue if:
//   - Environment variable is not set
//   - Value cannot be parsed as an integer
//   - Value is outside the [minValue, maxValue] range
//
// Invalid values trigger warning messages to stderr, or through the logger if provided.
func GetIntFromEnv(envVar string, defaultValue, minValue, maxValue int, debugLog *logger.Logger) int {
	envValue := os.Getenv(envVar) //nolint:osgetenvlibrary
	if envValue == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(envValue)
	if err != nil {
		warnf(debugLog, "Invalid %s value '%s' (must be a number), using default %d", envVar, envValue, defaultValue)
		return defaultValue
	}

	if val < minValue || val > maxValue {
		warnf(debugLog, "%s value %d is out of bounds (must be %d-%d), using default %d", envVar, val, minValue, maxValue, defaultValue)
		return defaultValue
	}

	debugf(debugLog, "Using %s=%d", envVar, val)
	return val
}

// GetBoolFromEnv reads a boolean value from an environment variable and returns
// defaultValue if the variable is not set or cannot be parsed as a boolean.
//
// Parameters:
//   - envVar: The environment variable name (e.g., "CI")
//   - defaultValue: The default value to return if env var is not set or invalid
//   - debugLog: Optional logger for debug output
//
// Returns the parsed boolean value, or defaultValue if:
//   - Environment variable is not set
//   - Value cannot be parsed as a boolean
//
// Invalid values trigger warning messages to stderr, or through the logger if provided.
func GetBoolFromEnv(envVar string, defaultValue bool, debugLog *logger.Logger) bool {
	envValue := os.Getenv(envVar) //nolint:osgetenvlibrary
	if envValue == "" {
		return defaultValue
	}

	val, err := strconv.ParseBool(envValue)
	if err != nil {
		warnf(debugLog, "Invalid %s value '%s' (must be a boolean), using default %t", envVar, envValue, defaultValue)
		return defaultValue
	}

	debugf(debugLog, "Using %s=%t", envVar, val)
	return val
}

// GetStringFromEnv reads a string value from an environment variable and
// returns defaultValue if the variable is not set.
//
// Parameters:
//   - envVar: The environment variable name (e.g., "GITHUB_TOKEN")
//   - defaultValue: The default value to return if env var is not set
//   - debugLog: Optional logger for debug output
//
// Returns the environment variable value, or defaultValue when the variable is
// not set. Empty string values are treated the same as unset variables.
func GetStringFromEnv(envVar, defaultValue string, debugLog *logger.Logger) string {
	envValue := os.Getenv(envVar) //nolint:osgetenvlibrary
	if envValue == "" {
		return defaultValue
	}

	debugf(debugLog, "Using %s from environment", envVar)
	return envValue
}
