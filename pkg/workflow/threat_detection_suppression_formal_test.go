//go:build !integration

package workflow

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Formal conformance model for specs/compiler-threat-detection-spec.md §6.4
// (False-Positive Handling) and §6.6 (Optimizer Failure Safeguards).
//
// The norms in §6.4 and §6.6 describe the behavior of the daily optimizer and of the
// `threat-detection-suppress` frontmatter key. This file models those norms as pure
// functions so that the normative thresholds (missing `reason` rejection, lock-file
// manifest shape, 10/20 business-day SLA windows, expiration handling, and the
// `OPTIMIZER_*` diagnostic shapes) are executable and regression-protected, mirroring
// the pattern used by awf_config_safeguards_formal_test.go for
// specs/awf-config-sources-spec.md §7.

const (
	ctrSuppressionSLABusinessDays        = 10
	ctrSuppressionEscalationBusinessDays = 20
)

// ctrSuppression models one entry of the `threat-detection-suppress` frontmatter list
// (§6.4 item 1).
type ctrSuppression struct {
	Rule    string
	Reason  string
	Expires time.Time // zero value means no expiration
	Created time.Time
	Owner   string
}

// ctrSLABreach models one `SLA_BREACH` daily-output entry (§6.4 item 4).
type ctrSLABreach struct {
	Rule            string
	Reason          string
	AgeBusinessDays int
	Owner           string
	Expires         string
}

// ctrValidateSuppression models the compiler validation required by §6.4 item 1:
// a suppression without a `rule` or with an absent/empty `reason` MUST NOT be accepted.
func ctrValidateSuppression(s ctrSuppression) error {
	if strings.TrimSpace(s.Rule) == "" {
		return errSuppressionMissingRule
	}
	if strings.TrimSpace(s.Reason) == "" {
		return errSuppressionMissingReason
	}
	return nil
}

var (
	errSuppressionMissingRule   = &ctrValidationError{message: "threat-detection-suppress entry is missing required 'rule' field"}
	errSuppressionMissingReason = &ctrValidationError{message: "threat-detection-suppress entry for rule is missing required 'reason' field"}
)

type ctrValidationError struct {
	message string
}

func (e *ctrValidationError) Error() string { return e.message }

// ctrLockManifestEntry models the lock-file manifest record required by §6.4 item 2.
type ctrLockManifestEntry struct {
	Rule    string
	Reason  string
	Expires string
}

// ctrLockManifest renders the audit-trail manifest entries for the active suppressions.
func ctrLockManifest(suppressions []ctrSuppression) []ctrLockManifestEntry {
	entries := make([]ctrLockManifestEntry, 0, len(suppressions))
	for _, s := range suppressions {
		expires := ""
		if !s.Expires.IsZero() {
			expires = s.Expires.UTC().Format(time.RFC3339)
		}
		entries = append(entries, ctrLockManifestEntry{Rule: s.Rule, Reason: s.Reason, Expires: expires})
	}
	return entries
}

// ctrSuppressionApproved models §6.4 item 2: a suppression absent from the lock-file
// manifest is unapproved and MUST be re-evaluated against the current CTR rule.
func ctrSuppressionApproved(s ctrSuppression, manifest []ctrLockManifestEntry) bool {
	for _, entry := range manifest {
		if entry.Rule == s.Rule && entry.Reason == s.Reason {
			return true
		}
	}
	return false
}

// ctrSuppressionActive models §6.4 item 6: an expired suppression is treated by the
// compiler as if it does not exist.
func ctrSuppressionActive(s ctrSuppression, now time.Time) bool {
	if s.Expires.IsZero() {
		return true
	}
	return !now.After(s.Expires)
}

