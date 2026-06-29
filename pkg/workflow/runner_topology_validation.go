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

	for _, line := range strings.Split(content, "\n") {
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
// Matches "sudo " at word boundaries but not inside comments or strings mentioning sudo.
func containsSudoCommand(line string) bool {
	// Look for sudo as a command (not in a YAML key or comment)
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return false
	}
	// Check for sudo at start of a command or after common shell operators
	for _, prefix := range []string{"sudo ", "sudo\t"} {
		if strings.Contains(trimmed, prefix) {
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
