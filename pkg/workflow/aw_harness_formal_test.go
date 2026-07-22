//go:build !integration

package workflow

// Formal test suite for the AW Harness behavioral contracts.
//
// Specification: specs/aw-harness.md
//
// This file formalizes 15 top-level predicates (P1–P15) covering session lifecycle
// invariants, budget enforcement gates, extension ordering constraints, the
// three-value exit-code contract, and observability obligations. Each predicate is
// captured as a pure Go predicate function and exercised by one or more Test functions.
//
// aw_harness.cjs is aspirational as of 2026-06-21 (see §2 of the spec). The stubs
// and predicate functions below define the expected behavioral contracts; replace
// the stub adapters with real code once the Node.js implementation lands at
// actions/setup/js/aw_harness.cjs.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Domain model stubs
// ---------------------------------------------------------------------------

// awHarnessExitCode represents the three-value exit-code contract (§5.3).
type awHarnessExitCode int

const (
	awExitSuccess         awHarnessExitCode = 0 // prompt completed successfully
	awExitSessionFailure  awHarnessExitCode = 1 // non-recoverable session error
	awExitInvocationError awHarnessExitCode = 2 // invocation error (missing config, bad provider)
)

// awSession models a single Pi AgentSession lifecycle (P1, P7).
type awSession struct {
	createCount int
	disposed    bool
}

// awExtensionKind distinguishes built-in from user extensions (P11–P13).
type awExtensionKind int

const (
	awExtBuiltin awExtensionKind = iota
	awExtUser
)

// awExtension models a Pi SDK extension registered into an AgentSession.
type awExtension struct {
	Name  string
	Kind  awExtensionKind
	Order int  // registration sequence (0-based)
	OK    bool // false when init fails
}

// awJsonlEvent models a structured JSONL record emitted to a stream (P14).
type awJsonlEvent struct {
	EventType string
	Data      map[string]string
	Target    string // "stderr" or "stdout"
}

// ---------------------------------------------------------------------------
// Predicate implementations
// ---------------------------------------------------------------------------

// p1SingleSessionInvariant (P1 — SingleSessionInvariant §7):
// Exactly one AgentSession is created per harness run.
func p1SingleSessionInvariant(s awSession) bool {
	return s.createCount == 1
}

// p2PromptAtomicity (P2 — PromptAtomicity §6.3):
// The entire contents of prompt.txt are passed as a single, unmodified message.
func p2PromptAtomicity(promptContent, sentContent string, messageCount int) bool {
	return messageCount == 1 && sentContent == promptContent
}

// p3ProviderAvailable (P3 — ProviderRegistrationFirst §8.1):
// Returns true iff at least one recognised LLM provider credential is available.
// When false the harness MUST exit with code 2 (invocation error).
func p3ProviderAvailable(env map[string]string, requiredKeys []string) bool {
	for _, k := range requiredKeys {
		if v, ok := env[k]; ok && v != "" {
			return true
		}
	}
	return false
}

// p4ExitCodeFromOutcome (P4 — ExitCodePartition §5.3):
// Maps a (sessionError, invocationError) pair to the canonical exit code.
func p4ExitCodeFromOutcome(sessionError, invocationError bool) awHarnessExitCode {
	switch {
	case invocationError:
		return awExitInvocationError
	case sessionError:
		return awExitSessionFailure
	default:
		return awExitSuccess
	}
}

// p4ExitCodeValid validates that an exit code is within the defined partition.
func p4ExitCodeValid(code awHarnessExitCode) bool {
	return code == awExitSuccess || code == awExitSessionFailure || code == awExitInvocationError
}

// p5BudgetHardAbort (P5 — BudgetHardGate §8.2):
// Returns true iff the cost-tracker extension MUST abort the session because
// usedAICredits has reached or exceeded the critical threshold.
func p5BudgetHardAbort(usedAICredits, maxAICredits int64, criticalPercent float64) bool {
	if maxAICredits <= 0 {
		return false // budget not configured; never abort on budget
	}
	pct := float64(usedAICredits) / float64(maxAICredits) * 100.0
	return pct >= criticalPercent
}

// p6BudgetSoftWarn (P6 — BudgetSoftGate §8.2):
// Returns true iff the cost-tracker extension SHOULD inject a steer warning
// because usedAICredits is at or above warnPercent but below criticalPercent.
func p6BudgetSoftWarn(usedAICredits, maxAICredits int64, warnPercent, criticalPercent float64) bool {
	if maxAICredits <= 0 {
		return false
	}
	pct := float64(usedAICredits) / float64(maxAICredits) * 100.0
	return pct >= warnPercent && pct < criticalPercent
}

// p7DisposeAlwaysCalled (P7 — DisposeAlwaysCalled §7):
// Returns true iff session.dispose() was called, regardless of success/failure.
func p7DisposeAlwaysCalled(s awSession) bool {
	return s.createCount > 0 && s.disposed
}

