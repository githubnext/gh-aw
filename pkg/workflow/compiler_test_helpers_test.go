package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func containsInNonCommentLines(content, search string) bool {
	lines := strings.SplitSeq(content, "\n")
	for line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		// Skip comment lines
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(line, search) {
			return true
		}
	}
	return false
}

// indexInNonCommentLines returns the index (relative to the original content) of the first
// occurrence of search that appears in a non-comment line. This is used for order comparisons
// where we need to verify step ordering while ignoring matches in comment lines (such as
// frontmatter embedded as comments). Returns -1 if not found.
func indexInNonCommentLines(content, search string) int {
	lines := strings.Split(content, "\n")
	offset := 0
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		// Skip comment lines
		if strings.HasPrefix(trimmed, "#") {
			offset += len(line) + 1 // +1 for newline
			continue
		}
		if idx := strings.Index(line, search); idx != -1 {
			return offset + idx
		}
		offset += len(line) + 1 // +1 for newline
	}
	return -1
}

func extractJobSection(yamlContent, jobName string) string {
	lines := strings.Split(yamlContent, "\n")
	var jobLines []string
	inJob := false
	jobPrefix := "  " + jobName + ":"

	for i, line := range lines {
		if strings.HasPrefix(line, jobPrefix) {
			inJob = true
			jobLines = append(jobLines, line)
			continue
		}

		if inJob {
			// If we hit another job at the same level (starts with "  " and ends with ":"), stop
			if strings.HasPrefix(line, "  ") && strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "    ") {
				break
			}
			// If we hit the end of jobs section, stop
			if strings.HasPrefix(line, "jobs:") && i > 0 {
				break
			}
			jobLines = append(jobLines, line)
		}
	}

	return strings.Join(jobLines, "\n")
}

func extractStepBlock(t *testing.T, jobSection, stepName, jobName string) string {
	t.Helper()

	lines := strings.Split(jobSection, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if !strings.HasPrefix(trimmed, "- name: "+stepName) {
			continue
		}

		stepIndent := len(line) - len(trimmed)
		var b strings.Builder
		for j := i; j < len(lines); j++ {
			cur := lines[j]
			curTrimmed := strings.TrimLeft(cur, " ")
			curIndent := len(cur) - len(curTrimmed)
			if j > i && curTrimmed != "" {
				if curIndent < stepIndent {
					break
				}
				if curIndent == stepIndent && strings.HasPrefix(curTrimmed, "- ") {
					break
				}
			}
			b.WriteString(cur + "\n")
		}
		return b.String()
	}

	require.FailNow(t, jobName+" should contain step", stepName)
	return ""
}

// TestExtractJobSection exercises the job-boundary parsing helper used throughout the
// test suite to isolate a single job's YAML from a full compiled lock file.
func TestExtractJobSection(t *testing.T) {
	t.Run("job at end of file", func(t *testing.T) {
		yamlContent := "jobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo hi\n"
		got := extractJobSection(yamlContent, "build")
		assert.Equal(t, "  build:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo hi\n", got)
	})

	t.Run("job followed by another job", func(t *testing.T) {
		yamlContent := "jobs:\n  activation:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo activation\n  agent:\n    runs-on: ubuntu-latest\n"
		got := extractJobSection(yamlContent, "activation")
		assert.Equal(t, "  activation:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo activation", got)
		assert.NotContains(t, got, "agent:")
	})

	t.Run("job with nested multi-level indentation", func(t *testing.T) {
		yamlContent := "jobs:\n  activation:\n    steps:\n      - name: Check\n        with:\n          nested:\n            deeply: true\n  agent:\n    runs-on: ubuntu-latest\n"
		got := extractJobSection(yamlContent, "activation")
		assert.Contains(t, got, "deeply: true")
		assert.NotContains(t, got, "agent:")
	})

	t.Run("job not present returns empty string", func(t *testing.T) {
		yamlContent := "jobs:\n  build:\n    runs-on: ubuntu-latest\n"
		got := extractJobSection(yamlContent, "missing")
		assert.Empty(t, got)
	})
}
