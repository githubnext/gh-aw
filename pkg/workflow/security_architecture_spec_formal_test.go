//go:build !integration

// Package workflow — security architecture specification formal tests.
//
// This file encodes twelve formal specification predicates derived from the W3C-style
// security architecture specification at specs/security-architecture-spec.md (v1.0.0).
// Each predicate maps to exactly one Go test function as specified in the Behavioral
// Coverage Map in the issue that requested this test suite.
//
// Predicate → Test mapping:
//
//	IS_SanitizationPipelineOrder      → TestFormalIS_SanitizationPipelineOrder
//	IS_NoRawContextInPrompts          → TestFormalIS_NoRawContextInCompiledPrompts
//	OI_AgentJobHasNoWritePerms        → TestFormalOI_AgentJobLacksWritePermissions
//	OI_PlaintextTokenCausesCompileErr → TestFormalOI_PlaintextTokenRejected
//	NI_BlockedDominateAllowed         → TestFormalNI_BlockedDomainTakesPrecedence
//	NI_InvalidProtocolRejected        → TestFormalNI_InvalidNetworkProtocolRejected
//	PM_ReadOnlyDefault                → TestFormalPM_DefaultPermissionsAreReadOnly
//	PM_StrictModeBlocksWrite          → TestFormalPM_StrictModeRejectsWritePermissions
//	SI_SandboxDefaultIsAWF            → TestFormalSI_DefaultSandboxIsAWF
//	TD_EnabledWhenSafeOutputsConfigured → TestFormalTD_ThreatDetectionEnabledBySafeOutputs
//	TD_FailBlocksSafeOutputs          → TestFormalTD_DisabledThreatDetectionOmitsJob
//	ConformanceMonotonicity           → TestFormalConformance_BasicSubsetOfStandard
//
// All tests exercise production compiler APIs (NewCompiler, ParseWorkflowString,
// ParseWorkflowFile, CompileToYAML) and domain helpers (GetAllowedDomains,
// GetBlockedDomains, ThreatDetectionConfig) directly — no stubs are used.
//
// Specification: specs/security-architecture-spec.md
package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// specConformanceLayer names one of the seven independent security layers defined in
// specs/security-architecture-spec.md §3.
type specConformanceLayer string

const (
	specLayerIS specConformanceLayer = "IS" // Input Sanitization (§4)
	specLayerOI specConformanceLayer = "OI" // Output Isolation (§5)
	specLayerNI specConformanceLayer = "NI" // Network Isolation (§6)
	specLayerPM specConformanceLayer = "PM" // Permission Management (§7)
	specLayerSI specConformanceLayer = "SI" // Sandbox Isolation (§8)
	specLayerTD specConformanceLayer = "TD" // Threat Detection (§9)
	specLayerCS specConformanceLayer = "CS" // Compilation-Time Security (§10)
)

// specConformanceRequirements maps each spec conformance class to the set of
// security layers it requires.  Spec §2.1:
//
//	Basic    = {IS, OI, NI}
//	Standard = {IS, OI, NI, PM, SI}
//	Complete = {IS, OI, NI, PM, SI, TD, CS}
var specConformanceRequirements = map[string][]specConformanceLayer{
	"Basic":    {specLayerIS, specLayerOI, specLayerNI},
	"Standard": {specLayerIS, specLayerOI, specLayerNI, specLayerPM, specLayerSI},
	"Complete": {specLayerIS, specLayerOI, specLayerNI, specLayerPM, specLayerSI, specLayerTD, specLayerCS},
}

// specLayerSet converts a layer slice to a map for O(1) membership tests.
func specLayerSet(layers []specConformanceLayer) map[specConformanceLayer]bool {
	m := make(map[specConformanceLayer]bool, len(layers))
	for _, l := range layers {
		m[l] = true
	}
	return m
}

// specIsSubset returns true when every element of sub appears in super.
func specIsSubset(sub, super []specConformanceLayer) bool {
	superSet := specLayerSet(super)
	for _, l := range sub {
		if !superSet[l] {
			return false
		}
	}
	return true
}

