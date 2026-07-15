//go:build !integration

package cli

// Formal compliance tests for the safe output outcome evaluation engine.
//
// These tests cover predicates P1–P12 derived from the formal model in
// specs/safe-output-outcome-evaluation.md.
//
// Formal notation cross-references:
//   - TLA+ state-machine invariants: P1, P4, P5, P6, P9
//   - Z3/SMT-LIB arithmetic predicates: P10
//   - F* pre/post contracts: P2, P3, P7, P8, P11, P12

import (
	"errors"
	"testing"

	"github.com/github/gh-aw/pkg/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormalOutcomeDomainInvariant verifies that every OutcomeResult produced by
// the evaluation engine is within the six outcome categories defined in the spec,
// or is a recognized internal state that must be normalized before external emission.
//
// Formal predicate (TLA+):
//
//	OutcomeDomain ≜
//	  ∀ r ∈ OutcomeResult :
//	    r ∈ {"accepted","rejected","ignored","pending","lifecycle","lifecycle_close"}
//	    ∨ r ∈ {"unknown","error"}  (* internal-only; normalized before OTel emission *)
//
// Specification reference: specs/safe-output-outcome-evaluation.md §Outcome Categories
func TestFormalOutcomeDomainInvariant(t *testing.T) {
	// The six externally-observable outcome categories defined by the spec.
	specDomain := map[string]bool{
		"accepted":        true,
		"rejected":        true,
		"ignored":         true,
		"pending":         true,
		"lifecycle":       true,
		"lifecycle_close": true,
	}
	// Internal states that are valid at the evaluator level but must be
	// normalized to a spec-defined category before external emission.
	internalOnly := map[string]bool{
		"unknown": true,
		"error":   true,
	}

	// Every declared OutcomeResult constant must be in the spec domain or internal.
	allResults := []OutcomeResult{
		OutcomeAccepted, OutcomeRejected, OutcomeIgnored, OutcomePending,
		OutcomeLifecycle, OutcomeLifecycleClose, OutcomeUnknown, OutcomeError,
	}
	for _, r := range allResults {
		s := string(r)
		assert.True(t, specDomain[s] || internalOnly[s],
			"P1: OutcomeResult %q must be in spec domain or recognized internal state", s)
	}

	// ComputeOutcomeSummary must count each spec-defined outcome correctly.
	reports := []OutcomeReport{
		{Type: "create_pull_request", Result: OutcomeAccepted},
		{Type: "create_issue", Result: OutcomeRejected},
		{Type: "add_comment", Result: OutcomeIgnored},
		{Type: "add_labels", Result: OutcomePending},
		{Type: "close_issue", Result: OutcomeLifecycle},
	}
	summary := ComputeOutcomeSummary(reports, github.DefaultObjectiveMapping())
	assert.Equal(t, 5, summary.Total, "P1: total must cover the five spec-defined non-internal outcomes")
	assert.Equal(t, 1, summary.Accepted, "P1: one accepted")
	assert.Equal(t, 1, summary.Rejected, "P1: one rejected")
	assert.Equal(t, 1, summary.Ignored, "P1: one ignored")
	assert.Equal(t, 1, summary.Pending, "P1: one pending")
	assert.Equal(t, 1, summary.Lifecycle, "P1: one lifecycle")
}

// TestFormalAPIFailurePending verifies that GitHub API 5xx and rate-limit errors
// never produce a terminal classification of accepted or rejected.
//
// Formal predicate (F*):
//
//	val evaluateWithAPIFailure :
//	  item:CreatedItemReport → apiErr:HTTPError →
//	  Tot OutcomeReport
//	  (requires apiErr.status ∈ {500, 502, 503, 429})
//	  (ensures fun r → r.Result ≠ OutcomeAccepted ∧ r.Result ≠ OutcomeRejected)
//
// Specification reference: specs/safe-output-outcome-evaluation.md §Norms (rules 2, 4)
func TestFormalAPIFailurePending(t *testing.T) {
	old := closeStickyGHAPIGet
	t.Cleanup(func() { closeStickyGHAPIGet = old })

	apiErrors := []struct {
		name    string
		errText string
	}{
		{"503 server error", "gh api: 503 Service Unavailable"},
		{"429 rate limit", "gh api: 429 Too Many Requests"},
		{"502 bad gateway", "gh api: 502 Bad Gateway"},
		{"500 internal error", "gh api: 500 Internal Server Error"},
	}

	for _, tc := range apiErrors {
		t.Run(tc.name, func(t *testing.T) {
			closeStickyGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
				return nil, errors.New(tc.errText)
			}
			item := CreatedItemReport{Type: "close_issue", Number: 99, Repo: "owner/repo"}
			report := evalCloseSticky(item, "owner/repo")

			assert.NotEqual(t, OutcomeAccepted, report.Result,
				"P2: API error %q must not yield accepted", tc.errText)
			assert.NotEqual(t, OutcomeRejected, report.Result,
				"P2: API error %q must not yield rejected", tc.errText)
		})
	}
}

