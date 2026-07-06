package intent

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var policyLog = logger.New("intent:policy")

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

// scopePriorityOrder maps scope names to their precedence rank.
// Higher rank means the scope takes precedence over lower ranks.
// Rules must be applied from highest to lowest precedence so that a higher-
// precedence scope (e.g. organization) seeds the policy before lower-precedence
// rules (e.g. intent) can only narrow it.
var scopePriorityOrder = map[string]int{
	"organization": 4, // highest: org-wide rules always dominate
	"repository":   3, // repo-specific constraints narrow org rules
	"intent":       2, // intent-level rules narrow within a repo
	"workflow":     1, // workflow defaults are lowest named scope
	"":             0, // unscoped rules are lowest priority of all
}

// ExecutionPolicy governs what an agent may do for a given intent.
//
// WARNING: PolicyCompiler is advisory only. All fields except Autonomy are
// compiled and recorded for audit but are NOT yet wired into runtime enforcement.
// Do not rely on this policy to gate actual tool calls or merge operations until
// Authorizer.AuthorizeTool is implemented and integrated into the execution path.
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
// Rules are sorted by scope precedence (organization > repository > intent > workflow)
// before merging; within the same scope, declaration order is preserved.
//
// WARNING: the compiled policy is advisory only. Runtime enforcement is not yet
// wired to the orchestrator — see the intent-attribution-agent-governance spec for
// the required follow-up before treating compiled policies as a security gate.
type PolicyCompiler struct {
	Rules []PolicyRule
}

// Compile returns the most restrictive policy produced by merging all matching rules.
// If no rules match, the safe fail-closed default is returned. When rules do match,
// they are first sorted by scope precedence so that organization constraints seed the
// policy before repository or intent rules can only narrow them. The first rule's
// policy acts as the baseline (so rules can express less-restrictive-than-safe values
// such as supervised autonomy or auto_merge_allowed=true); subsequent rules are merged
// using more-restrictive-wins logic. Fields left unset by all rules receive fail-closed
// defaults so that an incomplete rule set never grants open access.
func (c *PolicyCompiler) Compile(record IntentRecord, repo RepositoryContext) ExecutionPolicy {
	policyLog.Printf("Compiling execution policy: total_rules=%d, repo=%s/%s", len(c.Rules), repo.Owner, repo.Name)

	var matched []PolicyRule
	for _, rule := range c.Rules {
		if rule.When.Matches(record, repo) {
			matched = append(matched, rule)
		}
	}

	if len(matched) == 0 {
		policyLog.Print("No rules matched; returning fail-closed default policy")
		return safestDefaultPolicy()
	}

	// Sort by scope precedence (highest first) so that organization rules seed
	// the policy before repository or intent rules. slices.SortStableFunc preserves
	// declaration order within the same scope.
	slices.SortStableFunc(matched, func(a, b PolicyRule) int {
		return scopePriorityOrder[b.Scope] - scopePriorityOrder[a.Scope]
	})

	// Seed from the first matching rule's policy fragment, not from the safest default.
	// This allows higher-precedence rules to establish a less-restrictive-than-safe
	// baseline (e.g. supervised autonomy) that lower-precedence rules can only tighten.
	policy := matched[0].Set
	policy.RuleIDs = []string{matched[0].ID}
	policyLog.Printf("Matched %d rule(s); seeding from rule %s (scope=%q)", len(matched), matched[0].ID, matched[0].Scope)

	for _, rule := range matched[1:] {
		policyLog.Printf("Merging rule %s (scope=%q) into accumulated policy", rule.ID, rule.Scope)
		policy = mergePolicy(policy, rule.Set)
		policy.RuleIDs = append(policy.RuleIDs, rule.ID)
	}

	compiled := applyFailClosedDefaults(policy)
	policyLog.Printf("Compiled policy: autonomy=%q, write_scope=%q, human_approval=%v, rule_ids=%v", compiled.Autonomy, compiled.WriteScope, compiled.HumanApprovalRequired, compiled.RuleIDs)
	return compiled
}

