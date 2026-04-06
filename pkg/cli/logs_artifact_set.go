// This file provides command-line interface functionality for gh-aw.
// This file (logs_artifact_set.go) defines artifact set types and resolution logic
// for filtering artifact downloads in the logs and audit commands.
//
// Key responsibilities:
//   - Defining known artifact set names (all, agent, mcp, detection, github-api, activation)
//   - Mapping sets to concrete artifact name patterns
//   - Validating artifact set inputs from CLI flags and MCP arguments
//   - Determining whether a given artifact name matches an active filter

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// ArtifactSet is a named group of related artifacts that can be downloaded together.
// Using a named set allows callers to request only the artifacts they need for a
// specific analysis, rather than downloading all artifacts for a run.
type ArtifactSet string

const (
	// ArtifactSetAll downloads every artifact for the run (default behavior).
	ArtifactSetAll ArtifactSet = "all"

	// ArtifactSetActivation downloads the activation artifact (aw_info.json, prompt.txt,
	// and github_rate_limits.jsonl from the activation job).
	ArtifactSetActivation ArtifactSet = "activation"

	// ArtifactSetAgent downloads the unified agent artifact containing agent logs,
	// safe outputs, token usage, and agent-side github_rate_limits.jsonl.
	ArtifactSetAgent ArtifactSet = "agent"

	// ArtifactSetMCP downloads the firewall-audit-logs artifact containing MCP
	// gateway / proxy traffic logs.
	ArtifactSetMCP ArtifactSet = "mcp"

	// ArtifactSetDetection downloads the detection artifact containing threat
	// detection log output.
	ArtifactSetDetection ArtifactSet = "detection"

	// ArtifactSetGitHubAPI downloads the artifacts that contain GitHub API rate-limit
	// logs (github_rate_limits.jsonl), which are included in both the activation and
	// agent artifacts.
	ArtifactSetGitHubAPI ArtifactSet = "github-api"
)

// artifactSetArtifacts maps each named set to the list of artifact base names it includes.
// A nil value for ArtifactSetAll is intentional: it signals "no filter, download
// everything" and is handled specially in ResolveArtifactFilter (a nil return from
// ResolveArtifactFilter means no filter is active so the caller downloads all artifacts).
var artifactSetArtifacts = map[ArtifactSet][]string{
	ArtifactSetAll:        nil, // no filtering – download all artifacts
	ArtifactSetActivation: {constants.ActivationArtifactName},
	ArtifactSetAgent:      {constants.AgentArtifactName},
	ArtifactSetMCP:        {constants.FirewallAuditArtifactName},
	ArtifactSetDetection:  {constants.DetectionArtifactName},
	// github-api: both jobs upload github_rate_limits.jsonl; fetch both for a complete view.
	ArtifactSetGitHubAPI: {constants.ActivationArtifactName, constants.AgentArtifactName},
}

// ValidArtifactSetNames returns a sorted list of valid artifact set names,
// derived dynamically from the artifactSetArtifacts map to stay in sync automatically.
func ValidArtifactSetNames() []string {
	names := make([]string, 0, len(artifactSetArtifacts))
	for k := range artifactSetArtifacts {
		names = append(names, string(k))
	}
	sort.Strings(names)
	return names
}

// ValidateArtifactSets checks that every entry in sets is a known ArtifactSet name.
// Returns an error listing any unrecognized names.
func ValidateArtifactSets(sets []string) error {
	var unknown []string
	for _, s := range sets {
		if _, ok := artifactSetArtifacts[ArtifactSet(s)]; !ok {
			unknown = append(unknown, s)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown artifact set(s): %s; valid sets are: %s",
			strings.Join(unknown, ", "),
			strings.Join(ValidArtifactSetNames(), ", "))
	}
	return nil
}

// ResolveArtifactFilter converts a list of set names into a deduplicated list of
// artifact base names to download.  A nil or empty input, or any entry equal to
// ArtifactSetAll, returns nil (meaning: download every artifact – no filter applied).
func ResolveArtifactFilter(sets []string) []string {
	if len(sets) == 0 {
		return nil
	}

	// If "all" appears anywhere, disable filtering entirely.
	for _, s := range sets {
		if ArtifactSet(s) == ArtifactSetAll {
			return nil
		}
	}

	seen := make(map[string]bool)
	var names []string
	for _, s := range sets {
		for _, name := range artifactSetArtifacts[ArtifactSet(s)] {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

// artifactMatchesFilter reports whether the given artifact name should be downloaded
// given the active filter.
//
// A nil or empty filter means "accept everything".
//
// The match is satisfied when:
//  1. The artifact name exactly equals one of the filter entries, or
//  2. The artifact name ends with "-{filterEntry}" (workflow_call prefix pattern,
//     e.g. "abc123-agent" matches filter entry "agent").
func artifactMatchesFilter(name string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if name == f || strings.HasSuffix(name, "-"+f) {
			return true
		}
	}
	return false
}
