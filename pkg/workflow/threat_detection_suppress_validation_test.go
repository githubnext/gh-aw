package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// parseThreatDetectionSuppress unit tests
// ---------------------------------------------------------------------------

func TestParseThreatDetectionSuppress_Nil(t *testing.T) {
	result, err := parseThreatDetectionSuppress(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseThreatDetectionSuppress_EmptySlice(t *testing.T) {
	result, err := parseThreatDetectionSuppress([]any{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestParseThreatDetectionSuppress_NotASlice(t *testing.T) {
	_, err := parseThreatDetectionSuppress("not-a-slice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be an array")
}

func TestParseThreatDetectionSuppress_Valid(t *testing.T) {
	raw := []any{
		map[string]any{
			"rule":    "CTR-001",
			"reason":  "False positive in our context",
			"expires": "2027-01-01",
		},
	}
	result, err := parseThreatDetectionSuppress(raw)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "CTR-001", result[0].Rule)
	assert.Equal(t, "False positive in our context", result[0].Reason)
	assert.Equal(t, "2027-01-01", result[0].Expires)
}

func TestParseThreatDetectionSuppress_ValidMultiSegmentRule(t *testing.T) {
	raw := []any{
		map[string]any{
			"rule":   "CTR-INJ-001",
			"reason": "Acknowledged risk",
		},
	}
	result, err := parseThreatDetectionSuppress(raw)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "CTR-INJ-001", result[0].Rule)
	assert.Empty(t, result[0].Expires)
}

func TestParseThreatDetectionSuppress_MissingReason(t *testing.T) {
	raw := []any{
		map[string]any{
			"rule": "CTR-001",
			// reason intentionally absent
		},
	}
	_, err := parseThreatDetectionSuppress(raw)
	require.Error(t, err, "suppression without reason must cause a compile error")
	assert.Contains(t, err.Error(), "missing required field 'reason'")
	assert.Contains(t, err.Error(), "CTR-001")
}

func TestParseThreatDetectionSuppress_EmptyReason(t *testing.T) {
	raw := []any{
		map[string]any{
			"rule":   "CTR-002",
			"reason": "   ", // whitespace-only
		},
	}
	_, err := parseThreatDetectionSuppress(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'reason'")
}

func TestParseThreatDetectionSuppress_MissingRule(t *testing.T) {
	raw := []any{
		map[string]any{
			"reason": "No rule provided",
		},
	}
	_, err := parseThreatDetectionSuppress(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field 'rule'")
}

func TestParseThreatDetectionSuppress_InvalidRulePattern(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{"lowercase", "ctr-001"},
		{"no-prefix", "001"},
		{"spaces", "CTR 001"},
		{"empty", ""},
		{"just-ctr", "CTR"},
		{"ctr-lowercase-segment", "CTR-abc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := []any{
				map[string]any{
					"rule":   tc.rule,
					"reason": "Test suppression",
				},
			}
			_, err := parseThreatDetectionSuppress(raw)
			require.Error(t, err, "rule %q should be rejected", tc.rule)
		})
	}
}

func TestParseThreatDetectionSuppress_InvalidExpiresFormat(t *testing.T) {
	raw := []any{
		map[string]any{
			"rule":    "CTR-001",
			"reason":  "Valid reason",
			"expires": "31/12/2027", // wrong format
		},
	}
	_, err := parseThreatDetectionSuppress(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid ISO 8601 date")
}

func TestParseThreatDetectionSuppress_EntryNotAnObject(t *testing.T) {
	raw := []any{"CTR-001"}
	_, err := parseThreatDetectionSuppress(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "each entry must be an object")
}

// ---------------------------------------------------------------------------
// ThreatDetectionSuppression.IsExpired tests
// ---------------------------------------------------------------------------

func TestIsExpired_NoExpiry(t *testing.T) {
	sup := ThreatDetectionSuppression{Rule: "CTR-001", Reason: "Permanent"}
	assert.False(t, sup.IsExpired(time.Now()))
}

func TestIsExpired_FutureExpiry(t *testing.T) {
	sup := ThreatDetectionSuppression{Rule: "CTR-001", Reason: "Future", Expires: "2099-01-01"}
	assert.False(t, sup.IsExpired(time.Now()))
}

func TestIsExpired_PastExpiry(t *testing.T) {
	sup := ThreatDetectionSuppression{Rule: "CTR-001", Reason: "Expired", Expires: "2020-01-01"}
	assert.True(t, sup.IsExpired(time.Now()))
}

func TestIsExpired_InvalidExpiry(t *testing.T) {
	sup := ThreatDetectionSuppression{Rule: "CTR-001", Reason: "Bad date", Expires: "not-a-date"}
	// Invalid date strings should NOT be considered expired (safe default)
	assert.False(t, sup.IsExpired(time.Now()))
}

// ---------------------------------------------------------------------------
// buildSuppressionManifestEntries tests
// ---------------------------------------------------------------------------

func TestBuildSuppressionManifestEntries_Empty(t *testing.T) {
	entries := buildSuppressionManifestEntries(nil, time.Now())
	assert.Empty(t, entries)
}

func TestBuildSuppressionManifestEntries_ActiveWithExpiry(t *testing.T) {
	suppressions := []ThreatDetectionSuppression{
		{Rule: "CTR-001", Reason: "False positive", Expires: "2099-01-01"},
	}
	entries := buildSuppressionManifestEntries(suppressions, time.Now())
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0], "CTR-001")
	assert.Contains(t, entries[0], "False positive")
	assert.Contains(t, entries[0], "2099-01-01")
	assert.NotContains(t, entries[0], "SLA_BREACH")
}

func TestBuildSuppressionManifestEntries_PermanentFlagsSLABreach(t *testing.T) {
	suppressions := []ThreatDetectionSuppression{
		{Rule: "CTR-042", Reason: "Acknowledged permanently"},
	}
	entries := buildSuppressionManifestEntries(suppressions, time.Now())
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0], "SLA_BREACH")
	assert.Contains(t, entries[0], "CTR-042")
}