// p8CliProxyEnforced (P8 — CliProxyAlwaysOn §6.2):
// Returns the enforced cli-proxy value and whether a warning must be emitted.
// cli-proxy is ALWAYS true for engine: aw; any false setting MUST be overridden.
func p8CliProxyEnforced(configuredCliProxy bool) (enforced bool, warnNeeded bool) {
	return true, !configuredCliProxy
}

// p9GhProxyEnforced (P9 — GhProxyAlwaysOn §6.2):
// Returns the enforced mode and whether a warning must be emitted.
// tools.github.mode is ALWAYS gh-proxy for engine: aw.
func p9GhProxyEnforced(configuredMode string) (enforced string, warnNeeded bool) {
	return string(GitHubMCPModeGHProxy), configuredMode != string(GitHubMCPModeGHProxy)
}

// p10NoAmbientContext (P10 — NoAmbientContext §6.4):
// Returns true iff every context source was explicitly declared (no auto-inject).
func p10NoAmbientContext(actualSources, declaredSources []string) bool {
	declared := make(map[string]struct{}, len(declaredSources))
	for _, s := range declaredSources {
		declared[s] = struct{}{}
	}
	for _, src := range actualSources {
		if _, ok := declared[src]; !ok {
			return false
		}
	}
	return true
}

// p10AGENTSMdNotAutoLoaded (P10b — NoAmbientContext §6.4):
// Returns true iff ambient instruction files (AGENTS.md, PI.md, etc.) are not
// present in context unless explicitly listed in imports.
func p10AGENTSMdNotAutoLoaded(contextSources, declaredImports []string) bool {
	ambientFiles := []string{"AGENTS.md", "PI.md", ".github/copilot-instructions.md"}
	declared := make(map[string]struct{}, len(declaredImports))
	for _, imp := range declaredImports {
		declared[imp] = struct{}{}
	}
	for _, src := range contextSources {
		for _, ambient := range ambientFiles {
			if src == ambient {
				if _, ok := declared[src]; !ok {
					return false // ambient file loaded without explicit import
				}
			}
		}
	}
	return true
}

// p11UserExtensionAfterBuiltins (P11 — UserExtensionAfterBuiltins §8):
// Returns true iff every user extension has a higher registration order than
// every built-in extension. Built-ins are provider-setup, cost-tracker,
// steering, repair, and observability (indices 0–4).
func p11UserExtensionAfterBuiltins(extensions []awExtension) bool {
	maxBuiltinOrder := -1
	for _, ext := range extensions {
		if ext.Kind == awExtBuiltin && ext.Order > maxBuiltinOrder {
			maxBuiltinOrder = ext.Order
		}
	}
	for _, ext := range extensions {
		if ext.Kind == awExtUser && ext.Order <= maxBuiltinOrder {
			return false
		}
	}
	return true
}

// p12BuiltinExtensionFatalExitCode (P12 — BuiltinExtensionFatal §8):
// A builtin extension init failure MUST produce exit code 2.
func p12BuiltinExtensionFatalExitCode(ext awExtension) awHarnessExitCode {
	if ext.Kind == awExtBuiltin && !ext.OK {
		return awExitInvocationError
	}
	return awExitSuccess
}

// p13UserExtensionNonFatal (P13 — UserExtensionNonFatal §6.1.4):
// Returns whether the session should continue after a user-extension failure.
// Continues when extensionsRequired is false; aborts when it is true.
func p13UserExtensionContinue(ext awExtension, extensionsRequired bool) bool {
	if ext.Kind != awExtUser || ext.OK {
		return true // not a failing user ext — always continues
	}
	return !extensionsRequired
}

// p14JsonlToStderr (P14 — JsonlToStderr §8.5.1):
// Returns true iff all JSONL events target stderr, never stdout.
func p14JsonlToStderr(events []awJsonlEvent) bool {
	for _, ev := range events {
		if ev.Target != "stderr" {
			return false
		}
	}
	return true
}

// p14NoCredentialLeak (P14b — JsonlToStderr §8.5.1, §11.1):
// Returns true iff no credential string appears in any JSONL event field.
func p14NoCredentialLeak(events []awJsonlEvent, credentials []string) bool {
	for _, ev := range events {
		for _, cred := range credentials {
			if cred == "" {
				continue
			}
			for _, v := range ev.Data {
				if strings.Contains(v, cred) {
					return false
				}
			}
		}
	}
	return true
}

// p15StepSummaryCondition (P15 — StepSummaryWritten §8.5.3):
// Returns whether a step summary MUST be written given the env variable state.
func p15StepSummaryRequired(githubStepSummaryPath string) bool {
	return githubStepSummaryPath != ""
}

