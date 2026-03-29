package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

// apmAppTokenStepID is the step ID for the GitHub App token mint step used by APM dependencies.
const apmAppTokenStepID = "apm-app-token"

// getEffectiveAPMGitHubToken returns the GitHub token expression to use for APM pack authentication.
// Priority (highest to lowest):
//  1. Custom token from dependencies.github-token field
//  2. secrets.GH_AW_PLUGINS_TOKEN (token dedicated for plugin/package operations)
//  3. secrets.GH_AW_GITHUB_TOKEN (general-purpose gh-aw token)
//  4. secrets.GITHUB_TOKEN (default GitHub Actions token)
func getEffectiveAPMGitHubToken(customToken string) string {
	if customToken != "" {
		apmDepsLog.Print("Using custom APM GitHub token (from dependencies.github-token)")
		return customToken
	}
	apmDepsLog.Print("Using cascading APM GitHub token (GH_AW_PLUGINS_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN)")
	return "${{ secrets.GH_AW_PLUGINS_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}"
}

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

// buildAPMAppTokenInvalidationStep generates the step to invalidate the GitHub App token
// that was minted for APM cross-org repository access. This step always runs (even on failure)
// to ensure the token is properly cleaned up after the APM pack step completes.
func buildAPMAppTokenInvalidationStep() []string {
	var steps []string

	steps = append(steps, "      - name: Invalidate GitHub App token for APM\n")
	steps = append(steps, fmt.Sprintf("        if: always() && steps.%s.outputs.token != ''\n", apmAppTokenStepID))
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          TOKEN: ${{ steps.%s.outputs.token }}\n", apmAppTokenStepID))
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Revoking GitHub App installation token for APM...\"\n")
	steps = append(steps, "          # GitHub CLI will auth with the token being revoked.\n")
	steps = append(steps, "          gh api \\\n")
	steps = append(steps, "            --method DELETE \\\n")
	steps = append(steps, "            -H \"Authorization: token $TOKEN\" \\\n")
	steps = append(steps, "            /installation/token || echo \"Token revocation failed (token may be expired or invalid).\"\n")
	steps = append(steps, "          echo \"Token invalidation step complete.\"\n")

	return steps
}

// GenerateAPMPackStep generates the GitHub Actions step that installs APM packages
// from GitHub and packs them into a bundle in a single github-script step.
//
// The step uses two JavaScript modules from the gh-aw setup actions:
//  1. apm_install.cjs — downloads packages from GitHub using the REST API,
//     writes the installed files and apm.lock.yaml to APM_WORKSPACE.
//     This replaces the previous `pip install apm-cli && apm install` shell step.
//  2. apm_pack.cjs — reads the installed workspace, filters files by target,
//     creates the .tar.gz bundle, and emits the bundle-path output.
//
// The step id is "apm_pack" so the upload-artifact step can reference
// ${{ steps.apm_pack.outputs.bundle-path }}.
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

	// GITHUB_APM_PAT is the token used by apm_install.cjs for GitHub API access.
	// When a GitHub App is configured, use the minted app token; otherwise use the
	// cascading fallback token.
	hasGitHubAppToken := apmDeps.GitHubApp != nil

	var githubAPMPatExpr string
	if hasGitHubAppToken {
		githubAPMPatExpr = fmt.Sprintf("${{ steps.%s.outputs.token }}", apmAppTokenStepID)
	} else {
		githubAPMPatExpr = getEffectiveAPMGitHubToken(apmDeps.GitHubToken)
	}

	// Encode packages as a JSON array for the APM_PACKAGES env var.
	// json.Marshal on a []string value never returns an error.
	pkgsJSON, _ := json.Marshal(apmDeps.Packages)

	githubScriptRef := GetActionPin("actions/github-script")

	// Single github-script step: install packages from GitHub, then pack them.
	lines := []string{
		"      - name: Install and pack APM bundle",
		"        id: apm_pack",
		"        uses: " + githubScriptRef,
		"        env:",
		"          GITHUB_APM_PAT: " + githubAPMPatExpr,
		// APM_PACKAGES is a JSON array; single-quoting the value prevents YAML
		// from parsing it as an array literal (GitHub Actions env values must be strings).
		"          APM_PACKAGES: '" + string(pkgsJSON) + "'",
		"          APM_WORKSPACE: /tmp/gh-aw/apm-workspace",
		"          APM_BUNDLE_OUTPUT: /tmp/gh-aw/apm-bundle-output",
		"          APM_TARGET: " + target,
	}

	// Include any user-provided env vars (skip keys already set above)
	reserved := map[string]bool{
		"GITHUB_APM_PAT": true,
		"APM_PACKAGES":   true,
		"APM_WORKSPACE":  true,
		"APM_TARGET":     true,
	}
	if len(apmDeps.Env) > 0 {
		keys := make([]string, 0, len(apmDeps.Env))
		for k := range apmDeps.Env {
			if !reserved[k] {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("          %s: %s", k, apmDeps.Env[k]))
		}
	}

	lines = append(lines,
		"        with:",
		"          script: |",
		"            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');",
		"            setupGlobals(core, github, context, exec, io);",
		"            const { main: apmInstall } = require('"+SetupActionDestination+"/apm_install.cjs');",
		"            await apmInstall();",
		"            const { main: apmPack } = require('"+SetupActionDestination+"/apm_pack.cjs');",
		"            await apmPack();",
	)

	return GitHubActionStep(lines)
}

// GenerateAPMRestoreStep generates the GitHub Actions step that restores APM packages
// from a pre-packed bundle in the agent job.
//
// The restore step uses the JavaScript implementation in apm_unpack.cjs (actions/setup/js)
// via actions/github-script, removing the dependency on microsoft/apm-action for
// the unpack phase. Packing still uses microsoft/apm-action in the dedicated APM job.
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

	apmDepsLog.Printf("Generating APM restore step using JS unpacker (isolated=%v)", apmDeps.Isolated)

	lines := []string{
		"      - name: Restore APM dependencies",
		"        uses: " + GetActionPin("actions/github-script"),
		"        env:",
		"          APM_BUNDLE_DIR: /tmp/gh-aw/apm-bundle",
		"        with:",
		"          script: |",
		"            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');",
		"            setupGlobals(core, github, context, exec, io);",
		"            const { main } = require('" + SetupActionDestination + "/apm_unpack.cjs');",
		"            await main();",
	}

	return GitHubActionStep(lines)
}