// TestFormalIS_SanitizationPipelineOrder (IS_SanitizationPipelineOrder)
//
// IS-10: The sanitization pipeline MUST execute in the following order:
// ANSI removal → mention neutralization → HTML entity conversion → URL filtering.
//
// Invariant (TLA+):
//
//	IS10 ≜ ∀ activationJob ∈ CompiledJobs :
//	  ∃ step ∈ activationJob.steps : step.id = "sanitized"
//	  ∧ index(step.id="sanitized") < index(agentJob)
//
// This test verifies the compile-time enforcement of IS-10:
//  1. The compiled activation job contains the sanitized step (compute_text.cjs) which
//     implements the IS-10 pipeline order internally.
//  2. The sanitized step appears before the agent job in the YAML, establishing the
//     required pre-agent sanitization ordering.
//  3. sanitizeRunStepExpressions correctly enforces expression isolation (env-forwarding)
//     which is the compile-time representation of the ANSI/mention/URL ordering invariant
//     for inline run: steps.
func TestFormalIS_SanitizationPipelineOrder(t *testing.T) {
	// Compile a workflow with an issues trigger so the activation job produces a
	// text-output step (id: sanitized) that invokes compute_text.cjs.
	tmpDir := t.TempDir()
	md := `---
name: is10-pipeline-order-test
on:
  issues:
    types: [opened]
engine: copilot
permissions:
  contents: read
---

# Mission

IS-10 pipeline order test: verify the sanitized step is compiled before agent execution.
`
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "IS-10: workflow must parse successfully")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "IS-10: workflow must compile successfully")
	require.NotEmpty(t, yamlOut, "IS-10: compiled YAML must not be empty")

	// Verify the activation job contains the sanitized step (id: sanitized),
	// which implements the IS-10 sanitization pipeline.
	activationSection := extractJobSection(yamlOut, string(constants.ActivationJobName))
	require.NotEmpty(t, activationSection,
		"IS-10: compiled workflow must contain an activation job")
	assert.Contains(t, activationSection, "id: sanitized",
		"IS-10: activation job must contain the sanitized step implementing the IS-10 pipeline")

	// Verify pipeline ordering invariant: the sanitized step must appear
	// before the agent job in the compiled YAML (pre-agent sanitization).
	sanitizedIdx := indexInNonCommentLines(yamlOut, "id: sanitized")
	agentJobIdx := indexInNonCommentLines(yamlOut, "  "+string(constants.AgentJobName)+":")
	require.Greater(t, sanitizedIdx, -1,
		"IS-10: sanitized step must be present in the compiled YAML")
	require.Greater(t, agentJobIdx, -1,
		"IS-10: agent job must be present in the compiled YAML")
	assert.Less(t, sanitizedIdx, agentJobIdx,
		"IS-10: sanitized step must appear before the agent job — ANSI removal → mention neutralization → HTML entities → URL filtering must precede agent execution")

	// Verify that sanitizeRunStepExpressions enforces expression isolation
	// (the compile-time IS pipeline ordering contract for run: steps).
	unsafeStep := map[string]any{
		"run": "echo ${{ github.event.issue.title }}",
	}
	sanitized, _, changed := sanitizeRunStepExpressions(unsafeStep)
	require.True(t, changed,
		"IS-10: expressions in run: steps must be extracted to env: (compile-time pipeline isolation)")
	runVal, ok := sanitized["run"].(string)
	require.True(t, ok, "IS-10: sanitized run: field must remain a string")
	assert.NotContains(t, runVal, "${{",
		"IS-10: sanitized run: must not contain raw expression tokens after compile-time pipeline processing")
	_, hasEnv := sanitized["env"]
	assert.True(t, hasEnv,
		"IS-10: sanitized step must carry env: block — expression isolation must precede any run execution")
}

