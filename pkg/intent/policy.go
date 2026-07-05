package intent

import "slices"

// autonomyOrder maps autonomy level names to their restrictiveness rank.
// Higher rank means more restrictive.
var autonomyOrder = map[string]int{
	"propose_only": 3,
	"supervised":   2,
	"bounded":      1,
	"":             0,
}

// writeScopeOrder maps write_scope values to their restrictiveness rank.
// Higher rank means more restrictive.
var writeScopeOrder = map[string]int{
	"none":           3,
	"feature_branch": 2,
	"any_branch":     1,
	"":               0,
}

// ExecutionPolicy governs what an agent may do for a given intent.
type ExecutionPolicy struct {
	Autonomy string `json:"autonomy"`

	AllowedTools []string `json:"allowed_tools,omitempty"`
	DeniedTools  []string `json:"denied_tools,omitempty"`

	WriteScope string `json:"write_scope"`

	RequiredChecks []string `json:"required_checks,omitempty"`

	HumanApprovalRequired bool `json:"human_approval_required"`
	AutoMergeAllowed      bool `json:"auto_merge_allowed"`

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

// Matches returns true when the condition is satisfied by the given intent and repository.
// Labels are matched as flat strings. If a record carries labels like ["security","p1","critical"],
// the Domain/Priority/Risk fields each check for the presence of their value anywhere in that
// slice. Callers must ensure label values are unique across dimensions (e.g. no priority value
// that could collide with a domain value) to avoid false positives.
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
	if c.Org != "" && c.Org != repo.Org && c.Org != repo.Owner {
		return false
	}
	return true
}

// PolicyCompiler compiles a set of rules into an ExecutionPolicy for a given intent.
// Rules are applied in declaration order; each merged step preserves stricter
// higher-precedence constraints.
type PolicyCompiler struct {
	Rules []PolicyRule
}

// Compile returns the most restrictive policy produced by merging all matching rules.
// The base is always the safe default policy. Rules are processed in order so that
// earlier (higher-precedence) rules cannot be weakened by later (lower-precedence) rules.
func (c *PolicyCompiler) Compile(record IntentRecord, repo RepositoryContext) ExecutionPolicy {
	policy := safestDefaultPolicy()

	for _, rule := range c.Rules {
		if rule.When.Matches(record, repo) {
			policy = mergePolicy(policy, rule.Set)
			policy.RuleIDs = append(policy.RuleIDs, rule.ID)
		}
	}

	return policy
}

// safestDefaultPolicy returns the most restrictive ExecutionPolicy baseline.
// Unknown or ambiguous intent must not grant elevated authority.
func safestDefaultPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Autonomy:              "propose_only",
		WriteScope:            "none",
		HumanApprovalRequired: true,
		AutoMergeAllowed:      false,
		MaxAttempts:           1,
	}
}

// mergePolicy merges a new policy fragment into an existing accumulated policy,
// always preserving the more restrictive value for each field.
//
// The existing policy represents already-accumulated (higher-precedence) constraints.
// The incoming fragment represents a lower-precedence rule's desired settings.
// The result must never be less restrictive than the existing policy.
func mergePolicy(existing, incoming ExecutionPolicy) ExecutionPolicy {
	result := existing

	// Autonomy: keep the more restrictive level.
	if autonomyOrder[incoming.Autonomy] > autonomyOrder[existing.Autonomy] {
		result.Autonomy = incoming.Autonomy
	}

	// WriteScope: keep the more restrictive scope.
	if writeScopeOrder[incoming.WriteScope] > writeScopeOrder[existing.WriteScope] {
		result.WriteScope = incoming.WriteScope
	}

	// HumanApprovalRequired: true is more restrictive; use OR.
	if incoming.HumanApprovalRequired {
		result.HumanApprovalRequired = true
	}

	// AutoMergeAllowed: false is more restrictive; use AND.
	if !incoming.AutoMergeAllowed {
		result.AutoMergeAllowed = false
	}

	// MaxAttempts: lower is more restrictive; keep the minimum of both if both are set.
	if incoming.MaxAttempts > 0 {
		if result.MaxAttempts == 0 || incoming.MaxAttempts < result.MaxAttempts {
			result.MaxAttempts = incoming.MaxAttempts
		}
	}

	// RequiredChecks: union of both lists (adding checks is always more restrictive).
	for _, check := range incoming.RequiredChecks {
		if !slices.Contains(result.RequiredChecks, check) {
			result.RequiredChecks = append(result.RequiredChecks, check)
		}
	}

	// DeniedTools: union of both lists (denying more tools is always more restrictive).
	for _, tool := range incoming.DeniedTools {
		if !slices.Contains(result.DeniedTools, tool) {
			result.DeniedTools = append(result.DeniedTools, tool)
		}
	}

	// AllowedTools: if neither restricts tools, stay unrestricted.
	// If the existing policy restricts, keep its restriction (higher-precedence wins).
	// If only the incoming policy restricts, adopt that restriction.
	// If both restrict, use the intersection (more restrictive).
	if len(existing.AllowedTools) == 0 && len(incoming.AllowedTools) > 0 {
		result.AllowedTools = slices.Clone(incoming.AllowedTools)
	} else if len(existing.AllowedTools) > 0 && len(incoming.AllowedTools) > 0 {
		var intersection []string
		for _, tool := range existing.AllowedTools {
			if slices.Contains(incoming.AllowedTools, tool) {
				intersection = append(intersection, tool)
			}
		}
		result.AllowedTools = intersection
	}
	// If only existing restricts, result.AllowedTools already has the existing value.

	return result
}
