//go:build !integration

package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- Rule-model formal tests --------------------------------------------------
// These tests encode the structural invariants of the CTR rule catalog
// defined in specs/compiler-threat-detection-spec.md §4–5.
// They can be selected with `go test ./pkg/workflow/... -run Formal`.

// ctrRuleStub captures the minimum model fields required by P1 (RuleModelComplete).
type ctrRuleStub struct {
	id            string // e.g. "CTR-013"
	threatClass   string // e.g. "Argument Injection"
	detectionCond string // brief description of the detection condition
	action        string // "reject" | "warn" | "rewrite" | "warn/reject" | "reject/warn"
	evidence      string // observable diagnostic or stable CTR identifier
	implRef       string // primary implementation file reference
	deprecated    bool   // whether the rule is formally deprecated
}

// testCatalog mirrors the 21-rule normative catalog in spec §5.1.
// Each entry must have all non-empty required fields to satisfy P1.
var testCatalog = []ctrRuleStub{
	{"CTR-001", "Privilege Escalation", "write permissions in non-safe-outputs job", "reject", "CTR-001", "dangerous_permissions_validation.go", false},
	{"CTR-002", "Unpinned Action Integrity", "action referenced by tag or branch in strict mode", "reject", "CTR-002", "action_pins.go", false},
	{"CTR-003", "Unsafe Tool Scope Expansion", "wildcard tool permissions violating policy", "reject/warn", "CTR-003", "tools_validation.go", false},
	{"CTR-004", "Sandbox Bypass Configuration", "sandbox.agent: false in strict mode", "reject", "CTR-004", "sandbox_validation.go", false},
	{"CTR-005", "Unsafe Output Route", "direct write path bypassing safe-outputs", "reject", "CTR-005", "safe_outputs_validation.go", false},
	{"CTR-006", "Template Injection", "expression in run: without env-var indirection", "reject", "CTR-006", "template_injection_validation.go", false},
	{"CTR-007", "Markdown Content Security", "dangerous pattern in externally-sourced markdown", "reject", "CTR-007", "samples_validation.go", false},
	{"CTR-008", "Pull Request Target Safety", "unsafe checkout in pull_request_target workflow", "reject", "CTR-008", "pull_request_target_validation.go", false},
	{"CTR-009", "Shell Expansion in Safe-Outputs", "dangerous bash expansion in safe-outputs run:", "reject", "CTR-009", "safe_outputs_steps_shell_expansion_validation.go", false},
	{"CTR-010", "Expression Safety Allowlist", "expression not on approved allowlist", "reject", "CTR-010", "expression_safety_validation.go", false},
	{"CTR-011", "Network Firewall Configuration", "allow-urls without ssl-bump or wildcard domain", "reject", "CTR-011", "network_firewall_validation.go", false},
	{"CTR-012", "Safe-Outputs Wildcard Push Scope", "target: * without fetch pattern or access constraint", "warn", "CTR-012", "push_to_pull_request_branch_validation.go", false},
	{"CTR-013", "Argument Injection via Package/Image Names", "name starting with - in npm/pip/docker config", "reject", "CTR-013", "name_validation.go", false},
	{"CTR-014", "Supply Chain Attack via Install Scripts", "run-install-scripts: true in runtimes.node", "warn/reject", "CTR-014", "run_install_scripts_validation.go", false},
	{"CTR-015", "Allowed Label Glob Scope", `allowed-labels: ["*"] in safe-outputs config`, "reject", "CTR-015", "safe_outputs_allowed_labels_validation.go", false},
	{"CTR-016", "Compile-Time Manifest Drift", "new secret or action beyond approved manifest", "reject", "CTR-016", "safe_update_enforcement.go", false},
	{"CTR-017", "Secret Leakage via Environment Variables", "secrets expression in top-level env: or steps", "warn/reject", "CTR-017", "strict_mode_env_validation.go", false},
	{"CTR-018", "Version Integrity Bypass", "check-for-updates: false in frontmatter", "warn/reject", "CTR-018", "strict_mode_update_check_validation.go", false},
	{"CTR-019", "Cache-Memory Integrity Enforcement", "update_cache_memory without detection success guard", "rewrite", "CTR-019", "cache.go", false},
	{"CTR-020", "Conditional Import Security", "imports: entry with if field", "reject", "CTR-020", "pkg/parser/import_bfs.go", false},
	{"CTR-021", "Workflow Run Trigger Branch Scope", "workflow_run without branches restriction", "warn/reject", "CTR-021", "agent_validation.go", false},
}

