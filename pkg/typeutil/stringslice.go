package typeutil

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var stringSliceLog = logger.New("typeutil:stringslice")

// NormalizeStringSlice coerces a value that YAML/JSON decoding may produce as a
// scalar string, a []string, or a []any into a []string.
//
// The coercion rules are:
//   - string: returned as a single-element slice (GitHub Actions commonly allows a
//     scalar as shorthand for a one-element list, e.g. `needs: build`)
//   - []string: returned as a copy so callers can mutate the result safely
//   - []any: string elements are kept in order; non-string elements are skipped
//   - anything else (including nil): nil
//
// An empty []any yields an empty, non-nil slice so callers can distinguish
// "present but empty" from "absent". Callers that need trimming, empty-string
// filtering, deduplication, or sorting should apply it to the returned slice.
func NormalizeStringSlice(v any) []string {
	switch value := v.(type) {
	case string:
		return []string{value}
	case []string:
		return slices.Clone(value)
	case []any:
		result := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		if v != nil {
			stringSliceLog.Printf("NormalizeStringSlice: unexpected type %T, returning nil", v)
		}
		return nil
	}
}