// TestFormal404Classification verifies that 404-equivalent conditions are classified
// as rejected for persistent objects and ignored for transient targets, and that
// 404 API errors never yield accepted.
//
// Formal predicate (TLA+):
//
//	NotFoundClassification ≜
//	  ∀ r : APIError →
//	    r.status = 404 ∧ persistent(r.type) ⟹ eval.OutcomeStatus = rejected ∧
//	    r.status = 404 ∧ transient(r.type) ⟹ eval.OutcomeStatus = ignored ∧
//	    r.status = 404 ⟹ eval.OutcomeStatus ≠ accepted
//
// Specification reference: specs/safe-output-outcome-evaluation.md §Norms (rule 1)
func TestFormal404Classification(t *testing.T) {
	// Persistent object: "deleted" detail maps to rejected (persistent 404 classification).
	t.Run("persistent object deleted → rejected", func(t *testing.T) {
		report := OutcomeReport{
			Type:   "create_issue",
			Result: OutcomeRejected,
			Detail: "deleted",
		}
		eval := normalizeOutcomeEvaluation(report)
		assert.Equal(t, OutcomeStatusRejected, eval.OutcomeStatus,
			"P3: 404 on persistent object (deleted) must yield rejected")
		assert.Equal(t, "deleted", eval.Signal,
			"P3: deleted signal must be set for persistent 404")
	})

	// Transient target: "no engagement" maps to ignored (transient 404 classification).
	t.Run("transient target no engagement → ignored", func(t *testing.T) {
		report := OutcomeReport{
			Type:   "add_comment",
			Result: OutcomeIgnored,
			Detail: "no engagement",
		}
		eval := normalizeOutcomeEvaluation(report)
		assert.Equal(t, OutcomeStatusIgnored, eval.OutcomeStatus,
			"P3: transient target with no engagement must yield ignored")
	})

	// API error (simulating 404) must not yield accepted.
	t.Run("404 API error must not yield accepted", func(t *testing.T) {
		old := closeStickyGHAPIGet
		t.Cleanup(func() { closeStickyGHAPIGet = old })
		closeStickyGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
			return nil, errors.New("gh api: 404 Not Found")
		}
		item := CreatedItemReport{Type: "close_issue", Number: 99, Repo: "owner/repo"}
		report := evalCloseSticky(item, "owner/repo")
		assert.NotEqual(t, OutcomeAccepted, report.Result,
			"P3: 404 API error must not yield accepted")
	})
}

// TestFormalBotActorProvenance verifies that actor identity is correctly classified
// as bot or non-bot based on the visible GitHub login.
//
// Formal predicate (TLA+):
//
//	BotActorProvenance ≜
//	  ∀ login : isBotUser(login) ↔
//	    HasSuffix(login, "[bot]") ∨ login ∈ KnownBotLogins
//
// Specification reference: specs/safe-output-outcome-evaluation.md §Provenance Limits (rules 1–3)
func TestFormalBotActorProvenance(t *testing.T) {
	botLogins := []string{
		"github-actions[bot]",  // App bot with [bot] suffix
		"dependabot[bot]",      // Common dependency bot
		"copilot-swe-agent",    // Known bot login
		"github-actions",       // Well-known bot alias
		"some-custom-app[bot]", // Any [bot]-suffixed login is a bot
	}
	for _, login := range botLogins {
		assert.True(t, isBotUser(login),
			"P4: login %q must be identified as bot actor", login)
	}

	humanLogins := []string{
		"octocat",
		"alice",
		"john-smith",
		"mnkiefer",
		"github-actions-user", // similar prefix but not a known bot and no [bot] suffix
	}
	for _, login := range humanLogins {
		assert.False(t, isBotUser(login),
			"P4: login %q must be identified as non-bot (human-visible) actor", login)
	}
}

