package stringutil

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var versionLog = logger.New("stringutil:version")

// ParseVersionValue converts version values of various types to strings.
// Supports string, int, int64, uint64, and float64 types.
// Returns empty string for unsupported types.
func ParseVersionValue(version any) string {
	switch v := version.(type) {
	case string:
		return v
	case int, int64, uint64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		versionLog.Printf("ParseVersionValue: unsupported type %T, returning empty string", version)
		return ""
	}
}
