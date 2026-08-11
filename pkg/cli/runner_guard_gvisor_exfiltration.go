package cli

import (
	"os"
	"strings"
)

// runnerGuardSecretExfiltrationRule is the runner-guard rule that flags outbound HTTP
// requests to non-GitHub domains in privileged job contexts as possible secret exfiltration.
const runnerGuardSecretExfiltrationRule = "RGS-012"

// gvisorStepNameMarker is the exact step name emitted by generateGVisorInstallStep in
// pkg/workflow/copilot_engine_installation.go. The step downloads a pinned gVisor release from
// storage.googleapis.com and verifies it against a pinned SHA-512 checksum before installing
// it, so it never exfiltrates secrets.
const gvisorStepNameMarker = "Install gVisor (runsc)"

func filterExplicitRunnerGuardIgnores(findings []runnerGuardFinding, gitRoot string) []runnerGuardFinding {
	filtered := make([]runnerGuardFinding, 0, len(findings))
	fileLinesByPath := make(map[string][]string)

	for _, finding := range findings {
		resolvedPath := resolveRunnerGuardFilePath(gitRoot, finding.File)
		lines, ok := fileLinesByPath[resolvedPath]
		if !ok {
			lines = readWorkflowLines(resolvedPath)
			fileLinesByPath[resolvedPath] = lines
		}

		if findingHasRunnerGuardIgnore(lines, finding.Line, finding.RuleID) {
			runnerGuardLog.Printf("Suppressing %s finding with scoped annotation in %s", finding.RuleID, finding.File)
			continue
		}
		filtered = append(filtered, finding)
	}

	return filtered
}

func findingHasRunnerGuardIgnore(lines []string, lineNum int, ruleID string) bool {
	if lineNum <= 1 || lineNum > len(lines) || ruleID == "" {
		return false
	}

	findingIndex := lineNum - 1
	stepStart := -1
	stepIndent := -1
	for i := findingIndex; i >= 0; i-- {
		trimmed := strings.TrimLeft(lines[i], " ")
		if strings.HasPrefix(trimmed, "- ") {
			stepStart = i
			stepIndent = len(lines[i]) - len(trimmed)
			break
		}
	}
	if stepStart == -1 {
		return false
	}

	annotation := "# runner-guard:ignore " + ruleID
	for i := stepStart; i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " ")
		indent := len(lines[i]) - len(trimmed)
		if i > stepStart && indent == stepIndent && strings.HasPrefix(trimmed, "- ") {
			break
		}
		annotationIndex := strings.Index(lines[i], annotation)
		if annotationIndex == -1 {
			continue
		}
		remainder := lines[i][annotationIndex+len(annotation):]
		if remainder == "" || strings.HasPrefix(remainder, " ") || strings.HasPrefix(remainder, `\n`) {
			return true
		}
	}
	return false
}

// filterGvisorInstallFindings drops RGS-012 findings that point at the compiler-generated
// gVisor install step. That step downloads a fixed, version-pinned artifact into a file and
// verifies it against a published SHA-512 checksum before use; no data leaves the runner, so
// runner-guard's generic "outbound HTTP request in a privileged job" heuristic is a false
// positive here. The step is emitted verbatim into every workflow that enables the sandbox, so
// filtering by the surrounding file content (rather than per-workflow suppression comments)
// keeps the fix effective regardless of which line runner-guard attributes the finding to.
func filterGvisorInstallFindings(findings []runnerGuardFinding, gitRoot string) []runnerGuardFinding {
	filtered := make([]runnerGuardFinding, 0, len(findings))
	fileLinesByPath := make(map[string][]string)

	for _, finding := range findings {
		if finding.RuleID != runnerGuardSecretExfiltrationRule {
			filtered = append(filtered, finding)
			continue
		}

		resolvedPath := resolveRunnerGuardFilePath(gitRoot, finding.File)
		lines, ok := fileLinesByPath[resolvedPath]
		if !ok {
			lines = readWorkflowLines(resolvedPath)
			fileLinesByPath[resolvedPath] = lines
		}

		if findingInGvisorInstallStep(lines, finding.Line) {
			runnerGuardLog.Printf("Suppressing %s finding for gVisor install step in %s", finding.RuleID, finding.File)
			continue
		}
		filtered = append(filtered, finding)
	}

	return filtered
}

// readWorkflowLines reads the workflow file at path and splits it into lines. An empty or
// unreadable path returns nil, so findings are preserved rather than silently dropped.
func readWorkflowLines(path string) []string {
	if path == "" {
		return nil
	}

	// #nosec G304 -- path is produced by resolveRunnerGuardFilePath, which validates that the
	// resolved path stays within the repository root.
	content, err := os.ReadFile(path)
	if err != nil {
		runnerGuardLog.Printf("Failed to read workflow %s for gVisor step analysis: %v", path, err)
		return nil
	}

	return strings.Split(string(content), "\n")
}

// findingInGvisorInstallStep reports whether lineNum falls within the compiler-generated
// gVisor install step. It walks backward from lineNum to the nearest preceding GitHub Actions
// step boundary (a line matching "- name:" at the step's indentation) and checks whether that
// step is the gVisor install step. A lineNum of 0 or 1 (runner-guard's convention for an
// unknown/representative location) matches if the file contains the step at all, since the
// exact offending line cannot be pinpointed in that case.
func findingInGvisorInstallStep(lines []string, lineNum int) bool {
	if len(lines) == 0 {
		return false
	}

	if lineNum <= 1 {
		for _, line := range lines {
			if isStepNameLine(line, gvisorStepNameMarker) {
				return true
			}
		}
		return false
	}

	if lineNum > len(lines) {
		lineNum = len(lines)
	}

	for i := lineNum - 1; i >= 0; i-- {
		line := lines[i]
		if !isStepBoundaryLine(line) {
			continue
		}
		return isStepNameLine(line, gvisorStepNameMarker)
	}

	return false
}

// isStepBoundaryLine reports whether line marks the start of a GitHub Actions step, i.e. a
// (possibly indented) YAML sequence item starting with "name:" or "uses:".
func isStepBoundaryLine(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	if !strings.HasPrefix(trimmed, "- ") {
		return false
	}
	rest := strings.TrimSpace(trimmed[2:])
	return strings.HasPrefix(rest, "name:") || strings.HasPrefix(rest, "uses:")
}

// isStepNameLine reports whether a step boundary line's step name matches marker.
func isStepNameLine(line string, marker string) bool {
	trimmed := strings.TrimLeft(line, " ")
	rest := strings.TrimPrefix(trimmed, "- ")
	if !strings.HasPrefix(strings.TrimSpace(rest), "name:") {
		return false
	}
	return strings.Contains(rest, marker)
}