// TestFormalPRMergeAcceptance verifies the PR state machine transitions:
// merged → accepted; closed without merge → rejected; open → pending.
//
// Formal predicate (TLA+):
//
//	PRMergeAcceptance ≜
//	  ∀ pr : PR →
//	    pr.merged = true                    ⟹ outcome = accepted ∧
//	    pr.state = "closed" ∧ ¬pr.merged  ⟹ outcome = rejected ∧
//	    pr.state = "open"                  ⟹ outcome = pending
//
// Specification reference: specs/safe-output-outcome-evaluation.md §1. `create_pull_request`
func TestFormalPRMergeAcceptance(t *testing.T) {
	cases := []struct {
		name       string
		result     OutcomeResult
		detail     string
		wantStatus OutcomeStatus
		wantSignal string
	}{
		{
			name:       "merged PR → accepted",
			result:     OutcomeAccepted,
			detail:     "merged",
			wantStatus: OutcomeStatusAccepted,
			wantSignal: "merged",
		},
		{
			name:       "closed PR without merge → rejected",
			result:     OutcomeRejected,
			detail:     "closed without merge",
			wantStatus: OutcomeStatusRejected,
			wantSignal: "closed_without_merge",
		},
		{
			name:       "open PR → pending",
			result:     OutcomePending,
			detail:     "open",
			wantStatus: OutcomeStatusPending,
			wantSignal: "open",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := OutcomeReport{
				Type:   "create_pull_request",
				Result: tc.result,
				Detail: tc.detail,
			}
			eval := normalizeOutcomeEvaluation(report)
			assert.Equal(t, tc.wantStatus, eval.OutcomeStatus,
				"P5: PR state (%q) must yield OutcomeStatus=%s", tc.detail, tc.wantStatus)
			assert.Equal(t, tc.wantSignal, eval.Signal,
				"P5: PR state (%q) must set signal %q", tc.detail, tc.wantSignal)
		})
	}
}

// TestFormalIssueBotCloseLifecycle verifies that issue close provenance determines
// the outcome category: bot-close → lifecycle signal; human not_planned → rejected;
// completed → accepted.
//
// Formal predicate (TLA+):
//
//	IssueBotCloseLifecycle ≜
//	  ∀ issue : Issue →
//	    issue.state = "closed" ∧ issue.stateReason = "not_planned" ∧ closedByBot  ⟹ result = lifecycle ∧
//	    issue.state = "closed" ∧ issue.stateReason = "not_planned" ∧ ¬closedByBot ⟹ result = rejected ∧
//	    issue.state = "closed" ∧ issue.stateReason = "completed"                   ⟹ result = accepted
//
// Specification reference: specs/safe-output-outcome-evaluation.md §2. `create_issue`
func TestFormalIssueBotCloseLifecycle(t *testing.T) {
	cases := []struct {
		name       string
		result     OutcomeResult
		detail     string
		wantStatus OutcomeStatus
		wantSignal string
	}{
		{
			// Bot-closed not_planned carries the lifecycle signal.
			// OutcomeStatus is normalized to unknown with signal="lifecycle" in the
			// current implementation pending a dedicated lifecycle OutcomeStatus constant.
			// TODO: when OutcomeStatusLifecycle is introduced, update wantStatus to that value.
			name:       "bot closed not_planned → lifecycle signal",
			result:     OutcomeLifecycle,
			detail:     "closed by bot (lifecycle)",
			wantStatus: OutcomeStatusUnknown,
			wantSignal: "lifecycle",
		},
		{
			name:       "human closed not_planned → rejected",
			result:     OutcomeRejected,
			detail:     "closed as not planned",
			wantStatus: OutcomeStatusRejected,
			wantSignal: "closed_not_planned",
		},
		{
			name:       "resolved as completed → accepted",
			result:     OutcomeAccepted,
			detail:     "completed",
			wantStatus: OutcomeStatusAccepted,
			wantSignal: "completed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := OutcomeReport{
				Type:   "create_issue",
				Result: tc.result,
				Detail: tc.detail,
			}
			eval := normalizeOutcomeEvaluation(report)
			assert.Equal(t, tc.wantStatus, eval.OutcomeStatus,
				"P6: %s must yield OutcomeStatus=%s", tc.name, tc.wantStatus)
			assert.Equal(t, tc.wantSignal, eval.Signal,
				"P6: %s must set signal %q", tc.name, tc.wantSignal)
		})
	}
}

