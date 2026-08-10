//go:build !integration

package intent_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Formal conformance model for the drift escalation norm in
// specs/intent-attribution-agent-governance.md ("Sync Notes" → Escalation norm):
// a sync warning that persists across 3 or more consecutive CI runs without a
// corrective PR or explicit waiver MUST escalate to a compliance failure, the drift
// check MUST exit non-zero, and a tracking issue MUST be opened or updated.
//
// The counter is scoped per repository and per drifted key: consecutive runs are
// counted across CI runs of the drift-detection workflow within a single repository,
// and the streak resets when a run observes no drift for that key.

const driftEscalationThreshold = 3

// driftRun models the drift-detection outcome of a single CI run for one repository.
type driftRun struct {
	DriftedKeys []string
	// CorrectivePR is true when a corrective pull request landed for this run.
	CorrectivePR bool
	// Waived lists keys covered by an explicit waiver for this run.
	Waived []string
}

// driftState models the per-repository, per-key consecutive warning counters.
type driftState struct {
	Repository string
	Streaks    map[string]int
	FirstSeen  map[string]int
}

func newDriftState(repository string) *driftState {
	return &driftState{
		Repository: repository,
		Streaks:    map[string]int{},
		FirstSeen:  map[string]int{},
	}
}

// driftOutcome models the result of applying one CI run to the drift state.
type driftOutcome struct {
	EscalatedKeys []string
	ExitCode      int
	TrackingIssue *driftTrackingIssue
}

// driftTrackingIssue models the tracking issue that MUST be opened or updated on
// escalation.
type driftTrackingIssue struct {
	Repository       string
	Keys             []string
	FirstDetectedRun int
	Assignee         string
}

func contains(values []string, value string) bool {
	return slices.Contains(values, value)
}

// observeDriftRun applies a single CI run to the per-repository state and returns the
// escalation outcome for that run.
func (s *driftState) observeDriftRun(runIndex int, run driftRun, onCallMaintainer string) driftOutcome {
	// A corrective PR clears every outstanding warning for the repository.
	if run.CorrectivePR {
		s.Streaks = map[string]int{}
		s.FirstSeen = map[string]int{}
		return driftOutcome{ExitCode: 0}
	}

	seen := map[string]struct{}{}
	escalated := make([]string, 0)
	for _, key := range run.DriftedKeys {
		seen[key] = struct{}{}
		// An explicit waiver suppresses the warning and resets its streak.
		if contains(run.Waived, key) {
			delete(s.Streaks, key)
			delete(s.FirstSeen, key)
			continue
		}
		if _, ok := s.FirstSeen[key]; !ok {
			s.FirstSeen[key] = runIndex
		}
		s.Streaks[key]++
		if s.Streaks[key] >= driftEscalationThreshold {
			escalated = append(escalated, key)
		}
	}

	// Keys not observed in this run reset: the streak counts *consecutive* runs.
	for key := range s.Streaks {
		if _, ok := seen[key]; !ok {
			delete(s.Streaks, key)
			delete(s.FirstSeen, key)
		}
	}

	if len(escalated) == 0 {
		return driftOutcome{ExitCode: 0}
	}

	firstDetected := runIndex
	for _, key := range escalated {
		if first, ok := s.FirstSeen[key]; ok && first < firstDetected {
			firstDetected = first
		}
	}

	return driftOutcome{
		EscalatedKeys: escalated,
		ExitCode:      1,
		TrackingIssue: &driftTrackingIssue{
			Repository:       s.Repository,
			Keys:             escalated,
			FirstDetectedRun: firstDetected,
			Assignee:         onCallMaintainer,
		},
	}
}

// TestFormalDrift_EscalatesAtThirdConsecutiveRun asserts the escalation norm: the first
// two consecutive warnings stay warnings, and the third escalates to a compliance
// failure with a non-zero exit code and a tracking issue.
func TestFormalDrift_EscalatesAtThirdConsecutiveRun(t *testing.T) {
	state := newDriftState("github/gh-aw")
	run := driftRun{DriftedKeys: []string{"intent.security"}}

	first := state.observeDriftRun(1, run, "@on-call")
	assert.Empty(t, first.EscalatedKeys, "run 1 is a warning, not a failure")
	assert.Equal(t, 0, first.ExitCode)
	assert.Nil(t, first.TrackingIssue)

	second := state.observeDriftRun(2, run, "@on-call")
	assert.Empty(t, second.EscalatedKeys, "run 2 is still a warning")
	assert.Equal(t, 0, second.ExitCode)

	third := state.observeDriftRun(3, run, "@on-call")
	require.Equal(t, []string{"intent.security"}, third.EscalatedKeys, "run 3 must escalate")
	assert.Equal(t, 1, third.ExitCode, "escalation must fail the drift check with a non-zero exit code")

	require.NotNil(t, third.TrackingIssue, "escalation must open or update a tracking issue")
	assert.Equal(t, "github/gh-aw", third.TrackingIssue.Repository)
	assert.Equal(t, []string{"intent.security"}, third.TrackingIssue.Keys)
	assert.Equal(t, 1, third.TrackingIssue.FirstDetectedRun, "the tracking issue records the first-detected run")
	assert.Equal(t, "@on-call", third.TrackingIssue.Assignee, "the on-call maintainer is the default assignee")
}