// testIsConformant returns true when every rule in the catalog satisfies P1.
func testIsConformant(catalog []ctrRuleStub) bool {
	for _, r := range catalog {
		if r.id == "" || r.threatClass == "" || r.detectionCond == "" ||
			r.action == "" || r.evidence == "" || r.implRef == "" {
			return false
		}
	}
	return true
}

// ctrTestEntry represents a test-ID-to-rule mapping used by P8.
type ctrTestEntry struct {
	testID     string
	ruleID     string
	deprecated bool
}

func TestFormal_CTR016_NilManifestSkipsEnforcement(t *testing.T) {
	err := EnforceSafeUpdate(nil, []string{"MY_SECRET"}, []string{"evil-org/action@deadbeef # v1"}, "", false, false, false, false)
	require.NoError(t, err)
}

func TestFormal_CTR016_EmptyManifestRejectsNewSecret(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"MY_SECRET"}, nil, "", false, false, false, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "MY_SECRET")
}

func TestFormal_CTR016_GitHubTokenExempt_BareForm(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"GITHUB_TOKEN"}, nil, "", false, false, false, false)
	require.NoError(t, err)
}

func TestFormal_CTR016_GitHubTokenExempt_PrefixedForm(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"secrets.GITHUB_TOKEN"}, nil, "", false, false, false, false)
	require.NoError(t, err)
}

func TestFormal_CTR016_GhAwInternalSecretExempt(t *testing.T) {
	err := EnforceSafeUpdate(&GHAWManifest{Version: currentGHAWManifestVersion}, []string{"GH_AW_GITHUB_TOKEN"}, nil, "", false, false, false, false)
	require.NoError(t, err)
}

func TestFormal_CTR016_SecretPrefixNormalization(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Secrets: []string{"MY_SECRET"}}
	err := EnforceSafeUpdate(manifest, []string{"secrets.MY_SECRET"}, nil, "", false, false, false, false)
	require.NoError(t, err)
}

func TestFormal_CTR016_NewActionDriftRejected(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Actions: []GHAWManifestAction{{Repo: "actions/checkout", SHA: "abc1234", Version: "v4"}}}
	err := EnforceSafeUpdate(manifest, nil, []string{"actions/checkout@abc1234 # v4", "evil-org/steal@deadbeef # v1"}, "", false, false, false, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "evil-org/steal")
}

func TestFormal_CTR016_RemovedActionDriftRejected(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Actions: []GHAWManifestAction{{Repo: "my-org/approved-action", SHA: "abc1234", Version: "v1"}}}
	err := EnforceSafeUpdate(manifest, nil, []string{}, "", false, false, false, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "Previously-approved action")
	require.ErrorContains(t, err, "my-org/approved-action")
}

func TestFormal_CTR016_KnownActionPinUpdateAllowed(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Actions: []GHAWManifestAction{{Repo: "my-org/action", SHA: "abc1234", Version: "v1"}}}
	err := EnforceSafeUpdate(manifest, nil, []string{"my-org/action@def5678 # v2"}, "", false, false, false, false)
	require.NoError(t, err)
}

func TestFormal_CTR016_RedirectWhitespaceNormalization(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Redirect: "owner/repo/workflows/new.md@main"}
	err := EnforceSafeUpdate(manifest, nil, nil, "  owner/repo/workflows/new.md@main  ", false, false, false, false)
	require.NoError(t, err)
}

func TestFormal_CTR016_RedirectChangeRejected(t *testing.T) {
	manifest := &GHAWManifest{Version: currentGHAWManifestVersion, Redirect: "owner/repo/workflows/old.md@main"}
	err := EnforceSafeUpdate(manifest, nil, nil, "owner/repo/workflows/new.md@main", false, false, false, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "New redirect configured")
	require.ErrorContains(t, err, "Previously-approved redirect removed")
}

