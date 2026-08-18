package cli

import (
	"context"
	"maps"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var bootstrapInferenceLog = logger.New("cli:bootstrap_profile_inference")

// eventsExcludedFromGitHubAppInference lists workflow "on" triggers that do not
// correspond to a GitHub App webhook subscription and must never be inferred as
// required App events (they are driven by Actions scheduling/dispatch, not by a
// webhook delivered to an installed App).
var eventsExcludedFromGitHubAppInference = map[string]bool{
	"schedule":            true,
	"workflow_dispatch":   true,
	"repository_dispatch": true,
}

// inferBootstrapGitHubAppRequirements resolves every workflow reachable from sources
// (an aw.yml package) and merges their declared frontmatter permissions and trigger
// events into the minimal set required for a GitHub App to operate the package. This
// lets aw.yml manifests omit the github-app config[].permissions/events fields and
// still get a least-privilege App instead of the "metadata: read" fallback.
func inferBootstrapGitHubAppRequirements(ctx context.Context, sources []string) (map[string]string, []string, error) {
	if len(sources) == 0 {
		return nil, nil, nil
	}
	resolved, err := ResolveWorkflows(ctx, sources, false)
	if err != nil {
		return nil, nil, err
	}

	permissions := map[string]string{}
	eventSet := map[string]struct{}{}
	for _, candidate := range resolved.Workflows {
		if candidate == nil || candidate.IsActionWorkflow || candidate.IsPackageSkillFile || candidate.IsPackageAgentFile {
			continue
		}
		frontmatter, err := parser.ExtractFrontmatterFromContent(string(candidate.Content))
		if err != nil || frontmatter == nil {
			continue
		}
		mergeBootstrapPermissionsFromFrontmatter(permissions, frontmatter.Frontmatter["permissions"])
		for _, event := range bootstrapEventNamesFromOn(frontmatter.Frontmatter["on"]) {
			eventSet[event] = struct{}{}
		}
	}

	events := make([]string, 0, len(eventSet))
	for event := range eventSet {
		events = append(events, event)
	}
	sort.Strings(events)

	bootstrapInferenceLog.Printf("Inferred GitHub App requirements: permissions=%d, events=%d", len(permissions), len(events))
	if len(permissions) == 0 {
		permissions = nil
	}
	return permissions, events, nil
}

// mergeBootstrapPermissionsFromFrontmatter merges a single workflow's declared
// "permissions" frontmatter block into the running set, keeping the highest scope
// (write over read over none) seen across all workflows for each resource.
func mergeBootstrapPermissionsFromFrontmatter(merged map[string]string, raw any) {
	permMap, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for resource, value := range permMap {
		level, ok := value.(string)
		if !ok {
			continue
		}
		merged[resource] = mergeBootstrapPermissionLevel(merged[resource], strings.TrimSpace(level))
	}
}

func bootstrapPermissionLevelRank(level string) int {
	switch level {
	case "write":
		return 2
	case "read":
		return 1
	default:
		return 0
	}
}

// mergeBootstrapPermissionLevel returns the higher of two permission levels
// (write > read > none), so a resource needed as "write" by one workflow is never
// downgraded by another workflow that only needs "read".
func mergeBootstrapPermissionLevel(existing, incoming string) string {
	if existing == "" {
		return incoming
	}
	if bootstrapPermissionLevelRank(incoming) > bootstrapPermissionLevelRank(existing) {
		return incoming
	}
	return existing
}

// bootstrapEventNamesFromOn extracts the top-level trigger names from a workflow's
// "on" frontmatter value, which may be a string, a list of strings, or a mapping of
// trigger name to trigger configuration. Triggers that are not GitHub App webhook
// events (schedule, workflow_dispatch, repository_dispatch) are excluded.
func bootstrapEventNamesFromOn(raw any) []string {
	var names []string
	switch value := raw.(type) {
	case string:
		names = append(names, value)
	case []any:
		for _, item := range value {
			if name, ok := item.(string); ok {
				names = append(names, name)
			}
		}
	case map[string]any:
		for name := range value {
			names = append(names, name)
		}
	}

	filtered := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || eventsExcludedFromGitHubAppInference[name] {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
}

// mergeBootstrapGitHubAppRequirements combines explicitly declared manifest
// permissions/events (if any) with the inferred requirements from the package's
// resolved workflows, taking the union of events and the highest permission level
// per resource.
func mergeBootstrapGitHubAppRequirements(declaredPermissions map[string]string, declaredEvents []string, inferredPermissions map[string]string, inferredEvents []string) (map[string]string, []string) {
	merged := make(map[string]string, len(declaredPermissions)+len(inferredPermissions))
	maps.Copy(merged, declaredPermissions)
	for resource, level := range inferredPermissions {
		merged[resource] = mergeBootstrapPermissionLevel(merged[resource], level)
	}
	if len(merged) == 0 {
		merged = nil
	}

	eventSet := make(map[string]struct{}, len(declaredEvents)+len(inferredEvents))
	for _, event := range declaredEvents {
		eventSet[event] = struct{}{}
	}
	for _, event := range inferredEvents {
		eventSet[event] = struct{}{}
	}
	events := make([]string, 0, len(eventSet))
	for event := range eventSet {
		events = append(events, event)
	}
	sort.Strings(events)
	if len(events) == 0 {
		events = nil
	}

	return merged, events
}
