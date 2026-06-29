package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var runnerTopologyValidationLog = logger.New("workflow:runner_topology_validation")

// validateArcDindRootless checks that no generated step content uses sudo or other
// root-requiring operations when the workflow targets ARC/DinD topology.
// On ARC, the runner container does not have root access, so anything requiring
// root must already be handled at image build time.
func validateArcDindRootless(workflowData *WorkflowData) error {
	if !isArcDindTopology(workflowData) {
		return nil
	}
	runnerTopologyValidationLog.Print("Validating rootless execution for arc-dind topology")

	// Check custom steps, pre-steps, pre-agent-steps, and post-steps for sudo usage.
	checks := []struct {
		name    string
		content string
	}{
		{"steps", workflowData.CustomSteps},
		{"pre-steps", workflowData.PreSteps},
		{"pre-agent-steps", workflowData.PreAgentSteps},
		{"post-steps", workflowData.PostSteps},
	}

	for _, check := range checks {
		if check.content == "" {
			continue
		}
		if violations := findRootRequiringPatterns(check.content); len(violations) > 0 {
			return fmt.Errorf(
				"runner.topology is arc-dind but %s contain root-requiring operations (%s); "+
					"ARC runners do not have root access — remove sudo and privileged commands, "+
					"or use a pre-built sysroot image for system packages",
				check.name, strings.Join(violations, ", "),
			)
		}
	}

	return nil
}

// findRootRequiringPatterns scans step content for patterns that require root privileges.
// Returns a deduplicated list of violation descriptions found.
func findRootRequiringPatterns(content string) []string {
	var violations []string
	seen := map[string]bool{}

	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Skip YAML keys (e.g. "name:", "run:", "if:")
		if !strings.Contains(trimmed, "sudo") && !strings.Contains(trimmed, "apt-get") && !strings.Contains(trimmed, "apt ") {
			continue
		}

		if containsSudoCommand(trimmed) && !seen["sudo"] {
			seen["sudo"] = true
			violations = append(violations, "sudo")
		}
		if containsAptGetInstall(trimmed) && !seen["apt-get install"] {
			seen["apt-get install"] = true
			violations = append(violations, "apt-get install")
		}
	}

	return violations
}

// containsSudoCommand checks if a line contains a sudo invocation.
// It distinguishes between sudo used as a shell command and sudo merely
// mentioned in YAML metadata (e.g. "- name: avoid sudo operations").
// A line is flagged only when sudo appears as a command invocation:
//   - at the start of a shell command line,
//   - as the command in an inline "run: sudo ..." YAML field, or
//   - after a common shell operator (&& || ; | ().
func containsSudoCommand(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return false
	}

	// If this line is a YAML key-value, only inspect the value when the key is "run".
	// All other YAML keys (name, id, if, uses, with, env, …) are metadata and must
	// not cause false positives even when their value mentions sudo.
	rest := trimmed
	if strings.HasPrefix(rest, "- ") {
		rest = strings.TrimSpace(rest[2:])
	}
	if idx := strings.Index(rest, ":"); idx > 0 {
		key := rest[:idx]
		if isYAMLKey(key) {
			if key != "run" {
				return false // metadata field — not a command
			}
			// Inline run: value — check only the command portion
			cmd := strings.TrimSpace(rest[idx+1:])
			if cmd == "|" || cmd == ">" {
				return false // multiline block indicator, no inline command
			}
			return hasSudoInvocation(cmd)
		}
	}

	// Plain shell command line (inside a multiline run block)
	return hasSudoInvocation(trimmed)
}

// isYAMLKey reports whether s looks like a YAML mapping key (letters, digits, _ or -).
func isYAMLKey(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

// hasSudoInvocation reports whether cmd contains sudo used as a command
// (at start of the string, or immediately after a shell operator).
func hasSudoInvocation(cmd string) bool {
	if strings.HasPrefix(cmd, "sudo ") || strings.HasPrefix(cmd, "sudo\t") {
		return true
	}
	for _, op := range []string{"&& sudo ", "|| sudo ", "; sudo ", "| sudo ", "( sudo "} {
		if strings.Contains(cmd, op) {
			return true
		}
	}
	return false
}

// containsAptGetInstall checks if a line contains apt-get install.
func containsAptGetInstall(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return false
	}
	return strings.Contains(trimmed, "apt-get install") || strings.Contains(trimmed, "apt install")
}
