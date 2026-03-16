package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

// apmArtifactBaseName returns the artifact base name for a given APMDependenciesInfo.
// When GitHub App auth is NOT configured (simple case) the legacy name "apm" is used to
// preserve backward compatibility with previously compiled lock files.
// When GitHub App auth IS configured, the artifact is named "apm-0".
func apmArtifactBaseName(deps *APMDependenciesInfo) string {
	if !deps.HasGitHubApp() {
		return constants.APMArtifactName // "apm" — backward compat
	}
	return constants.APMArtifactName + "-0"
}

// apmPackStepID returns the step ID for the APM pack step.
// The simple-case (no GitHub App) keeps the legacy "apm_pack" step ID.
func apmPackStepID(deps *APMDependenciesInfo) string {
	if !deps.HasGitHubApp() {
		return "apm_pack" // legacy
	}
	return "apm_pack_0"
}

// apmAppTokenStepID returns the step ID for the GitHub App token mint step.
func apmAppTokenStepID() string {
	return "apm-app-token-0"
}

// generateAPMAppTokenMintStep generates the step that mints a short-lived GitHub App
// installation token scoped for use in the APM pack step.
func (c *Compiler) generateAPMAppTokenMintStep(app *GitHubAppConfig, stepID string) []string {
	effectiveOwner := app.Owner
	if effectiveOwner == "" {
		effectiveOwner = "${{ github.repository_owner }} (default)"
	}
	apmDepsLog.Printf("Generating APM GitHub App token mint step: id=%s, owner=%s", stepID, effectiveOwner)
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token for APM dependencies\n")
	steps = append(steps, fmt.Sprintf("        id: %s\n", stepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	if len(app.Repositories) == 1 && app.Repositories[0] == "*" {
		// Org-wide access: omit repositories field
	} else if len(app.Repositories) == 1 {
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	} else if len(app.Repositories) > 1 {
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range app.Repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	} else {
		steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")
	}

	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")
	return steps
}

// GenerateAPMPackStep generates the GitHub Actions step that installs APM packages and
// packs them into a bundle in the activation job. The step always uses isolated:true because
// the activation job has no repo context to preserve.
//
// This is the simple (backward-compatible) form: no GitHub App auth.
// For the GitHub App case use generateAPMPackStepWithToken.
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

	apmDepsLog.Printf("Generating APM pack step: %d packages, target=%s", len(apmDeps.Packages), target)

	stepID := apmPackStepID(apmDeps)
	actionRef := GetActionPin("microsoft/apm-action")

	lines := []string{
		"      - name: Install and pack APM dependencies",
		"        id: " + stepID,
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
		"          working-directory: /tmp/gh-aw/apm-workspace",
	)

	return GitHubActionStep(lines)
}

// generateAPMPackStepWithToken generates the APM pack step for the GitHub App auth case.
// It sets the GITHUB_TOKEN env to the token minted by the preceding token-mint step.
// Uses step ID "apm_pack_0" (distinguished from the no-auth "apm_pack") so artifact
// references remain consistent when workflows have a github-app configured.
func generateAPMPackStepWithToken(apmDeps *APMDependenciesInfo, target string, tokenStepID string, data *WorkflowData) GitHubActionStep {
	if len(apmDeps.Packages) == 0 {
		apmDepsLog.Print("No APM packages to pack")
		return GitHubActionStep{}
	}

	apmDepsLog.Printf("Generating APM pack step with token: %d packages, target=%s, tokenStep=%s",
		len(apmDeps.Packages), target, tokenStepID)

	actionRef := GetActionPin("microsoft/apm-action")

	lines := []string{
		"      - name: Install and pack APM dependencies",
		"        id: " + apmPackStepID(apmDeps),
		"        uses: " + actionRef,
	}

	if tokenStepID != "" {
		lines = append(lines,
			"        env:",
			"          GITHUB_TOKEN: ${{ steps."+tokenStepID+".outputs.token }}",
		)
	}

	lines = append(lines,
		"        with:",
		"          dependencies: |",
	)

	for _, dep := range apmDeps.Packages {
		lines = append(lines, "            - "+dep)
	}

	lines = append(lines,
		"          isolated: 'true'",
		"          pack: 'true'",
		"          archive: 'true'",
		"          target: "+target,
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

	apmDepsLog.Printf("Generating APM restore step (isolated=%v)", apmDeps.Isolated)

	actionRef := GetActionPin("microsoft/apm-action")

	lines := []string{
		"      - name: Restore APM dependencies",
		"        uses: " + actionRef,
		"        with:",
		"          bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
	}

	if apmDeps.Isolated {
		lines = append(lines, "          isolated: 'true'")
	}

	return GitHubActionStep(lines)
}
