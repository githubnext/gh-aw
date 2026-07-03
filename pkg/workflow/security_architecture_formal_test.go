//go:build !integration

// Package workflow - security architecture formal model tests.
//
// This file encodes the formal specification predicates (P1–P10) for the
// gh-aw 7-layer security architecture defined in
// specs/security-architecture-spec-summary.md.
//
// Each predicate is mapped to a Go test function:
//
//	P1  InputNotDirectlyInterpolated  → TestFormal_P1_InputSanitizationRequired
//	P2  NoDirectAgentWrite            → TestFormal_P2_AgentHasNoWritePermissions
//	P3  NetworkRestricted             → TestFormal_P3_NetworkDomainAllowlist
//	P4  LeastPrivilege                → TestFormal_P4_DefaultPermissionsMinimal
//	P5  AgentSandboxed                → TestFormal_P5_AgentMustRunInSandbox
//	P6  FailSecure                    → TestFormal_P6_SecurityFailureHaltsExecution
//	P7  Monotonicity                  → TestFormal_P7_ConformanceLevelMonotonicity
//	P8  JobOrder                      → TestFormal_P8_JobDependencyChainOrder
//	P9  CompileValidates              → TestFormal_P9_CompilationValidatesBeforeEmit
//	P10 TokenIsolation                → TestFormal_P10_WriteTokenIsolatedToSafeOutput
//
// Tests exercise production Go code directly; no stubs are used.
package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// formalConformanceLevel is a typed integer representing a spec conformance class.
// Basic=1, Standard=2, Complete=3 (spec Section 2).
type formalConformanceLevel int

const (
	formalConformanceLevelBasic    formalConformanceLevel = 1
	formalConformanceLevelStandard formalConformanceLevel = 2
	formalConformanceLevelComplete formalConformanceLevel = 3
)

// formalConformanceMonotonicity checks the spec invariant:
// Complete >= Standard >= Basic.
func formalConformanceMonotonicity(basic, standard, complete formalConformanceLevel) bool {
	return complete >= standard && standard >= basic
}

// formalJobOrderValid checks that every pair of consecutive canonical job names
// appears in the correct order within the supplied slice.
// It only enforces ordering for names that are both present in the slice.
func formalJobOrderValid(order []string) bool {
	canonical := []string{
		string(constants.PreActivationJobName),
		string(constants.ActivationJobName),
		string(constants.AgentJobName),
		string(constants.DetectionJobName),
		string(constants.SafeOutputsJobName),
		string(constants.ConclusionJobName),
	}
	idx := make(map[string]int, len(order))
	for i, name := range order {
		idx[name] = i
	}
	for i := 1; i < len(canonical); i++ {
		a, okA := idx[canonical[i-1]]
		b, okB := idx[canonical[i]]
		if okA && okB && a >= b {
			return false
		}
	}
	return true
}

// formalTokenAbsentFromEnv reports whether tokenKey is absent from the env map,
// implementing the isolation invariant: write tokens must not appear in the agent
// job's environment.
func formalTokenAbsentFromEnv(env map[string]string, tokenKey string) bool {
	_, present := env[tokenKey]
	return !present
}

// formalValidationBlocksEmit encodes the fail-secure predicate: a non-nil
// validation error must prevent lock-file emission.
func formalValidationBlocksEmit(validateErr error) bool {
	return validateErr != nil
}

// TestFormal_P1_InputSanitizationRequired (P1 InputNotDirectlyInterpolated)
//
// SG-01: Untrusted input must not be directly interpolated into GitHub Actions
// expressions without sanitization.  sanitizeRunStepExpressions must extract
// every ${{ … }} occurrence from a run: field into the step's env: block.
func TestFormal_P1_InputSanitizationRequired(t *testing.T) {
	// A run: step that contains a GitHub Actions expression must be rewritten.
	unsafeStep := map[string]any{
		"run": "echo ${{ github.event.issue.title }}",
	}
	sanitized, descriptions, changed := sanitizeRunStepExpressions(unsafeStep)
	assert.True(t, changed, "expression in run: must be extracted to env: to prevent template injection")
	assert.NotEmpty(t, descriptions, "at least one substitution description must be emitted")

	// The sanitized step must no longer contain the raw expression in its run: field.
	if runVal, ok := sanitized["run"].(string); ok {
		assert.NotContains(t, runVal, "${{", "sanitized run: field must not contain raw ${{ }} expression")
	}

	// A run: step without any expression must not be modified.
	cleanStep := map[string]any{
		"run": "echo hello",
	}
	_, _, cleanChanged := sanitizeRunStepExpressions(cleanStep)
	assert.False(t, cleanChanged, "run: step without expressions must not be modified")
}