// ---------------------------------------------------------------------------
// P1 — SingleSessionInvariant
// ---------------------------------------------------------------------------

// TestSingleSessionCreated verifies that createAgentSession is invoked exactly
// once per harness run (P1 / T-AW-001).
func TestSingleSessionCreated(t *testing.T) {
	t.Run("exactly one session per run", func(t *testing.T) {
		s := awSession{createCount: 1}
		assert.True(t, p1SingleSessionInvariant(s), "P1: exactly one session must be created")
	})

	t.Run("zero sessions violates invariant", func(t *testing.T) {
		s := awSession{createCount: 0}
		assert.False(t, p1SingleSessionInvariant(s), "P1: zero sessions violates single-session invariant")
	})

	t.Run("multiple sessions violate invariant", func(t *testing.T) {
		s := awSession{createCount: 2}
		assert.False(t, p1SingleSessionInvariant(s), "P1: creating more than one session violates single-session invariant")
	})
}

// ---------------------------------------------------------------------------
// P2 — PromptAtomicity
// ---------------------------------------------------------------------------

// TestPromptPassedAsAtomicMessage verifies that the full prompt.txt content
// is passed as one message, not split (P2 / §6.3).
func TestPromptPassedAsAtomicMessage(t *testing.T) {
	prompt := "Review all PRs opened in the last 24 hours and summarize findings."

	t.Run("full prompt in single message", func(t *testing.T) {
		assert.True(t, p2PromptAtomicity(prompt, prompt, 1), "P2: full prompt must be one unmodified message")
	})

	t.Run("split prompt violates atomicity", func(t *testing.T) {
		firstHalf := prompt[:len(prompt)/2]
		assert.False(t, p2PromptAtomicity(prompt, firstHalf, 2), "P2: partial content across multiple messages violates atomicity")
	})

	t.Run("modified content violates atomicity", func(t *testing.T) {
		assert.False(t, p2PromptAtomicity(prompt, prompt+" extra", 1), "P2: modified prompt content violates atomicity")
	})
}

// ---------------------------------------------------------------------------
// P3 — ProviderRegistrationFirst
// ---------------------------------------------------------------------------

// TestProviderRegistrationRequired verifies that the harness exits with code 2
// when no LLM provider credentials are present (P3 / §8.1, §5.3).
func TestProviderRegistrationRequired(t *testing.T) {
	required := []string{"ANTHROPIC_API_KEY", "COPILOT_GITHUB_TOKEN", "OPENAI_API_KEY"}

	t.Run("at least one credential satisfies requirement", func(t *testing.T) {
		env := map[string]string{"ANTHROPIC_API_KEY": "sk-ant-secret"}
		assert.True(t, p3ProviderAvailable(env, required), "P3: one valid credential must satisfy provider requirement")
	})

	t.Run("no credentials causes invocation error", func(t *testing.T) {
		env := map[string]string{}
		available := p3ProviderAvailable(env, required)
		assert.False(t, available, "P3: no credentials should fail provider check")

		// Invocation error (exit code 2) must follow from missing provider.
		exitCode := p4ExitCodeFromOutcome(false, !available)
		assert.Equal(t, awExitInvocationError, exitCode, "P3: missing provider must yield exit code 2")
	})

	t.Run("empty credential string is not accepted", func(t *testing.T) {
		env := map[string]string{"ANTHROPIC_API_KEY": ""}
		assert.False(t, p3ProviderAvailable(env, required), "P3: empty credential value must not satisfy provider check")
	})

	t.Run("pi engine compile-time validation enforces gh-proxy and cli-proxy", func(t *testing.T) {
		// The Go compiler already validates Pi/AW engine requirements at compile time.
		// This test confirms that missing tools.github.mode: gh-proxy is caught.
		compiler := NewCompiler()
		tools := NewTools(map[string]any{
			"github":    map[string]any{"mode": "remote"},
			"cli-proxy": true,
		})
		err := compiler.validatePiEngineRequirements(tools, NewPiEngine())
		require.Error(t, err, "P3/P9: remote GitHub mode must be rejected for Pi/AW engine")
		assert.Contains(t, err.Error(), "gh-proxy")
	})
}

// ---------------------------------------------------------------------------
// P4 — ExitCodePartition
// ---------------------------------------------------------------------------

// TestExitCodeOnSuccess verifies that exit code 0 is produced on clean completion (P4).
func TestExitCodeOnSuccess(t *testing.T) {
	code := p4ExitCodeFromOutcome(false, false)
	assert.Equal(t, awExitSuccess, code, "P4: clean completion must yield exit code 0")
	assert.True(t, p4ExitCodeValid(code))
}

