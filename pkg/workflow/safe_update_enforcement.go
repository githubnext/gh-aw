package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeUpdateLog = logger.New("workflow:safe_update")

// githubTokenSecret is the one secret that is always permitted in safe update mode.
// Stored without the "secrets." prefix to match manifest storage format.
const githubTokenSecret = "GITHUB_TOKEN"

// ghAwInternalSecrets lists secrets that are automatically injected by the gh-aw
// compiler as part of standard tool and engine configurations (e.g. GitHub MCP server,
// Copilot engine). These are infrastructure secrets managed by gh-aw itself, not
// user- or AI-authored content, so they are always permitted in safe update mode.
var ghAwInternalSecrets = map[string]bool{
	"GH_AW_GITHUB_TOKEN":            true,
	"GH_AW_GITHUB_MCP_SERVER_TOKEN": true,
	"GH_AW_AGENT_TOKEN":             true,
	"GH_AW_CI_TRIGGER_TOKEN":        true,
	"GH_AW_PROJECT_GITHUB_TOKEN":    true,
	"COPILOT_GITHUB_TOKEN":          true,
}

// EnforceSafeUpdate validates that no new restricted secrets or unapproved action
// changes have been introduced compared to those recorded in the existing manifest.
//
// manifest is the gh-aw-manifest extracted from the current lock file before
// recompilation. When nil (no lock file exists yet), it is treated as an empty
// manifest so that all non-GITHUB_TOKEN secrets and all custom actions are rejected
// on a first-time safe-update compilation.
//
// secretNames contains the raw names produced by CollectSecretReferences (i.e.
// they may or may not carry the "secrets." prefix; both forms are normalized
// via normalizeSecretName before comparison).
//
// actionRefs contains the raw action reference strings produced by CollectActionReferences,
// e.g. "actions/checkout@abc1234 # v4".
//
// Returns a structured, actionable error when violations are found.
func EnforceSafeUpdate(manifest *GHAWManifest, secretNames []string, actionRefs []string) error {
	if manifest == nil {
		// No prior lock file – treat as an empty manifest so safe-update enforcement
		// blocks any secrets (other than GITHUB_TOKEN) and any custom actions on the
		// first compilation, matching the principle of least privilege.
		safeUpdateLog.Print("No existing manifest found; treating as empty manifest for safe update enforcement")
		manifest = &GHAWManifest{Version: currentGHAWManifestVersion}
	}

	secretViolations := collectSecretViolations(manifest, secretNames)
	addedActions, removedActions := collectActionViolations(manifest, actionRefs)

	if len(secretViolations) == 0 && len(addedActions) == 0 && len(removedActions) == 0 {
		safeUpdateLog.Printf("Safe update check passed (%d secret(s), %d action(s) verified)",
			len(secretNames), len(actionRefs))
		return nil
	}

	if len(secretViolations) > 0 {
		safeUpdateLog.Printf("Safe update violation: %d new secret(s) detected: %s",
			len(secretViolations), strings.Join(secretViolations, ", "))
	}
	if len(addedActions) > 0 {
		safeUpdateLog.Printf("Safe update violation: %d new action(s) added: %s",
			len(addedActions), strings.Join(addedActions, ", "))
	}
	if len(removedActions) > 0 {
		safeUpdateLog.Printf("Safe update violation: %d action(s) removed: %s",
			len(removedActions), strings.Join(removedActions, ", "))
	}

	return buildSafeUpdateError(secretViolations, addedActions, removedActions)
}

// collectSecretViolations returns the normalized secret names that are new (not in the
// previous manifest) and are not among the always-allowed secrets (GITHUB_TOKEN and
// gh-aw-internal secrets automatically injected by the compiler).
func collectSecretViolations(manifest *GHAWManifest, secretNames []string) []string {
	known := make(map[string]bool, len(manifest.Secrets))
	for _, s := range manifest.Secrets {
		known[s] = true
	}

	var violations []string
	for _, name := range secretNames {
		full := normalizeSecretName(name)
		if full == githubTokenSecret {
			continue
		}
		if ghAwInternalSecrets[full] {
			continue
		}
		if known[full] {
			continue
		}
		violations = append(violations, full)
	}
	sort.Strings(violations)
	return violations
}

