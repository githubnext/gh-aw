package intent

import "slices"

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

// PolicyCompiler holds policy rules for callers that still exchange policy compiler
// configuration data.
//
// WARNING: the compiled policy is advisory only. Runtime enforcement is not yet
// wired to the orchestrator — see the intent-attribution-agent-governance spec for
// the required follow-up before treating compiled policies as a security gate.
type PolicyCompiler struct {
	Rules []PolicyRule
}

// Compile applies the compiler's rules to rec and repo and returns the resulting
// ExecutionPolicy. Unlinked and ambiguous records always receive the safest
// policy regardless of configured rules (fail-closed). For all other statuses
// the first matching rule seeds the accumulator directly; subsequent matching
// rules are merged with stricter-wins semantics. If no rules match, the safest
// default policy is returned.
func (c PolicyCompiler) Compile(rec IntentRecord, repo RepositoryContext) ExecutionPolicy {
	// Fail-closed for indeterminate statuses: unlinked and ambiguous records
	// must never receive a relaxed policy from a matching wildcard rule.
	if rec.Status == AttributionUnlinked || rec.Status == AttributionAmbiguous {
		return safestDefaultPolicy()
	}

	var accumulated ExecutionPolicy
	matched := false
	for _, rule := range c.Rules {
		if !rule.matches(rec, repo) {
			continue
		}
		if !matched {
			// Seed the accumulator with a deep copy of the first matching
			// rule's policy so that permissive values (e.g. auto_merge: true,
			// max_attempts: 5) are not silently discarded by the safest-default
			// base, and so that pointer/slice fields cannot alias rule.Set.
			accumulated = deepCopyPolicy(rule.Set)
			accumulated.RuleIDs = []string{rule.ID}
			matched = true
		} else {
			accumulated = mergePolicy(accumulated, rule.Set)
			accumulated.RuleIDs = append(accumulated.RuleIDs, rule.ID)
		}
	}
	if !matched {
		return safestDefaultPolicy()
	}
	return accumulated
}

// safestDefaultPolicy returns the most restrictive execution policy: propose-only,
// no write scope, human approval required, auto-merge denied, and a single attempt.
func safestDefaultPolicy() ExecutionPolicy {
	f := false
	return ExecutionPolicy{
		Autonomy:              "propose_only",
		WriteScope:            "none",
		HumanApprovalRequired: true,
		AutoMergeAllowed:      &f,
		MaxAttempts:           1,
	}
}

// matches reports whether the rule's condition is satisfied by rec and repo.
// Empty condition fields act as wildcards. Domain, Priority, and Risk are
// matched against the record's labels. Org is matched against both the
// repository org and the repository owner.
func (r PolicyRule) matches(rec IntentRecord, repo RepositoryContext) bool {
	if r.When.Domain != "" && !slices.Contains(rec.Labels, r.When.Domain) {
		return false
	}
	if r.When.Priority != "" && !slices.Contains(rec.Labels, r.When.Priority) {
		return false
	}
	if r.When.Risk != "" && !slices.Contains(rec.Labels, r.When.Risk) {
		return false
	}
	if r.When.Org != "" && r.When.Org != repo.Org && r.When.Org != repo.Owner {
		return false
	}
	return true
}

// deepCopyPolicy returns an independent copy of p with pointer and slice fields
// freshly allocated, so that mutations to the copy cannot affect the original.
func deepCopyPolicy(p ExecutionPolicy) ExecutionPolicy {
	result := p
	if p.AutoMergeAllowed != nil {
		v := *p.AutoMergeAllowed
		result.AutoMergeAllowed = &v
	}
	result.AllowedTools = cloneStrings(p.AllowedTools)
	result.DeniedTools = cloneStrings(p.DeniedTools)
	result.RequiredChecks = cloneStrings(p.RequiredChecks)
	result.RuleIDs = cloneStrings(p.RuleIDs)
	return result
}

// mergePolicy overlays fragment onto base, preserving the stricter value for each
// field. String fields adopt the fragment value when non-empty. Boolean gates are
// ORed (human approval) or ANDed (auto-merge). Numeric limits take the minimum.
func mergePolicy(base, fragment ExecutionPolicy) ExecutionPolicy {
	result := base
	if fragment.Autonomy != "" {
		result.Autonomy = fragment.Autonomy
	}
	if fragment.WriteScope != "" {
		result.WriteScope = fragment.WriteScope
	}
	if fragment.HumanApprovalRequired {
		result.HumanApprovalRequired = true
	}
	if fragment.AutoMergeAllowed != nil {
		if result.AutoMergeAllowed == nil || (!*fragment.AutoMergeAllowed && *result.AutoMergeAllowed) {
			v := *fragment.AutoMergeAllowed
			result.AutoMergeAllowed = &v
		}
	}
	if fragment.MaxAttempts > 0 && fragment.MaxAttempts < result.MaxAttempts {
		result.MaxAttempts = fragment.MaxAttempts
	}
	result.AllowedTools = append(cloneStrings(base.AllowedTools), fragment.AllowedTools...)
	result.DeniedTools = append(cloneStrings(base.DeniedTools), fragment.DeniedTools...)
	result.RequiredChecks = append(cloneStrings(base.RequiredChecks), fragment.RequiredChecks...)
	return result
}