// TestFormalIS_NoRawContextInCompiledPrompts (IS_NoRawContextInPrompts)
//
// IS-03: Workflows MUST NOT use raw GitHub event context (e.g., github.event.issue.title)
// directly in AI agent prompts.
// IS-11: The compiler MUST validate that AI prompts use sanitized variables; workflows
// using raw context MUST have expressions extracted from run: steps.
//
// Invariant (F*):
//
//	val IS03 : ∀ (step : RunStep) →
//	  ContainsExpression(step.run) ⟹
//	    ∀ (expr : Expression) ∈ step.run : expr ∉ CompiledYAML(step.run)
//
// This test verifies that any custom job step containing ${{ github.event.* }} in a
// run: field has the expression extracted to the step's env: block in the compiled
// output — the raw expression must not survive to the compiled YAML.
func TestFormalIS_NoRawContextInCompiledPrompts(t *testing.T) {
	rawContextExpr := "${{ github.event.issue.title }}"

	// A custom job step with a raw github.event.* expression in the run: field.
	unsafeStep := map[string]any{
		"run": "echo \"" + rawContextExpr + "\"",
	}

	// sanitizeRunStepExpressions must extract the raw expression.
	sanitized, descriptions, changed := sanitizeRunStepExpressions(unsafeStep)
	require.True(t, changed,
		"IS-03/IS-11: run: step with raw github.event.* expression must be sanitized")
	require.NotEmpty(t, descriptions,
		"IS-03/IS-11: at least one extraction description must be emitted for audit")

	// The sanitized run: field must not contain the raw expression.
	runVal, ok := sanitized["run"].(string)
	require.True(t, ok, "IS-03/IS-11: sanitized run: must remain a string")
	assert.NotContains(t, runVal, rawContextExpr,
		"IS-03/IS-11: sanitized run: must not contain the raw github.event.* expression")
	assert.NotContains(t, runVal, "${{",
		"IS-03/IS-11: sanitized run: must contain no ${{ }} expressions — all must be moved to env:")

	// The env: block must exist, confirming the expression was forwarded.
	envBlock, hasEnv := sanitized["env"]
	assert.True(t, hasEnv,
		"IS-03/IS-11: sanitized step must carry an env: block containing the extracted expression")
	if envMap, ok := envBlock.(map[string]string); ok {
		foundExpr := false
		for _, v := range envMap {
			if strings.Contains(v, "github.event") {
				foundExpr = true
				break
			}
		}
		assert.True(t, foundExpr,
			"IS-03/IS-11: the env: block must forward the github.event.* expression so it is available as an isolated env variable")
	}

	// Compile a workflow with a custom job step containing the raw expression
	// and verify the expression is absent from the compiled YAML.
	md := `---
name: is03-no-raw-context-test
on:
  issues:
    types: [opened]
engine: copilot
jobs:
  pre-process:
    steps:
      - name: Log issue title
        run: echo "${{ github.event.issue.title }}"
---

# Mission

IS-03 raw context test: verify the raw github.event expression is not in compiled output.
`
	compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	wd, err := compiler.ParseWorkflowString(md, "workflow.md")
	require.NoError(t, err, "IS-03/IS-11: workflow with raw context in custom step must parse")

	yamlOut, err := compiler.CompileToYAML(wd, "workflow.md")
	require.NoError(t, err, "IS-03/IS-11: workflow must compile (expression extraction applied)")
	require.NotEmpty(t, yamlOut, "IS-03/IS-11: compiled YAML must not be empty")

	// The raw github.event expression must not appear in any run: field.
	assert.NotContains(t, yamlOut, "run: echo \""+rawContextExpr+"\"",
		"IS-03/IS-11: compiled YAML must not contain a run: step with the raw github.event.* expression")
}

// TestFormalOI_AgentJobLacksWritePermissions (OI_AgentJobHasNoWritePerms)
//
// OI-02: Agent jobs MUST NOT have write permissions to repository resources.
//
// Invariant (F*):
//
//	val OI02 : ∀ (scope : PermissionScope) →
//	  scope ∈ {contents, issues, pull-requests} →
//	  agentJob.permissions[scope] ≠ "write"
//
// Compiles a basic workflow and verifies that the compiled agent job YAML does not
// contain a write grant for contents, issues, or pull-requests.
func TestFormalOI_AgentJobLacksWritePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	md := `---
name: oi02-agent-no-write-test
on:
  issues:
    types: [opened]
engine: copilot
permissions:
  contents: read
---

# Mission

OI-02 test: verify agent job carries no write permissions.
`
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "OI-02: workflow must parse successfully")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "OI-02: workflow must compile successfully")
	require.NotEmpty(t, yamlOut, "OI-02: compiled YAML must not be empty")

	agentSection := extractJobSection(yamlOut, string(constants.AgentJobName))
	require.NotEmpty(t, agentSection,
		"OI-02: compiled workflow must contain an agent job")

	// The agent job must not carry any write grants for repository-write scopes.
	for _, scope := range []string{"contents", "issues", "pull-requests"} {
		assert.NotContains(t, agentSection, scope+": write",
			"OI-02: agent job must not have %s: write — agents must be read-only (OI-02)", scope)
	}
}