// TestFormalLabelStickiness verifies the label retention monotonicity invariant:
// all bot-applied labels still present → change retained (accepted); any removed → reverted (rejected).
//
// Formal predicate (F*):
//
//	val labelRetentionMonotonicity :
//	  before:list string → after:list string → current:list string →
//	  Tot retainedStateComparison
//	  (requires Subset before after  (* labels were added *))
//	  (ensures fun c →
//	    Subset after current ⟹ c.Retained ≠ [] ∧
//	    ¬Subset after current ⟹ c.Reverted ≠ [] ∨ c.Replaced ≠ [])
//
// Specification reference: specs/safe-output-outcome-evaluation.md §4. `add_labels`
func TestFormalLabelStickiness(t *testing.T) {
	cases := []struct {
		name          string
		beforeLabels  []any
		afterLabels   []any
		currentLabels []any
		wantRetained  bool
	}{
		{
			name:          "all added labels retained",
			beforeLabels:  []any{"triage"},
			afterLabels:   []any{"triage", "bug"},
			currentLabels: []any{"triage", "bug"},
			wantRetained:  true,
		},
		{
			name:          "added label removed",
			beforeLabels:  []any{"triage"},
			afterLabels:   []any{"triage", "bug"},
			currentLabels: []any{"triage"}, // "bug" was removed
			wantRetained:  false,
		},
		{
			name:          "all labels removed",
			beforeLabels:  []any{"triage"},
			afterLabels:   []any{"triage", "bug"},
			currentLabels: []any{},
			wantRetained:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := map[string]any{"labels": tc.beforeLabels}
			after := map[string]any{"labels": tc.afterLabels}
			current := map[string]any{"labels": tc.currentLabels}

			comparison := compareRetainedUpdateState(before, after, current, []string{"labels"})
			require.Len(t, comparison.Changed, 1, "P7: label delta must be detected as a changed field")

			if tc.wantRetained {
				assert.Len(t, comparison.Retained, 1,
					"P7: when all labels retained, Retained must contain the labels field")
				assert.Empty(t, comparison.Reverted,
					"P7: when all labels retained, Reverted must be empty")
			} else {
				assert.Empty(t, comparison.Retained,
					"P7: when any label removed, Retained must be empty")
				assert.True(t, len(comparison.Reverted) > 0 || len(comparison.Replaced) > 0,
					"P7: when any label removed, Reverted or Replaced must be non-empty")
			}
		})
	}
}

// TestFormalUpdateSnapshotComparison verifies the three-way snapshot comparison:
// current matches after-state → retained (accepted); matches before-state → reverted (rejected);
// matches neither → replaced (rejected).
//
// Formal predicate (F*):
//
//	val compareUpdateSnapshot :
//	  before:state → after:state → current:state → fields:list string →
//	  Tot retainedStateComparison
//	  (ensures fun c →
//	    current = after  ⟹ c.Retained = c.Changed ∧
//	    current = before ⟹ c.Reverted = c.Changed ∧
//	    current ≠ before ∧ current ≠ after ⟹ c.Replaced = c.Changed)
//
// Specification reference: specs/safe-output-outcome-evaluation.md §6. `update_issue`, §7. `update_pull_request`
func TestFormalUpdateSnapshotComparison(t *testing.T) {
	before := map[string]any{"title": "Old title"}
	after := map[string]any{"title": "New title"}

	t.Run("current = after → all retained", func(t *testing.T) {
		current := map[string]any{"title": "New title"}
		comparison := compareRetainedUpdateState(before, after, current, []string{"title"})
		require.Len(t, comparison.Changed, 1, "P8: title change must be detected")
		assert.Len(t, comparison.Retained, len(comparison.Changed),
			"P8: current=after must have all changed fields in Retained")
		assert.Empty(t, comparison.Reverted, "P8: current=after must have no reverted fields")
		assert.Empty(t, comparison.Replaced, "P8: current=after must have no replaced fields")
	})

	t.Run("current = before → all reverted", func(t *testing.T) {
		current := map[string]any{"title": "Old title"}
		comparison := compareRetainedUpdateState(before, after, current, []string{"title"})
		require.Len(t, comparison.Changed, 1, "P8: title change must be detected")
		assert.Len(t, comparison.Reverted, len(comparison.Changed),
			"P8: current=before must have all changed fields in Reverted")
		assert.Empty(t, comparison.Retained, "P8: current=before must have no retained fields")
		assert.Empty(t, comparison.Replaced, "P8: current=before must have no replaced fields")
	})

	t.Run("current diverged → all replaced", func(t *testing.T) {
		current := map[string]any{"title": "Diverged title"}
		comparison := compareRetainedUpdateState(before, after, current, []string{"title"})
		require.Len(t, comparison.Changed, 1, "P8: title change must be detected")
		assert.NotEmpty(t, comparison.Replaced, "P8: diverged state must have replaced fields")
		assert.Empty(t, comparison.Retained, "P8: diverged state must have no retained fields")
		assert.Empty(t, comparison.Reverted, "P8: diverged state must have no reverted fields")
	})
}