// githubActionsOrg is the owner whose actions are always trusted and never flagged
// as unapproved additions, regardless of what was recorded in the manifest.
const githubActionsOrg = "actions"

// isActionsOrgRepo reports whether a repo string belongs to the trusted "actions" org
// (i.e. has the form "actions/<name>").
func isActionsOrgRepo(repo string) bool {
	return strings.HasPrefix(repo, githubActionsOrg+"/")
}

// collectActionViolations compares the new action refs against the previous manifest
// and returns two sorted slices: repos that were added and repos that were removed.
// The comparison uses the action repo as the key, so SHA/version changes to an
// already-approved repo are not flagged.
// Actions belonging to the "actions/" GitHub org are always trusted and never flagged.
func collectActionViolations(manifest *GHAWManifest, actionRefs []string) (added []string, removed []string) {
	// Build known repo set from previous manifest.
	knownRepos := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		knownRepos[a.Repo] = true
	}

	// Build new repo set from the freshly compiled action refs.
	newActions := parseActionRefs(actionRefs)
	newRepos := make(map[string]bool, len(newActions))
	for _, a := range newActions {
		newRepos[a.Repo] = true
	}

	// Find additions: repos present in the new compilation but absent from the manifest.
	// Actions from the trusted "actions/" org are always allowed and never flagged.
	for repo := range newRepos {
		if isActionsOrgRepo(repo) {
			continue
		}
		if !knownRepos[repo] {
			added = append(added, repo)
		}
	}

	// Find removals: repos present in the previous manifest but absent from the new compilation.
	// Actions from the trusted "actions/" org are always allowed, so their removal is not flagged.
	for repo := range knownRepos {
		if isActionsOrgRepo(repo) {
			continue
		}
		if !newRepos[repo] {
			removed = append(removed, repo)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// buildSafeUpdateError creates a clear, structured error message that names the
// offending secrets and actions and tells the user how to remediate.
func buildSafeUpdateError(secretViolations, addedActions, removedActions []string) error {
	var sb strings.Builder
	sb.WriteString("safe update mode detected unapproved changes\n")

	if len(secretViolations) > 0 {
		sb.WriteString("\nNew restricted secret(s):\n  - ")
		sb.WriteString(strings.Join(secretViolations, "\n  - "))
	}
	if len(addedActions) > 0 {
		sb.WriteString("\nNew unapproved action(s):\n  - ")
		sb.WriteString(strings.Join(addedActions, "\n  - "))
	}
	if len(removedActions) > 0 {
		sb.WriteString("\nPreviously-approved action(s) removed:\n  - ")
		sb.WriteString(strings.Join(removedActions, "\n  - "))
	}

	sb.WriteString("\n\nRemediation options:\n  1. Use an interactive agentic flow (e.g. Copilot CLI) to review and approve the changes.\n  2. Remove the --safe-update flag to allow the change.\n  3. Revert the unapproved changes from your workflow if they were added unintentionally.")
	return fmt.Errorf("%s", sb.String())
}

// buildSafeUpdateWarningPrompt wraps the raw safe update violation message in a
// structured prompt that:
//   - Asks the calling agent to review each new secret / action for malicious use
//   - Asks the agent to include a security review note in the pull request description
//     so that human reviewers are aware of the changes
func buildSafeUpdateWarningPrompt(violationMsg string) string {
	var sb strings.Builder
	sb.WriteString("⚠️  SECURITY REVIEW REQUIRED — safe update mode detected unapproved changes:\n\n")
	sb.WriteString(violationMsg)
	sb.WriteString("\n\n")
	sb.WriteString("Action required for the calling agent:\n")
	sb.WriteString("  1. Review every new secret and action listed above for potential malicious use\n")
	sb.WriteString("     (e.g. credential exfiltration, supply-chain attacks, unexpected network access).\n")
	sb.WriteString("  2. Add a security review note to the pull request description that:\n")
	sb.WriteString("     - Lists the new secrets and actions introduced by this compilation.\n")
	sb.WriteString("     - Confirms you have reviewed each one and explains why it is safe to add.\n")
	sb.WriteString("     - Flags any item you cannot fully verify so a human reviewer can inspect it.\n")
	return sb.String()
}