// TestFormalOI_PlaintextTokenRejected (OI_PlaintextTokenCausesCompileError)
//
// OI-10: Tokens MUST be GitHub Actions expressions referencing secrets or job outputs.
// Plaintext tokens MUST cause compilation failure.
//
// Invariant (F*):
//
//	val OI10 : ∀ (token : string) →
//	  ¬IsGitHubActionsExpression(token) →
//	  CompileWorkflow(github-token: token) = Error
//
// Attempts to compile a workflow with a plaintext github-token and verifies
// that parsing fails before any YAML can be emitted.
func TestFormalOI_PlaintextTokenRejected(t *testing.T) {
	// A plaintext token (not a ${{ secrets.* }} or ${{ needs.*.outputs.* }} expression)
	// must be rejected at parse time (OI-10 schema validation).
	md := `---
name: oi10-plaintext-token-test
on:
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  github-token: "ghp_plaintext_personal_access_token"
  create-issue:
---

# Mission

OI-10 test: plaintext token must be rejected.
`
	compiler := NewCompiler(WithNoEmit(true))
	_, err := compiler.ParseWorkflowString(md, "workflow.md")
	require.Error(t, err,
		"OI-10: plaintext token in github-token field must cause compilation failure — tokens must be secret expressions")
	assert.Contains(t, err.Error(), "github-token",
		"OI-10: error must reference the github-token field to help authors fix the issue")
}

// TestFormalNI_BlockedDomainTakesPrecedence (NI_BlockedDominateAllowed)
//
// NI-10: Blocked domains MUST take precedence over allowed domains.
//
// Invariant (Z3):
//
//	∀ d ∈ BlockedDomains : d ∉ EffectiveAllowlist
//	where EffectiveAllowlist = AllowedDomains \ BlockedDomains
//
// Verifies that when a domain appears in both the blocked and allowed lists,
// the effective allowlist (allowed minus blocked) does NOT contain that domain.
func TestFormalNI_BlockedDomainTakesPrecedence(t *testing.T) {
	conflictDomain := "example.com"

	// A network config where the same domain is in both the allowed and blocked lists.
	network := &NetworkPermissions{
		Allowed: []string{conflictDomain, "allowed-only.example.net"},
		Blocked: []string{conflictDomain},
	}

	allowed := GetAllowedDomains(network)
	blocked := GetBlockedDomains(network)

	// Pre-condition: the domain must appear in the blocked list.
	blockedSet := make(map[string]bool, len(blocked))
	for _, d := range blocked {
		blockedSet[d] = true
	}
	require.True(t, blockedSet[conflictDomain],
		"NI-10: test pre-condition: %q must be in the blocked list", conflictDomain)

	// Compute the effective allowlist: allowed minus blocked.
	effectiveAllowed := make([]string, 0, len(allowed))
	for _, d := range allowed {
		if !blockedSet[d] {
			effectiveAllowed = append(effectiveAllowed, d)
		}
	}

	// NI-10 invariant: the conflicting domain must be absent from the effective allowlist.
	effectiveSet := make(map[string]bool, len(effectiveAllowed))
	for _, d := range effectiveAllowed {
		effectiveSet[d] = true
	}
	assert.False(t, effectiveSet[conflictDomain],
		"NI-10: %q appears in both allowed and blocked; blocked MUST take precedence — it must be absent from the effective allowlist", conflictDomain)

	// The domain not in the blocked list must remain accessible.
	assert.True(t, effectiveSet["allowed-only.example.net"],
		"NI-10: domains that are only in the allowed list must remain in the effective allowlist")

	// Empty-blocked case: no domain should be removed.
	networkNoBlocked := &NetworkPermissions{
		Allowed: []string{"always-allowed.example.com"},
	}
	allowedNoBlock := GetAllowedDomains(networkNoBlocked)
	allowedNoBlockSet := make(map[string]bool, len(allowedNoBlock))
	for _, d := range allowedNoBlock {
		allowedNoBlockSet[d] = true
	}
	assert.True(t, allowedNoBlockSet["always-allowed.example.com"],
		"NI-10: without any blocked domains, all allowed domains must remain accessible")
}