// ctrBusinessDaysBetween counts whole business days (Mon–Fri) elapsed between start and now.
func ctrBusinessDaysBetween(start, now time.Time) int {
	if !now.After(start) {
		return 0
	}
	days := 0
	cursor := start.UTC()
	for {
		cursor = cursor.AddDate(0, 0, 1)
		if cursor.After(now.UTC()) {
			break
		}
		if cursor.Weekday() != time.Saturday && cursor.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}

// ctrSLABreaches models §6.4 item 4: suppressions older than 10 business days MUST be
// emitted in daily output as `SLA_BREACH` entries carrying rule, reason,
// age_business_days, owner, and expires.
func ctrSLABreaches(suppressions []ctrSuppression, now time.Time) []ctrSLABreach {
	breaches := make([]ctrSLABreach, 0)
	for _, s := range suppressions {
		if !ctrSuppressionActive(s, now) {
			continue
		}
		age := ctrBusinessDaysBetween(s.Created, now)
		if age <= ctrSuppressionSLABusinessDays {
			continue
		}
		expires := ""
		if !s.Expires.IsZero() {
			expires = s.Expires.UTC().Format(time.RFC3339)
		}
		breaches = append(breaches, ctrSLABreach{
			Rule:            s.Rule,
			Reason:          s.Reason,
			AgeBusinessDays: age,
			Owner:           s.Owner,
			Expires:         expires,
		})
	}
	return breaches
}

// ctrRequiresEscalation models §6.4 item 5: suppressions older than 20 business days for
// MUST-level controls require a follow-up sync action in the same daily output.
func ctrRequiresEscalation(s ctrSuppression, mustLevel bool, now time.Time) bool {
	if !mustLevel {
		return false
	}
	return ctrBusinessDaysBetween(s.Created, now) > ctrSuppressionEscalationBusinessDays
}

// ctrOptimizerDegraded models the `OPTIMIZER_DEGRADED` diagnostic entry of §6.6
// Failure Mode 1.
type ctrOptimizerDegraded struct {
	Endpoints  []string
	ErrorClass string
	Timestamp  time.Time
}

// ctrOptimizerTimeout models the `OPTIMIZER_TIMEOUT` entry of §6.6 Failure Mode 2.
type ctrOptimizerTimeout struct {
	LastCompletedStep string
	UnevaluatedRules  []string
}

// ctrOptimizerRateLimited models the `OPTIMIZER_RATE_LIMITED` entry of §6.6 Failure Mode 3.
type ctrOptimizerRateLimited struct {
	Endpoints  []string
	RetryAfter string
}

// ctrMayOpenPullRequest models §6.6 Failure Mode 1 item 2 and Failure Mode 2 item 2:
// no pull request or spec update may be produced from a degraded or timed-out run.
func ctrMayOpenPullRequest(degraded bool, timedOut bool, rateLimited bool) bool {
	return !degraded && !timedOut && !rateLimited
}

// ctrCountsAsCompletedCycle models §6.6 Failure Mode 3 item 3: a rate-limited run MUST
// NOT count as a completed threat-coverage cycle.
func ctrCountsAsCompletedCycle(degraded bool, timedOut bool, rateLimited bool) bool {
	return !degraded && !timedOut && !rateLimited
}

// ctrBackoffDelays models the exponential back-off policy of §6.6 Failure Mode 1 item 3
// (initial delay 10s, maximum delay 5m, maximum 3 attempts).
func ctrBackoffDelays() []time.Duration {
	const (
		initial = 10 * time.Second
		maximum = 5 * time.Minute
		retries = 3
	)
	delays := make([]time.Duration, 0, retries)
	delay := initial
	for range retries {
		if delay > maximum {
			delay = maximum
		}
		delays = append(delays, delay)
		delay *= 2
	}
	return delays
}

// --- §6.4 False-Positive Handling ---

// T-CTR-6.4-1: a suppression missing `reason` is rejected by the compiler.
func TestFormalCTR64_SuppressionRequiresReason(t *testing.T) {
	valid := ctrSuppression{Rule: "CTR-006", Reason: "expression is a constant literal"}
	require.NoError(t, ctrValidateSuppression(valid))

	missingReason := ctrSuppression{Rule: "CTR-006"}
	err := ctrValidateSuppression(missingReason)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reason")

	blankReason := ctrSuppression{Rule: "CTR-006", Reason: "   "}
	require.Error(t, ctrValidateSuppression(blankReason))

	missingRule := ctrSuppression{Reason: "documented false positive"}
	err = ctrValidateSuppression(missingRule)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rule")
}

// T-CTR-6.4-2: every active suppression is recorded in the lock-file manifest with
// rule, reason, and expires.
func TestFormalCTR64_LockManifestRecordsSuppressionAuditTrail(t *testing.T) {
	expires := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	suppressions := []ctrSuppression{
		{Rule: "CTR-006", Reason: "constant literal expression", Expires: expires},
		{Rule: "CTR-014", Reason: "vendored package with pinned integrity"},
	}

	manifest := ctrLockManifest(suppressions)
	require.Len(t, manifest, 2)

	assert.Equal(t, "CTR-006", manifest[0].Rule)
	assert.Equal(t, "constant literal expression", manifest[0].Reason)
	assert.Equal(t, "2026-09-01T00:00:00Z", manifest[0].Expires)

	assert.Equal(t, "CTR-014", manifest[1].Rule)
	assert.Empty(t, manifest[1].Expires, "an omitted expires field renders as empty, not as a synthesized date")
}

// T-CTR-6.4-3: a suppression absent from the lock-file manifest is unapproved.
func TestFormalCTR64_SuppressionAbsentFromManifestIsUnapproved(t *testing.T) {
	recorded := ctrSuppression{Rule: "CTR-006", Reason: "constant literal expression"}
	manifest := ctrLockManifest([]ctrSuppression{recorded})

	assert.True(t, ctrSuppressionApproved(recorded, manifest))
	assert.False(t, ctrSuppressionApproved(ctrSuppression{Rule: "CTR-011", Reason: "internal domain"}, manifest))
	assert.False(t, ctrSuppressionApproved(ctrSuppression{Rule: "CTR-006", Reason: "edited after approval"}, manifest),
		"editing the reason invalidates the recorded approval")
}

// T-CTR-6.4-4: expired suppressions are treated as if they do not exist.
func TestFormalCTR64_ExpiredSuppressionIsInactive(t *testing.T) {
	expires := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	s := ctrSuppression{Rule: "CTR-006", Reason: "constant literal expression", Expires: expires}

	assert.True(t, ctrSuppressionActive(s, expires.Add(-time.Second)))
	assert.True(t, ctrSuppressionActive(s, expires), "the expiration instant itself is still valid")
	assert.False(t, ctrSuppressionActive(s, expires.Add(time.Second)))
	assert.True(t, ctrSuppressionActive(ctrSuppression{Rule: "CTR-006", Reason: "no expiry"}, expires.Add(time.Hour)))
}

// T-CTR-6.4-5: suppressions older than 10 business days are emitted as SLA_BREACH
// entries with the full documented shape.
func TestFormalCTR64_SLABreachEmissionShape(t *testing.T) {
	// 2026-08-10 is a Monday; 2026-07-24 is the Friday 11 business days earlier.
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	created := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	require.Equal(t, 11, ctrBusinessDaysBetween(created, now))

	suppressions := []ctrSuppression{
		{
			Rule:    "CTR-001",
			Reason:  "permission required by external reusable workflow",
			Created: created,
			Owner:   "@on-call",
			Expires: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Rule:    "CTR-011",
			Reason:  "internal domain resolved via proxy",
			Created: now.AddDate(0, 0, -3),
			Owner:   "@on-call",
		},
	}

	breaches := ctrSLABreaches(suppressions, now)
	require.Len(t, breaches, 1, "only the suppression older than 10 business days breaches the SLA")

	breach := breaches[0]
	assert.Equal(t, "CTR-001", breach.Rule)
	assert.Equal(t, "permission required by external reusable workflow", breach.Reason)
	assert.Equal(t, 11, breach.AgeBusinessDays)
	assert.Equal(t, "@on-call", breach.Owner)
	assert.Equal(t, "2026-09-01T00:00:00Z", breach.Expires)
}

// T-CTR-6.4-6: the SLA boundary is exclusive at exactly 10 business days, and expired
// suppressions are not reported as breaches.
func TestFormalCTR64_SLABoundaryAndExpiredSuppressions(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	tenBusinessDaysAgo := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	require.Equal(t, 10, ctrBusinessDaysBetween(tenBusinessDaysAgo, now))

	atBoundary := ctrSuppression{Rule: "CTR-001", Reason: "documented", Created: tenBusinessDaysAgo, Owner: "@on-call"}
	assert.Empty(t, ctrSLABreaches([]ctrSuppression{atBoundary}, now), "exactly 10 business days is within SLA")

	expired := ctrSuppression{
		Rule:    "CTR-001",
		Reason:  "documented",
		Created: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
		Expires: now.AddDate(0, 0, -1),
		Owner:   "@on-call",
	}
	assert.Empty(t, ctrSLABreaches([]ctrSuppression{expired}, now), "expired suppressions do not exist for SLA purposes")
}

// T-CTR-6.4-7: MUST-level suppressions older than 20 business days require escalation.
func TestFormalCTR64_EscalationAfterTwentyBusinessDays(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	twentyBusinessDaysAgo := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	require.Equal(t, 20, ctrBusinessDaysBetween(twentyBusinessDaysAgo, now))

	atBoundary := ctrSuppression{Rule: "CTR-001", Reason: "documented", Created: twentyBusinessDaysAgo}
	assert.False(t, ctrRequiresEscalation(atBoundary, true, now))

	beyondBoundary := ctrSuppression{Rule: "CTR-001", Reason: "documented", Created: twentyBusinessDaysAgo.AddDate(0, 0, -1)}
	assert.True(t, ctrRequiresEscalation(beyondBoundary, true, now))
	assert.False(t, ctrRequiresEscalation(beyondBoundary, false, now), "non-MUST-level controls do not escalate")
}

// T-CTR-6.4-8: weekend days are excluded from the business-day age computation.
func TestFormalCTR64_BusinessDayAgeExcludesWeekends(t *testing.T) {
	friday := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	monday := time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)

	assert.Equal(t, 1, ctrBusinessDaysBetween(friday, monday))
	assert.Equal(t, 0, ctrBusinessDaysBetween(monday, friday), "a future creation date yields zero age")
	assert.Equal(t, 0, ctrBusinessDaysBetween(friday, friday))
}

