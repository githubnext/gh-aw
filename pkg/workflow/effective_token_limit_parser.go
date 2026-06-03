package workflow

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var effectiveTokenLimitLog = logger.New("workflow:effective_token_limit_parser")

// normalizePositiveEffectiveTokenLimit converts positive integer-like values
// into a canonical base-10 string.
//
// Supported inputs:
//   - positive integers
//   - positive numeric strings with optional K/M suffixes
//
// K/M suffix strings are expanded to plain base-10 (for example, "100M"
// becomes "100000000").
//
// It returns the normalized base-10 value and true when parsing succeeds.
// It returns an empty string and false when the value is not a valid positive
// effective-token limit.
func normalizePositiveEffectiveTokenLimit(raw any) (string, bool) {
	if val, ok := typeutil.ParseIntValue(raw); ok && val > 0 {
		return strconv.Itoa(val), true
	}

	rawStr, ok := raw.(string)
	if !ok {
		effectiveTokenLimitLog.Printf("Rejecting effective-token limit: unsupported type %T", raw)
		return "", false
	}

	trimmed := strings.TrimSpace(rawStr)
	if trimmed == "" {
		return "", false
	}

	normalized, ok := typeutil.NormalizeInt64KMSuffix(trimmed)
	if !ok {
		effectiveTokenLimitLog.Printf("Rejecting effective-token limit: %q is not a valid positive value", trimmed)
		return "", false
	}
	effectiveTokenLimitLog.Printf("Normalized effective-token limit %q to %s", trimmed, normalized)
	return normalized, true
}

// parseMaxEffectiveTokenLimitValue parses max-effective-tokens from either an
// integer, -1 string sentinel, or positive K/M-suffixed string.
//
// It returns the parsed limit value and a success boolean. A false success
// value means the input was not a supported max-effective-tokens value.
func parseMaxEffectiveTokenLimitValue(raw any) (int64, bool) {
	if val, ok := typeutil.ParseIntValue(raw); ok && val != 0 {
		return int64(val), true
	}

	rawStr, ok := raw.(string)
	if !ok {
		return 0, false
	}

	trimmed := strings.TrimSpace(rawStr)
	if trimmed == "-1" {
		effectiveTokenLimitLog.Print("Parsed max-effective-tokens sentinel -1 (unlimited)")
		return -1, true
	}

	parsed, ok := typeutil.ParseInt64KMSuffix(trimmed)
	if !ok {
		effectiveTokenLimitLog.Printf("Rejecting max-effective-tokens: %q is not a supported value", trimmed)
		return 0, false
	}
	return parsed, true
}