// TestFormalDrift_StreakResetsWhenDriftClears asserts that the counter measures
// *consecutive* runs: a clean run in between prevents escalation.
func TestFormalDrift_StreakResetsWhenDriftClears(t *testing.T) {
	state := newDriftState("github/gh-aw")
	drifted := driftRun{DriftedKeys: []string{"intent.security"}}

	state.observeDriftRun(1, drifted, "@on-call")
	state.observeDriftRun(2, drifted, "@on-call")
	clean := state.observeDriftRun(3, driftRun{}, "@on-call")
	assert.Equal(t, 0, clean.ExitCode)

	fourth := state.observeDriftRun(4, drifted, "@on-call")
	assert.Empty(t, fourth.EscalatedKeys, "the streak restarts after a clean run")
	assert.Equal(t, 0, fourth.ExitCode)
}

// TestFormalDrift_CorrectivePRAndWaiverSuppressEscalation asserts that a corrective PR
// or an explicit waiver prevents escalation.
func TestFormalDrift_CorrectivePRAndWaiverSuppressEscalation(t *testing.T) {
	withPR := newDriftState("github/gh-aw")
	drifted := driftRun{DriftedKeys: []string{"intent.security"}}
	withPR.observeDriftRun(1, drifted, "@on-call")
	withPR.observeDriftRun(2, drifted, "@on-call")
	withPR.observeDriftRun(3, driftRun{DriftedKeys: []string{"intent.security"}, CorrectivePR: true}, "@on-call")
	after := withPR.observeDriftRun(4, drifted, "@on-call")
	assert.Equal(t, 0, after.ExitCode, "a corrective PR clears outstanding warnings")

	withWaiver := newDriftState("github/gh-aw")
	waived := driftRun{DriftedKeys: []string{"intent.security"}, Waived: []string{"intent.security"}}
	withWaiver.observeDriftRun(1, drifted, "@on-call")
	withWaiver.observeDriftRun(2, drifted, "@on-call")
	waivedOutcome := withWaiver.observeDriftRun(3, waived, "@on-call")
	assert.Equal(t, 0, waivedOutcome.ExitCode, "an explicit waiver suppresses the warning")
	assert.Empty(t, waivedOutcome.EscalatedKeys)
}

// TestFormalDrift_CounterScopeIsPerRepositoryAndKey asserts the documented counting
// scope: streaks accumulate per repository and per drifted key, never across
// repositories.
func TestFormalDrift_CounterScopeIsPerRepositoryAndKey(t *testing.T) {
	repoA := newDriftState("github/gh-aw")
	repoB := newDriftState("github/other")
	drifted := driftRun{DriftedKeys: []string{"intent.security"}}

	repoA.observeDriftRun(1, drifted, "@on-call")
	repoA.observeDriftRun(2, drifted, "@on-call")
	outcomeB := repoB.observeDriftRun(3, drifted, "@on-call")
	assert.Equal(t, 0, outcomeB.ExitCode, "another repository's runs do not advance this repository's streak")

	// Distinct keys maintain independent streaks within one repository.
	repoC := newDriftState("github/gh-aw")
	repoC.observeDriftRun(1, driftRun{DriftedKeys: []string{"intent.security"}}, "@on-call")
	repoC.observeDriftRun(2, driftRun{DriftedKeys: []string{"intent.security", "intent.docs"}}, "@on-call")
	outcome := repoC.observeDriftRun(3, driftRun{DriftedKeys: []string{"intent.security", "intent.docs"}}, "@on-call")

	require.Equal(t, []string{"intent.security"}, outcome.EscalatedKeys,
		"only the key seen in 3 consecutive runs escalates")
	assert.Equal(t, 1, outcome.ExitCode)
}
