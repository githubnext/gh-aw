//go:build !integration

package workflow

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAllScriptFilenamesReturnsSortedResults(t *testing.T) {
	filenames := GetAllScriptFilenames()
	assert.True(t, sort.StringsAreSorted(filenames), "GetAllScriptFilenames should return sorted filenames")
}

func TestGetAllScriptFilenamesReturnsOnlyCJSFiles(t *testing.T) {
	filenames := GetAllScriptFilenames()
	for _, filename := range filenames {
		assert.Truef(t, strings.HasSuffix(filename, ".cjs"), "GetAllScriptFilenames should only return .cjs files, got %q", filename)
	}
}