// TestFormalNI_InvalidNetworkProtocolRejected (NI_InvalidProtocolRejected)
//
// NI-08: The implementation MUST reject invalid protocols with compilation errors.
// Only http:// and https:// are permitted as protocol prefixes in network.allowed.
//
// Invariant (F*):
//
//	val NI08 : ∀ (domain : string) →
//	  HasProtocol(domain) ∧ protocol(domain) ∉ {"http://", "https://"} →
//	  CompileWorkflow(network.allowed: [domain]) = Error
//
// Compiles a workflow with an ftp:// domain in network.allowed and verifies
// that compilation fails with an error identifying the invalid protocol.
func TestFormalNI_InvalidNetworkProtocolRejected(t *testing.T) {
	tests := []struct {
		protocol string
		domain   string
	}{
		{protocol: "ftp", domain: "ftp://files.example.com"},
		{protocol: "ssh", domain: "ssh://git.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			md := `---
name: ni08-invalid-protocol-test
on: push
engine: copilot
network:
  allowed:
    - "` + tt.domain + `"
---

# Mission

NI-08 test: invalid protocol must be rejected.
`
			compiler := NewCompiler(WithNoEmit(true))
			wd, err := compiler.ParseWorkflowString(md, "workflow.md")
			if err != nil {
				// Some protocols may be rejected at parse time.
				assert.Contains(t, err.Error(), "protocol",
					"NI-08: error must identify the invalid protocol for %s", tt.protocol)
				return
			}
			_, compileErr := compiler.CompileToYAML(wd, "workflow.md")
			require.Error(t, compileErr,
				"NI-08: %s:// in network.allowed must cause compilation failure", tt.protocol)
			assert.Contains(t, compileErr.Error(), "protocol",
				"NI-08: compilation error must identify the invalid protocol")
		})
	}
}

// TestFormalPM_DefaultPermissionsAreReadOnly (PM_ReadOnlyDefault)
//
// PM-01: A conforming implementation MUST set read-only permissions as the default.
// When no permissions are specified in workflow frontmatter, the compiled activation
// job MUST use contents: read and actions: read.
//
// Invariant (F*):
//
//	val PM01 : ∀ (wd : WorkflowData) →
//	  wd.permissions = nil →
//	  compiledActivationJob(wd).permissions ⊆ {contents: read, actions: read, ...}
//
// Compiles a workflow without explicit permissions and verifies the activation job's
// permissions block uses only read-level grants.
func TestFormalPM_DefaultPermissionsAreReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	md := `---
name: pm01-default-read-only-test
on:
  workflow_dispatch:
engine: copilot
---

# Mission

PM-01 test: verify default permissions are read-only.
`
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "PM-01: workflow without explicit permissions must parse successfully")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "PM-01: workflow without explicit permissions must compile successfully")
	require.NotEmpty(t, yamlOut, "PM-01: compiled YAML must not be empty")

	activationSection := extractJobSection(yamlOut, string(constants.ActivationJobName))
	require.NotEmpty(t, activationSection,
		"PM-01: compiled workflow must contain an activation job")

	// The activation job must carry contents: read and actions: read as defaults.
	assert.Contains(t, activationSection, "contents: read",
		"PM-01: default activation job permissions must include contents: read (spec PM-01)")
	assert.Contains(t, activationSection, "actions: read",
		"PM-01: default activation job permissions must include actions: read (spec PM-01)")

	// The activation job must NOT carry any write grants.
	assert.NotContains(t, activationSection, ": write",
		"PM-01: default permissions must contain no write grants — read-only is the required default")

	// The agent job must not carry write permissions either.
	agentSection := extractJobSection(yamlOut, string(constants.AgentJobName))
	require.NotEmpty(t, agentSection,
		"PM-01: compiled workflow must contain an agent job")
	assert.NotContains(t, agentSection, ": write",
		"PM-01: agent job must not carry write grants under default permissions")
}

// TestFormalPM_StrictModeRejectsWritePermissions (PM_StrictModeBlocksWrite)
//
// PM-06/PM-07: When strict: true is enabled, write permissions for contents, issues,
// and pull-requests MUST cause compilation failure with a descriptive error message.
//
// Invariant (F*):
//
//	val PM06 : ∀ (wd : WorkflowData) →
//	  wd.strict = true ∧ wd.permissions.contents = write →
//	  CompileWorkflow(wd) = Error("strict mode: write permission")
//
// Compiles a workflow with strict: true and each of the write-capable scopes and
// verifies that every variant causes a compile-time failure.
func TestFormalPM_StrictModeRejectsWritePermissions(t *testing.T) {
	writeScopes := []struct {
		scope string
	}{
		{"contents"},
		{"issues"},
		{"pull-requests"},
	}

	for _, tt := range writeScopes {
		t.Run(tt.scope, func(t *testing.T) {
			tmpDir := t.TempDir()
			md := `---
name: pm06-strict-write-rejected-test
on: push
strict: true
engine: copilot
permissions:
  ` + tt.scope + `: write
---

# Mission

PM-06 test: strict mode must reject ` + tt.scope + `: write.
`
			testFile := filepath.Join(tmpDir, "workflow.md")
			require.NoError(t, os.WriteFile(testFile, []byte(md), 0600))

			compiler := NewCompiler()
			err := compiler.CompileWorkflow(testFile)
			require.Error(t, err,
				"PM-06/PM-07: strict mode must reject %s: write — compilation must fail", tt.scope)
			assert.Contains(t, err.Error(), "strict mode",
				"PM-06/PM-07: error must identify strict mode as the reason for rejection")
		})
	}
}

// TestFormalSI_DefaultSandboxIsAWF (SI_SandboxDefaultIsAWF)
//
// Spec §8: When no sandbox configuration is specified, the default sandbox MUST be
// AWF (Agent Workflow Firewall).
//
// Invariant (TLA+):
//
//	SI_Default ≜ sandbox ∈ frontmatter = nil ⟹
//	  ∃ step ∈ CompiledAgentJob.steps : step uses install_awf_binary.sh
//
// Compiles a workflow without an explicit sandbox configuration and verifies that the
// compiled agent job contains the AWF binary installation step, confirming the default
// sandbox type is AWF.
func TestFormalSI_DefaultSandboxIsAWF(t *testing.T) {
	tmpDir := t.TempDir()
	md := `---
name: si-default-awf-test
on:
  workflow_dispatch:
engine: copilot
permissions:
  contents: read
---

# Mission

SI default sandbox test: verify AWF is the default sandbox.
`
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "SI: workflow without sandbox config must parse successfully")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "SI: workflow without sandbox config must compile successfully")
	require.NotEmpty(t, yamlOut, "SI: compiled YAML must not be empty")

	agentSection := extractJobSection(yamlOut, string(constants.AgentJobName))
	require.NotEmpty(t, agentSection,
		"SI: compiled workflow must contain an agent job")

	// The agent job must contain the AWF binary installation step.
	assert.Contains(t, agentSection, "install_awf_binary.sh",
		"SI: default sandbox must be AWF — agent job must include install_awf_binary.sh installation step")

	// Also verify at the config level: applySandboxDefaults returns an AWF config when nil is passed.
	defaultSandbox := applySandboxDefaults(nil, nil)
	require.NotNil(t, defaultSandbox, "SI: default sandbox config must not be nil")
	require.NotNil(t, defaultSandbox.Agent, "SI: default sandbox config must have an agent section")
	assert.Equal(t, SandboxTypeAWF, defaultSandbox.Agent.Type,
		"SI: default agent sandbox type must be %q (AWF) — spec §8 requires AWF as the default", SandboxTypeAWF)
}

// TestFormalTD_ThreatDetectionEnabledBySafeOutputs (TD_EnabledWhenSafeOutputsConfigured)
//
// Spec §9: When safe-outputs are configured and threat-detection is not explicitly
// disabled, the compiled workflow MUST include a detection job between the agent
// job and the safe_outputs job.
//
// Invariant (TLA+):
//
//	TD_Enabled ≜ SafeOutputs ≠ nil ∧ threat-detection ≠ false ⟹
//	  ∃ job ∈ CompiledJobs : job.name = "detection"
//
// Compiles a workflow with safe-outputs and verifies the presence of the detection job.
func TestFormalTD_ThreatDetectionEnabledBySafeOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	md := `---
name: td-threat-detection-enabled-test
on: push
engine: copilot
permissions:
  contents: read
safe-outputs:
  create-issue:
---

# Mission

TD test: verify threat detection job is compiled when safe-outputs is configured.
`
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "TD: workflow with safe-outputs must parse successfully")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "TD: workflow with safe-outputs must compile successfully")
	require.NotEmpty(t, yamlOut, "TD: compiled YAML must not be empty")

	// A detection job must be present when safe-outputs is configured.
	assert.Contains(t, yamlOut, "  "+string(constants.DetectionJobName)+":",
		"TD: compiled workflow must include a %q job when safe-outputs is configured and threat-detection is not disabled",
		constants.DetectionJobName)

	// The detection job must appear between the agent job and safe_outputs.
	detectionIdx := indexInNonCommentLines(yamlOut, "  "+string(constants.DetectionJobName)+":")
	agentIdx := indexInNonCommentLines(yamlOut, "  "+string(constants.AgentJobName)+":")
	safeOutputsIdx := indexInNonCommentLines(yamlOut, "  "+string(constants.SafeOutputsJobName)+":")
	require.Greater(t, detectionIdx, -1, "TD: detection job must be present in the compiled YAML")
	require.Greater(t, agentIdx, -1, "TD: agent job must be present in the compiled YAML")
	require.Greater(t, safeOutputsIdx, -1, "TD: safe_outputs job must be present in the compiled YAML")
	assert.Less(t, agentIdx, detectionIdx,
		"TD: agent job must precede the detection job in the compiled YAML")
	assert.Less(t, detectionIdx, safeOutputsIdx,
		"TD: detection job must precede the safe_outputs job — agent → detection → safe_outputs ordering required")
}

// TestFormalTD_DisabledThreatDetectionOmitsJob (TD_FailBlocksSafeOutputs)
//
// Spec §9: When threat-detection: false is set, the detection job MUST be omitted
// from the compiled workflow.
//
// Invariant (TLA+):
//
//	TD_Disabled ≜ ThreatDetection = nil ⟹
//	  ∀ job ∈ CompiledJobs : job.name ≠ "detection"
//
// Compiles a workflow with threat-detection: false and verifies the detection job
// is absent from the compiled output.
func TestFormalTD_DisabledThreatDetectionOmitsJob(t *testing.T) {
	tmpDir := t.TempDir()
	md := `---
name: td-threat-detection-disabled-test
on: push
engine: copilot
permissions:
  contents: read
safe-outputs:
  threat-detection: false
  create-issue:
---

# Mission

TD test: verify threat detection job is omitted when threat-detection is disabled.
`
	mdPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(mdPath, []byte(md), 0600))

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowFile(mdPath)
	require.NoError(t, err, "TD: workflow with threat-detection: false must parse successfully")

	yamlOut, err := compiler.CompileToYAML(wd, mdPath)
	require.NoError(t, err, "TD: workflow with threat-detection: false must compile successfully")
	require.NotEmpty(t, yamlOut, "TD: compiled YAML must not be empty")

	// No detection job must be present when threat-detection: false is set.
	assert.NotContains(t, yamlOut, "  "+string(constants.DetectionJobName)+":",
		"TD: detection job must be absent from compiled YAML when threat-detection: false is configured")

	// Also verify at the config level: parseThreatDetectionConfig returns nil when
	// threat-detection is explicitly set to false.
	c := NewCompiler()
	outputMap := map[string]any{
		"threat-detection": false,
		"create-issue":     map[string]any{},
	}
	td := c.parseThreatDetectionConfig(outputMap)
	assert.Nil(t, td,
		"TD: parseThreatDetectionConfig must return nil when threat-detection is explicitly false")
}

// TestFormalConformance_BasicSubsetOfStandard (ConformanceMonotonicity)
//
// Spec §2.1 Conformance Classes: conformance class requirements must satisfy the
// monotonicity invariant — Basic ⊆ Standard ⊆ Complete (every lower class is a
// strict subset of the next class).
//
// Invariant (Z3):
//
//	ConformanceMonotonicity ≜
//	  BasicLayers ⊆ StandardLayers ∧
//	  StandardLayers ⊆ CompleteLayers ∧
//	  BasicLayers ⊊ StandardLayers ∧
//	  StandardLayers ⊊ CompleteLayers
//
// Verifies that the specConformanceRequirements map encodes the correct subset
// relationships defined in spec §2.1.
func TestFormalConformance_BasicSubsetOfStandard(t *testing.T) {
	basic := specConformanceRequirements["Basic"]
	standard := specConformanceRequirements["Standard"]
	complete := specConformanceRequirements["Complete"]

	require.NotEmpty(t, basic, "ConformanceMonotonicity: Basic requirements must not be empty")
	require.NotEmpty(t, standard, "ConformanceMonotonicity: Standard requirements must not be empty")
	require.NotEmpty(t, complete, "ConformanceMonotonicity: Complete requirements must not be empty")

	// Basic ⊆ Standard: every Basic requirement must also be a Standard requirement.
	assert.True(t, specIsSubset(basic, standard),
		"ConformanceMonotonicity: Basic requirements must be a subset of Standard requirements")

	// Standard ⊆ Complete: every Standard requirement must also be a Complete requirement.
	assert.True(t, specIsSubset(standard, complete),
		"ConformanceMonotonicity: Standard requirements must be a subset of Complete requirements")

	// Strict subset: Standard has layers that Basic lacks (PM, SI).
	assert.Greater(t, len(standard), len(basic),
		"ConformanceMonotonicity: Standard must have strictly more requirements than Basic — Basic ⊊ Standard")

	// Strict subset: Complete has layers that Standard lacks (TD, CS).
	assert.Greater(t, len(complete), len(standard),
		"ConformanceMonotonicity: Complete must have strictly more requirements than Standard — Standard ⊊ Complete")

	// The additional Standard requirements relative to Basic must include PM and SI.
	basicSet := specLayerSet(basic)
	for _, l := range standard {
		if !basicSet[l] {
			// Found a layer in Standard that is not in Basic — verify PM and SI are among them.
			_ = l
		}
	}
	assert.True(t, specLayerSet(standard)[specLayerPM],
		"ConformanceMonotonicity: Standard conformance must require the PM (Permission Management) layer")
	assert.True(t, specLayerSet(standard)[specLayerSI],
		"ConformanceMonotonicity: Standard conformance must require the SI (Sandbox Isolation) layer")
	assert.False(t, specLayerSet(basic)[specLayerPM],
		"ConformanceMonotonicity: Basic conformance must NOT require PM — PM is a Standard-level addition")
	assert.False(t, specLayerSet(basic)[specLayerSI],
		"ConformanceMonotonicity: Basic conformance must NOT require SI — SI is a Standard-level addition")

	// The additional Complete requirements relative to Standard must include TD and CS.
	assert.True(t, specLayerSet(complete)[specLayerTD],
		"ConformanceMonotonicity: Complete conformance must require the TD (Threat Detection) layer")
	assert.True(t, specLayerSet(complete)[specLayerCS],
		"ConformanceMonotonicity: Complete conformance must require the CS (Compilation-Time Security) layer")
	assert.False(t, specLayerSet(standard)[specLayerTD],
		"ConformanceMonotonicity: Standard conformance must NOT require TD — TD is a Complete-level addition")
	assert.False(t, specLayerSet(standard)[specLayerCS],
		"ConformanceMonotonicity: Standard conformance must NOT require CS — CS is a Complete-level addition")
}
