package workflow

import (
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

	// Build env block: always add GITHUB_TOKEN (app token takes priority over cascading fallback)
	// plus any user-provided env vars.
	// If github-app is configured, GITHUB_TOKEN is set from the minted app token, so any
	// user-supplied GITHUB_TOKEN key is skipped to avoid a duplicate / conflicting entry.
	hasGitHubAppToken := apmDeps.GitHubApp != nil
	hasUserEnv := len(apmDeps.Env) > 0
	lines = append(lines, "        env:")
	if hasGitHubAppToken {
		lines = append(lines,
			fmt.Sprintf("          GITHUB_TOKEN: ${{ steps.%s.outputs.token }}", apmAppTokenStepID),
		)
	} else {
		// No github-app: use cascading token fallback (custom token or GH_AW_PLUGINS_TOKEN cascade)
		lines = append(lines,
			"          GITHUB_TOKEN: "+getEffectiveAPMGitHubToken(apmDeps.GitHubToken),
		)
	}
	if hasUserEnv {
		keys := make([]string, 0, len(apmDeps.Env))
		for k := range apmDeps.Env {
			// Skip GITHUB_TOKEN when github-app provides it to avoid duplicate keys
			if hasGitHubAppToken && k == "GITHUB_TOKEN" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("          %s: %s", k, apmDeps.Env[k]))
		}
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
