package intent

import (
	"slices"
)

// ExecutionPolicy governs what an agent may do for a given intent.
type ExecutionPolicy struct {
	Autonomy string `json:"autonomy"`

	// AllowedTools controls which tools the agent may call.
	// nil means unrestricted; []string{} (non-nil empty) means deny-all; non-empty
	// means restricted to the listed tools. JSON omitempty cannot preserve the
	// nil-vs-empty distinction; callers must check AllowedTools != nil at runtime.
	AllowedTools []string `json:"allowed_tools,omitempty"`
	DeniedTools  []string `json:"denied_tools,omitempty"`

	WriteScope string `json:"write_scope"`

	RequiredChecks []string `json:"required_checks,omitempty"`

	HumanApprovalRequired bool `json:"human_approval_required"`

	// AutoMergeAllowed uses a pointer so that an unset rule fragment (nil) is
	// distinguishable from an explicit denial (false). The merge logic only applies
	// the AND (more-restrictive) step when at least one side has an explicit value.
	// nil means the rule did not express a preference; false is an explicit denial;
	// true is an explicit grant.
	AutoMergeAllowed *bool `json:"auto_merge_allowed,omitempty"`

	MaxAttempts int `json:"max_attempts"`

	RuleIDs []string `json:"rule_ids,omitempty"`
}

// RepositoryContext carries repository-level context used when matching policy rules.
type RepositoryContext struct {
	Owner      string `json:"owner,omitempty"`
	Name       string `json:"name,omitempty"`
	Visibility string `json:"visibility,omitempty"` // "public" or "private"
	Org        string `json:"org,omitempty"`
}

// PolicyRule pairs a match condition with a policy fragment to apply.
type PolicyRule struct {
	ID    string          `json:"id"`
	Scope string          `json:"scope,omitempty"` // "organization", "repository", "intent", or "workflow"
	When  PolicyCondition `json:"when"`
	Set   ExecutionPolicy `json:"set"`
}

// PolicyCondition describes when a rule applies.
type PolicyCondition struct {
	Domain   string `json:"domain,omitempty"`
	Priority string `json:"priority,omitempty"`
	Risk     string `json:"risk,omitempty"`
	Org      string `json:"org,omitempty"`
}

// PolicyCompiler holds policy rules and compiles them into a single ExecutionPolicy
// using more-restrictive-wins merge semantics.
type PolicyCompiler struct {
	Rules []PolicyRule
}

// Compile assembles a single ExecutionPolicy from the ordered rules using
// more-restrictive-wins merge semantics.
//
// Unlinked and ambiguous attribution always produce safestDefaultPolicy()
// regardless of configured rules, enforcing fail-closed behaviour.
//
// When one or more rules match the intent record and repository context, the
// first matching rule's values define the initial policy; each subsequent
// matching rule applies more-restrictive-wins on top. Any field not touched
// by any matching rule falls back to the safest default value.
//
// When no rules match the result is safestDefaultPolicy().
func (c PolicyCompiler) Compile(record IntentRecord, repo RepositoryContext) ExecutionPolicy {
	// Fail closed for unresolved attribution — no rules can relax this.
	if record.Status == AttributionUnlinked || record.Status == AttributionAmbiguous {
		return safestDefaultPolicy()
	}

	safe := safestDefaultPolicy()
	var accumulated *ExecutionPolicy
	var ruleIDs []string

	for _, rule := range c.Rules {
		if !rule.When.Matches(record, repo) {
			continue
		}
		if rule.ID != "" {
			ruleIDs = append(ruleIDs, rule.ID)
		}
		if accumulated == nil {
			// First matching rule: adopt its values as the initial policy.
			p := rule.Set
			accumulated = &p
		} else {
			// Subsequent rules: keep the more restrictive value for each field.
			merged := mergePolicy(*accumulated, rule.Set)
			accumulated = &merged
		}
	}

	if accumulated == nil {
		return safe
	}

	policy := *accumulated
	policy.RuleIDs = ruleIDs

	// Fill in fields that no matching rule addressed with the safe default.
	if policy.Autonomy == "" {
		policy.Autonomy = safe.Autonomy
	}
	if policy.WriteScope == "" {
		policy.WriteScope = safe.WriteScope
	}

	return policy
}

// Matches reports whether c applies to the given intent record and repository context.
// An empty PolicyCondition matches everything.
func (c PolicyCondition) Matches(record IntentRecord, repo RepositoryContext) bool {
	if c.Domain != "" && !slices.Contains(record.Labels, c.Domain) {
		return false
	}
	if c.Priority != "" && !slices.Contains(record.Labels, c.Priority) {
		return false
	}
	if c.Risk != "" && !slices.Contains(record.Labels, c.Risk) {
		return false
	}
	if c.Org != "" && repo.Org != c.Org {
		return false
	}
	return true
}

// safestDefaultPolicy returns the most restrictive possible policy:
// propose_only autonomy, no writes, human approval required, no auto-merge,
// and a single-attempt limit.
func safestDefaultPolicy() ExecutionPolicy {
	autoMerge := false
	return ExecutionPolicy{
		Autonomy:              "propose_only",
		WriteScope:            "none",
		HumanApprovalRequired: true,
		AutoMergeAllowed:      &autoMerge,
		MaxAttempts:           1,
	}
}

