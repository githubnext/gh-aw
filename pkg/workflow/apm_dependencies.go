package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

// resolveAPMActionRef resolves the microsoft/apm-action reference using the action pin manager.
// It uses GetActionPinWithData for dynamic resolution (via the ActionResolver when available),
// falling back to hardcoded pins in action_pins.json.
func resolveAPMActionRef(data *WorkflowData) string {
	actionRepo := "microsoft/apm-action"
	actionVersion := string(constants.DefaultAPMActionVersion)
	pinnedRef, err := GetActionPinWithData(actionRepo, actionVersion, data)
	if err != nil || pinnedRef == "" {
		apmDepsLog.Printf("Failed to resolve %s@%s via pin manager: %v, using tag reference", actionRepo, actionVersion, err)
		return fmt.Sprintf("%s@%s", actionRepo, actionVersion)
	}
	apmDepsLog.Printf("Resolved %s@%s to %s via pin manager", actionRepo, actionVersion, pinnedRef)
	return pinnedRef
}

// GenerateAPMPackStep generates the GitHub Actions step that installs APM packages and
// packs them into a bundle in the activation job. The step always uses isolated:true because
// the activation job has no repo context to preserve.
//
// Parameters:
//   - apmDeps: APM dependency configuration extracted from frontmatter
//   - target:  APM target derived from the agentic engine (e.g. "copilot", "claude", "all")
//   - data:    WorkflowData used for action pin resolution
//
// Returns a GitHubActionStep, or an empty step if apmDeps is nil or has no packages.
func GenerateAPMPackStep(apmDeps *APMDependenciesInfo, target string, data *WorkflowData) GitHubActionStep {
	if apmDeps == nil || len(apmDeps.Packages) == 0 {
		apmDepsLog.Print("No APM dependencies to pack")
		return GitHubActionStep{}
	}

	apmVersion := apmDeps.GetAPMVersion()
	apmDepsLog.Printf("Generating APM pack step: %d packages, target=%s, apm-version=%s", len(apmDeps.Packages), target, apmVersion)

	actionRef := resolveAPMActionRef(data)

	lines := []string{
		"      - name: Install and pack APM dependencies",
		"        id: apm_pack",
		"        uses: " + actionRef,
		"        with:",
		"          dependencies: |",
	}

	for _, dep := range apmDeps.Packages {
		lines = append(lines, "            - "+dep)
	}

	lines = append(lines,
		"          isolated: 'true'",
		"          pack: 'true'",
		"          archive: 'true'",
		"          target: "+target,
		"          apm-version: "+apmVersion,
		"          working-directory: /tmp/gh-aw/apm-workspace",
	)

	return GitHubActionStep(lines)
}

// GenerateAPMRestoreStep generates the GitHub Actions step that restores APM packages
// from a pre-packed bundle in the agent job.
//
// Parameters:
//   - apmDeps: APM dependency configuration extracted from frontmatter
//   - data:    WorkflowData used for action pin resolution
//
// Returns a GitHubActionStep, or an empty step if apmDeps is nil or has no packages.
func GenerateAPMRestoreStep(apmDeps *APMDependenciesInfo, data *WorkflowData) GitHubActionStep {
	if apmDeps == nil || len(apmDeps.Packages) == 0 {
		apmDepsLog.Print("No APM dependencies to restore")
		return GitHubActionStep{}
	}

	apmVersion := apmDeps.GetAPMVersion()
	apmDepsLog.Printf("Generating APM restore step (isolated=%v, apm-version=%s)", apmDeps.Isolated, apmVersion)

	actionRef := resolveAPMActionRef(data)

	lines := []string{
		"      - name: Restore APM dependencies",
		"        uses: " + actionRef,
		"        with:",
		"          bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
		"          apm-version: " + apmVersion,
	}

	if apmDeps.Isolated {
		lines = append(lines, "          isolated: 'true'")
	}

	return GitHubActionStep(lines)
}
