package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeUpdateLog = logger.New("workflow:safe_update")

// githubTokenSecret is the one secret that is always permitted in safe update mode.
const githubTokenSecret = "secrets.GITHUB_TOKEN"

// EnforceSafeUpdate validates that no new restricted secrets or unapproved action
// changes have been introduced compared to those recorded in the existing manifest.
//
// manifest is the gh-aw-manifest extracted from the current lock file before
// recompilation; it may be nil when no lock file (and therefore no previous
// manifest) exists yet, in which case enforcement is skipped.
//
// secretNames contains the raw names produced by CollectSecretReferences (i.e.
// WITHOUT the "secrets." prefix, e.g. "GITHUB_TOKEN").
//
// actionRefs contains the raw action reference strings produced by CollectActionReferences,
// e.g. "actions/checkout@abc1234 # v4".
//
// Returns a structured, actionable error when violations are found.
func EnforceSafeUpdate(manifest *GHAWManifest, secretNames []string, actionRefs []string) error {
	if manifest == nil {
		// No prior manifest – this is a first-time compilation; nothing to enforce.
		safeUpdateLog.Print("No existing manifest found; skipping safe update enforcement")
		return nil
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
// previous manifest) and are not the always-allowed GITHUB_TOKEN.
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
		if known[full] {
			continue
		}
		violations = append(violations, full)
	}
	sort.Strings(violations)
	return violations
}

// collectActionViolations compares the new action refs against the previous manifest
// and returns two sorted slices: repos that were added and repos that were removed.
// The comparison uses the action repo as the key, so SHA/version changes to an
// already-approved repo are not flagged.
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
	for repo := range newRepos {
		if !knownRepos[repo] {
			added = append(added, repo)
		}
	}

	// Find removals: repos present in the previous manifest but absent from the new compilation.
	for repo := range knownRepos {
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
	sb.WriteString("safe update mode rejected compilation: unapproved changes were introduced\n")

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