// applyFailClosedDefaults fills in safe fail-closed values for any ExecutionPolicy field
// that rules left unset (at its zero value). This ensures an incomplete rule set never
// inadvertently grants open access.
//
// String fields (Autonomy, WriteScope) and MaxAttempts use non-zero safe defaults, so
// an unset zero value is detected and replaced. HumanApprovalRequired is also set here:
// its safe default (true) differs from its zero value (false), so any compiled policy
// that has not explicitly set it via the OR merge path is upgraded to true (fail-closed).
// AutoMergeAllowed uses *bool; nil (unset) is replaced with false (deny auto-merge).
func applyFailClosedDefaults(p ExecutionPolicy) ExecutionPolicy {
	safe := safestDefaultPolicy()
	if p.Autonomy == "" {
		p.Autonomy = safe.Autonomy
	}
	if p.WriteScope == "" {
		p.WriteScope = safe.WriteScope
	}
	// HumanApprovalRequired: override false (zero) to the safe default (true).
	// Rules that want to allow unapproved execution must set AutoMergeAllowed instead;
	// this field cannot currently be relaxed below the fail-closed default.
	if !p.HumanApprovalRequired {
		p.HumanApprovalRequired = safe.HumanApprovalRequired
	}
	// AutoMergeAllowed: nil means no rule expressed a preference; default to false.
	if p.AutoMergeAllowed == nil {
		p.AutoMergeAllowed = safe.AutoMergeAllowed
	}
	if p.MaxAttempts == 0 {
		p.MaxAttempts = safe.MaxAttempts
	}
	return p
}

// safestDefaultPolicy returns the most restrictive ExecutionPolicy baseline.
// Unknown or ambiguous intent must not grant elevated authority.
func safestDefaultPolicy() ExecutionPolicy {
	return ExecutionPolicy{
		Autonomy:              "propose_only",
		WriteScope:            "none",
		HumanApprovalRequired: true,
		AutoMergeAllowed:      new(false),
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

	// AutoMergeAllowed uses *bool so that "not set" (nil) is distinct from "explicitly
	// denied" (false). More-restrictive-wins semantics:
	//   nil + nil     → nil  (neither expressed a preference)
	//   nil + *v      → *v   (adopt the incoming explicit value)
	//   *v + nil      → *v   (keep the existing explicit value)
	//   *true + *true → *true
	//   any + *false  → *false (false is more restrictive)
	//   *false + any  → *false
	switch {
	case existing.AutoMergeAllowed == nil && incoming.AutoMergeAllowed == nil:
		// leave result.AutoMergeAllowed as nil
	case existing.AutoMergeAllowed == nil:
		result.AutoMergeAllowed = incoming.AutoMergeAllowed
	case incoming.AutoMergeAllowed == nil:
		// result already holds existing.AutoMergeAllowed from `result := existing` above
	default:
		// both sides have explicit values; AND (false wins)
		v := *existing.AutoMergeAllowed && *incoming.AutoMergeAllowed
		result.AutoMergeAllowed = new(v)
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

	// AllowedTools uses nil vs non-nil to distinguish unrestricted from restricted:
	//   nil          → unrestricted (no tool filter)
	//   []string{}   → deny-all (empty set; more restrictive than any non-empty set)
	//   non-empty    → restricted to the listed tools
	//
	// More-restrictive-wins semantics:
	//   unrestricted + unrestricted → unrestricted (nil)
	//   unrestricted + restricted   → adopt the restriction
	//   restricted   + unrestricted → keep the restriction
	//   restricted_A + restricted_B → intersection(A, B); empty intersection → deny-all ([]string{})
	//   any combination with deny-all → deny-all
	existingRestricts := existing.AllowedTools != nil
	incomingRestricts := incoming.AllowedTools != nil

	switch {
	case !existingRestricts && !incomingRestricts:
		result.AllowedTools = nil
	case !existingRestricts:
		result.AllowedTools = slices.Clone(incoming.AllowedTools)
	case !incomingRestricts:
		// result was initialized from existing at the top of this function (`result := existing`),
		// so result.AllowedTools already holds the existing restriction. No change needed.
	case len(existing.AllowedTools) == 0 || len(incoming.AllowedTools) == 0:
		// At least one side is deny-all; deny-all is always more restrictive.
		result.AllowedTools = []string{}
	default:
		// Both sides restrict to non-empty sets; use the intersection.
		var intersection []string
		for _, tool := range existing.AllowedTools {
			if slices.Contains(incoming.AllowedTools, tool) {
				intersection = append(intersection, tool)
			}
		}
		// An empty intersection means no tool satisfies both restrictions: deny all.
		// Use []string{} (non-nil) to signal deny-all rather than nil (unrestricted).
		if intersection == nil {
			result.AllowedTools = []string{}
		} else {
			result.AllowedTools = intersection
		}
	}

	return result
}