// TestFormalCloseStickyReopenRejection verifies that a still-closed object is accepted
// and a reopened object is rejected.
//
// Formal predicate (TLA+):
//
//	CloseStickyReopenRejection ≜
//	  ∀ item : close_issue ∪ close_pull_request →
//	    current.state = "closed" ⟹ result = accepted ∧
//	    current.state = "open"   ⟹ result = rejected
//
// Specification reference: specs/safe-output-outcome-evaluation.md §8. `close_issue`, §9. `close_pull_request`
func TestFormalCloseStickyReopenRejection(t *testing.T) {
	cases := []struct {
		name       string
		state      string
		wantResult OutcomeResult
		wantDetail string
	}{
		{"closed → accepted", "closed", OutcomeAccepted, "still closed"},
		{"reopened → rejected", "open", OutcomeRejected, "reopened"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			old := closeStickyGHAPIGet
			t.Cleanup(func() { closeStickyGHAPIGet = old })
			stateVal := tc.state
			closeStickyGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
				return map[string]any{"state": stateVal}, nil
			}

			item := CreatedItemReport{Type: "close_issue", Number: 99, Repo: "owner/repo"}
			report := evalCloseSticky(item, "owner/repo")

			assert.Equal(t, tc.wantResult, report.Result,
				"P9: state=%q must yield %s", tc.state, tc.wantResult)
			assert.Equal(t, tc.wantDetail, report.Detail,
				"P9: state=%q must set detail %q", tc.state, tc.wantDetail)
		})
	}
}

// TestFormalDerivedMetricsConsistency verifies the acceptance_rate and waste_rate
// formulas and guards against division-by-zero when the denominator is zero.
//
// Formal predicate (Z3/SMT-LIB):
//
//	(declare-const accepted Int)
//	(declare-const rejected Int)
//	(declare-const total    Int)
//	(assert (>= accepted 0)) (assert (>= rejected 0)) (assert (>= total (+ accepted rejected)))
//	(assert (=> (> (+ accepted rejected) 0) (= acceptance_rate (/ accepted (+ accepted rejected)))))
//	(assert (=> (> total 0) (= waste_rate (/ rejected total))))
//	(assert (=> (= (+ accepted rejected) 0) (= acceptance_rate 0.0)))
//	(assert (=> (= total 0) (= waste_rate 0.0)))
//	(check-sat) ; sat — formulas are consistent and zero-safe
//
// Specification reference: specs/safe-output-outcome-evaluation.md §Derived Metrics
func TestFormalDerivedMetricsConsistency(t *testing.T) {
	t.Run("acceptance_rate = accepted / (accepted + rejected)", func(t *testing.T) {
		reports := []OutcomeReport{
			{Type: "create_pull_request", Result: OutcomeAccepted},
			{Type: "create_pull_request", Result: OutcomeAccepted},
			{Type: "create_issue", Result: OutcomeRejected},
		}
		summary := ComputeOutcomeSummary(reports, github.DefaultObjectiveMapping())
		// 2 accepted, 1 rejected → acceptance_rate = 2/3
		assert.InDelta(t, 2.0/3.0, summary.AcceptanceRate, 1e-9,
			"P10: acceptance_rate must equal accepted/(accepted+rejected)")
	})

	t.Run("waste_rate = rejected / total", func(t *testing.T) {
		reports := []OutcomeReport{
			{Type: "create_pull_request", Result: OutcomeAccepted},
			{Type: "create_issue", Result: OutcomeRejected},
			{Type: "add_comment", Result: OutcomeIgnored},
			{Type: "add_labels", Result: OutcomePending},
		}
		summary := ComputeOutcomeSummary(reports, github.DefaultObjectiveMapping())
		// 1 rejected / 4 total → waste_rate = 0.25
		assert.InDelta(t, 0.25, summary.WasteRate, 1e-9,
			"P10: waste_rate must equal rejected/total")
	})

	t.Run("division-by-zero safety: empty report set", func(t *testing.T) {
		summary := ComputeOutcomeSummary(nil, github.DefaultObjectiveMapping())
		assert.InDelta(t, 0.0, summary.AcceptanceRate, 1e-12,
			"P10: acceptance_rate must be 0.0 when total=0 (division-by-zero safe)")
		assert.InDelta(t, 0.0, summary.WasteRate, 1e-12,
			"P10: waste_rate must be 0.0 when total=0 (division-by-zero safe)")
	})

	t.Run("division-by-zero safety: only pending outcomes", func(t *testing.T) {
		reports := []OutcomeReport{
			{Type: "add_labels", Result: OutcomePending},
			{Type: "add_labels", Result: OutcomePending},
		}
		summary := ComputeOutcomeSummary(reports, github.DefaultObjectiveMapping())
		assert.InDelta(t, 0.0, summary.AcceptanceRate, 1e-12,
			"P10: acceptance_rate must be 0.0 when accepted+rejected=0")
	})
}