func TestBuildSuppressionManifestEntries_ExpiredSkipped(t *testing.T) {
	suppressions := []ThreatDetectionSuppression{
		{Rule: "CTR-001", Reason: "Expired", Expires: "2020-01-01"},
	}
	entries := buildSuppressionManifestEntries(suppressions, time.Now())
	assert.Empty(t, entries, "expired suppressions should not appear in manifest")
}

func TestBuildSuppressionManifestEntries_Mixed(t *testing.T) {
	suppressions := []ThreatDetectionSuppression{
		{Rule: "CTR-001", Reason: "Expired", Expires: "2020-01-01"},
		{Rule: "CTR-002", Reason: "Active with expiry", Expires: "2099-01-01"},
		{Rule: "CTR-003", Reason: "Permanent, no expiry"},
	}
	entries := buildSuppressionManifestEntries(suppressions, time.Now())
	// Expired entry is skipped; 2 remain.
	require.Len(t, entries, 2)
	// Active with expiry: no SLA_BREACH
	assert.Contains(t, entries[0], "CTR-002")
	assert.NotContains(t, entries[0], "SLA_BREACH")
	// Permanent: SLA_BREACH
	assert.Contains(t, entries[1], "SLA_BREACH")
	assert.Contains(t, entries[1], "CTR-003")
}

// ---------------------------------------------------------------------------
// ParseFrontmatterConfig integration tests
// ---------------------------------------------------------------------------

func TestParseFrontmatterConfig_ThreatDetectionSuppress_Valid(t *testing.T) {
	frontmatter := map[string]any{
		"name":   "test-workflow",
		"engine": "copilot",
		"prompt": "Do something",
		"threat-detection-suppress": []any{
			map[string]any{
				"rule":    "CTR-001",
				"reason":  "False positive",
				"expires": "2099-12-31",
			},
		},
	}
	config, err := ParseFrontmatterConfig(frontmatter)
	require.NoError(t, err)
	require.Len(t, config.ThreatDetectionSuppress, 1)
	assert.Equal(t, "CTR-001", config.ThreatDetectionSuppress[0].Rule)
	assert.Equal(t, "False positive", config.ThreatDetectionSuppress[0].Reason)
	assert.Equal(t, "2099-12-31", config.ThreatDetectionSuppress[0].Expires)
}

func TestParseFrontmatterConfig_ThreatDetectionSuppress_MissingReason_CausesError(t *testing.T) {
	frontmatter := map[string]any{
		"name":   "test-workflow",
		"engine": "copilot",
		"prompt": "Do something",
		"threat-detection-suppress": []any{
			map[string]any{
				"rule": "CTR-001",
				// reason absent
			},
		},
	}
	_, err := ParseFrontmatterConfig(frontmatter)
	require.Error(t, err, "suppression without reason must be rejected during ParseFrontmatterConfig")
	assert.Contains(t, err.Error(), "reason")
}

func TestParseFrontmatterConfig_ThreatDetectionSuppress_InvalidRule_CausesError(t *testing.T) {
	frontmatter := map[string]any{
		"name":   "test-workflow",
		"engine": "copilot",
		"prompt": "Do something",
		"threat-detection-suppress": []any{
			map[string]any{
				"rule":   "NOT-VALID",
				"reason": "Some reason",
			},
		},
	}
	_, err := ParseFrontmatterConfig(frontmatter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CTR-*")
}

func TestParseFrontmatterConfig_ThreatDetectionSuppress_Absent(t *testing.T) {
	frontmatter := map[string]any{
		"name":   "test-workflow",
		"engine": "copilot",
		"prompt": "Do something",
	}
	config, err := ParseFrontmatterConfig(frontmatter)
	require.NoError(t, err)
	assert.Empty(t, config.ThreatDetectionSuppress)
}
