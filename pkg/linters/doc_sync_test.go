//go:build !integration

package linters_test

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocGo_CountMatchesBullets validates that the "All N active analyzers:"
// header count in doc.go matches the actual number of bullet entries.
// This prevents the header from silently drifting from the bullet list
// (as seen in gh-aw#40436, gh-aw#45185, gh-aw#46131).
func TestDocGo_CountMatchesBullets(t *testing.T) {
	f, err := os.Open("doc.go")
	require.NoError(t, err, "doc.go must be present in pkg/linters")
	defer f.Close() //nolint:errcheck

	headerRe := regexp.MustCompile(`// All (\d+) active analyzers:`)
	var headerCount int
	var bulletCount int
	var foundHeader bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if m := headerRe.FindStringSubmatch(line); m != nil {
			n, parseErr := strconv.Atoi(m[1])
			require.NoError(t, parseErr)
			headerCount = n
			foundHeader = true
		}
		if strings.HasPrefix(line, "//   - ") {
			bulletCount++
		}
	}
	require.NoError(t, scanner.Err())
	require.True(t, foundHeader, "doc.go must contain an '// All N active analyzers:' header")

	assert.Equal(t, headerCount, bulletCount,
		"doc.go header says %d analyzers but %d bullet entries were found; "+
			"update the header or add/remove the missing bullets",
		headerCount, bulletCount)
}
