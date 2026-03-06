package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

// GenerateAPMDependenciesStep generates a GitHub Actions step that installs APM packages
// using the microsoft/apm-setup action. The step is emitted when the workflow frontmatter
// contains a non-empty `dependencies` list in microsoft/apm format.
//
// Parameters:
//   - dependencies: List of APM package references (e.g., "microsoft/apm-sample-package",
//     "github/awesome-copilot/skills/review-and-refactor")
//   - data: WorkflowData used for action pin resolution
//
// Returns a GitHubActionStep, or an empty step if the dependencies list is empty.
func GenerateAPMDependenciesStep(dependencies []string, data *WorkflowData) GitHubActionStep {
	if len(dependencies) == 0 {
		apmDepsLog.Print("No APM dependencies to install")
		return GitHubActionStep{}
	}

	apmDepsLog.Printf("Generating APM dependencies step: %d packages", len(dependencies))

	// Resolve the pinned action reference for microsoft/apm-setup.
	actionRef := getAPMSetupActionRef(data)

	// Build step lines. The `dependencies` input uses a YAML block scalar (`|`)
	// so each package is written as an indented list item on its own line.
	lines := []string{
		"      - name: Install APM dependencies",
		"        uses: " + actionRef,
		"        with:",
		"          dependencies: |",
	}

	for _, dep := range dependencies {
		lines = append(lines, "            - "+dep)
	}

	return GitHubActionStep(lines)
}

// getAPMSetupActionRef returns the pinned or default action reference for microsoft/apm-setup.
func getAPMSetupActionRef(data *WorkflowData) string {
	pinned, err := GetActionPinWithData("microsoft/apm-setup", "v1", data)
	if err != nil || pinned == "" {
		return "microsoft/apm-setup@v1"
	}
	return pinned
}
