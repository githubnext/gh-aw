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

// EnforceSafeUpdate validates that no new restricted secrets have been introduced
// compared to those recorded in the existing manifest.
//
// manifest is the GHAW manifest extracted from the current lock file before
// recompilation; it may be nil when no lock file (and therefore no previous
// manifest) exists yet, in which case enforcement is skipped.
//
// secretNames contains the raw names produced by CollectSecretReferences (i.e.
// WITHOUT the "secrets." prefix, e.g. "GITHUB_TOKEN").
//
// Returns a structured, actionable error when violations are found.
func EnforceSafeUpdate(manifest *GHAWManifest, secretNames []string) error {
	if manifest == nil {
		// No prior manifest – this is a first-time compilation; nothing to enforce.
		safeUpdateLog.Print("No existing manifest found; skipping safe update enforcement")
		return nil
	}

	// Build the set of previously known secrets from the manifest.
	known := make(map[string]bool, len(manifest.Secrets))
	for _, s := range manifest.Secrets {
		known[s] = true
	}

	// Identify newly introduced secrets that are not in the allowed set.
	var violations []string
	for _, name := range secretNames {
		full := normalizeSecretName(name)
		if full == githubTokenSecret {
			// Always permitted regardless of manifest state.
			continue
		}
		if known[full] {
			// Previously recorded – permitted.
			continue
		}
		violations = append(violations, full)
	}

	if len(violations) == 0 {
		safeUpdateLog.Printf("Safe update check passed (%d secret(s) verified)", len(secretNames))
		return nil
	}

	sort.Strings(violations)
	safeUpdateLog.Printf("Safe update violation: %d new secret(s) detected: %s",
		len(violations), strings.Join(violations, ", "))

	return buildSafeUpdateError(violations)
}

// buildSafeUpdateError creates a clear, structured error message that names the
// offending secrets and tells the user how to remediate.
func buildSafeUpdateError(violations []string) error {
	offending := strings.Join(violations, "\n  - ")
	return fmt.Errorf(
		"safe update mode rejected compilation: new restricted secret(s) were introduced\n\nOffending secret(s):\n  - %s\n\nRemediation options:\n  1. Use an interactive agentic flow (e.g. Copilot CLI) to review and approve the new secret.\n  2. Remove the --safe-update flag (or set safe-update: false in frontmatter) to allow the change.\n  3. Remove the new secret reference from your workflow if it was added unintentionally.",
		offending,
	)
}
