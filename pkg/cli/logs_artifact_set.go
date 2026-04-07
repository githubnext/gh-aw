// This file provides command-line interface functionality for gh-aw.
// This file (logs_artifact_set.go) defines artifact set types and resolution logic
// for filtering artifact downloads in the logs and audit commands.
//
// Key responsibilities:
//   - Defining known artifact set names (all, agent, mcp, firewall, detection, github-api, activation)
//   - Mapping sets to concrete artifact name patterns
//   - Validating artifact set inputs from CLI flags and MCP arguments
//   - Determining whether a given artifact name matches an active filter
//   - Finding which filter entries are missing from a previously-downloaded run folder

package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var artifactSetLog = logger.New("cli:logs_artifact_set")

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

	// ArtifactSetMCP downloads the firewall-audit-logs artifact to access MCP
	// gateway traffic logs (gateway.jsonl, rpc-messages.jsonl) containing tool
	// calls, server negotiations, and RPC request/response pairs.
	ArtifactSetMCP ArtifactSet = "mcp"

	// ArtifactSetFirewall downloads the firewall-audit-logs artifact to access
	// AWF network policy data: domain allow/deny decisions, firewall audit trail,
	// and token-usage proxy logs.
	ArtifactSetFirewall ArtifactSet = "firewall"

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
	ArtifactSetFirewall:   {constants.FirewallAuditArtifactName},
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
	artifactSetLog.Printf("Validating %d artifact set(s): %s", len(sets), strings.Join(sets, ", "))
	var unknown []string
	for _, s := range sets {
		if _, ok := artifactSetArtifacts[ArtifactSet(s)]; !ok {
			unknown = append(unknown, s)
		}
	}
	if len(unknown) > 0 {
		artifactSetLog.Printf("Unknown artifact set(s) rejected: %s", strings.Join(unknown, ", "))
		return fmt.Errorf("unknown artifact set(s): %s; valid sets are: %s",
			strings.Join(unknown, ", "),
			strings.Join(ValidArtifactSetNames(), ", "))
	}
	artifactSetLog.Print("All artifact sets are valid")
	return nil
}

// ResolveArtifactFilter converts a list of set names into a deduplicated list of
// artifact base names to download.  A nil or empty input, or any entry equal to
// ArtifactSetAll, returns nil (meaning: download every artifact – no filter applied).
func ResolveArtifactFilter(sets []string) []string {
	if len(sets) == 0 {
		artifactSetLog.Print("No artifact sets specified, downloading all artifacts")
		return nil
	}

	// If "all" appears anywhere, disable filtering entirely.
	for _, s := range sets {
		if ArtifactSet(s) == ArtifactSetAll {
			artifactSetLog.Print("Artifact set 'all' specified, downloading all artifacts")
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
	artifactSetLog.Printf("Resolved artifact filter: sets=%v -> artifacts=%v", sets, names)
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

// findMissingFilterEntries checks which entries of the given artifact filter do not yet
// have a corresponding directory on disk in outputDir.
//
// For each filter entry (e.g. "firewall-audit-logs"), the function looks for:
//  1. A directory exactly named {entry} inside outputDir, or
//  2. Any directory whose name ends with "-{entry}" (workflow_call prefix pattern,
//     e.g. "abc123-agent" for filter entry "agent").
//
// Entries that have a matching directory are considered already-downloaded.
// Entries without a match are returned as "missing" — they still need to be fetched.
func findMissingFilterEntries(filter []string, outputDir string) []string {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		// If we can't read the directory, assume everything is missing.
		return filter
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}

	var missing []string
	for _, f := range filter {
		found := false
		for _, d := range dirs {
			// Mirror the artifactMatchesFilter logic: accept exact match or any directory
			// ending in "-{f}", which covers the workflow_call prefix pattern where GitHub
			// Actions prepends a short hash (e.g. "abc123-agent"). Note that this means a
			// hypothetical directory named "super-agent" would satisfy filter entry "agent",
			// but in practice artifact directories in a run folder only come from GitHub
			// Actions downloads and follow the "{hash}-{base}" or exact-base patterns.
			if d == f || strings.HasSuffix(d, "-"+f) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		artifactSetLog.Printf("Missing artifact entries in %s: %v", outputDir, missing)
	} else {
		artifactSetLog.Printf("All %d artifact filter entries present in %s", len(filter), outputDir)
	}
	return missing
}
