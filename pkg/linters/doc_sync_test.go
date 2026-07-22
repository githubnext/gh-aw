//go:build !integration

package linters_test

import (
	"bufio"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/linters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	docBulletRe   = regexp.MustCompile(`^//\s+-\s+([a-z0-9-]+)\s+—`)
	readmeTableRe = regexp.MustCompile(`^\|\s+` + "`" + `([a-z0-9-]+)` + "`" + `\s+\|`)
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

func TestDocGo_AnalyzersMatchREADME(t *testing.T) {
	docSet := parseDocBulletSet(t)
	readmeSet := parseReadmeSubpackageSet(t)
	assert.Equal(t, sortedKeys(docSet), sortedKeys(readmeSet),
		"doc.go analyzer bullets and README Subpackages table must list the same analyzers; update both files together")
}

func TestDocSurfacesMatchRegistryAndSpecList(t *testing.T) {
	registrySet := make(map[string]struct{})
	for _, analyzer := range linters.All() {
		registrySet[analyzer.Name] = struct{}{}
	}

	docAsAnalyzerNames := normalizedDocSlugSetToAnalyzerNames(parseDocBulletSet(t))
	readmeSlugs := parseReadmeSubpackageSet(t)
	readmeAsAnalyzerNames := normalizedDocSlugSetToAnalyzerNames(readmeSlugs)

	assert.Equal(t, sortedKeys(registrySet), sortedKeys(docAsAnalyzerNames),
		"doc.go analyzer bullets must match linters.All(); update pkg/linters/doc.go when adding/removing linters")
	assert.Equal(t, sortedKeys(registrySet), sortedKeys(readmeAsAnalyzerNames),
		"README Subpackages analyzer entries must match linters.All(); update pkg/linters/README.md when adding/removing linters")

	documentedLabels := make(map[string]struct{})
	for _, d := range documentedAnalyzers() {
		documentedLabels[d.label] = struct{}{}
	}
	assert.Equal(t, sortedKeys(readmeSlugs), sortedKeys(documentedLabels),
		"documentedAnalyzers() labels in spec_test.go must match README Subpackages table labels exactly")
}

func parseDocBulletSet(t *testing.T) map[string]struct{} {
	t.Helper()

	docBytes, err := os.ReadFile("doc.go")
	require.NoError(t, err, "doc.go must be present in pkg/linters")

	docSet := make(map[string]struct{})
	for line := range strings.SplitSeq(string(docBytes), "\n") {
		if m := docBulletRe.FindStringSubmatch(line); m != nil {
			docSet[m[1]] = struct{}{}
		}
	}

	return docSet
}

func parseReadmeSubpackageSet(t *testing.T) map[string]struct{} {
	t.Helper()

	readmeBytes, err := os.ReadFile("README.md")
	require.NoError(t, err, "README.md must be present in pkg/linters")

	readmeSet := make(map[string]struct{})
	for line := range strings.SplitSeq(string(readmeBytes), "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		m := readmeTableRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if m[1] == "internal" {
			continue
		}
		readmeSet[m[1]] = struct{}{}
	}

	return readmeSet
}

func normalizedDocSlugSetToAnalyzerNames(slugSet map[string]struct{}) map[string]struct{} {
	normalized := make(map[string]struct{}, len(slugSet))
	for slug := range slugSet {
		normalized[docSlugToAnalyzerName(slug)] = struct{}{}
	}
	return normalized
}

func docSlugToAnalyzerName(slug string) string {
	switch slug {
	case "panic-in-library-code":
		return "panicinlibrarycode"
	default:
		return slug
	}
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
