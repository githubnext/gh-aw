package workflow

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/typeutil"
)

// normalizePositiveEffectiveTokenLimit converts positive integer-like values into
// a canonical base-10 string. Supported inputs are positive integers and numeric
// strings with optional K/M suffixes.
func normalizePositiveEffectiveTokenLimit(raw any) (string, bool) {
	if val, ok := typeutil.ParseIntValue(raw); ok && val > 0 {
		return strconv.Itoa(val), true
	}

	rawStr, ok := raw.(string)
	if !ok {
		return "", false
	}

	trimmed := strings.TrimSpace(rawStr)
	if trimmed == "" {
		return "", false
	}

	normalized, ok := typeutil.NormalizeInt64KMSuffix(trimmed)
	if !ok {
		return "", false
	}
	return normalized, true
}

// parseMaxEffectiveTokenLimitValue parses max-effective-tokens from either an
// integer, -1 string sentinel, or positive K/M-suffixed string.
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
		return -1, true
	}

	parsed, ok := typeutil.ParseInt64KMSuffix(trimmed)
	if !ok {
		return 0, false
	}
	return parsed, true
}
