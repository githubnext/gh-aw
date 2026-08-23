//go:build !integration

package workflow

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// This formal test suite validates the suppression lifecycle norms in
// spec §6.4 (False-Positive Handling, T-CTR-024/025/029) against the
// concrete implementation in threat_detection_suppression.go and the
// compiled lock-file manifest emitted by generateWorkflowHeader.
//
// The rule deprecation lifecycle (spec §5.4) is verified against the
// specification artifacts in threat_detection_deprecation_policy_formal_test.go.

func TestFormal_SuppressionRequiresRuleAndReason(t *testing.T) {
	cases := []struct {
		name    string
		reason  string
		wantErr bool
	}{
		{name: "missing reason", reason: "", wantErr: true},
		{name: "whitespace-only reason", reason: "   ", wantErr: true},
		{name: "well-formed reason", reason: "safe because inputs are static", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateThreatDetectionSuppressions([]ThreatDetectionSuppression{
				{Rule: "CTR-001", Reason: tc.reason},
			})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFormal_SuppressionRuleFormatWellFormed(t *testing.T) {
	cases := []struct {
		name    string
		rule    string
		wantErr bool
	}{
		{name: "empty rule", rule: "", wantErr: true},
		{name: "malformed rule (no digits)", rule: "CTR-", wantErr: true},
		{name: "malformed rule (too few digits)", rule: "CTR-01", wantErr: true},
		{name: "malformed rule (lowercase)", rule: "ctr-001", wantErr: true},
		{name: "well-formed rule", rule: "CTR-001", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateThreatDetectionSuppressions([]ThreatDetectionSuppression{
				{Rule: tc.rule, Reason: "valid reason"},
			})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFormal_SuppressionExpiresISO8601OrAbsent(t *testing.T) {
	cases := []struct {
		name    string
		expires string
		wantErr bool
	}{
		{name: "absent expires", expires: "", wantErr: false},
		{name: "valid ISO 8601 date", expires: "2026-12-31", wantErr: false},
		{name: "non-ISO format", expires: "12/31/2026", wantErr: true},
		{name: "invalid calendar date", expires: "2026-02-30", wantErr: true},
		{name: "invalid month", expires: "2026-13-01", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateThreatDetectionSuppressions([]ThreatDetectionSuppression{
				{Rule: "CTR-001", Reason: "valid reason", Expires: tc.expires},
			})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFormal_ActiveSuppressionRetainsAuditFields(t *testing.T) {
	value := []any{
		map[string]any{
			"rule":    "CTR-011",
			"reason":  "documented exception for internal domain",
			"expires": "2026-12-31",
		},
	}

	parsed, err := parseThreatDetectionSuppressions(value)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.Equal(t, "CTR-011", parsed[0].Rule)
	require.Equal(t, "documented exception for internal domain", parsed[0].Reason)
	require.Equal(t, "2026-12-31", parsed[0].Expires)

	// T-CTR-025 requires the compiled lock file to retain the audit fields,
	// so assert the serialized gh-aw-manifest, not just the parsed struct.
	var header strings.Builder
	require.NoError(t, (&Compiler{}).generateWorkflowHeader(&header, &WorkflowData{
		RawFrontmatter: map[string]any{
			"on":                        map[string]any{"schedule": "daily"},
			"threat-detection-suppress": value,
		},
	}, "", "", nil, nil))
	require.Contains(t, header.String(), `"threat_detection_suppressions":[{"rule":"CTR-011","reason":"documented exception for internal domain","expires":"2026-12-31"}]`)
}

func TestFormal_ExpiredSuppressionTreatedAsAbsent(t *testing.T) {
	now := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	suppressions := []ThreatDetectionSuppression{
		{Rule: "CTR-011", Reason: "past exception", Expires: "2026-03-14"},
	}

	require.False(t, isThreatDetectionRuleSuppressed(suppressions, "CTR-011", now))
	require.Empty(t, activeThreatDetectionSuppressions(suppressions, now))
}

func TestFormal_SuppressionBoundaryDayStillActive(t *testing.T) {
	suppressions := []ThreatDetectionSuppression{
		{Rule: "CTR-011", Reason: "boundary-day exception", Expires: "2026-03-15"},
	}

	sameDay := time.Date(2026, time.March, 15, 23, 59, 59, 0, time.UTC)
	require.True(t, isThreatDetectionRuleSuppressed(suppressions, "CTR-011", sameDay))

	nextDay := time.Date(2026, time.March, 16, 0, 0, 0, 0, time.UTC)
	require.False(t, isThreatDetectionRuleSuppressed(suppressions, "CTR-011", nextDay))
}

func TestFormal_DiagnosticSuppressionRequiresMatchingRule(t *testing.T) {
	now := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	suppressions := []ThreatDetectionSuppression{
		{Rule: "CTR-011", Reason: "unrelated rule suppressed"},
	}

	diagnosticForDifferentRule := &threatDetectionDiagnosticError{
		Rule: "CTR-012",
		Err:  errors.New("wildcard push scope misconfigured"),
	}
	require.False(t, isThreatDetectionDiagnosticSuppressed(diagnosticForDifferentRule, suppressions, now))

	diagnosticForSuppressedRule := &threatDetectionDiagnosticError{
		Rule: "CTR-011",
		Err:  errors.New("firewall dependency missing"),
	}
	require.True(t, isThreatDetectionDiagnosticSuppressed(diagnosticForSuppressedRule, suppressions, now))
}
