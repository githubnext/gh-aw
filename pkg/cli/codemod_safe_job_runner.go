package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeJobRunnerCodemodLog = logger.New("cli:codemod_safe_job_runner")

func getSafeJobRunnerCodemod() Codemod {
	return Codemod{
		ID:           "safe-job-runner-to-runs-on",
		Name:         "Rename safe-outputs.jobs runner to runs-on",
		Description:  "Renames deprecated safe-outputs.jobs.<job>.runner fields to runs-on.",
		IntroducedIn: "1.5.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			newContent, applied, err := applyFrontmatterLineTransform(content, renameSafeJobRunnerKeys)
			if applied {
				safeJobRunnerCodemodLog.Print("Renamed safe-job runner fields to runs-on")
			}
			return newContent, applied, err
		},
	}
}

func renameSafeJobRunnerKeys(lines []string) ([]string, bool) {
	result := append([]string(nil), lines...)
	modified := false

	for i := range lines {
		if !hasYAMLKey(strings.TrimSpace(lines[i]), "safe-outputs") {
			continue
		}

		safeOutputsIndent := len(getIndentation(lines[i]))
		childIndent := -1
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimSpace(lines[j])
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}

			indent := len(getIndentation(lines[j]))
			if indent <= safeOutputsIndent {
				break
			}
			if childIndent == -1 {
				childIndent = indent
			}
			if indent != childIndent || !hasYAMLKey(trimmed, "jobs") {
				continue
			}

			if renameSafeJobRunnerKeysInJobsBlock(result, lines, j) {
				modified = true
			}
			break
		}
	}

	return result, modified
}

func renameSafeJobRunnerKeysInJobsBlock(result, lines []string, jobsLine int) bool {
	jobsIndent := len(getIndentation(lines[jobsLine]))
	jobIndent := -1
	jobStarts := []int{}
	blockEnd := jobsLine + 1

	for i := jobsLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			blockEnd = i + 1
			continue
		}

		indent := len(getIndentation(lines[i]))
		if indent <= jobsIndent {
			blockEnd = i
			break
		}
		blockEnd = i + 1
		if jobIndent == -1 {
			jobIndent = indent
		}
		if indent == jobIndent {
			jobStarts = append(jobStarts, i)
		}
	}

	modified := false
	for i, start := range jobStarts {
		end := blockEnd
		if i+1 < len(jobStarts) {
			end = jobStarts[i+1]
		}
		if renameSafeJobRunnerKeyInJob(result, lines, start, end) {
			modified = true
		}
	}
	return modified
}

func renameSafeJobRunnerKeyInJob(result, lines []string, start, end int) bool {
	jobIndent := len(getIndentation(lines[start]))
	fieldIndent := -1
	runnerLine := -1
	hasRunsOn := false

	for i := start + 1; i < end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(getIndentation(lines[i]))
		if indent <= jobIndent {
			break
		}
		if fieldIndent == -1 {
			fieldIndent = indent
		}
		if indent != fieldIndent {
			continue
		}
		if hasYAMLKey(trimmed, "runs-on") {
			hasRunsOn = true
		}
		if hasYAMLKey(trimmed, "runner") {
			runnerLine = i
		}
	}

	if runnerLine == -1 || hasRunsOn {
		return false
	}
	replacement, replaced := findAndReplaceInLine(lines[runnerLine], "runner", "runs-on")
	if replaced {
		result[runnerLine] = replacement
	}
	return replaced
}

func hasYAMLKey(line, key string) bool {
	return strings.HasPrefix(line, key+":")
}