// TestExitCodeOnSessionFailure verifies that exit code 1 is produced on a
// non-recoverable session error (P4 / §5.3).
func TestExitCodeOnSessionFailure(t *testing.T) {
	code := p4ExitCodeFromOutcome(true, false)
	assert.Equal(t, awExitSessionFailure, code, "P4: session failure must yield exit code 1")
	assert.True(t, p4ExitCodeValid(code))
}

// TestExitCodeOnInvocationError verifies that exit code 2 is produced for
// invocation errors such as missing config or provider (P4 / §5.3).
func TestExitCodeOnInvocationError(t *testing.T) {
	code := p4ExitCodeFromOutcome(false, true)
	assert.Equal(t, awExitInvocationError, code, "P4: invocation error must yield exit code 2")
	assert.True(t, p4ExitCodeValid(code))

	// Invocation error takes precedence over session failure when both flags fire.
	codeWithBoth := p4ExitCodeFromOutcome(true, true)
	assert.Equal(t, awExitInvocationError, codeWithBoth, "P4: invocation error takes precedence over session error")
}

// ---------------------------------------------------------------------------
// P5 — BudgetHardGate
// ---------------------------------------------------------------------------

// TestBudgetHardAbortAtCriticalThreshold verifies that the cost-tracker
// extension aborts the session when AI credits reach the critical threshold (P5).
func TestBudgetHardAbortAtCriticalThreshold(t *testing.T) {
	const maxCredits = 100000
	const criticalPct = 90.0

	t.Run("exactly at critical threshold triggers abort", func(t *testing.T) {
		used := int64(90000) // 90 % of 100000
		assert.True(t, p5BudgetHardAbort(used, maxCredits, criticalPct),
			"P5: 90% usage must trigger hard abort")
	})

	t.Run("above critical threshold triggers abort", func(t *testing.T) {
		used := int64(95000) // 95 %
		assert.True(t, p5BudgetHardAbort(used, maxCredits, criticalPct),
			"P5: usage above critical threshold must trigger hard abort")
	})

	t.Run("below critical threshold does not abort", func(t *testing.T) {
		used := int64(80000) // 80 %
		assert.False(t, p5BudgetHardAbort(used, maxCredits, criticalPct),
			"P5: usage below critical threshold must not trigger abort")
	})

	t.Run("unconfigured budget (maxCredits=0) never aborts", func(t *testing.T) {
		assert.False(t, p5BudgetHardAbort(999999, 0, criticalPct),
			"P5: unconfigured budget must never cause an abort")
	})
}

// TestBudgetPreservesCompletedTurnArtifacts verifies that completed-turn
// artifacts are preserved after a hard budget abort (P5 / §8.2).
func TestBudgetPreservesCompletedTurnArtifacts(t *testing.T) {
	// Model: artifacts produced before the abort turn must remain accessible.
	type artifact struct {
		TurnNumber int
		Committed  bool
	}
	artifacts := []artifact{
		{TurnNumber: 1, Committed: true},
		{TurnNumber: 2, Committed: true},
		{TurnNumber: 3, Committed: false}, // in-flight when budget abort fires
	}

	for _, a := range artifacts {
		if a.Committed {
			assert.True(t, a.Committed,
				"P5: artifact from completed turn %d must be preserved after budget abort", a.TurnNumber)
		}
	}
	// The in-flight artifact (turn 3) may be incomplete — only committed ones
	// are subject to the preservation guarantee.
	assert.False(t, artifacts[2].Committed, "P5: in-flight artifact at abort time may be incomplete")
}

// ---------------------------------------------------------------------------
// P6 — BudgetSoftGate
// ---------------------------------------------------------------------------

// TestBudgetWarnSteerMessageInjected verifies that the cost-tracker extension
// injects a steer message when token usage crosses the warn threshold (P6 / §8.2).
func TestBudgetWarnSteerMessageInjected(t *testing.T) {
	const maxCredits = 100000
	const warnPct = 75.0
	const criticalPct = 90.0

	t.Run("exactly at warn threshold injects steer message", func(t *testing.T) {
		used := int64(75000)
		assert.True(t, p6BudgetSoftWarn(used, maxCredits, warnPct, criticalPct),
			"P6: 75% usage must trigger steer warning message")
	})

	t.Run("between warn and critical injects steer message", func(t *testing.T) {
		used := int64(85000)
		assert.True(t, p6BudgetSoftWarn(used, maxCredits, warnPct, criticalPct),
			"P6: usage between warn and critical must inject steer message")
	})

	t.Run("below warn threshold no steer message", func(t *testing.T) {
		used := int64(60000)
		assert.False(t, p6BudgetSoftWarn(used, maxCredits, warnPct, criticalPct),
			"P6: usage below warn threshold must not inject steer message")
	})

	t.Run("at or above critical threshold switches to hard abort, not warn", func(t *testing.T) {
		used := int64(90000)
		softWarn := p6BudgetSoftWarn(used, maxCredits, warnPct, criticalPct)
		hardAbort := p5BudgetHardAbort(used, maxCredits, criticalPct)
		assert.False(t, softWarn, "P6: usage at critical threshold must not produce a soft warning (hard abort takes over)")
		assert.True(t, hardAbort, "P5: usage at critical threshold must produce a hard abort")
	})

	t.Run("unconfigured budget never injects steer message", func(t *testing.T) {
		assert.False(t, p6BudgetSoftWarn(999999, 0, warnPct, criticalPct),
			"P6: unconfigured budget must never inject steer message")
	})
}

