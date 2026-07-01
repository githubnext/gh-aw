package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
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

// GHAWManifestContainer represents a Docker container image referenced in a compiled workflow.
// It records the original mutable tag, the resolved SHA-256 digest (when available),
// and the full pinned reference that combines both.
type GHAWManifestContainer struct {
	Image       string `json:"image"`                  // Original tag, e.g. "node:lts-alpine"
	Digest      string `json:"digest,omitempty"`       // SHA-256 digest, e.g. "sha256:abc123..."
	PinnedImage string `json:"pinned_image,omitempty"` // Full ref, e.g. "node:lts-alpine@sha256:abc123..."
}

// GHAWManifestResolutionFailure represents an action-ref pinning failure captured
// during compilation. These failures are embedded for lock-file auditing.
type GHAWManifestResolutionFailure struct {
	Repo      string `json:"repo"`
	Ref       string `json:"ref"`
	ErrorType string `json:"error_type"`
}

// GHAWManifest is the single-line JSON payload embedded as a "# gh-aw-manifest: ..."
// comment in generated lock files. It records the secrets, external actions, and
// container images that were detected at the time the lock file was last compiled
// so that subsequent compilations can detect newly introduced secrets when safe
// update mode is enabled.
type GHAWManifest struct {
	Version            int                             `json:"version"`
	Secrets            []string                        `json:"secrets"`
	Actions            []GHAWManifestAction            `json:"actions"`
	Skills             []string                        `json:"skills,omitempty"`              // frontmatter skill specs (owner/repo@sha or owner/repo/skill/path@sha), sorted
	ResolutionFailures []GHAWManifestResolutionFailure `json:"resolution_failures,omitempty"` // unresolved action-ref pinning failures
	Containers         []GHAWManifestContainer         `json:"containers,omitempty"`          // container images used, with digest when available
	Redirect           string                          `json:"redirect,omitempty"`            // frontmatter redirect target for moved workflows
}

// NewGHAWManifest builds a GHAWManifest from the raw secret names, action reference
// strings, container image references, and skill specs produced at compile time.
//
// secretNames entries may include or omit the "secrets." prefix; the prefix is always
// stripped before storage so the manifest contains plain names (e.g. "GITHUB_TOKEN").
// actionRefs entries follow the format produced by CollectActionReferences, e.g.
//
//	"actions/checkout@abc1234 # v4"
//
// containers is the list of container image entries with full digest info (when available).
// skillSpecs is the list of skill references from the workflow frontmatter.
func NewGHAWManifest(secretNames []string, actionRefs []string, failures []GHAWManifestResolutionFailure, containers []GHAWManifestContainer, redirect string, skillSpecs []string) *GHAWManifest {
	safeUpdateManifestLog.Printf("Building gh-aw-manifest: raw_secrets=%d, raw_actions=%d, containers=%d, skills=%d", len(secretNames), len(actionRefs), len(containers), len(skillSpecs))

	// Normalize secret names to full "secrets.NAME" form and deduplicate.
	seen := make(map[string]struct {
	})
	secrets := make([]string, 0, len(secretNames))
	for _, name := range secretNames {
		full := normalizeSecretName(name)
		if !setutil.Contains(seen, full) {
			seen[full] = struct {
			}{}
			secrets = append(secrets, full)
		}
	}
	sort.Strings(secrets)

	actions := parseActionRefs(actionRefs)
	resolutionFailures := normalizeResolutionFailures(failures)

	// Deduplicate container entries by image name and sort for deterministic output.
	seenContainers := make(map[string]struct {
	}, len(containers))
	sortedContainers := make([]GHAWManifestContainer, 0, len(containers))
	for _, c := range containers {
		if c.Image != "" && !setutil.Contains(seenContainers, c.Image) {
			seenContainers[c.Image] = struct {
			}{}
			sortedContainers = append(sortedContainers, c)
		}
	}
	slices.SortFunc(sortedContainers, func(a, b GHAWManifestContainer) int {
		switch {
		case a.Image < b.Image:
			return -1
		case a.Image > b.Image:
			return 1
		default:
			return 0
		}
	})

	safeUpdateManifestLog.Printf("Manifest built: version=%d, secrets=%d, actions=%d, containers=%d, skills=%d",
		currentGHAWManifestVersion, len(secrets), len(actions), len(sortedContainers), len(skillSpecs))

	// Deduplicate and sort skill specs for deterministic output.
	seenSkills := make(map[string]struct{}, len(skillSpecs))
	sortedSkills := make([]string, 0, len(skillSpecs))
	for _, s := range skillSpecs {
		if s != "" && !setutil.Contains(seenSkills, s) {
			seenSkills[s] = struct{}{}
			sortedSkills = append(sortedSkills, s)
		}
	}
	sort.Strings(sortedSkills)
	if len(sortedSkills) == 0 {
		sortedSkills = nil // keep JSON output clean: omitempty omits nil but not empty slice
	}

	return &GHAWManifest{
		Version:            currentGHAWManifestVersion,
		Secrets:            secrets,
		Actions:            actions,
		Skills:             sortedSkills,
		ResolutionFailures: resolutionFailures,
		Containers:         sortedContainers,
		Redirect:           strings.TrimSpace(redirect),
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
	seen := make(map[string]struct {
	})
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
		if setutil.Contains(seen, key) {
			continue
		}
		seen[key] = struct {
		}{}

		actions = append(actions, GHAWManifestAction{
			Repo:    repo,
			SHA:     sha,
			Version: version,
		})
	}

	// Sort for deterministic output.
	slices.SortFunc(actions, func(a, b GHAWManifestAction) int {
		if a.Repo != b.Repo {
			if a.Repo < b.Repo {
				return -1
			}
			return 1
		}
		switch {
		case a.SHA < b.SHA:
			return -1
		case a.SHA > b.SHA:
			return 1
		default:
			return 0
		}
	})

	return actions
}

func normalizeResolutionFailures(failures []GHAWManifestResolutionFailure) []GHAWManifestResolutionFailure {
	type failureKey struct {
		Repo      string
		Ref       string
		ErrorType string
	}
	seen := make(map[failureKey]bool)
	normalized := make([]GHAWManifestResolutionFailure, 0, len(failures))
	for _, f := range failures {
		repo := strings.TrimSpace(f.Repo)
		ref := strings.TrimSpace(f.Ref)
		errorType := strings.TrimSpace(f.ErrorType)
		if repo == "" || ref == "" || errorType == "" {
			continue
		}
		key := failureKey{Repo: repo, Ref: ref, ErrorType: errorType}
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, GHAWManifestResolutionFailure{
			Repo:      repo,
			Ref:       ref,
			ErrorType: errorType,
		})
	}
	slices.SortFunc(normalized, func(a, b GHAWManifestResolutionFailure) int {
		if a.Repo != b.Repo {
			if a.Repo < b.Repo {
				return -1
			}
			return 1
		}
		if a.Ref != b.Ref {
			if a.Ref < b.Ref {
				return -1
			}
			return 1
		}
		switch {
		case a.ErrorType < b.ErrorType:
			return -1
		case a.ErrorType > b.ErrorType:
			return 1
		default:
			return 0
		}
	})
	return normalized
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
