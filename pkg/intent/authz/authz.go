// Package authz provides runtime enforcement of compiled ExecutionPolicy values.
//
// Gate usage behind [Authorizer.FeatureEnabled] until the policy model has been
// validated in production. When FeatureEnabled is false every enforcement method
// returns nil (advisory-only mode), preserving the previous behaviour.
package authz

import (
	"errors"
	"slices"

	"github.com/github/gh-aw/pkg/intent"
)

// Sentinel errors returned by Authorizer enforcement methods.
var (
	// ErrToolDenied is returned when a tool appears in ExecutionPolicy.DeniedTools.
	ErrToolDenied = errors.New("authz: tool is explicitly denied by policy")

	// ErrToolNotAllowed is returned when AllowedTools is non-nil and the tool is
	// not present in the list.
	ErrToolNotAllowed = errors.New("authz: tool is not in the allowed-tools list")

	// ErrWriteScopeExceeded is returned when the requested write scope exceeds
	// the scope permitted by the policy.
	ErrWriteScopeExceeded = errors.New("authz: write scope exceeds policy limit")

	// ErrHumanApprovalRequired is returned when the policy requires a human
	// approval gate before the operation may proceed.
	ErrHumanApprovalRequired = errors.New("authz: human approval is required before this operation")

	// ErrAttemptsExceeded is returned when currentAttempts has reached
	// ExecutionPolicy.MaxAttempts.
	ErrAttemptsExceeded = errors.New("authz: maximum attempts exceeded")
)

// Authorizer enforces a compiled [intent.ExecutionPolicy] at runtime.
//
// # Feature flag
//
// FeatureEnabled gates enforcement. Set it to true in production once the
// policy model has been validated end-to-end. Until then, every method
// returns nil so callers can be wired in without changing existing behaviour.
type Authorizer struct {
	// FeatureEnabled controls whether enforcement is active.
	// false → all methods return nil (advisory-only mode).
	// true  → policy constraints are enforced; violations return errors.
	FeatureEnabled bool
}

// AuthorizeTool returns an error if policy disallows calling the named tool.
//
//   - If policy.DeniedTools contains tool, [ErrToolDenied] is returned.
//   - If policy.AllowedTools is non-nil and does not contain tool,
//     [ErrToolNotAllowed] is returned.
//   - A nil AllowedTools slice means unrestricted; any tool not denied is allowed.
//   - An empty (non-nil) AllowedTools slice means deny-all.
func (a Authorizer) AuthorizeTool(policy intent.ExecutionPolicy, tool string) error {
	if !a.FeatureEnabled {
		return nil
	}
	if slices.Contains(policy.DeniedTools, tool) {
		return ErrToolDenied
	}
	if policy.AllowedTools != nil && !slices.Contains(policy.AllowedTools, tool) {
		return ErrToolNotAllowed
	}
	return nil
}

// AuthorizeWriteScope returns [ErrWriteScopeExceeded] when the requested write
// scope is broader than the scope the policy allows.
//
// Valid scopes from least to most permissive: "none", "feature_branch",
// "any_branch". An unrecognised scope string is treated as "none".
func (a Authorizer) AuthorizeWriteScope(policy intent.ExecutionPolicy, requestedScope string) error {
	if !a.FeatureEnabled {
		return nil
	}
	if writeScopeLevel(requestedScope) > writeScopeLevel(policy.WriteScope) {
		return ErrWriteScopeExceeded
	}
	return nil
}

// CheckHumanApproval returns [ErrHumanApprovalRequired] when the policy
// requires a human approval gate before the operation may proceed.
func (a Authorizer) CheckHumanApproval(policy intent.ExecutionPolicy) error {
	if !a.FeatureEnabled {
		return nil
	}
	if policy.HumanApprovalRequired {
		return ErrHumanApprovalRequired
	}
	return nil
}

// CheckAttemptLimit returns [ErrAttemptsExceeded] when currentAttempts has
// reached the MaxAttempts limit.
//
// A MaxAttempts value of 0 means unlimited; no error is returned in that case.
func (a Authorizer) CheckAttemptLimit(policy intent.ExecutionPolicy, currentAttempts int) error {
	if !a.FeatureEnabled {
		return nil
	}
	if policy.MaxAttempts > 0 && currentAttempts >= policy.MaxAttempts {
		return ErrAttemptsExceeded
	}
	return nil
}

// writeScopeLevel maps a write scope string to a numeric level.
// Higher values represent broader (less restrictive) write permissions.
func writeScopeLevel(scope string) int {
	switch scope {
	case "feature_branch":
		return 1
	case "any_branch":
		return 2
	default: // "none", empty, or unknown — most restrictive
		return 0
	}
}