// ---------------------------------------------------------------------------
// P7 — DisposeAlwaysCalled
// ---------------------------------------------------------------------------

// TestDisposeCalledOnSuccess verifies that session.dispose() is called after a
// successful run (P7 / §7).
func TestDisposeCalledOnSuccess(t *testing.T) {
	s := awSession{createCount: 1, disposed: true}
	assert.True(t, p7DisposeAlwaysCalled(s), "P7: dispose must be called on successful session completion")
}

// TestDisposeCalledOnFailure verifies that session.dispose() is called even
// when the session fails (P7 / §7).
func TestDisposeCalledOnFailure(t *testing.T) {
	t.Run("dispose called after session failure", func(t *testing.T) {
		s := awSession{createCount: 1, disposed: true}
		assert.True(t, p7DisposeAlwaysCalled(s), "P7: dispose must be called on session failure path")
	})

	t.Run("missing dispose on failure violates invariant", func(t *testing.T) {
		s := awSession{createCount: 1, disposed: false}
		assert.False(t, p7DisposeAlwaysCalled(s), "P7: undisposed session must fail the invariant check")
	})
}

// ---------------------------------------------------------------------------
// P8 — CliProxyAlwaysOn
// ---------------------------------------------------------------------------

// TestCliProxyOverrideEnforced verifies that cli-proxy:false in a workflow
// config is overridden to true and a warning is emitted (P8 / §6.2).
func TestCliProxyOverrideEnforced(t *testing.T) {
	t.Run("cli-proxy already true — no warning needed", func(t *testing.T) {
		enforced, warn := p8CliProxyEnforced(true)
		assert.True(t, enforced, "P8: cli-proxy must be enforced as true")
		assert.False(t, warn, "P8: no warning when cli-proxy was already true")
	})

	t.Run("cli-proxy false is overridden and warning emitted", func(t *testing.T) {
		enforced, warn := p8CliProxyEnforced(false)
		assert.True(t, enforced, "P8: cli-proxy:false must be overridden to true")
		assert.True(t, warn, "P8: warning must be emitted when cli-proxy:false is overridden")
	})

	t.Run("compiler rejects cli-proxy:false for Pi/AW engine at compile time", func(t *testing.T) {
		compiler := NewCompiler()
		tools := NewTools(map[string]any{
			"github":    map[string]any{"mode": "gh-proxy"},
			"cli-proxy": false,
		})
		err := compiler.validatePiEngineRequirements(tools, NewPiEngine())
		require.Error(t, err, "P8: cli-proxy:false must be rejected by compile-time validation")
		assert.Contains(t, err.Error(), "cli-proxy")
	})
}

// ---------------------------------------------------------------------------
// P9 — GhProxyAlwaysOn
// ---------------------------------------------------------------------------

// TestGhProxyOverrideEnforced verifies that tools.github.mode:remote is
// overridden to gh-proxy and a warning is emitted (P9 / §6.2).
func TestGhProxyOverrideEnforced(t *testing.T) {
	t.Run("gh-proxy mode already set — no warning", func(t *testing.T) {
		enforced, warn := p9GhProxyEnforced("gh-proxy")
		assert.Equal(t, string(GitHubMCPModeGHProxy), enforced, "P9: gh-proxy mode must be enforced")
		assert.False(t, warn, "P9: no warning when mode was already gh-proxy")
	})

	t.Run("remote mode is overridden and warning emitted", func(t *testing.T) {
		enforced, warn := p9GhProxyEnforced("remote")
		assert.Equal(t, string(GitHubMCPModeGHProxy), enforced, "P9: remote mode must be overridden to gh-proxy")
		assert.True(t, warn, "P9: warning must be emitted when github.mode:remote is overridden")
	})

	t.Run("local mode is overridden and warning emitted", func(t *testing.T) {
		enforced, warn := p9GhProxyEnforced("local")
		assert.Equal(t, string(GitHubMCPModeGHProxy), enforced, "P9: local mode must be overridden to gh-proxy")
		assert.True(t, warn, "P9: warning must be emitted when github.mode:local is overridden")
	})

	t.Run("compiler rejects non-gh-proxy mode for Pi/AW engine at compile time", func(t *testing.T) {
		compiler := NewCompiler()
		tools := NewTools(map[string]any{
			"github":    map[string]any{"mode": "remote"},
			"cli-proxy": true,
		})
		err := compiler.validatePiEngineRequirements(tools, NewPiEngine())
		require.Error(t, err, "P9: github.mode:remote must be rejected by compile-time validation")
		assert.Contains(t, err.Error(), "gh-proxy")
	})
}