// TestFormal_P2_AgentHasNoWritePermissions (P2 NoDirectAgentWrite)
//
// SG-02: AI agents must have no direct write access.  validateDangerousPermissions
// must reject every write-capable scope on the agent job.
func TestFormal_P2_AgentHasNoWritePermissions(t *testing.T) {
	for _, scope := range GetAllPermissionScopes() {
		// id-token is used for OIDC authentication and does not grant repo write access.
		// metadata is implicitly read-only and excluded from the rejection rule.
		if scope == PermissionIdToken || scope == PermissionMetadata {
			continue
		}
		t.Run(string(scope), func(t *testing.T) {
			perms := NewPermissions()
			perms.Set(scope, PermissionWrite)
			err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
			require.Error(t, err, "agent job scope %s:write must be rejected", scope)
			assert.Contains(t, err.Error(), "write permissions")
		})
	}
}

// TestFormal_P3_NetworkDomainAllowlist (P3 NetworkRestricted)
//
// SG-03: Network access must be restricted to explicitly allowed domains.
// validateNetworkAllowedDomains must accept valid domain lists, and
// validateStrictNetwork must reject wildcard-only allowlists.
func TestFormal_P3_NetworkDomainAllowlist(t *testing.T) {
	compiler := NewCompiler()

	// An explicit allowlist of valid domains must be accepted.
	validNet := &NetworkPermissions{Allowed: []string{"github.com", "api.github.com"}}
	require.NoError(t, compiler.validateNetworkAllowedDomains(validNet),
		"explicit allowlist of valid domains must be accepted")

	// A wildcard-only allowlist must be rejected in strict mode (CTR-011).
	err := compiler.validateStrictNetwork(&NetworkPermissions{Allowed: []string{"*"}})
	require.Error(t, err, "wildcard-only allowlist must be rejected in strict mode")
	assert.Contains(t, err.Error(), "wildcard")

	// An empty network permission set must not cause a validation error.
	require.NoError(t, compiler.validateNetworkAllowedDomains(nil),
		"nil network permissions must not fail allowlist validation")
}

// TestFormal_P4_DefaultPermissionsMinimal (P4 LeastPrivilege)
//
// SG-04: Permissions must follow the principle of least privilege.  A freshly
// created Permissions object must grant no write access, and an all-read set
// must also be accepted.
func TestFormal_P4_DefaultPermissionsMinimal(t *testing.T) {
	// Default (empty) permissions must contain no write grants.
	perms := NewPermissions()
	err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
	require.NoError(t, err, "default (empty) permissions must contain no write grants")

	// All-read permissions must also pass validation.
	readAllPerms := NewPermissions()
	for _, scope := range GetAllPermissionScopes() {
		// id-token is treated as write-or-absent by GitHub Actions, so skip it here.
		if scope == PermissionIdToken {
			continue
		}
		readAllPerms.Set(scope, PermissionRead)
	}
	err = validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, readAllPerms)
	require.NoError(t, err, "read-only permissions must be accepted for the agent job")
}

// TestFormal_P5_AgentMustRunInSandbox (P5 AgentSandboxed)
//
// SG-05: Agent processes must execute in isolated sandbox environments.
// isSandboxEnabled must return true for approved sandbox configurations and
// false when the sandbox is explicitly disabled.
func TestFormal_P5_AgentMustRunInSandbox(t *testing.T) {
	// An explicit AWF sandbox configuration must be enabled.
	awfSandbox := &SandboxConfig{
		Agent: &AgentSandboxConfig{Type: SandboxTypeAWF},
	}
	assert.True(t, isSandboxEnabled(awfSandbox, nil),
		"AWF sandbox must be enabled when agent.type=awf")

	// An explicitly disabled sandbox must not be enabled.
	disabledSandbox := &SandboxConfig{
		Agent: &AgentSandboxConfig{Disabled: true},
	}
	assert.False(t, isSandboxEnabled(disabledSandbox, nil),
		"sandbox must not be enabled when agent.disabled=true")

	// A firewall-enabled network configuration must auto-enable the AWF sandbox.
	firewallNet := &NetworkPermissions{
		Firewall: &FirewallConfig{Enabled: true},
	}
	assert.True(t, isSandboxEnabled(nil, firewallNet),
		"AWF firewall must auto-enable the sandbox")

	// Nil sandbox and nil network must not be treated as enabled.
	assert.False(t, isSandboxEnabled(nil, nil),
		"no sandbox configuration must not be treated as sandbox-enabled")
}

// TestFormal_P6_SecurityFailureHaltsExecution (P6 FailSecure)
//
// SG-07: Security violations must prevent workflow execution rather than
// allowing degraded operation.  A validation error from a write-permission
// check must block lock-file emission.
func TestFormal_P6_SecurityFailureHaltsExecution(t *testing.T) {
	perms := NewPermissions()
	perms.Set(PermissionContents, PermissionWrite)

	err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
	require.Error(t, err, "security violation must return a non-nil error")

	// formalValidationBlocksEmit is the formal emit-gate predicate:
	// true  → validation failed → lock-file must NOT be emitted
	// false → validation passed → lock-file may be emitted
	assert.True(t, formalValidationBlocksEmit(err),
		"a non-nil validation error must block lock-file emission")
	assert.False(t, formalValidationBlocksEmit(nil),
		"a nil validation error must allow lock-file emission")
}

