package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

// apmAppTokenStepID is the step ID for the GitHub App token mint step used by APM dependencies.
const apmAppTokenStepID = "apm-app-token"

// buildAPMAppTokenMintStep generates the step to mint a GitHub App installation access token
// for use by the APM pack step to access cross-org private repositories.
//
// Parameters:
//   - app:              GitHub App configuration containing app-id, private-key, owner, and repositories
//   - fallbackRepoExpr: expression used as the repositories value when app.Repositories is empty.
//     Pass "${{ steps.resolve-host-repo.outputs.target_repo_name }}" for workflow_call relay
//     workflows so the token is scoped to the platform (host) repo rather than the caller repo.
//     Pass "" to use the default "${{ github.event.repository.name }}" fallback.
//
// Returns a slice of YAML step lines.
func buildAPMAppTokenMintStep(app *GitHubAppConfig, fallbackRepoExpr string) []string {
	apmDepsLog.Printf("Building APM GitHub App token mint step: owner=%s, repos=%d", app.Owner, len(app.Repositories))
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token for APM dependencies\n")
	steps = append(steps, fmt.Sprintf("        id: %s\n", apmAppTokenStepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner - default to current repository owner if not specified
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	// Add repositories - behavior depends on configuration:
	// - If repositories is ["*"], omit the field to allow org-wide access
	// - If repositories is a single value, use inline format
	// - If repositories has multiple values, use block scalar format
	// - If repositories is empty/not specified, default to the current repository
	if len(app.Repositories) == 1 && app.Repositories[0] == "*" {
		// Org-wide access: omit repositories field entirely
		apmDepsLog.Print("Using org-wide GitHub App token for APM (repositories: *)")
	} else if len(app.Repositories) == 1 {
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	} else if len(app.Repositories) > 1 {
		steps = append(steps, "          repositories: |-\n")
		reposCopy := make([]string, len(app.Repositories))
		copy(reposCopy, app.Repositories)
		sort.Strings(reposCopy)
		for _, repo := range reposCopy {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	} else {
		// No explicit repositories: use fallback expression, or default to the triggering repo's name.
		// For workflow_call relay scenarios the caller passes steps.resolve-host-repo.outputs.target_repo_name
		// so the token is scoped to the platform (host) repo name rather than the full owner/repo slug.
		repoExpr := fallbackRepoExpr
		if repoExpr == "" {
			repoExpr = "${{ github.event.repository.name }}"
		}
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", repoExpr))
	}

	// Always add github-api-url from environment variable
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	return steps
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

	apmDepsLog.Printf("Generating APM pack step: %d packages, target=%s", len(apmDeps.Packages), target)

	actionRef := GetActionPin("microsoft/apm-action")

	lines := []string{
		"      - name: Install and pack APM dependencies",
		"        id: apm_pack",
		"        uses: " + actionRef,
	}

	// Inject the minted GitHub App token as GITHUB_TOKEN so APM can access cross-org private repos
	if apmDeps.GitHubApp != nil {
		lines = append(lines,
			"        env:",
			fmt.Sprintf("          GITHUB_TOKEN: ${{ steps.%s.outputs.token }}", apmAppTokenStepID),
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
		"          apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
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
		"          apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
	}

	if apmDeps.Isolated {
		lines = append(lines, "          isolated: 'true'")
	}

	return GitHubActionStep(lines)
}
