package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

// GenerateAPMDependenciesStep generates a GitHub Actions step that installs APM packages
// using the microsoft/apm-action action. The step is emitted when the workflow frontmatter
// contains a non-empty `dependencies` list in microsoft/apm format.
//
// Parameters:
//   - apmDeps: APM dependency configuration extracted from frontmatter
//   - data: WorkflowData used for action pin resolution
//
// Returns a GitHubActionStep, or an empty step if apmDeps is nil or has no packages.
func GenerateAPMDependenciesStep(apmDeps *APMDependenciesInfo, data *WorkflowData) GitHubActionStep {
	if apmDeps == nil || len(apmDeps.Packages) == 0 {
		apmDepsLog.Print("No APM dependencies to install")
		return GitHubActionStep{}
	}

	apmDepsLog.Printf("Generating APM dependencies step: %d packages", len(apmDeps.Packages))

	// Resolve the pinned action reference for microsoft/apm-action.
	actionRef := GetActionPin("microsoft/apm-action")

	// Build step lines. The `dependencies` input uses a YAML block scalar (`|`)
	// so each package is written as an indented list item on its own line.
	lines := []string{
		"      - name: Install APM dependencies",
		"        uses: " + actionRef,
		"        with:",
		"          dependencies: |",
	}

	for _, dep := range apmDeps.Packages {
		lines = append(lines, "            - "+dep)
	}

	return GitHubActionStep(lines)
}
