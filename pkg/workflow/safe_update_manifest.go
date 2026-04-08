package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeUpdateManifestLog = logger.New("workflow:safe_update_manifest")

// ghawManifestPattern matches a "# gh-aw-manifest: {...}" line in a lock file header.
var ghawManifestPattern = regexp.MustCompile(`#\s*gh-aw-manifest:\s*(\{.+\})`)

// currentGHAWManifestVersion is the current schema version for the GHAW manifest header.
const currentGHAWManifestVersion = 1

// GHAWManifestAction represents a single GitHub Action referenced in a compiled workflow.
type GHAWManifestAction struct {
	Repo    string `json:"repo"`
	SHA     string `json:"sha"`
	Version string `json:"version,omitempty"`
}

// GHAWManifest is the single-line JSON payload embedded as a "# gh-aw-manifest: ..."
// comment in generated lock files. It records the secrets and external actions that
// were detected at the time the lock file was last compiled so that subsequent
// compilations can detect newly introduced secrets when safe update mode is enabled.
type GHAWManifest struct {
	Version int                  `json:"version"`
	Secrets []string             `json:"secrets"`
	Actions []GHAWManifestAction `json:"actions"`
}

// NewGHAWManifest builds a GHAWManifest from the raw secret names and action reference
// strings produced by CollectSecretReferences and CollectActionReferences.
//
// secretNames entries may include or omit the "secrets." prefix; the prefix is always
// stripped before storage so the manifest contains plain names (e.g. "GITHUB_TOKEN").
// actionRefs entries follow the format produced by CollectActionReferences, e.g.
//
//	"actions/checkout@abc1234 # v4"
func NewGHAWManifest(secretNames []string, actionRefs []string) *GHAWManifest {
	safeUpdateManifestLog.Printf("Building gh-aw-manifest: raw_secrets=%d, raw_actions=%d", len(secretNames), len(actionRefs))

	// Normalize secret names to full "secrets.NAME" form and deduplicate.
	seen := make(map[string]bool)
	secrets := make([]string, 0, len(secretNames))
	for _, name := range secretNames {
		full := normalizeSecretName(name)
		if !seen[full] {
			seen[full] = true
			secrets = append(secrets, full)
		}
	}
	sort.Strings(secrets)

	actions := parseActionRefs(actionRefs)
	safeUpdateManifestLog.Printf("Manifest built: version=%d, secrets=%d, actions=%d", currentGHAWManifestVersion, len(secrets), len(actions))

	return &GHAWManifest{
		Version: currentGHAWManifestVersion,
		Secrets: secrets,
		Actions: actions,
	}
}

// normalizeSecretName ensures a secret identifier is stored as a plain name
// without the "secrets." prefix (e.g. "GITHUB_TOKEN" not "secrets.GITHUB_TOKEN").
// If the input already carries the "secrets." prefix it is stripped; otherwise
// the name is returned unchanged.
func normalizeSecretName(name string) string {
	return strings.TrimPrefix(name, "secrets.")
}

// parseActionRefs converts the action reference strings returned by
// CollectActionReferences into structured GHAWManifestAction values.
//
// Accepted formats (produced by actionReferencePattern):
//
//	"actions/checkout@abc1234 # v4"   → repo=actions/checkout, sha=abc1234, version=v4
//	"actions/checkout@v4"             → repo=actions/checkout, sha=v4, version=v4
func parseActionRefs(refs []string) []GHAWManifestAction {
	seen := make(map[string]bool)
	actions := make([]GHAWManifestAction, 0, len(refs))

	for _, raw := range refs {
		ref := raw

		// Extract optional inline comment (e.g. "# v4") for the human-readable version tag.
		comment := ""
		if idx := strings.Index(ref, " # "); idx >= 0 {
			comment = strings.TrimSpace(ref[idx+3:])
			ref = strings.TrimSpace(ref[:idx])
		}

		// Split on the last "@" to separate repo from sha/version.
		at := strings.LastIndex(ref, "@")
		if at < 0 {
			continue
		}
		repo := ref[:at]
		sha := ref[at+1:]
		version := comment
		if version == "" {
			version = sha
		}

		key := repo + "@" + sha
		if seen[key] {
			continue
		}
		seen[key] = true

		actions = append(actions, GHAWManifestAction{
			Repo:    repo,
			SHA:     sha,
			Version: version,
		})
	}

	// Sort for deterministic output.
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Repo != actions[j].Repo {
			return actions[i].Repo < actions[j].Repo
		}
		return actions[i].SHA < actions[j].SHA
	})

	return actions
}

// ToJSON serialises the manifest to a compact, single-line JSON string suitable
// for embedding in a YAML comment header.
func (m *GHAWManifest) ToJSON() (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to serialize gh-aw-manifest: %w", err)
	}
	return string(data), nil
}

// ExtractGHAWManifestFromLockFile parses the gh-aw-manifest from a lock file's
// comment header. Returns nil (with no error) when no manifest line is present,
// which is the normal state for lock files generated before this feature was
// introduced.
func ExtractGHAWManifestFromLockFile(content string) (*GHAWManifest, error) {
	matches := ghawManifestPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, nil
	}
	var m GHAWManifest
	if err := json.Unmarshal([]byte(matches[1]), &m); err != nil {
		return nil, fmt.Errorf("failed to parse gh-aw-manifest JSON: %w", err)
	}
	safeUpdateManifestLog.Printf("Extracted gh-aw-manifest: version=%d secrets=%d actions=%d",
		m.Version, len(m.Secrets), len(m.Actions))
	return &m, nil
}