// mergePolicy returns a new ExecutionPolicy that applies more-restrictive-wins
// semantics between base and override:
//
//   - DeniedTools and RequiredChecks: union (deny everything denied by either side)
//   - AllowedTools: intersection when both constrain; adopt restriction when only
//     one side constrains; nil (unrestricted) when neither constrains
//   - HumanApprovalRequired: OR (require approval if either side requires it)
//   - AutoMergeAllowed: AND (block auto-merge if either side blocks it)
//   - MaxAttempts: min of non-zero values; 0 means unlimited
//   - WriteScope: more-restrictive scope wins
//   - Autonomy: more-restrictive autonomy wins
func mergePolicy(base, override ExecutionPolicy) ExecutionPolicy {
	result := base

	result.DeniedTools = unionStrings(base.DeniedTools, override.DeniedTools)
	result.AllowedTools = mergeAllowedTools(base.AllowedTools, override.AllowedTools)
	result.RequiredChecks = unionStrings(base.RequiredChecks, override.RequiredChecks)

	if override.HumanApprovalRequired {
		result.HumanApprovalRequired = true
	}

	result.AutoMergeAllowed = mergeAutoMerge(base.AutoMergeAllowed, override.AutoMergeAllowed)
	result.MaxAttempts = mergeMaxAttempts(base.MaxAttempts, override.MaxAttempts)
	result.WriteScope = moreRestrictiveWriteScope(base.WriteScope, override.WriteScope)
	result.Autonomy = moreRestrictiveAutonomy(base.Autonomy, override.Autonomy)

	return result
}

// mergeAllowedTools combines two AllowedTools slices using more-restrictive-wins:
//   - nil × nil → nil (unrestricted)
//   - nil × slice → slice (adopt the constraint)
//   - slice × nil → slice (adopt the constraint)
//   - slice × slice → intersection (keep only tools allowed by both)
func mergeAllowedTools(base, override []string) []string {
	switch {
	case base == nil && override == nil:
		return nil
	case base == nil:
		return cloneStrings(override)
	case override == nil:
		return cloneStrings(base)
	default:
		return intersectStrings(base, override)
	}
}

// mergeAutoMerge combines two AutoMergeAllowed pointers using AND semantics:
// auto-merge is blocked if either side explicitly blocks it.
func mergeAutoMerge(base, override *bool) *bool {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		v := *override
		return &v
	}
	if override == nil {
		v := *base
		return &v
	}
	v := *base && *override
	return &v
}

// mergeMaxAttempts returns the smaller of two attempt limits.
// 0 means unlimited; a non-zero value always wins over 0.
func mergeMaxAttempts(base, override int) int {
	switch {
	case base == 0:
		return override
	case override == 0:
		return base
	case base <= override:
		return base
	default:
		return override
	}
}

// writeScopeLevel maps a write scope string to a numeric level.
// Higher numbers represent broader (less restrictive) write permissions.
func writeScopeLevel(scope string) int {
	switch scope {
	case "feature_branch":
		return 1
	case "any_branch":
		return 2
	default: // "none", empty, or unknown — treat as most restrictive
		return 0
	}
}

// moreRestrictiveWriteScope returns the more restrictive of two write scopes.
func moreRestrictiveWriteScope(a, b string) string {
	if writeScopeLevel(a) <= writeScopeLevel(b) {
		return a
	}
	return b
}

// autonomyLevel maps an autonomy string to a numeric level.
// Higher numbers represent broader (less restrictive) autonomy.
func autonomyLevel(autonomy string) int {
	switch autonomy {
	case "supervised":
		return 1
	case "bounded":
		return 2
	default: // "propose_only", empty, or unknown — treat as most restrictive
		return 0
	}
}

// moreRestrictiveAutonomy returns the more restrictive of two autonomy values.
// An empty string is treated as "propose_only" (most restrictive).
func moreRestrictiveAutonomy(a, b string) string {
	if autonomyLevel(a) <= autonomyLevel(b) {
		return a
	}
	return b
}

// unionStrings returns a deduplicated union of a and b preserving order.
func unionStrings(a, b []string) []string {
	if len(a) == 0 {
		return cloneStrings(b)
	}
	if len(b) == 0 {
		return cloneStrings(a)
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	for _, s := range b {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	return result
}

// intersectStrings returns elements present in both a and b, preserving the
// order from a. When a is non-nil, the returned slice is always non-nil
// (an empty non-nil slice when no elements are in common), which preserves
// the deny-all semantics of AllowedTools: a non-nil empty AllowedTools means
// "deny all tools", and intersecting it with any list must remain non-nil empty,
// not nil (which would mean "unrestricted").
func intersectStrings(a, b []string) []string {
	bSet := make(map[string]struct{}, len(b))
	for _, s := range b {
		bSet[s] = struct{}{}
	}
	result := make([]string, 0, len(a))
	for _, s := range a {
		if _, ok := bSet[s]; ok {
			result = append(result, s)
		}
	}
	return result
}
