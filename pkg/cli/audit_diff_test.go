//go:build !integration

package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFirewallDiff_NewDomains(t *testing.T) {
	run1 := &FirewallAnalysis{
		TotalRequests:   5,
		AllowedRequests: 5,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}
	run2 := &FirewallAnalysis{
		TotalRequests:   20,
		AllowedRequests: 17,
		BlockedRequests: 3,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":        {Allowed: 5, Blocked: 0},
			"registry.npmjs.org:443":    {Allowed: 15, Blocked: 0},
			"telemetry.example.com:443": {Allowed: 0, Blocked: 2},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Equal(t, int64(100), diff.Run1ID, "Run1ID should match")
	assert.Equal(t, int64(200), diff.Run2ID, "Run2ID should match")
	assert.Len(t, diff.NewDomains, 2, "Should have 2 new domains")
	assert.Empty(t, diff.RemovedDomains, "Should have no removed domains")
	assert.Empty(t, diff.StatusChanges, "Should have no status changes")

	// Check new domains are sorted
	assert.Equal(t, "registry.npmjs.org:443", diff.NewDomains[0].Domain, "First new domain should be registry.npmjs.org")
	assert.Equal(t, "new", diff.NewDomains[0].Status, "Status should be 'new'")
	assert.Equal(t, "allowed", diff.NewDomains[0].Run2Status, "Registry should be allowed")
	assert.False(t, diff.NewDomains[0].IsAnomaly, "Allowed new domain should not be anomaly")

	assert.Equal(t, "telemetry.example.com:443", diff.NewDomains[1].Domain, "Second new domain should be telemetry.example.com")
	assert.Equal(t, "denied", diff.NewDomains[1].Run2Status, "Telemetry should be denied")
	assert.True(t, diff.NewDomains[1].IsAnomaly, "New denied domain should be anomaly")
	assert.Equal(t, "new denied domain", diff.NewDomains[1].AnomalyNote, "Anomaly note should explain the issue")

	// Check summary
	assert.Equal(t, 2, diff.Summary.NewDomainCount, "Summary should show 2 new domains")
	assert.True(t, diff.Summary.HasAnomalies, "Should have anomalies")
	assert.Equal(t, 1, diff.Summary.AnomalyCount, "Should have 1 anomaly")
}

func TestComputeFirewallDiff_RemovedDomains(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":       {Allowed: 5, Blocked: 0},
			"old-api.internal.com:443": {Allowed: 8, Blocked: 0},
		},
	}
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Len(t, diff.RemovedDomains, 1, "Should have 1 removed domain")
	assert.Equal(t, "old-api.internal.com:443", diff.RemovedDomains[0].Domain, "Removed domain should be old-api.internal.com")
	assert.Equal(t, "removed", diff.RemovedDomains[0].Status, "Status should be 'removed'")
	assert.Equal(t, "allowed", diff.RemovedDomains[0].Run1Status, "Domain was allowed in run 1")
	assert.Equal(t, 8, diff.RemovedDomains[0].Run1Allowed, "Domain had 8 allowed requests")
	assert.Equal(t, 1, diff.Summary.RemovedDomainCount, "Summary should show 1 removed domain")
}

func TestComputeFirewallDiff_StatusChanges(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"staging.api.com:443":    {Allowed: 10, Blocked: 0},
			"legacy.service.com:443": {Allowed: 0, Blocked: 5},
		},
	}
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"staging.api.com:443":    {Allowed: 0, Blocked: 3},
			"legacy.service.com:443": {Allowed: 7, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Len(t, diff.StatusChanges, 2, "Should have 2 status changes")

	// legacy.service.com: denied → allowed (anomaly: previously denied, now allowed)
	legacyEntry := findDiffEntry(diff.StatusChanges, "legacy.service.com:443")
	require.NotNil(t, legacyEntry, "Should find legacy.service.com in status changes")
	assert.Equal(t, "denied", legacyEntry.Run1Status, "Was denied in run 1")
	assert.Equal(t, "allowed", legacyEntry.Run2Status, "Now allowed in run 2")
	assert.True(t, legacyEntry.IsAnomaly, "Should be flagged as anomaly")
	assert.Equal(t, "previously denied, now allowed", legacyEntry.AnomalyNote, "Anomaly note should explain the flip")

	// staging.api.com: allowed → denied (anomaly)
	stagingEntry := findDiffEntry(diff.StatusChanges, "staging.api.com:443")
	require.NotNil(t, stagingEntry, "Should find staging.api.com in status changes")
	assert.Equal(t, "allowed", stagingEntry.Run1Status, "Was allowed in run 1")
	assert.Equal(t, "denied", stagingEntry.Run2Status, "Now denied in run 2")
	assert.True(t, stagingEntry.IsAnomaly, "Should be flagged as anomaly")

	assert.Equal(t, 2, diff.Summary.StatusChangeCount, "Summary should show 2 status changes")
	assert.True(t, diff.Summary.HasAnomalies, "Should have anomalies")
}