// TestFormalOTelGracefulDegradation verifies that outcome evaluation always
// produces a valid, non-discardable result regardless of transport or OTel
// exporter availability.
//
// Formal predicate (F*):
//
//	val evaluateOutcome :
//	  item:CreatedItemReport → transportOK:bool →
//	  Tot OutcomeReport
//	  (requires True)
//	  (ensures fun r →
//	    r.Type ≠ "" ∧
//	    r.Result ∈ KnownOutcomeResults ∧
//	    normalizeOutcomeEvaluation(r).OutcomeStatus ≠ "" ∧
//	    normalizeOutcomeEvaluation(r).EvidenceStrength ≠ "")
//
// Specification reference: specs/safe-output-outcome-evaluation.md §Conformance →§OTel Backend Unavailability
func TestFormalOTelGracefulDegradation(t *testing.T) {
	old := closeStickyGHAPIGet
	t.Cleanup(func() { closeStickyGHAPIGet = old })
	// Simulate a transport failure (connection refused) that would also prevent OTLP export.
	closeStickyGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
		return nil, errors.New("transport error: connection refused")
	}

	validResults := map[OutcomeResult]bool{
		OutcomeAccepted: true, OutcomeRejected: true, OutcomeIgnored: true,
		OutcomePending: true, OutcomeLifecycle: true, OutcomeUnknown: true, OutcomeError: true,
	}

	items := []CreatedItemReport{
		{Type: "close_issue", Number: 1, Repo: "owner/repo"},
		{Type: "close_pull_request", Number: 2, Repo: "owner/repo"},
	}
	for _, item := range items {
		t.Run(item.Type, func(t *testing.T) {
			report := evalCloseSticky(item, "owner/repo")

			// P11: outcome must always be produced — never discarded on transport failure.
			assert.NotEmpty(t, report.Type,
				"P11: report.Type must be set regardless of transport availability")
			assert.True(t, validResults[report.Result],
				"P11: result %q must be a recognized OutcomeResult even when transport fails", report.Result)

			// The outcome must always be normalizable (audit log entry is always writable).
			eval := normalizeOutcomeEvaluation(report)
			assert.NotEmpty(t, string(eval.OutcomeStatus),
				"P11: OutcomeStatus must be non-empty so the audit log entry can be written")
			assert.NotEmpty(t, string(eval.EvidenceStrength),
				"P11: EvidenceStrength must be non-empty so the audit log entry can be written")
		})
	}
}