// --- §6.6 Optimizer Failure Safeguards ---

// T-CTR-6.6-1: OPTIMIZER_DEGRADED records endpoints, error class, and UTC timestamp.
func TestFormalCTR66_OptimizerDegradedDiagnosticShape(t *testing.T) {
	timestamp := time.Date(2026, 8, 10, 6, 30, 0, 0, time.UTC)
	entry := ctrOptimizerDegraded{
		Endpoints:  []string{"GET /repos/{owner}/{repo}/code-scanning/alerts"},
		ErrorClass: "503",
		Timestamp:  timestamp,
	}

	require.NotEmpty(t, entry.Endpoints)
	assert.NotEmpty(t, entry.ErrorClass)
	assert.Equal(t, time.UTC, entry.Timestamp.Location())
	assert.False(t, ctrMayOpenPullRequest(true, false, false), "a degraded run must not open a pull request")
	assert.False(t, ctrCountsAsCompletedCycle(true, false, false))
}

// T-CTR-6.6-2: OPTIMIZER_TIMEOUT records the last completed step and unevaluated rules.
func TestFormalCTR66_OptimizerTimeoutDiagnosticShape(t *testing.T) {
	entry := ctrOptimizerTimeout{
		LastCompletedStep: "evaluate-ctr-rules",
		UnevaluatedRules:  []string{"CTR-019", "CTR-020", "CTR-021"},
	}

	assert.NotEmpty(t, entry.LastCompletedStep)
	require.NotEmpty(t, entry.UnevaluatedRules)
	assert.False(t, ctrMayOpenPullRequest(false, true, false), "a timed-out run must discard in-progress artifacts")
	assert.False(t, ctrCountsAsCompletedCycle(false, true, false))
}

