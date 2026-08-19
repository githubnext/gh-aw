package cli

import (
	"context"
	"maps"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var bootstrapInferenceLog = logger.New("cli:bootstrap_profile_inference")

// inferBootstrapGitHubAppRequirements resolves every workflow reachable from sources
// (an aw.yml package) and merges their required GitHub App manifest permissions and
// webhook events into the minimal set required for a GitHub App to operate the
// package. This lets aw.yml manifests omit the github-app config[].permissions/events
// fields and still get a least-privilege App instead of the "metadata: read"
// fallback.
//
// Requirement derivation is delegated to pkg/workflow (ComputeGitHubAppManifestPermissions
// and NormalizeGitHubAppWebhookEvents), the same code path used to determine safe-outputs
// permissions and trigger normalization for standalone .md workflows, so aw.yml packages
// and directly-added .md workflows are held to identical permission/event rules.
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
		safeOutputs := bootstrapSafeOutputsConfigFromFrontmatter(frontmatter.Frontmatter)
		for resource, level := range workflow.ComputeGitHubAppManifestPermissions(frontmatter.Frontmatter["permissions"], safeOutputs) {
			permissions[resource] = mergeBootstrapPermissionLevel(permissions[resource], level)
		}
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

// bootstrapSafeOutputsConfigFromFrontmatter builds a minimal *workflow.SafeOutputsConfig
// from a workflow's raw "safe-outputs" frontmatter map, using the handler key names
// present (e.g. "create-issue", "add-comment"). This reuses workflow.SafeOutputsConfigFromKeys,
// the same helper the interactive workflow builder uses to compute safe-outputs-derived
// permissions for newly generated .md workflows, so package workflows are scoped with the
// same rules (e.g. an "issues: read" workflow with "create-issue" configured still yields
// "issues: write").
func bootstrapSafeOutputsConfigFromFrontmatter(frontmatter map[string]any) *workflow.SafeOutputsConfig {
	safeOutputsRaw, ok := frontmatter["safe-outputs"].(map[string]any)
	if !ok || len(safeOutputsRaw) == 0 {
		return nil
	}
	keys := make([]string, 0, len(safeOutputsRaw))
	for key := range safeOutputsRaw {
		keys = append(keys, key)
	}
	return workflow.SafeOutputsConfigFromKeys(keys)
}

// bootstrapPermissionLevelRank ranks permission levels so higher-privilege scopes are
// never downgraded when merging requirements across multiple workflows or between
// declared aw.yml values and inferred ones.
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

// bootstrapEventNamesFromOn extracts the GitHub App webhook events required by a
// workflow's "on" frontmatter value. It delegates to workflow.NormalizeGitHubAppWebhookEvents,
// the same trigger-normalization code path used elsewhere in the compiler, so
// compiler-only keys (slash_command, label_command, reaction, status-comment),
// command-trigger shorthands (e.g. "on: /my-bot"), and pull_request_target are all
// handled consistently rather than being reimplemented here.
func bootstrapEventNamesFromOn(raw any) []string {
	return workflow.NormalizeGitHubAppWebhookEvents(raw)
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