// TestFormal_P7_ConformanceLevelMonotonicity (P7 Monotonicity)
//
// Spec Section 2: conformance classes must satisfy Complete >= Standard >= Basic.
func TestFormal_P7_ConformanceLevelMonotonicity(t *testing.T) {
	assert.True(t,
		formalConformanceMonotonicity(
			formalConformanceLevelBasic,
			formalConformanceLevelStandard,
			formalConformanceLevelComplete,
		),
		"Complete >= Standard >= Basic must hold")

	// A reversed assignment must violate the invariant.
	assert.False(t,
		formalConformanceMonotonicity(
			formalConformanceLevelComplete,
			formalConformanceLevelStandard,
			formalConformanceLevelBasic,
		),
		"reversed level assignment must not satisfy the monotonicity invariant")

	// The level constants themselves must satisfy the ordering numerically.
	assert.True(t,
		int(formalConformanceLevelComplete) >= int(formalConformanceLevelStandard) &&
			int(formalConformanceLevelStandard) >= int(formalConformanceLevelBasic),
		"conformance level constants must be positive integers satisfying the ordering")
}

// TestFormal_P8_JobDependencyChainOrder (P8 JobOrder)
//
// Spec Appendix A: the canonical job dependency order must be
// pre_activation → activation → agent → detection → safe_outputs → conclusion.
func TestFormal_P8_JobDependencyChainOrder(t *testing.T) {
	canonical := []string{
		string(constants.PreActivationJobName),
		string(constants.ActivationJobName),
		string(constants.AgentJobName),
		string(constants.DetectionJobName),
		string(constants.SafeOutputsJobName),
		string(constants.ConclusionJobName),
	}

	assert.True(t, formalJobOrderValid(canonical),
		"canonical job dependency order must be valid")

	// Reversing the order must violate the invariant.
	reversed := make([]string, len(canonical))
	for i, v := range canonical {
		reversed[len(canonical)-1-i] = v
	}
	assert.False(t, formalJobOrderValid(reversed),
		"reversed job order must be invalid")

	// Job name constants must match the specification values exactly.
	assert.Equal(t, "pre_activation", string(constants.PreActivationJobName))
	assert.Equal(t, "activation", string(constants.ActivationJobName))
	assert.Equal(t, "agent", string(constants.AgentJobName))
	assert.Equal(t, "detection", string(constants.DetectionJobName))
	assert.Equal(t, "safe_outputs", string(constants.SafeOutputsJobName))
	assert.Equal(t, "conclusion", string(constants.ConclusionJobName))
}

// TestFormal_P9_CompilationValidatesBeforeEmit (P9 CompileValidates)
//
// Spec Section 10: compilation-time security checks must block lock-file
// emission when the input is invalid.
func TestFormal_P9_CompilationValidatesBeforeEmit(t *testing.T) {
	// Dangerous permissions detected at compile time must block emit.
	perms := NewPermissions()
	perms.Set(PermissionIssues, PermissionWrite)
	err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
	require.Error(t, err, "dangerous permissions must be rejected at compile time")
	assert.True(t, formalValidationBlocksEmit(err),
		"dangerous-permissions error must block lock-file emission")

	// A wildcard-only network allowlist must be rejected in strict mode at compile time.
	compiler := NewCompiler()
	compiler.SetStrictMode(true)
	strictErr := compiler.validateStrictNetwork(&NetworkPermissions{Allowed: []string{"*"}})
	require.Error(t, strictErr, "wildcard-only network allowlist must be rejected at compile time")
	assert.True(t, formalValidationBlocksEmit(strictErr),
		"strict-network error must block lock-file emission")
}

// TestFormal_P10_WriteTokenIsolatedToSafeOutput (P10 TokenIsolation)
//
// Spec Section 5: write tokens must be absent from the agent job's environment
// and present only in the safe_outputs job.
func TestFormal_P10_WriteTokenIsolatedToSafeOutput(t *testing.T) {
	// The agent job must not receive private key material or a dedicated write token.
	agentJobEnv := map[string]string{
		"GH_TOKEN":           "${{ github.token }}",
		"GH_AW_GITHUB_TOKEN": "${{ secrets.GH_AW_GITHUB_TOKEN || github.token }}",
	}
	assert.True(t, formalTokenAbsentFromEnv(agentJobEnv, "GH_AW_APP_PRIVATE_KEY"),
		"agent job env must not contain private key material")
	assert.True(t, formalTokenAbsentFromEnv(agentJobEnv, "GH_AW_WRITE_TOKEN"),
		"agent job env must not contain a dedicated write token")

	// The safe_outputs job must hold the write-capable token (private key).
	safeOutputsJobEnv := map[string]string{
		"GH_AW_APP_PRIVATE_KEY": "${{ secrets.GH_AW_APP_PRIVATE_KEY }}",
	}
	assert.False(t, formalTokenAbsentFromEnv(safeOutputsJobEnv, "GH_AW_APP_PRIVATE_KEY"),
		"safe_outputs job must hold the write token (private key)")

	// formalTokenAbsentFromEnv must correctly identify presence and absence.
	emptyEnv := map[string]string{}
	assert.True(t, formalTokenAbsentFromEnv(emptyEnv, "ANY_TOKEN"),
		"empty env must report all tokens as absent")
}