// T-CTR-6.6-3: OPTIMIZER_RATE_LIMITED records affected endpoints and the retry hint,
// and the run is not counted as a completed threat-coverage cycle.
func TestFormalCTR66_OptimizerRateLimitedDiagnosticShape(t *testing.T) {
	entry := ctrOptimizerRateLimited{
		Endpoints:  []string{"GET /search/issues"},
		RetryAfter: "60",
	}

	require.NotEmpty(t, entry.Endpoints)
	assert.NotEmpty(t, entry.RetryAfter)
	assert.False(t, ctrCountsAsCompletedCycle(false, false, true))
	assert.False(t, ctrMayOpenPullRequest(false, false, true))
}

// T-CTR-6.6-4: a healthy run is permitted to open a pull request and counts as a cycle.
func TestFormalCTR66_HealthyRunIsUnaffected(t *testing.T) {
	assert.True(t, ctrMayOpenPullRequest(false, false, false))
	assert.True(t, ctrCountsAsCompletedCycle(false, false, false))
}

// T-CTR-6.6-5: the retry policy uses at most 3 attempts with exponential back-off from
// 10 seconds and capped at 5 minutes.
func TestFormalCTR66_DegradedRetryBackoffPolicy(t *testing.T) {
	delays := ctrBackoffDelays()

	require.Len(t, delays, 3)
	assert.Equal(t, 10*time.Second, delays[0])
	for _, delay := range delays {
		assert.LessOrEqual(t, delay, 5*time.Minute)
	}
	for i := 1; i < len(delays); i++ {
		assert.Greater(t, delays[i], delays[i-1], "back-off must be strictly increasing until the cap")
	}
}
