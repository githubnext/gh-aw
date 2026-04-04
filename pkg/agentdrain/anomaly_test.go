//go:build !integration

package agentdrain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnomalyDetector_Analyze(t *testing.T) {
	tests := []struct {
		name              string
		simThreshold      float64
		rareThreshold     int
		result            *MatchResult
		isNew             bool
		cluster           *Cluster
		wantIsNewTemplate bool
		wantNewCluster    bool
		wantLowSimilarity bool
		wantRareCluster   bool
		wantScore         float64
		wantReason        string
	}{
		{
			// isNew=true → both IsNewTemplate and NewClusterCreated; size=1 ≤ rareThreshold=2 → RareCluster.
			// score = (1.0 + 0.3) / 2.0 = 0.65
			name:              "new template creates cluster and is also rare",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 1.0},
			isNew:             true,
			cluster:           &Cluster{ID: 1, Template: []string{"stage=plan"}, Size: 1},
			wantIsNewTemplate: true,
			wantNewCluster:    true,
			wantLowSimilarity: false,
			wantRareCluster:   true,
			wantScore:         0.65,
		},
		{
			// isNew=false, size=5 > rareThreshold=2 → not rare; similarity=0.2 < threshold=0.4 → LowSimilarity.
			// score = 0.7 / 2.0 = 0.35
			name:              "low similarity below threshold",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.2},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a", "b", "c"}, Size: 5},
			wantIsNewTemplate: false,
			wantNewCluster:    false,
			wantLowSimilarity: true,
			wantRareCluster:   false,
			wantScore:         0.35,
		},
		{
			// isNew=false, size=1 ≤ rareThreshold=2 → RareCluster; similarity=0.9 ≥ threshold → not low.
			// score = 0.3 / 2.0 = 0.15
			name:              "rare cluster with high similarity",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.9},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 1},
			wantIsNewTemplate: false,
			wantNewCluster:    false,
			wantLowSimilarity: false,
			wantRareCluster:   true,
			wantScore:         0.15,
		},
		{
			// isNew=false, size=100 > rareThreshold=2, similarity=0.9 ≥ threshold → no anomalies.
			name:              "normal event has no anomaly",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.9},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a", "b"}, Size: 100},
			wantIsNewTemplate: false,
			wantNewCluster:    false,
			wantLowSimilarity: false,
			wantRareCluster:   false,
			wantScore:         0.0,
			wantReason:        "no anomaly detected",
		},
		{
			// similarity == threshold → 0.4 < 0.4 is false → not low similarity (boundary condition).
			name:              "similarity exactly at threshold is not flagged as low",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.4},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 5},
			wantIsNewTemplate: false,
			wantNewCluster:    false,
			wantLowSimilarity: false,
			wantRareCluster:   false,
			wantScore:         0.0,
			wantReason:        "no anomaly detected",
		},
		{
			// similarity just below threshold → 0.39 < 0.4 is true → LowSimilarity (boundary condition).
			// score = 0.7 / 2.0 = 0.35
			name:              "similarity just below threshold is flagged as low",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.39},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 5},
			wantIsNewTemplate: false,
			wantNewCluster:    false,
			wantLowSimilarity: true,
			wantRareCluster:   false,
			wantScore:         0.35,
		},
		{
			// Combined: isNew=false, size=1 ≤ 2 → RareCluster; similarity=0.2 < 0.4 → LowSimilarity.
			// score = (0.7 + 0.3) / 2.0 = 0.5
			name:              "combined low similarity and rare cluster",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.2},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 1},
			wantIsNewTemplate: false,
			wantNewCluster:    false,
			wantLowSimilarity: true,
			wantRareCluster:   true,
			wantScore:         0.5,
		},
		{
			// nil cluster: the rare-cluster check is guarded; RareCluster must stay false.
			name:              "nil cluster does not trigger rare cluster flag",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 0, Similarity: 0.9},
			isNew:             false,
			cluster:           nil,
			wantIsNewTemplate: false,
			wantNewCluster:    false,
			wantLowSimilarity: false,
			wantRareCluster:   false,
			wantScore:         0.0,
			wantReason:        "no anomaly detected",
		},
		{
			// Max achievable score: isNew=true + rare (size=5 ≤ rareThreshold=10).
			// LowSimilarity is never set when isNew=true, so ceiling is (1.0+0.3)/2.0=0.65.
			name:              "max score achieved with new template and rare cluster",
			simThreshold:      0.4,
			rareThreshold:     10,
			result:            &MatchResult{ClusterID: 1, Similarity: 1.0},
			isNew:             true,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 5},
			wantIsNewTemplate: true,
			wantNewCluster:    true,
			wantLowSimilarity: false,
			wantRareCluster:   true,
			wantScore:         0.65,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewAnomalyDetector(tt.simThreshold, tt.rareThreshold)

			report := d.Analyze(tt.result, tt.isNew, tt.cluster)

			require.NotNil(t, report, "Analyze should always return a non-nil report")
			assert.Equal(t, tt.wantIsNewTemplate, report.IsNewTemplate, "IsNewTemplate mismatch")
			assert.Equal(t, tt.wantNewCluster, report.NewClusterCreated, "NewClusterCreated mismatch")
			assert.Equal(t, tt.wantLowSimilarity, report.LowSimilarity, "LowSimilarity mismatch")
			assert.Equal(t, tt.wantRareCluster, report.RareCluster, "RareCluster mismatch")
			assert.InDelta(t, tt.wantScore, report.AnomalyScore, 1e-9, "AnomalyScore mismatch")
			if tt.wantReason != "" {
				assert.Equal(t, tt.wantReason, report.Reason, "Reason mismatch")
			}
		})
	}
}

func TestAnalyzeEvent(t *testing.T) {
	cfg := DefaultConfig()
	m, err := NewMiner(cfg)
	require.NoError(t, err, "NewMiner should succeed")
	require.NotNil(t, m, "NewMiner should return a non-nil miner")

	evt := AgentEvent{
		Stage:  "plan",
		Fields: map[string]string{"action": "start", "model": "gpt-4"},
	}

	// First occurrence → new template.
	result, report, err := m.AnalyzeEvent(evt)
	require.NoError(t, err, "AnalyzeEvent should not fail on first event")
	require.NotNil(t, result, "AnalyzeEvent should return a non-nil result on first event")
	require.NotNil(t, report, "AnalyzeEvent should return a non-nil report on first event")
	assert.True(t, report.IsNewTemplate, "first event should be detected as a new template")

	// Second occurrence of the same event → not new.
	result2, report2, err := m.AnalyzeEvent(evt)
	require.NoError(t, err, "AnalyzeEvent should not fail on second identical event")
	require.NotNil(t, result2, "AnalyzeEvent should return a non-nil result on second event")
	require.NotNil(t, report2, "AnalyzeEvent should return a non-nil report on second event")
	assert.False(t, report2.IsNewTemplate, "second identical event should not be detected as a new template")
}