func TestFormal_CTR001_WritePermissionsRejected(t *testing.T) {
	// PermissionIdToken: id-token:write is allowed for OIDC auth and does not grant repo write access.
	// PermissionMetadata: metadata is always implicitly read-only, so it is excluded from the write-rejection rule.
	for _, scope := range GetAllPermissionScopes() {
		if scope == PermissionIdToken || scope == PermissionMetadata {
			continue
		}
		t.Run(string(scope), func(t *testing.T) {
			perms := NewPermissions()
			perms.Set(scope, PermissionWrite)
			err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
			require.Error(t, err)
			require.ErrorContains(t, err, "write permissions")
		})
	}
}

func TestFormal_CTR001_ReadOnlyPermissionsAllowed(t *testing.T) {
	perms := NewPermissions()
	for _, scope := range GetAllPermissionScopes() {
		// PermissionIdToken is intentionally omitted because GitHub Actions treats it as write-or-absent, not read-or-write.
		if scope == PermissionIdToken {
			continue
		}
		perms.Set(scope, PermissionRead)
	}
	err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
	require.NoError(t, err)
}

func TestFormal_CTR001_EmptyPermissionsAllowed(t *testing.T) {
	err := validateDangerousPermissions(&WorkflowData{Permissions: ""}, NewPermissions())
	require.NoError(t, err)
}

func TestFormal_CTR011_AllowURLsRequiresSSLBump(t *testing.T) {
	err := validateNetworkFirewallConfig(&NetworkPermissions{
		Firewall: &FirewallConfig{
			AllowURLs: []string{"https://github.com/githubnext/*"},
			SSLBump:   false,
		},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "allow-urls requires ssl-bump: true")
}

func TestFormal_CTR011_AllowURLsWithSSLBumpAllowed(t *testing.T) {
	err := validateNetworkFirewallConfig(&NetworkPermissions{
		Firewall: &FirewallConfig{
			AllowURLs: []string{"https://github.com/githubnext/*"},
			SSLBump:   true,
		},
	})
	require.NoError(t, err)
}

func TestFormal_CTR011_WildcardOnlyDomainRejected(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateStrictNetwork(&NetworkPermissions{Allowed: []string{"*"}})
	require.Error(t, err)
	require.ErrorContains(t, err, "wildcard '*' is not allowed")
}

func TestFormal_CTR015_WildcardLabelRejected(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(&SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{AllowedLabels: []string{"*"}},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "CTR-015")
}

func TestFormal_CTR015_WildcardLabelRejected_CreateDiscussion(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(&SafeOutputsConfig{
		CreateDiscussions: &CreateDiscussionsConfig{AllowedLabels: []string{"*"}},
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "CTR-015")
}

func TestFormal_CTR015_SpecificLabelsAllowed(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(&SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{AllowedLabels: []string{"bug", "team-*"}},
	})
	require.NoError(t, err)
}

func TestFormal_CTR015_NilConfigAllowed(t *testing.T) {
	compiler := NewCompiler()
	err := compiler.validateSafeOutputsAllowedLabelsGlobScope(nil)
	require.NoError(t, err)
}

func TestFormal_CTR014_StrictModeEnabledRejected(t *testing.T) {
	compiler := NewCompiler()
	compiler.SetStrictMode(true)
	err := compiler.validateRunInstallScripts(&WorkflowData{RunInstallScripts: true})
	require.Error(t, err)
	require.ErrorContains(t, err, "strict mode")
}

func TestFormal_CTR014_DisabledAlwaysAllowed(t *testing.T) {
	t.Run("strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		compiler.SetStrictMode(true)
		err := compiler.validateRunInstallScripts(&WorkflowData{RunInstallScripts: false})
		require.NoError(t, err)
	})

	t.Run("non-strict mode", func(t *testing.T) {
		compiler := NewCompiler()
		err := compiler.validateRunInstallScripts(&WorkflowData{RunInstallScripts: false})
		require.NoError(t, err)
	})
}