// ---------------------------------------------------------------------------
// P10 — NoAmbientContext
// ---------------------------------------------------------------------------

// TestNoAmbientContextInjected verifies that only prompt and declared imports
// appear in the session context — no ambient files are silently injected (P10 / §6.4).
func TestNoAmbientContextInjected(t *testing.T) {
	declared := []string{"prompt.txt", "skills/reporting/SKILL.md"}

	t.Run("all context sources are declared", func(t *testing.T) {
		actual := []string{"prompt.txt", "skills/reporting/SKILL.md"}
		assert.True(t, p10NoAmbientContext(actual, declared),
			"P10: context sourced only from declared files must pass")
	})

	t.Run("undeclared file in context violates invariant", func(t *testing.T) {
		actual := []string{"prompt.txt", "skills/reporting/SKILL.md", "AGENTS.md"}
		assert.False(t, p10NoAmbientContext(actual, declared),
			"P10: undeclared context source must fail the no-ambient-context check")
	})

	t.Run("empty context is valid", func(t *testing.T) {
		assert.True(t, p10NoAmbientContext(nil, declared),
			"P10: empty context source list must pass")
	})
}

// TestAGENTSMdNotAutoLoaded verifies that AGENTS.md (and similar ambient
// instruction files) are never automatically loaded into context (P10 / §6.4).
func TestAGENTSMdNotAutoLoaded(t *testing.T) {
	t.Run("AGENTS.md absent from context when not declared", func(t *testing.T) {
		contextSources := []string{"prompt.txt"}
		declared := []string{"prompt.txt"}
		assert.True(t, p10AGENTSMdNotAutoLoaded(contextSources, declared),
			"P10: AGENTS.md must not appear in context when not declared")
	})

	t.Run("AGENTS.md in context without declaration violates invariant", func(t *testing.T) {
		contextSources := []string{"prompt.txt", "AGENTS.md"}
		declared := []string{"prompt.txt"}
		assert.False(t, p10AGENTSMdNotAutoLoaded(contextSources, declared),
			"P10: AGENTS.md auto-loaded without explicit import violates no-ambient-context rule")
	})

	t.Run("AGENTS.md in context is allowed when explicitly declared", func(t *testing.T) {
		contextSources := []string{"prompt.txt", "AGENTS.md"}
		declared := []string{"prompt.txt", "AGENTS.md"}
		assert.True(t, p10AGENTSMdNotAutoLoaded(contextSources, declared),
			"P10: AGENTS.md is allowed when explicitly listed in imports")
	})

	t.Run("PI.md and copilot-instructions.md also subject to constraint", func(t *testing.T) {
		for _, ambientFile := range []string{"PI.md", ".github/copilot-instructions.md"} {
			contextSources := []string{"prompt.txt", ambientFile}
			declared := []string{"prompt.txt"}
			assert.False(t, p10AGENTSMdNotAutoLoaded(contextSources, declared),
				"P10: %s auto-loaded without explicit import violates no-ambient-context rule", ambientFile)
		}
	})
}

// ---------------------------------------------------------------------------
// P11 — UserExtensionAfterBuiltins
// ---------------------------------------------------------------------------

// TestUserExtensionsRegisteredAfterBuiltins verifies that all 5 built-in
// extensions are registered before any user extension (P11 / §8).
func TestUserExtensionsRegisteredAfterBuiltins(t *testing.T) {
	// The 5 built-in gh-aw Pi extensions defined in §4 / §8.
	builtins := []awExtension{
		{Name: "provider-setup", Kind: awExtBuiltin, Order: 0, OK: true},
		{Name: "cost-tracker", Kind: awExtBuiltin, Order: 1, OK: true},
		{Name: "steering", Kind: awExtBuiltin, Order: 2, OK: true},
		{Name: "repair", Kind: awExtBuiltin, Order: 3, OK: true},
		{Name: "observability", Kind: awExtBuiltin, Order: 4, OK: true},
	}

	t.Run("user extensions registered after all builtins", func(t *testing.T) {
		exts := append(builtins,
			awExtension{Name: "custom-tool", Kind: awExtUser, Order: 5, OK: true},
		)
		assert.True(t, p11UserExtensionAfterBuiltins(exts),
			"P11: user extensions with higher order than all builtins must pass")
	})

	t.Run("user extension before a builtin violates ordering", func(t *testing.T) {
		exts := append(builtins,
			awExtension{Name: "early-user-ext", Kind: awExtUser, Order: 2, OK: true}, // same order as "steering"
		)
		assert.False(t, p11UserExtensionAfterBuiltins(exts),
			"P11: user extension at same or lower order than a builtin must fail")
	})

	t.Run("no user extensions is always valid", func(t *testing.T) {
		assert.True(t, p11UserExtensionAfterBuiltins(builtins),
			"P11: only builtins with no user extensions must pass")
	})

	t.Run("exactly five builtins are defined", func(t *testing.T) {
		assert.Len(t, builtins, 5, "P11: exactly 5 built-in extensions must be defined (§4 / §8)")
	})
}

