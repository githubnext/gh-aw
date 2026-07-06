package intent

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
