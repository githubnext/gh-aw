package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

const (
	// apmAppTokenStepID is the step ID for the GitHub App token mint step.
	apmAppTokenStepID = "apm-app-token-0"
)

// apmPackStepID returns the step ID for the APM pack step.
// The no-app case keeps the legacy "apm_pack" step ID for backward compatibility.
func apmPackStepID(deps *APMDependenciesInfo) string {
	if deps.HasGitHubApp() {
		return "apm_pack_0"
	}
	return "apm_pack"
}

// generateAPMAppTokenMintStep generates the step that mints a short-lived GitHub App
// installation token scoped for use in the APM pack step.
func (c *Compiler) generateAPMAppTokenMintStep(app *GitHubAppConfig) []string {
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	apmDepsLog.Printf("Generating APM GitHub App token mint step: owner=%s", owner)

	steps := []string{
		"      - name: Generate GitHub App token for APM dependencies\n",
		"        id: " + apmAppTokenStepID + "\n",
		fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")),
		"        with:\n",
		fmt.Sprintf("          app-id: %s\n", app.AppID),
		fmt.Sprintf("          private-key: %s\n", app.PrivateKey),
		fmt.Sprintf("          owner: %s\n", owner),
	}

	switch {
	case len(app.Repositories) == 1 && app.Repositories[0] == "*":
		// Org-wide access: omit repositories field
	case len(app.Repositories) == 1:
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	case len(app.Repositories) > 1:
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range app.Repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	default:
		steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")
	}

	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")
	return steps
}

// generateAPMPackStep generates the GitHub Actions step that installs APM packages and
// packs them into a bundle in the activation job. The step always uses isolated:true because
// the activation job has no repo context to preserve.
//
// When tokenStepID is non-empty, a GITHUB_TOKEN env override is added so the pack step
// uses the freshly minted GitHub App token instead of the default workflow token.
// When a GitHub App is configured the step ID becomes "apm_pack_0" and the artifact is
// named "apm-0"; otherwise the legacy "apm_pack" / "apm" names are kept for backward compat.
func generateAPMPackStep(apmDeps *APMDependenciesInfo, target string, tokenStepID string) GitHubActionStep {
	if apmDeps == nil || len(apmDeps.Packages) == 0 {
		apmDepsLog.Print("No APM dependencies to pack")
		return GitHubActionStep{}
	}

	// Step ID differs between the legacy (no-app) and app-auth cases.
	stepID := apmPackStepID(apmDeps)

	apmDepsLog.Printf("Generating APM pack step: id=%s, %d packages, target=%s", stepID, len(apmDeps.Packages), target)

	lines := []string{
		"      - name: Install and pack APM dependencies",
		"        id: " + stepID,
		"        uses: " + GetActionPin("microsoft/apm-action"),
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

// apmArtifactName returns the full artifact name for the APM bundle.
// Legacy (no GitHub App): "apm". With GitHub App: "apm-0".
func apmArtifactName(deps *APMDependenciesInfo, prefix string) string {
	base := constants.APMArtifactName // "apm"
	if deps.HasGitHubApp() {
		base += "-0"
	}
	return prefix + base
}

// GenerateAPMRestoreStep generates the GitHub Actions step that restores APM packages
// from a pre-packed bundle in the agent job.
//
// Returns a GitHubActionStep, or an empty step if apmDeps is nil or has no packages.
func GenerateAPMRestoreStep(apmDeps *APMDependenciesInfo) GitHubActionStep {
	if apmDeps == nil || len(apmDeps.Packages) == 0 {
		apmDepsLog.Print("No APM dependencies to restore")
		return GitHubActionStep{}
	}

	apmDepsLog.Printf("Generating APM restore step (isolated=%v)", apmDeps.Isolated)

	lines := []string{
		"      - name: Restore APM dependencies",
		"        uses: " + GetActionPin("microsoft/apm-action"),
		"        with:",
		"          bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
	}
	if apmDeps.Isolated {
		lines = append(lines, "          isolated: 'true'")
	}
	return GitHubActionStep(lines)
}