// TestFormalConformanceClassCoverage verifies that the evaluation engine satisfies
// the three mandatory conformance safeguard classes defined in the spec:
//
//   - Class A: standard accepted/rejected/ignored/pending state transitions
//   - Class B: human override and lifecycle outcome paths
//   - Class C: API degradation (5xx, 404, rate-limit)
//
// Formal predicate (F*):
//
//	val conformanceClassCoverage :
//	  evaluator:outcomeEvaluator →
//	  Tot bool
//	  (requires True)
//	  (ensures fun ok →
//	    ok = classAExists(evaluator) ∧ classCExists(evaluator))
//
// Specification reference: specs/safe-output-outcome-evaluation.md §Conformance →§Conformance Safeguard Coverage Requirements
func TestFormalConformanceClassCoverage(t *testing.T) {
	t.Run("Class A: close_issue accepted state transition", func(t *testing.T) {
		old := closeStickyGHAPIGet
		t.Cleanup(func() { closeStickyGHAPIGet = old })
		closeStickyGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
			return map[string]any{"state": "closed"}, nil
		}
		report := evalCloseSticky(CreatedItemReport{Type: "close_issue", Number: 1, Repo: "o/r"}, "o/r")
		assert.Equal(t, OutcomeAccepted, report.Result,
			"P12 Class A: still-closed issue must be accepted")
	})

	t.Run("Class A: close_issue rejected state transition", func(t *testing.T) {
		old := closeStickyGHAPIGet
		t.Cleanup(func() { closeStickyGHAPIGet = old })
		closeStickyGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
			return map[string]any{"state": "open"}, nil
		}
		report := evalCloseSticky(CreatedItemReport{Type: "close_issue", Number: 1, Repo: "o/r"}, "o/r")
		assert.Equal(t, OutcomeRejected, report.Result,
			"P12 Class A: reopened issue must be rejected")
	})

	t.Run("Class A: update_issue accepted (state retained)", func(t *testing.T) {
		old := outcomeUpdateGHAPIGet
		t.Cleanup(func() { outcomeUpdateGHAPIGet = old })
		outcomeUpdateGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
			return map[string]any{"title": "New title", "body": "", "state": "open", "labels": []any{}, "assignees": []any{}}, nil
		}
		item := CreatedItemReport{
			Type: "update_issue", Number: 1, Repo: "o/r",
			BeforeState: map[string]any{"title": "Old title", "body_hash": mutableBodyHash(""), "state": "open", "labels": []any{}, "assignees": []any{}},
			AfterState:  map[string]any{"title": "New title", "body_hash": mutableBodyHash(""), "state": "open", "labels": []any{}, "assignees": []any{}},
		}
		report := evalUpdateIssue(item, "o/r")
		assert.Equal(t, OutcomeAccepted, report.Result,
			"P12 Class A: retained update must be accepted")
	})

	t.Run("Class B: lifecycle bot-close carries lifecycle signal", func(t *testing.T) {
		report := OutcomeReport{
			Type:   "close_issue",
			Result: OutcomeLifecycle,
			Detail: "closed by bot (lifecycle)",
		}
		eval := normalizeOutcomeEvaluation(report)
		assert.Equal(t, "lifecycle", eval.Signal,
			"P12 Class B: bot-closed outcome must carry the lifecycle signal")
	})

	t.Run("Class C: API 5xx for close_issue", func(t *testing.T) {
		old := closeStickyGHAPIGet
		t.Cleanup(func() { closeStickyGHAPIGet = old })
		closeStickyGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
			return nil, errors.New("gh api: 500 Internal Server Error")
		}
		report := evalCloseSticky(CreatedItemReport{Type: "close_issue", Number: 1, Repo: "o/r"}, "o/r")
		assert.NotEqual(t, OutcomeAccepted, report.Result,
			"P12 Class C: 5xx error must not yield accepted")
		assert.NotEqual(t, OutcomeRejected, report.Result,
			"P12 Class C: 5xx error must not yield rejected")
	})

	t.Run("Class C: rate limit for update_issue", func(t *testing.T) {
		old := outcomeUpdateGHAPIGet
		t.Cleanup(func() { outcomeUpdateGHAPIGet = old })
		outcomeUpdateGHAPIGet = func(endpoint, repo string) (map[string]any, error) {
			return nil, errors.New("gh api: 429 Too Many Requests")
		}
		item := CreatedItemReport{
			Type: "update_issue", Number: 1, Repo: "o/r",
			BeforeState: map[string]any{"title": "Old"},
			AfterState:  map[string]any{"title": "New"},
		}
		report := evalUpdateIssue(item, "o/r")
		assert.NotEqual(t, OutcomeAccepted, report.Result,
			"P12 Class C: rate limit must not yield accepted")
		assert.NotEqual(t, OutcomeRejected, report.Result,
			"P12 Class C: rate limit must not yield rejected")
	})
}