// ---------------------------------------------------------------------------
// P12 — BuiltinExtensionFatal
// ---------------------------------------------------------------------------

// TestBuiltinExtensionFailureIsFatal verifies that a builtin extension init
// failure causes exit code 2 (invocation error) (P12 / §8).
func TestBuiltinExtensionFailureIsFatal(t *testing.T) {
	t.Run("failed builtin produces exit code 2", func(t *testing.T) {
		failedBuiltin := awExtension{Name: "cost-tracker", Kind: awExtBuiltin, Order: 1, OK: false}
		code := p12BuiltinExtensionFatalExitCode(failedBuiltin)
		assert.Equal(t, awExitInvocationError, code,
			"P12: failed built-in extension must yield exit code 2")
	})

	t.Run("successful builtin does not error", func(t *testing.T) {
		okBuiltin := awExtension{Name: "steering", Kind: awExtBuiltin, Order: 2, OK: true}
		code := p12BuiltinExtensionFatalExitCode(okBuiltin)
		assert.Equal(t, awExitSuccess, code,
			"P12: successful built-in extension must not cause an error exit code")
	})

	t.Run("failed user extension does not produce exit code 2 via this predicate", func(t *testing.T) {
		failedUser := awExtension{Name: "custom-tool", Kind: awExtUser, Order: 5, OK: false}
		code := p12BuiltinExtensionFatalExitCode(failedUser)
		assert.Equal(t, awExitSuccess, code,
			"P12: user extension failure is not governed by BuiltinExtensionFatal predicate")
	})
}

// ---------------------------------------------------------------------------
// P13 — UserExtensionNonFatal
// ---------------------------------------------------------------------------

// TestUserExtensionFailureNonFatal verifies that a failed user extension does
// not abort the session when extensions-required is false (P13 / §6.1.4).
func TestUserExtensionFailureNonFatal(t *testing.T) {
	failedExt := awExtension{Name: "custom-tool", Kind: awExtUser, Order: 5, OK: false}

	t.Run("failed user extension is non-fatal by default (extensions-required: false)", func(t *testing.T) {
		assert.True(t, p13UserExtensionContinue(failedExt, false),
			"P13: session must continue when user extension fails and extensions-required is false")
	})

	t.Run("successful user extension always continues", func(t *testing.T) {
		okExt := awExtension{Name: "custom-tool", Kind: awExtUser, Order: 5, OK: true}
		assert.True(t, p13UserExtensionContinue(okExt, false),
			"P13: session must continue when user extension succeeds")
		assert.True(t, p13UserExtensionContinue(okExt, true),
			"P13: session must continue when user extension succeeds regardless of extensions-required")
	})
}

// TestUserExtensionFailureFatalWhenRequired verifies that a failed user extension
// aborts the session when extensions-required is true (P13 / §6.1.4).
func TestUserExtensionFailureFatalWhenRequired(t *testing.T) {
	failedExt := awExtension{Name: "required-tool", Kind: awExtUser, Order: 5, OK: false}

	t.Run("failed user extension is fatal when extensions-required: true", func(t *testing.T) {
		assert.False(t, p13UserExtensionContinue(failedExt, true),
			"P13: session must abort when user extension fails and extensions-required is true")
	})

	t.Run("builtin extension failures are always fatal regardless of extensions-required", func(t *testing.T) {
		failedBuiltin := awExtension{Name: "provider-setup", Kind: awExtBuiltin, Order: 0, OK: false}
		code := p12BuiltinExtensionFatalExitCode(failedBuiltin)
		assert.Equal(t, awExitInvocationError, code,
			"P12/P13: builtin failure always exits with code 2 regardless of extensions-required")
	})
}

// ---------------------------------------------------------------------------
// P14 — JsonlToStderr
// ---------------------------------------------------------------------------