func TestComputeFirewallDiff_VolumeChanges(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":  {Allowed: 23, Blocked: 0},
			"cdn.example.com:443": {Allowed: 50, Blocked: 0},
		},
	}
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":  {Allowed: 89, Blocked: 0},
			"cdn.example.com:443": {Allowed: 55, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	// api.github.com: 23 → 89 = +287% (over 100% threshold)
	assert.Len(t, diff.VolumeChanges, 1, "Should have 1 volume change (api.github.com, not cdn)")
	assert.Equal(t, "api.github.com:443", diff.VolumeChanges[0].Domain, "Volume change should be for api.github.com")
	assert.Equal(t, "+287%", diff.VolumeChanges[0].VolumeChange, "Volume change should be +287%")

	// cdn.example.com: 50 → 55 = +10% (under threshold, not flagged)
	assert.Equal(t, 1, diff.Summary.VolumeChangeCount, "Summary should show 1 volume change")
	assert.False(t, diff.Summary.HasAnomalies, "Volume changes alone should not create anomalies")
}

func TestComputeFirewallDiff_BothNil(t *testing.T) {
	diff := computeFirewallDiff(100, 200, nil, nil)

	assert.Empty(t, diff.NewDomains, "Should have no new domains")
	assert.Empty(t, diff.RemovedDomains, "Should have no removed domains")
	assert.Empty(t, diff.StatusChanges, "Should have no status changes")
	assert.Empty(t, diff.VolumeChanges, "Should have no volume changes")
	assert.False(t, diff.Summary.HasAnomalies, "Should have no anomalies")
}

func TestComputeFirewallDiff_Run1Nil(t *testing.T) {
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, nil, run2)

	assert.Len(t, diff.NewDomains, 1, "All run2 domains should be new")
	assert.Equal(t, "api.github.com:443", diff.NewDomains[0].Domain, "New domain should be api.github.com")
}

func TestComputeFirewallDiff_Run2Nil(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, nil)

	assert.Len(t, diff.RemovedDomains, 1, "All run1 domains should be removed")
	assert.Equal(t, "api.github.com:443", diff.RemovedDomains[0].Domain, "Removed domain should be api.github.com")
}

func TestComputeFirewallDiff_NoChanges(t *testing.T) {
	stats := map[string]DomainRequestStats{
		"api.github.com:443": {Allowed: 5, Blocked: 0},
	}
	run1 := &FirewallAnalysis{RequestsByDomain: stats}
	run2 := &FirewallAnalysis{RequestsByDomain: stats}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Empty(t, diff.NewDomains, "Should have no new domains")
	assert.Empty(t, diff.RemovedDomains, "Should have no removed domains")
	assert.Empty(t, diff.StatusChanges, "Should have no status changes")
	assert.Empty(t, diff.VolumeChanges, "Should have no volume changes")
}

func TestComputeFirewallDiff_CompleteScenario(t *testing.T) {
	run1 := &FirewallAnalysis{
		TotalRequests:   46,
		AllowedRequests: 38,
		BlockedRequests: 8,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":       {Allowed: 23, Blocked: 0},
			"old-api.internal.com:443": {Allowed: 8, Blocked: 0},
			"staging.api.com:443":      {Allowed: 7, Blocked: 0},
			"blocked.example.com:443":  {Allowed: 0, Blocked: 8},
		},
	}
	run2 := &FirewallAnalysis{
		TotalRequests:   108,
		AllowedRequests: 106,
		BlockedRequests: 2,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":        {Allowed: 89, Blocked: 0},
			"registry.npmjs.org:443":    {Allowed: 15, Blocked: 0},
			"telemetry.example.com:443": {Allowed: 0, Blocked: 2},
			"staging.api.com:443":       {Allowed: 0, Blocked: 0}, // no requests (edge case)
			"blocked.example.com:443":   {Allowed: 0, Blocked: 0}, // no longer any requests (edge case)
		},
	}

	diff := computeFirewallDiff(12345, 12346, run1, run2)

	// Verify new domains
	assert.Len(t, diff.NewDomains, 2, "Should have 2 new domains")

	// Verify removed domains
	assert.Len(t, diff.RemovedDomains, 1, "Should have 1 removed domain (old-api.internal.com)")

	// api.github.com: 23 → 89 = +287%
	assert.GreaterOrEqual(t, len(diff.VolumeChanges), 1, "Should have at least 1 volume change")
}

