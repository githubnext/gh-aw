package workflow

import (
	"strings"

	actionpins "github.com/github/gh-aw/pkg/actionpins"
	"github.com/github/gh-aw/pkg/logger"
)

var actionPinsLog = logger.New("workflow:action_pins")

// Type aliases — callers within pkg/workflow use these names directly.

// ActionYAMLInput is defined in pkg/actionpins; aliased here so all files in
// pkg/workflow (action_cache.go, safe_outputs_actions.go, …) can reference
// the type without an explicit import.
type ActionYAMLInput = actionpins.ActionYAMLInput

// ActionPin is the pinned GitHub Action type from pkg/actionpins.
type ActionPin = actionpins.ActionPin

// ActionPinsData is the embedded JSON structure from pkg/actionpins.
type ActionPinsData = actionpins.ActionPinsData

// --------------------------------------------------------------------------
// Package-private helpers used throughout pkg/workflow
// --------------------------------------------------------------------------

// formatActionReference formats an action reference with repo, SHA, and version.
func formatActionReference(repo, sha, version string) string {
	return actionpins.FormatReference(repo, sha, version)
}

// formatActionCacheKey generates a cache key for action resolution.
func formatActionCacheKey(repo, version string) string {
	return actionpins.FormatCacheKey(repo, version)
}

// extractActionRepo extracts the action repository from a uses string.
func extractActionRepo(uses string) string {
	return actionpins.ExtractRepo(uses)
}

// extractActionVersion extracts the version from a uses string.
func extractActionVersion(uses string) string {
	return actionpins.ExtractVersion(uses)
}

// getActionPin returns the pinned reference for the latest version of the repo
// using only the embedded pins (no WorkflowData required).
func getActionPin(repo string) string {
	pins := actionpins.GetActionPinsByRepo(repo)
	if len(pins) == 0 {
		actionPinsLog.Printf("No embedded pins found for repo: %s", repo)
		return ""
	}
	return actionpins.FormatReference(repo, pins[0].SHA, pins[0].Version)
}

// getActionPins returns all loaded action pins (sorted by version descending).
// Package-private wrapper used by tests in this package.
func getActionPins() []ActionPin {
	return actionpins.GetActionPins()
}

// getCachedActionPinFromResolver returns the pinned action reference for repo,
// preferring dynamic resolution via resolver over the embedded pins.
// For use within pkg/workflow when only a resolver is available (no WorkflowData).
func getCachedActionPinFromResolver(repo string, resolver ActionSHAResolver) string {
	ctx := &actionpins.PinContext{}
	if resolver != nil {
		ctx.Resolver = resolver
	}
	return actionpins.GetCachedActionPin(repo, ctx)
}

// --------------------------------------------------------------------------
// Package-private API — delegates to pkg/actionpins with a PinContext from WorkflowData
// --------------------------------------------------------------------------

// getActionPinByRepo returns the latest ActionPin for a given repository, if any.
func getActionPinByRepo(repo string) (ActionPin, bool) {
	return actionpins.GetActionPinByRepo(repo)
}

// getActionPinWithData returns the pinned action reference for a given action@version,
// delegating to pkg/actionpins with a PinContext built from WorkflowData.
func getActionPinWithData(actionRepo, version string, data *WorkflowData) (string, error) {
	return actionpins.ResolveActionPin(actionRepo, version, data.PinContext())
}

// getCachedActionPin returns the pinned action reference for a given repository,
// preferring the dynamic resolver from WorkflowData over the embedded pins.
func getCachedActionPin(repo string, data *WorkflowData) string {
	return actionpins.GetCachedActionPin(repo, data.PinContext())
}

// --------------------------------------------------------------------------
// Step-level helpers that depend on WorkflowStep (stay in pkg/workflow)
// --------------------------------------------------------------------------

// applyActionPinToTypedStep applies SHA pinning to a WorkflowStep if it uses an action.
// Returns a modified copy of the step with pinned references.
// If the step doesn't use an action or the action is not pinned, returns the original step.
func applyActionPinToTypedStep(step *WorkflowStep, data *WorkflowData) *WorkflowStep {
	if step == nil || !step.IsUsesStep() {
		return step
	}

	actionRepo := extractActionRepo(step.Uses)
	if actionRepo == "" {
		return step
	}

	version := extractActionVersion(step.Uses)
	if version == "" {
		return step
	}

	// Strip the comment suffix before checking if it's already a SHA.
	// Uses strings like "repo@sha # version" are treated as already-pinned.
	rawVersion, _, _ := strings.Cut(version, " ")

	pinnedRef, err := getActionPinWithData(actionRepo, rawVersion, data)
	if err != nil || pinnedRef == "" {
		actionPinsLog.Printf("Skipping pin for %s@%s: no pin available", actionRepo, rawVersion)
		return step
	}

	actionPinsLog.Printf("Pinned action: %s@%s -> %s", actionRepo, rawVersion, pinnedRef)
	result := step.Clone()
	result.Uses = pinnedRef
	return result
}

// applyActionPinsToTypedSteps applies SHA pinning to a slice of typed WorkflowStep pointers.
// Returns a new slice with pinned references.
func applyActionPinsToTypedSteps(steps []*WorkflowStep, data *WorkflowData) []*WorkflowStep {
	if steps == nil {
		return nil
	}

	result := make([]*WorkflowStep, 0, len(steps))
	for _, step := range steps {
		if step == nil {
			result = append(result, nil)
			continue
		}
		result = append(result, applyActionPinToTypedStep(step, data))
	}
	return result
}