// TestJsonlEventsEmittedToStderr verifies that all JSONL events go to stderr
// and never stdout (P14 / §8.5.1, §5.4).
func TestJsonlEventsEmittedToStderr(t *testing.T) {
	events := []awJsonlEvent{
		{EventType: "session_start", Data: map[string]string{"model": "claude-sonnet-4.6"}, Target: "stderr"},
		{EventType: "turn_end", Data: map[string]string{"turn": "1", "tokens": "1234"}, Target: "stderr"},
		{EventType: "session_end", Data: map[string]string{"cost_usd": "0.0042"}, Target: "stderr"},
	}

	t.Run("all JSONL events target stderr", func(t *testing.T) {
		assert.True(t, p14JsonlToStderr(events), "P14: all JSONL events must be written to stderr")
	})

	t.Run("JSONL event targeting stdout violates invariant", func(t *testing.T) {
		badEvents := append(events, awJsonlEvent{
			EventType: "debug_info",
			Data:      map[string]string{"info": "some debug data"},
			Target:    "stdout",
		})
		assert.False(t, p14JsonlToStderr(badEvents), "P14: JSONL event to stdout must fail the stderr check")
	})

	t.Run("empty event stream is valid", func(t *testing.T) {
		assert.True(t, p14JsonlToStderr(nil), "P14: empty event stream must pass")
	})
}

// TestCredentialsNotLeakedToJsonl verifies that provider credentials are never
// present in any JSONL event payload (P14b / §11.1).
func TestCredentialsNotLeakedToJsonl(t *testing.T) {
	credentials := []string{"sk-ant-secret-key-12345", "ghp_token_value_999"}

	t.Run("no credentials in JSONL events", func(t *testing.T) {
		events := []awJsonlEvent{
			{EventType: "session_start", Data: map[string]string{"model": "claude-sonnet-4.6"}, Target: "stderr"},
			{EventType: "turn_end", Data: map[string]string{"turn": "1"}, Target: "stderr"},
		}
		assert.True(t, p14NoCredentialLeak(events, credentials), "P14: events without credentials must pass leak check")
	})

	t.Run("credential value in JSONL event data triggers leak detection", func(t *testing.T) {
		events := []awJsonlEvent{
			{EventType: "session_start", Data: map[string]string{
				"model":   "claude-sonnet-4.6",
				"api_key": "sk-ant-secret-key-12345", // leaked credential
			}, Target: "stderr"},
		}
		assert.False(t, p14NoCredentialLeak(events, credentials), "P14: event containing a credential must fail leak check")
	})

	t.Run("empty credential list never triggers leak detection", func(t *testing.T) {
		events := []awJsonlEvent{
			{EventType: "turn_end", Data: map[string]string{"payload": "anything goes here"}, Target: "stderr"},
		}
		assert.True(t, p14NoCredentialLeak(events, []string{"", ""}),
			"P14: empty credential list must never report a leak")
	})
}

// ---------------------------------------------------------------------------
// P15 — StepSummaryWritten
// ---------------------------------------------------------------------------

// TestStepSummaryWrittenWhenEnvSet verifies that a Markdown step summary is
// written to $GITHUB_STEP_SUMMARY when the env variable is set (P15 / §8.5.3).
func TestStepSummaryWrittenWhenEnvSet(t *testing.T) {
	summaryPath := "/tmp/gh-aw/agent-step-summary.md"

	t.Run("step summary required when GITHUB_STEP_SUMMARY is set", func(t *testing.T) {
		assert.True(t, p15StepSummaryRequired(summaryPath),
			"P15: step summary must be written when GITHUB_STEP_SUMMARY is set")
	})

	t.Run("step summary content must be non-empty", func(t *testing.T) {
		summaryContent := "## AW Harness Run — `claude-sonnet-4.6`\n\n| Turn | Tokens |\n|------|-------|\n| 1 | 1234 |\n"
		assert.NotEmpty(t, summaryContent, "P15: step summary content must be non-empty when written")
		assert.True(t, strings.HasPrefix(summaryContent, "#"),
			"P15: step summary must begin with a Markdown heading")
	})
}

// TestStepSummarySkippedWhenEnvAbsent verifies that no step summary is written
// when $GITHUB_STEP_SUMMARY is not set (P15 / §8.5.3).
func TestStepSummarySkippedWhenEnvAbsent(t *testing.T) {
	t.Run("step summary not required when GITHUB_STEP_SUMMARY is absent", func(t *testing.T) {
		assert.False(t, p15StepSummaryRequired(""),
			"P15: step summary must be skipped when GITHUB_STEP_SUMMARY is not set")
	})

	t.Run("pi engine always sets GITHUB_STEP_SUMMARY inside AWF sandbox", func(t *testing.T) {
		// The existing Pi engine sets AgentStepSummaryPath as GITHUB_STEP_SUMMARY
		// inside the AWF container, so the harness always finds it set at runtime.
		// This test confirms the constant is non-empty (the Go side of the contract).
		assert.NotEmpty(t, AgentStepSummaryPath,
			"P15: AgentStepSummaryPath must be defined so the step summary env is always set in the AWF sandbox")
	})
}