func TestDomainStatus(t *testing.T) {
	tests := []struct {
		name     string
		stats    DomainRequestStats
		expected string
	}{
		{name: "allowed only", stats: DomainRequestStats{Allowed: 5, Blocked: 0}, expected: "allowed"},
		{name: "denied only", stats: DomainRequestStats{Allowed: 0, Blocked: 3}, expected: "denied"},
		{name: "mixed", stats: DomainRequestStats{Allowed: 2, Blocked: 1}, expected: "mixed"},
		{name: "zero requests", stats: DomainRequestStats{Allowed: 0, Blocked: 0}, expected: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domainStatus(tt.stats)
			assert.Equal(t, tt.expected, result, "Domain status should match")
		})
	}
}

func TestFormatVolumeChange(t *testing.T) {
	tests := []struct {
		name     string
		total1   int
		total2   int
		expected string
	}{
		{name: "increase 287%", total1: 23, total2: 89, expected: "+287%"},
		{name: "decrease 50%", total1: 100, total2: 50, expected: "-50%"},
		{name: "double", total1: 10, total2: 20, expected: "+100%"},
		{name: "from zero", total1: 0, total2: 10, expected: "+∞"},
		{name: "no change", total1: 10, total2: 10, expected: "+0%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVolumeChange(tt.total1, tt.total2)
			assert.Equal(t, tt.expected, result, "Volume change format should match")
		})
	}
}

func TestFirewallDiffJSONSerialization(t *testing.T) {
	diff := computeFirewallDiff(100, 200, &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}, &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":  {Allowed: 5, Blocked: 0},
			"new.example.com:443": {Allowed: 3, Blocked: 0},
		},
	})

	data, err := json.MarshalIndent(diff, "", "  ")
	require.NoError(t, err, "Should serialize diff to JSON")

	var parsed FirewallDiff
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Should deserialize diff from JSON")

	assert.Equal(t, int64(100), parsed.Run1ID, "Run1ID should survive serialization")
	assert.Equal(t, int64(200), parsed.Run2ID, "Run2ID should survive serialization")
	assert.Len(t, parsed.NewDomains, 1, "Should have 1 new domain after deserialization")
	assert.Equal(t, "new.example.com:443", parsed.NewDomains[0].Domain, "New domain should match")
}

func TestStatusEmoji(t *testing.T) {
	assert.Equal(t, "✅", statusEmoji("allowed"), "Allowed should show checkmark")
	assert.Equal(t, "❌", statusEmoji("denied"), "Denied should show X")
	assert.Equal(t, "⚠️", statusEmoji("mixed"), "Mixed should show warning")
	assert.Equal(t, "❓", statusEmoji("unknown"), "Unknown should show question mark")
	assert.Equal(t, "❓", statusEmoji(""), "Empty should show question mark")
}

func TestIsEmptyDiff(t *testing.T) {
	emptyDiff := &FirewallDiff{}
	assert.True(t, isEmptyDiff(emptyDiff), "Empty diff should be detected")

	nonEmptyDiff := &FirewallDiff{
		NewDomains: []DomainDiffEntry{{Domain: "test.com"}},
	}
	assert.False(t, isEmptyDiff(nonEmptyDiff), "Non-empty diff should not be detected as empty")
}

// findDiffEntry is a test helper to find a domain in a list of diff entries
func findDiffEntry(entries []DomainDiffEntry, domain string) *DomainDiffEntry {
	for i := range entries {
		if entries[i].Domain == domain {
			return &entries[i]
		}
	}
	return nil
}