// --- P1: RuleModelComplete ---------------------------------------------------

// TestRuleCatalog_AllFieldsPresent verifies that every rule in the catalog
// has non-empty ID, threat class, detection condition, valid action, evidence,
// and implementation reference.
func TestRuleCatalog_AllFieldsPresent(t *testing.T) {
	validActions := map[string]bool{
		"reject": true, "warn": true, "rewrite": true,
		"warn/reject": true, "reject/warn": true,
	}
	for _, rule := range testCatalog {
		t.Run(rule.id, func(t *testing.T) {
			require.NotEmpty(t, rule.id, "rule ID must not be empty")
			require.NotEmpty(t, rule.threatClass, "threat class must not be empty")
			require.NotEmpty(t, rule.detectionCond, "detection condition must not be empty")
			require.True(t, validActions[rule.action], "action %q is not one of reject/warn/rewrite/warn-reject", rule.action)
			require.NotEmpty(t, rule.evidence, "evidence must not be empty")
			require.NotEmpty(t, rule.implRef, "implementation reference must not be empty")
		})
	}
}

// TestRuleCatalog_EmptyCatalogVacuouslyValid confirms that an empty rule list
// is vacuously conformant — no rule can violate P1 if there are no rules.
func TestRuleCatalog_EmptyCatalogVacuouslyValid(t *testing.T) {
	require.True(t, testIsConformant(nil))
	require.True(t, testIsConformant([]ctrRuleStub{}))
}

// --- P2: DeterministicResponse -----------------------------------------------

// TestCompilerResponse_Deterministic verifies that evaluating the same rule with
// the same input twice produces identical diagnostics.
func TestCompilerResponse_Deterministic(t *testing.T) {
	names := []string{"-exploit"}
	err1 := rejectHyphenPrefixPackages(names, "pip")
	err2 := rejectHyphenPrefixPackages(names, "pip")
	require.Error(t, err1)
	require.Error(t, err2)
	require.Equal(t, err1.Error(), err2.Error(), "compiler response must be deterministic for identical inputs")
}

// --- P3: SecureOutcome -------------------------------------------------------

// TestCompilerResponse_RejectsOrRewritesSafely verifies that a triggered rule
// always returns an error (rejects) and never silently passes unsafe input.
func TestCompilerResponse_RejectsOrRewritesSafely(t *testing.T) {
	tests := []struct {
		name    string
		trigger func() error
	}{
		{
			name: "CTR-013 hyphen-prefix package name",
			trigger: func() error {
				return rejectHyphenPrefixPackages([]string{"-evil"}, "pip")
			},
		},
		{
			name: "CTR-001 write permissions",
			trigger: func() error {
				perms := NewPermissions()
				perms.Set(PermissionContents, PermissionWrite)
				return validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
			},
		},
		{
			name: "CTR-011 allow-urls without ssl-bump",
			trigger: func() error {
				return validateNetworkFirewallConfig(&NetworkPermissions{
					Firewall: &FirewallConfig{
						AllowURLs: []string{"https://example.com"},
						SSLBump:   false,
					},
				})
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.trigger()
			require.Error(t, err, "triggered rule must not silently pass through: %s", tc.name)
		})
	}
}

// --- P4: NoHyphenPrefixPassthrough (CTR-013) ---------------------------------

