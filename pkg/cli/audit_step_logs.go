// This file provides command-line interface functionality for gh-aw.
// This file (audit_step_logs.go) attaches failure excerpts from downloaded GitHub Actions
// step logs to the jobs reported by `gh aw audit`.
//
// Key responsibilities:
//   - Locating the workflow-logs/{job}/{num}_{step}.txt file for a given job step
//   - Extracting a bounded excerpt of the failure output for failed steps
//   - Exposing that excerpt as JobStep.ErrorExcerpt so audit consumers (including the
//     MCP audit tool) can root-cause failures such as safe_outputs job errors
package cli

import (
	"cmp"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/stringutil"
)

// maxStepErrorExcerptLen bounds the size of a per-step failure excerpt so audit
// payloads stay small enough for MCP consumers.
const maxStepErrorExcerptLen = 2000

// maxStepLogTailBytes bounds how much of a step log file is read when building an
// excerpt. Failure output is written at the end of the step, so reading the tail is
// sufficient and keeps memory use bounded for very large logs.
const maxStepLogTailBytes int64 = 64 * 1024

// attachStepErrorExcerpts fills in ErrorExcerpt for every failed step of every job by
// reading the matching step log from the downloaded workflow-logs/ directory.
// It is a no-op when the run directory has no workflow logs.
func attachStepErrorExcerpts(jobs []JobData, logsPath string) {
	if logsPath == "" || !hasFailedStep(jobs) {
		return
	}
	workflowLogsDir := filepath.Join(logsPath, "workflow-logs")
	jobDirs, err := os.ReadDir(workflowLogsDir)
	if err != nil {
		auditReportLog.Printf("No workflow-logs directory for step excerpts: %v", err)
		return
	}

	jobDirsByName := make(map[string]string, len(jobDirs))
	for _, entry := range jobDirs {
		if entry.IsDir() {
			jobDirsByName[normalizeLogName(entry.Name())] = filepath.Join(workflowLogsDir, entry.Name())
		}
	}

	for i := range jobs {
		jobDir, ok := jobDirsByName[normalizeLogName(jobs[i].Name)]
		if !ok {
			continue
		}
		stepLogs := indexStepLogFiles(jobDir)
		stepLogOffsets := make(map[string]int, len(stepLogs))
		for j := range jobs[i].Steps {
			stepName := normalizeLogName(jobs[i].Steps[j].Name)
			stepPaths := stepLogs[stepName]
			stepLogOffset := stepLogOffsets[stepName]
			stepLogOffsets[stepName] = stepLogOffset + 1
			if !isFailureConclusion(jobs[i].Steps[j].Conclusion) {
				continue
			}
			if stepLogOffset >= len(stepPaths) {
				continue
			}
			excerpt := extractStepFailureExcerpt(stepPaths[stepLogOffset])
			if excerpt != "" {
				jobs[i].Steps[j].ErrorExcerpt = excerpt
				auditReportLog.Printf("Attached error excerpt for step %s/%s", jobs[i].Name, jobs[i].Steps[j].Name)
			}
		}
	}
}

func hasFailedStep(jobs []JobData) bool {
	for _, job := range jobs {
		for _, step := range job.Steps {
			if isFailureConclusion(step.Conclusion) {
				return true
			}
		}
	}
	return false
}

type numberedStepLog struct {
	number int
	path   string
}

// indexStepLogFiles maps normalized step names to their log file paths in step-number order.
func indexStepLogFiles(jobDir string) map[string][]string {
	entries, err := os.ReadDir(jobDir)
	if err != nil {
		return nil
	}
	numberedLogs := make(map[string][]numberedStepLog, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		num, stepName := parseStepFilename(entry.Name())
		if num <= 0 {
			continue
		}
		key := normalizeLogName(stepName)
		numberedLogs[key] = append(numberedLogs[key], numberedStepLog{
			number: num,
			path:   filepath.Join(jobDir, entry.Name()),
		})
	}

	index := make(map[string][]string, len(numberedLogs))
	for name, logs := range numberedLogs {
		slices.SortFunc(logs, func(a, b numberedStepLog) int {
			return cmp.Compare(a.number, b.number)
		})
		paths := make([]string, len(logs))
		for i, log := range logs {
			paths[i] = log.path
		}
		index[name] = paths
	}
	return index
}

// normalizeLogName lowercases a job or step name and collapses every run of
// non-alphanumeric characters into a single hyphen so that GitHub Actions log file
// and directory names can be matched against API-reported names.
func normalizeLogName(name string) string {
	var sb strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			sb.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(sb.String(), "-")
}

// extractStepFailureExcerpt returns a truncated excerpt of a failed step's log, preferring
// GitHub Actions ##[error] annotations and falling back to the tail of the log.
func extractStepFailureExcerpt(path string) string {
	tail := readLogTail(path, maxStepLogTailBytes)
	if tail == "" {
		return ""
	}

	var errorLines []string
	sawErrorAnnotation := false
	for line := range strings.SplitSeq(tail, "\n") {
		if strings.Contains(line, "##[error]") {
			sawErrorAnnotation = true
			stripped := stripGHALogTimestamps(line)
			if stripped != "" && !isAgentToolResultAnnotation(stripped) {
				errorLines = append(errorLines, stripped)
			}
		}
	}
	if len(errorLines) > 0 {
		return stringutil.Truncate(strings.Join(errorLines, "\n"), maxStepErrorExcerptLen)
	}
	if sawErrorAnnotation {
		return ""
	}

	content := strings.TrimSpace(stripGHALogTimestamps(tail))
	if content == "" {
		return ""
	}
	if len(content) > maxStepErrorExcerptLen {
		// Keep the end of the log, which holds the failure output.
		content = "..." + content[len(content)-maxStepErrorExcerptLen:]
	}
	return content
}

// readLogTail reads at most maxBytes from the end of the file at path.
func readLogTail(path string, maxBytes int64) string {
	f, err := os.Open(path)
	if err != nil {
		auditReportLog.Printf("Failed to open step log %s: %v", path, err)
		return ""
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		auditReportLog.Printf("Failed to stat step log %s: %v", path, err)
		return ""
	}
	if fi.Size() > maxBytes {
		if _, err := f.Seek(-maxBytes, io.SeekEnd); err != nil {
			auditReportLog.Printf("Failed to seek step log %s: %v", path, err)
			return ""
		}
	}

	data, err := io.ReadAll(f)
	if err != nil {
		auditReportLog.Printf("Failed to read step log %s: %v", path, err)
		return ""
	}
	return string(data)
}
