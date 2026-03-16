package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var apmDepsLog = logger.New("workflow:apm_dependencies")

// APMPackageGroup represents a group of APM packages that share the same effective GitHub App config.
// Packages with per-package overrides are in their own group; packages without overrides share
// the default github-app group (or the "no-auth" group when no default is configured).
type APMPackageGroup struct {
	GitHubApp *GitHubAppConfig // nil = use GITHUB_TOKEN; non-nil = mint a fresh token
	Packages  []string         // Package sources in this group
	Index     int              // Group index for deterministic step/artifact naming
}

// GetPackageGroups returns the package groups derived from the dependency configuration.
// When no GitHub App is configured (simple case), a single group with no auth is returned.
// When a GitHub App is configured, entries are grouped by their effective app config so that
// packages sharing the same credentials are packed together in one APM step.
func (a *APMDependenciesInfo) GetPackageGroups() []APMPackageGroup {
	return groupAPMEntriesByApp(a)
}

// groupAPMEntriesByApp groups APM package entries by their effective GitHub App config.
// Entries without a per-package override use the default GitHubApp (or no app).
// Entries with identical AppID+PrivateKey combinations are placed in the same group.
func groupAPMEntriesByApp(deps *APMDependenciesInfo) []APMPackageGroup {
	if !deps.HasGitHubApp() {
		// Simple case: no GitHub App configured — single group, no token minting needed
		return []APMPackageGroup{{Packages: deps.Packages, Index: 0}}
	}

	type groupData struct {
		app      *GitHubAppConfig
		packages []string
	}

	groupMap := make(map[string]*groupData) // key = apmAppGroupKey
	var groupOrder []string                 // preserves insertion order for determinism

	for _, entry := range deps.Entries {
		effectiveApp := entry.GitHubApp
		if effectiveApp == nil {
			effectiveApp = deps.GitHubApp // fall back to the top-level default
		}
		key := apmAppGroupKey(effectiveApp)
		if _, exists := groupMap[key]; !exists {
			groupMap[key] = &groupData{app: effectiveApp}
			groupOrder = append(groupOrder, key)
		}
		groupMap[key].packages = append(groupMap[key].packages, entry.Source)
	}

	groups := make([]APMPackageGroup, len(groupOrder))
	for i, key := range groupOrder {
		groups[i] = APMPackageGroup{
			GitHubApp: groupMap[key].app,
			Packages:  groupMap[key].packages,
			Index:     i,
		}
	}
	return groups
}

// apmAppGroupKey returns a stable string key for a GitHubAppConfig used to group packages.
// A nil config (= use GITHUB_TOKEN) maps to an empty string.
func apmAppGroupKey(app *GitHubAppConfig) string {
	if app == nil {
		return ""
	}
	return app.AppID + "|" + app.PrivateKey
}

// apmArtifactBaseName returns the artifact base name for a given APMDependenciesInfo and group index.
// When GitHub App auth is NOT configured (simple case) the legacy name "apm" is used to
// preserve backward compatibility with previously compiled lock files.
// When GitHub App auth IS configured, artifacts are named "apm-N".
func apmArtifactBaseName(deps *APMDependenciesInfo, groupIndex int) string {
	if !deps.HasGitHubApp() {
		return constants.APMArtifactName // "apm" — backward compat
	}
	return fmt.Sprintf("%s-%d", constants.APMArtifactName, groupIndex)
}

// apmPackStepID returns the step ID for the APM pack step of a given group.
// The simple-case (no GitHub App) keeps the legacy "apm_pack" step ID.
func apmPackStepID(deps *APMDependenciesInfo, groupIndex int) string {
	if !deps.HasGitHubApp() {
		return "apm_pack" // legacy
	}
	return fmt.Sprintf("apm_pack_%d", groupIndex)
}

// apmAppTokenStepID returns the step ID for the GitHub App token mint step for a given group.
func apmAppTokenStepID(groupIndex int) string {
	return fmt.Sprintf("apm-app-token-%d", groupIndex)
}

// generateAPMAppTokenMintStep generates the step that mints a short-lived GitHub App
// installation token scoped for use in the APM pack step.
// groupIndex is included in the step name to avoid duplicate-step validation errors when
// multiple groups each require their own token.
func (c *Compiler) generateAPMAppTokenMintStep(app *GitHubAppConfig, stepID string, groupIndex int) []string {
	effectiveOwner := app.Owner
	if effectiveOwner == "" {
		effectiveOwner = "${{ github.repository_owner }} (default)"
	}
	apmDepsLog.Printf("Generating APM GitHub App token mint step: id=%s, owner=%s", stepID, effectiveOwner)
	var steps []string

	steps = append(steps, fmt.Sprintf("      - name: Generate GitHub App token for APM dependencies (%d)\n", groupIndex))
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
// This is the simple (backward-compatible) form: no GitHub App auth, single group.
// For the multi-group case use GenerateAPMPackStepForGroup.
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

	stepID := apmPackStepID(apmDeps, 0)
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

// GenerateAPMPackStepForGroup generates the APM pack step for a specific package group.
// Used when GitHub App auth is configured (Option B), where each group may use a different token.
//
// Parameters:
//   - group:       APMPackageGroup containing the packages and optional GitHub App config
//   - target:      APM target (e.g. "copilot", "claude", "all")
//   - tokenStepID: Step ID of the preceding token-mint step (empty = use default GITHUB_TOKEN)
//   - data:        WorkflowData used for action pin resolution
func GenerateAPMPackStepForGroup(group APMPackageGroup, target string, tokenStepID string, data *WorkflowData) GitHubActionStep {
	if len(group.Packages) == 0 {
		apmDepsLog.Print("No APM packages in group to pack")
		return GitHubActionStep{}
	}

	apmDepsLog.Printf("Generating APM pack step for group %d: %d packages, target=%s, tokenStep=%s",
		group.Index, len(group.Packages), target, tokenStepID)

	stepID := fmt.Sprintf("apm_pack_%d", group.Index)
	workDir := fmt.Sprintf("/tmp/gh-aw/apm-workspace-%d", group.Index)
	actionRef := GetActionPin("microsoft/apm-action")

	lines := []string{
		fmt.Sprintf("      - name: Install and pack APM dependencies (%d)", group.Index),
		"        id: " + stepID,
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

	for _, dep := range group.Packages {
		lines = append(lines, "            - "+dep)
	}

	lines = append(lines,
		"          isolated: 'true'",
		"          pack: 'true'",
		"          archive: 'true'",
		"          target: "+target,
		"          working-directory: "+workDir,
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
