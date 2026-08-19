// This file provides the single, canonical code path used to derive the minimal
// GitHub App manifest requirements (permissions and webhook events) a workflow
// needs to operate. It is shared by every caller that must determine this "config"
// from a workflow's frontmatter — whether the workflow is reached directly as a
// standalone .md file or resolved from an aw.yml package manifest — so the same
// permission scopes, safe-outputs handling, and trigger normalization rules apply
// regardless of how the workflow was discovered.
package workflow

import (
	"sort"
	"strings"
)

// permissionScopesWithoutAppManifestEquivalent lists Actions-only permission scopes
// that have no corresponding GitHub App manifest "default_permissions" entry and
// must never be copied into a bootstrap App manifest. id-token and attestations are
// ephemeral Actions/OIDC token claims; models and copilot-requests gate
// Actions-hosted AI features rather than API access granted to an installed App.
var permissionScopesWithoutAppManifestEquivalent = map[PermissionScope]bool{
	PermissionIdToken:         true,
	PermissionAttestations:    true,
	PermissionModels:          true,
	PermissionCopilotRequests: true,
}

// GitHubAppManifestPermissionKey converts a workflow permission scope (as used in
// GitHub Actions frontmatter, e.g. "pull-requests") to the corresponding GitHub App
// manifest "default_permissions" key (e.g. "pull_requests"). It returns false when
// the scope has no GitHub App manifest equivalent (id-token, attestations, models,
// copilot-requests) and must be omitted from the manifest.
func GitHubAppManifestPermissionKey(scope PermissionScope) (string, bool) {
	if permissionScopesWithoutAppManifestEquivalent[scope] {
		return "", false
	}
	return strings.ReplaceAll(string(scope), "-", "_"), true
}

// ComputeGitHubAppManifestPermissions derives the GitHub App manifest
// "default_permissions" map (keyed by manifest permission name, e.g.
// "pull_requests") required to operate a workflow, given its top-level frontmatter
// "permissions" value and its already parsed/merged safe-outputs configuration
// (which itself accounts for imports and per-handler apps). Standard Actions token
// scopes honor shorthand (read-all/write-all/"all: read") via Permissions.Get;
// GitHub App-only scopes (e.g. administration) are only included when explicitly
// declared, matching validateGitHubAppOnlyPermissions. Scopes with no GitHub App
// manifest equivalent, and scopes at "none", are omitted.
func ComputeGitHubAppManifestPermissions(frontmatterPermissions any, safeOutputs *SafeOutputsConfig) map[string]string {
	permissions := NewPermissionsParserFromValue(frontmatterPermissions).ToPermissions()
	// Merge() treats an explicit permissions map (even an empty one) as authoritative
	// and clears any read-all/write-all shorthand on the receiver, so only merge when
	// the safe-outputs config actually contributes permission scopes.
	if safeOutputsPermissions := ComputePermissionsForSafeOutputs(safeOutputs); len(safeOutputsPermissions.permissions) > 0 {
		permissions.Merge(safeOutputsPermissions)
	}

	result := make(map[string]string)
	addManifestPermission := func(scope PermissionScope, level PermissionLevel, ok bool) {
		if !ok || level == "" || level == PermissionNone {
			return
		}
		key, supported := GitHubAppManifestPermissionKey(scope)
		if !supported {
			return
		}
		result[key] = string(level)
	}

	for _, scope := range GetAllPermissionScopes() {
		level, ok := permissions.Get(scope)
		addManifestPermission(scope, level, ok)
	}
	// GitHub App-only scopes must not be derived from read-all/write-all/"all: read"
	// shorthand: only an explicit declaration in frontmatter grants them.
	for _, scope := range GetAllGitHubAppOnlyScopes() {
		level, ok := permissions.GetExplicit(scope)
		addManifestPermission(scope, level, ok)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// nonWebhookOnTriggers lists workflow "on:" triggers that Actions supports but that
// are not deliverable to a GitHub App webhook subscription: they are driven by
// Actions scheduling/dispatch/composition, not by a webhook event delivered to an
// installed App.
var nonWebhookOnTriggers = map[string]bool{
	"schedule":            true,
	"workflow_dispatch":   true,
	"repository_dispatch": true,
	"workflow_call":       true,
}

// commandTriggerDefaultWebhookEvents lists the underlying GitHub webhook events a
// "command:" trigger (or its "/name" shorthand, expanded to "slash_command" by the
// compiler) listens to by default, when it does not restrict itself with an
// explicit "events:" list.
var commandTriggerDefaultWebhookEvents = []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"}

// labelCommandDefaultWebhookEvents lists the underlying GitHub webhook events a
// "label_command:" trigger listens to by default.
var labelCommandDefaultWebhookEvents = []string{"issues", "pull_request", "discussion"}

// IsKnownGitHubWebhookEvent reports whether name is a recognized GitHub Actions/App
// webhook event type.
func IsKnownGitHubWebhookEvent(name string) bool {
	return isKnownGitHubEvent(name)
}

// rawOnSectionTriggerNames extracts the top-level trigger key(s) from a workflow's
// "on" frontmatter value, which may be a string, a list of strings, or a mapping of
// trigger name to trigger configuration.
func rawOnSectionTriggerNames(onValue any) []string {
	var names []string
	switch value := onValue.(type) {
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
	return names
}

// NormalizeGitHubAppWebhookEvents extracts the set of GitHub App webhook events that
// a workflow's "on:" frontmatter section requires. It expands gh-aw's compiler-only
// command trigger shorthands ("/name" slash command strings, "command:", and
// "label_command:") to their underlying webhook events, maps pull_request_target to
// the pull_request webhook, and excludes both non-webhook Actions triggers
// (schedule, workflow_dispatch, repository_dispatch, workflow_call) and gh-aw's
// on:-section extension keys (reaction, status-comment, github-app, bots, roles,
// etc.) that are not themselves webhook events.
func NormalizeGitHubAppWebhookEvents(onValue any) []string {
	eventSet := map[string]struct{}{}
	addEvents := func(events []string) {
		for _, event := range events {
			eventSet[event] = struct{}{}
		}
	}

	for _, name := range rawOnSectionTriggerNames(onValue) {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		if strings.HasPrefix(name, "/") {
			// Slash command shorthand, e.g. "on: /my-bot".
			addEvents(commandTriggerDefaultWebhookEvents)
			continue
		}
		if strings.HasPrefix(name, "label-command ") {
			addEvents(labelCommandDefaultWebhookEvents)
			continue
		}

		switch name {
		case "slash_command", "command":
			addEvents(commandTriggerDefaultWebhookEvents)
			continue
		case "label_command":
			addEvents(labelCommandDefaultWebhookEvents)
			continue
		}

		if nonWebhookOnTriggers[name] || ghAwOnSectionKeys[name] {
			continue
		}
		if !isKnownGitHubEvent(name) {
			continue
		}
		if name == "pull_request_target" {
			name = "pull_request"
		}
		eventSet[name] = struct{}{}
	}

	events := make([]string, 0, len(eventSet))
	for event := range eventSet {
		events = append(events, event)
	}
	sort.Strings(events)
	return events
}
