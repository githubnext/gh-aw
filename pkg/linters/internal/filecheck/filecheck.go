package filecheck

import (
	"path/filepath"
	"strings"
)

// IsTestFile reports whether filename is a Go test file path.
func IsTestFile(filename string) bool {
	return strings.HasSuffix(filepath.Base(filename), "_test.go")
}