// TestFormal_CTR013_RejectHyphenPrefixPackages is a table-driven test of the
// rejectHyphenPrefixPackages helper that underpins CTR-013 argument-injection detection.
func TestFormal_CTR013_RejectHyphenPrefixPackages(t *testing.T) {
	tests := []struct {
		name      string
		names     []string
		kind      string
		wantError bool
	}{
		{name: "valid single name", names: []string{"requests"}, kind: "pip", wantError: false},
		{name: "valid npm scoped name", names: []string{"@scope/pkg"}, kind: "npm", wantError: false},
		{name: "valid list with version", names: []string{"flask==2.0"}, kind: "pip", wantError: false},
		{name: "hyphen-prefix single", names: []string{"-exploit"}, kind: "pip", wantError: true},
		{name: "double-dash flag", names: []string{"--privileged"}, kind: "docker", wantError: true},
		{name: "mixed valid and invalid", names: []string{"requests", "-evil"}, kind: "pip", wantError: true},
		{name: "all valid list", names: []string{"flask", "django", "requests"}, kind: "pip", wantError: false},
		{name: "all invalid list", names: []string{"-a", "-b"}, kind: "npx", wantError: true},
		{name: "empty list", names: []string{}, kind: "pip", wantError: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := rejectHyphenPrefixPackages(tc.names, tc.kind)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRejectHyphenPrefixPackages_NilInput verifies that a nil names slice does
// not panic and returns no error (vacuously clean).
func TestRejectHyphenPrefixPackages_NilInput(t *testing.T) {
	require.NotPanics(t, func() {
		err := rejectHyphenPrefixPackages(nil, "pip")
		require.NoError(t, err)
	})
}

// --- P5: WorkflowsFieldMandatory (CTR-021) -----------------------------------

// TestValidateWorkflowRunHasWorkflows_HardErrorBothModes verifies that a missing
// or empty workflows field always produces an error regardless of strict mode.
func TestValidateWorkflowRunHasWorkflows_HardErrorBothModes(t *testing.T) {
	tests := []struct {
		name        string
		runMap      map[string]any
		wantError   bool
		errContains string
	}{
		{
			name:        "missing workflows key",
			runMap:      map[string]any{"types": []any{"completed"}},
			wantError:   true,
			errContains: "non-empty workflows field",
		},
		{
			name:        "empty workflows slice",
			runMap:      map[string]any{"workflows": []any{}},
			wantError:   true,
			errContains: "non-empty workflows field",
		},
		{
			name:        "whitespace-only workflow names",
			runMap:      map[string]any{"workflows": []any{" ", ""}},
			wantError:   true,
			errContains: "non-empty workflows field",
		},
		{
			name:      "valid workflows list",
			runMap:    map[string]any{"workflows": []any{"CI"}},
			wantError: false,
		},
		{
			name:      "valid string workflows entry",
			runMap:    map[string]any{"workflows": "CI"},
			wantError: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// validateWorkflowRunHasWorkflows is mode-agnostic: always errors
			// on missing/empty workflows regardless of strict mode.
			err := validateWorkflowRunHasWorkflows(tc.runMap, "test.md")
			if tc.wantError {
				require.Error(t, err)
				if tc.errContains != "" {
					require.ErrorContains(t, err, tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// --- P6: BranchScopeModeSensitive (CTR-021) ----------------------------------

// TestWorkflowRunBranchScope_ModeSensitivity verifies that a workflow_run trigger
// without branches: warns in non-strict mode and rejects in strict mode, while
// a trigger with branches: passes in both modes.
func TestWorkflowRunBranchScope_ModeSensitivity(t *testing.T) {
	t.Run("no branches non-strict emits warning not error", func(t *testing.T) {
		c := NewCompiler()
		c.SetStrictMode(false)
		before := c.GetWarningCount()
		err := c.emitWorkflowRunMissingBranches("test.md")
		require.NoError(t, err)
		require.Equal(t, before+1, c.GetWarningCount())
	})

	t.Run("no branches strict mode returns error", func(t *testing.T) {
		c := NewCompiler()
		c.SetStrictMode(true)
		err := c.emitWorkflowRunMissingBranches("test.md")
		require.Error(t, err)
		require.ErrorContains(t, err, "branch restrictions")
	})

	t.Run("branches present passes in non-strict mode", func(t *testing.T) {
		c := NewCompiler()
		c.SetStrictMode(false)
		workflowData := &WorkflowData{
			On: "on:\n  workflow_run:\n    workflows: [\"CI\"]\n    types: [completed]\n    branches:\n      - main\n",
		}
		err := c.validateWorkflowRunBranches(workflowData, "test.md")
		require.NoError(t, err)
	})

	t.Run("branches present passes in strict mode", func(t *testing.T) {
		c := NewCompiler()
		c.SetStrictMode(true)
		workflowData := &WorkflowData{
			On: "on:\n  workflow_run:\n    workflows: [\"CI\"]\n    types: [completed]\n    branches:\n      - main\n",
		}
		err := c.validateWorkflowRunBranches(workflowData, "test.md")
		require.NoError(t, err)
	})
}

// --- P7: DeprecationRetainsEntry --------------------------------------------

// TestDeprecatedRule_CatalogRowRetained verifies that formally deprecating a rule
// keeps it as a catalog entry (the row must not be deleted, only annotated).
func TestDeprecatedRule_CatalogRowRetained(t *testing.T) {
	deprecated := ctrRuleStub{
		id:            "CTR-DEPRECATED",
		threatClass:   "Legacy Threat",
		detectionCond: "old pattern",
		action:        "reject",
		evidence:      "CTR-DEPRECATED",
		implRef:       "old_file.go",
		deprecated:    true,
	}
	catalog := append(testCatalog, deprecated)
	found := false
	for _, r := range catalog {
		if r.id == "CTR-DEPRECATED" {
			found = true
			require.True(t, r.deprecated, "catalog entry must carry the deprecated flag")
		}
	}
	require.True(t, found, "deprecated rule must remain as a catalog entry")
}

// --- P8: DeprecatedTestsNotRequired -----------------------------------------

// TestDeprecatedRule_TestsExcludedFromConformance verifies that test entries
// mapped to a deprecated rule are excluded from the required-for-conformance set.
func TestDeprecatedRule_TestsExcludedFromConformance(t *testing.T) {
	allTests := []ctrTestEntry{
		{testID: "T-CTR-001", ruleID: "CTR-001", deprecated: false},
		{testID: "T-CTR-021", ruleID: "CTR-021", deprecated: false},
		{testID: "T-CTR-DEPRECATED", ruleID: "CTR-DEPRECATED", deprecated: true},
	}
	required := make([]ctrTestEntry, 0, len(allTests))
	for _, entry := range allTests {
		if !entry.deprecated {
			required = append(required, entry)
		}
	}
	for _, entry := range required {
		require.False(t, entry.deprecated, "required-for-conformance test must not be deprecated")
	}
	require.Len(t, required, 2, "only non-deprecated tests should be required for conformance")
}

// --- P9: VersionSyncInvariant -----------------------------------------------

// TestSpecVersion_HasCompatibilityRow verifies that every published spec version
// listed below has a corresponding row in the spec compatibility table (§2).
func TestSpecVersion_HasCompatibilityRow(t *testing.T) {
	specBytes, err := os.ReadFile("../../specs/compiler-threat-detection-spec.md")
	if err != nil {
		t.Skipf("spec file not available: %v", err)
	}
	specContent := string(specBytes)

	// All versions that MUST appear in the spec §2 compatibility table.
	knownVersions := []string{
		"1.0.8", "1.0.9", "1.0.10", "1.0.11", "1.0.12",
		"1.0.13", "1.0.14", "1.0.15", "1.0.16", "1.0.17", "1.0.18",
	}
	for _, v := range knownVersions {
		t.Run(v, func(t *testing.T) {
			require.Contains(t, specContent, "| `"+v+"`",
				"spec version %s must have a compatibility table row", v)
		})
	}
}

// --- P10: Conformance (edge case) -------------------------------------------

// TestConformance_FailsOnAnyViolatedPredicate verifies that a single rule with
// a missing required field (here: empty evidence) makes the overall catalog
// non-conformant, encoding the property that Conforms ⟺ ∀r ∈ catalog. P1(r).
func TestConformance_FailsOnAnyViolatedPredicate(t *testing.T) {
	badRule := ctrRuleStub{
		id:            "CTR-BAD",
		threatClass:   "Bad Threat",
		detectionCond: "some condition",
		action:        "reject",
		evidence:      "", // violates P1: evidence must not be empty
		implRef:       "some_file.go",
	}
	require.False(t, testIsConformant([]ctrRuleStub{badRule}),
		"catalog containing a rule with an empty evidence field must not conform")
}
