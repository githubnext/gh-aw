//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formalThreatSuppressionNeedsSLABreach(ageBusinessDays int) bool {
	return ageBusinessDays > 10
}

func formalThreatSuppressionNeedsEscalation(ageBusinessDays int, mustLevel bool) bool {
	return mustLevel && ageBusinessDays > 20
}

func optimizerWorkflowSource(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "..", ".github", "workflows", "daily-compiler-threat-spec-optimizer.md"))
	require.NoError(t, err)
	return string(data)
}

func TestThreatSuppression_TCTR024_ReasonRequired(t *testing.T) {
	validFrontmatter := map[string]any{
		"on": map[string]any{"schedule": "daily"},
		"threat-detection-suppress": []any{map[string]any{
			"rule":   "CTR-006",
			"reason": "Expression is passed through a trusted environment variable.",
		}},
	}
	require.NoError(t, parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(validFrontmatter, "/tmp/threat-suppression-valid.md"))
	_, err := ParseFrontmatterConfig(validFrontmatter)
	require.NoError(t, err)

	missingReasonFrontmatter := map[string]any{
		"on":                        map[string]any{"schedule": "daily"},
		"threat-detection-suppress": []any{map[string]any{"rule": "CTR-006"}},
	}
	require.Error(t, parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(missingReasonFrontmatter, "/tmp/threat-suppression-missing-reason.md"))

	whitespaceReasonFrontmatter := map[string]any{
		"on": map[string]any{"schedule": "daily"},
		"threat-detection-suppress": []any{map[string]any{
			"rule": "CTR-006", "reason": "  ",
		}},
	}
	require.Error(t, parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(whitespaceReasonFrontmatter, "/tmp/threat-suppression-whitespace-reason.md"))

	require.NoError(t, validateThreatDetectionSuppressions([]ThreatDetectionSuppression{{
		Rule:   "CTR-006",
		Reason: "Expression is passed through a trusted environment variable.",
	}}))
	require.Error(t, validateThreatDetectionSuppressions([]ThreatDetectionSuppression{{Rule: "CTR-006"}}))
	require.Error(t, validateThreatDetectionSuppressions([]ThreatDetectionSuppression{{Rule: "CTR-006", Reason: "  "}}))
	require.Error(t, validateThreatDetectionSuppressions([]ThreatDetectionSuppression{{Rule: "invalid", Reason: "Safe"}}))

	var header strings.Builder
	err = (&Compiler{}).generateWorkflowHeader(&header, &WorkflowData{
		RawFrontmatter: missingReasonFrontmatter,
	}, "", "", nil, nil)
	require.ErrorContains(t, err, "reason must not be empty")
}

func TestThreatSuppression_TCTR025_AuditFields(t *testing.T) {
	suppression := ThreatDetectionSuppression{Rule: "CTR-006", Reason: "Reviewed safe", Expires: "2026-08-31"}
	manifest := GHAWManifest{ThreatDetectionSuppressions: activeThreatDetectionSuppressions(
		[]ThreatDetectionSuppression{suppression},
		time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	)}
	require.Len(t, manifest.ThreatDetectionSuppressions, 1)
	assert.Equal(t, suppression, manifest.ThreatDetectionSuppressions[0])
}

func TestThreatSuppression_TCTR026Through028_SLAAndEscalation(t *testing.T) {
	assert.False(t, formalThreatSuppressionNeedsSLABreach(10))
	assert.True(t, formalThreatSuppressionNeedsSLABreach(11))
	assert.False(t, formalThreatSuppressionNeedsEscalation(21, false))
	assert.False(t, formalThreatSuppressionNeedsEscalation(20, true))
	assert.True(t, formalThreatSuppressionNeedsEscalation(21, true))

	source := optimizerWorkflowSource(t)
	for _, field := range []string{"SLA_BREACH", "rule", "reason", "age_business_days", "owner", "expires"} {
		assert.Contains(t, source, field)
	}
}

func TestThreatSuppression_TCTR029_ExpiresHandling(t *testing.T) {
	suppression := ThreatDetectionSuppression{Rule: "CTR-006", Reason: "Reviewed safe", Expires: "2026-08-15"}
	require.NoError(t, validateThreatDetectionSuppressions([]ThreatDetectionSuppression{suppression}))
	assert.Len(t, activeThreatDetectionSuppressions(
		[]ThreatDetectionSuppression{suppression},
		time.Date(2026, 8, 15, 23, 59, 59, 0, time.UTC),
	), 1)
	assert.Empty(t, activeThreatDetectionSuppressions(
		[]ThreatDetectionSuppression{suppression},
		time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC),
	))
	require.Error(t, validateThreatDetectionSuppressions([]ThreatDetectionSuppression{{
		Rule: "CTR-006", Reason: "Safe", Expires: "08/15/2026",
	}}))

	invalidDateFrontmatter := map[string]any{
		"on": map[string]any{"schedule": "daily"},
		"threat-detection-suppress": []any{map[string]any{
			"rule": "CTR-006", "reason": "Safe", "expires": "2026-99-99",
		}},
	}
	require.Error(t, parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(invalidDateFrontmatter, "/tmp/threat-suppression-invalid-date.md"))
}

func TestThreatOptimizer_TCTR030Through038_FailureDiagnostics(t *testing.T) {
	source := optimizerWorkflowSource(t)
	assert.Contains(t, source, "schedule: daily")
	assert.Contains(t, source, "timeout-minutes: 30")
	assert.Contains(t, source, "RATE_LIMIT_RETRY_CONFIG")

	requiredShapes := map[string][]string{
		"OPTIMIZER_DEGRADED":     {"endpoints", "error_class", "failed_at"},
		"OPTIMIZER_TIMEOUT":      {"last_completed_step", "unevaluated_rules", "failed_at"},
		"OPTIMIZER_RATE_LIMITED": {"endpoints", "retry_after", "failed_at"},
	}
	for diagnostic, fields := range requiredShapes {
		t.Run(diagnostic, func(t *testing.T) {
			assert.Contains(t, source, `"diagnostic":"`+diagnostic+`"`)
			for _, field := range fields {
				assert.Contains(t, source, `"`+field+`"`)
			}
		})
	}

	assert.Contains(t, source, "Do not emit a noop or create/update a pull request from incomplete threat-coverage data.")
	assert.Contains(t, source, "discard partial artifacts")
	assert.Contains(t, source, "does not count as a completed coverage cycle")
}
