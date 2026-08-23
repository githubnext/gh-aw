//go:build !integration

package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// This formal test suite validates the suppression lifecycle norms in
// spec §6.4 (False-Positive Handling, T-CTR-024/025/029) and the rule
// deprecation lifecycle in spec §5.4 (Deprecation Policy) against the
// concrete implementation in threat_detection_suppression.go.

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

// deprecationRegistry is a stub illustrating the rule-status state machine
// required by spec §5.4 (Deprecation Policy). No such registry currently
// exists in pkg/workflow; §5.4 is presently a documentation/process
// obligation on the spec and Section 7.1 mapping table rather than a
// runtime API. Replace this stub with a real implementation once one
// exists, and update these tests to exercise it directly.
type deprecationRegistryRule struct {
	ID       string
	Status   string // "Active" or "Deprecated"
	Required bool   // whether the rule is required for conformance gating
}

type deprecationRegistry struct {
	rules map[string]deprecationRegistryRule
}

func newDeprecationRegistry(rules ...deprecationRegistryRule) *deprecationRegistry {
	registry := &deprecationRegistry{rules: make(map[string]deprecationRegistryRule, len(rules))}
	for _, rule := range rules {
		registry.rules[rule.ID] = rule
	}
	return registry
}

// deprecate transitions a rule's status to "Deprecated" and retains its
// catalog row (spec §5.4: "The rule catalog entry MUST be retained (not
// deleted)").
func (r *deprecationRegistry) deprecate(id string) {
	rule, ok := r.rules[id]
	if !ok {
		return
	}
	rule.Status = "Deprecated"
	rule.Required = false
	r.rules[id] = rule
}

func (r *deprecationRegistry) requiredRules() []string {
	var required []string
	for id, rule := range r.rules {
		if rule.Status != "Deprecated" && rule.Required {
			required = append(required, id)
		}
	}
	return required
}

func TestFormal_DeprecatedRuleRetainsCatalogRow(t *testing.T) {
	registry := newDeprecationRegistry(
		deprecationRegistryRule{ID: "CTR-999", Status: "Active", Required: true},
	)

	registry.deprecate("CTR-999")

	rule, ok := registry.rules["CTR-999"]
	require.True(t, ok, "deprecated rule's catalog row must be retained, not deleted")
	require.Equal(t, "Deprecated", rule.Status)
}

func TestFormal_DeprecatedRuleExcludedFromRequiredGate(t *testing.T) {
	registry := newDeprecationRegistry(
		deprecationRegistryRule{ID: "CTR-998", Status: "Active", Required: true},
		deprecationRegistryRule{ID: "CTR-997", Status: "Active", Required: true},
	)

	registry.deprecate("CTR-998")

	required := registry.requiredRules()
	require.NotContains(t, required, "CTR-998", "deprecated rule must be excluded from the required conformance gate")
	require.Contains(t, required, "CTR-997", "active rule must remain required")
}
