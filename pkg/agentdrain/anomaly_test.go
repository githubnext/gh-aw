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
			wantReason:        "new log template discovered; rare cluster (few observations)",
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
			wantReason:        "low similarity to known template",
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
			wantReason:        "rare cluster (few observations)",
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
			wantReason:        "low similarity to known template",
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
			wantReason:        "low similarity to known template; rare cluster (few observations)",
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
			wantReason:        "new log template discovered; rare cluster (few observations)",
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
			assert.Equal(t, tt.wantReason, report.Reason, "Reason mismatch")
		})
	}
}

func TestNewAnomalyDetector_ThresholdBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		simThreshold  float64
		rareThreshold int
	}{
		{
			name:          "zero thresholds are preserved",
			simThreshold:  0.0,
			rareThreshold: 0,
		},
		{
			name:          "negative thresholds are preserved",
			simThreshold:  -0.1,
			rareThreshold: -1,
		},
		{
			name:          "upper-bound similarity threshold is preserved",
			simThreshold:  1.0,
			rareThreshold: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewAnomalyDetector(tt.simThreshold, tt.rareThreshold)
			require.NotNil(t, detector, "NewAnomalyDetector should return a non-nil detector")
			assert.InDelta(t, tt.simThreshold, detector.threshold, 1e-12, "similarity threshold should be stored as provided")
			assert.Equal(t, tt.rareThreshold, detector.rareThreshold, "rare cluster threshold should be stored as provided")
		})
	}
}

func TestBuildReason(t *testing.T) {
	tests := []struct {
		name          string
		isNewTemplate bool
		lowSimilarity bool
		rareCluster   bool
		wantReason    string
	}{
		{
			name:          "no flags set",
			isNewTemplate: false,
			lowSimilarity: false,
			rareCluster:   false,
			wantReason:    "no anomaly detected",
		},
		{
			name:          "new template only",
			isNewTemplate: true,
			lowSimilarity: false,
			rareCluster:   false,
			wantReason:    "new log template discovered",
		},
		{
			name:          "low similarity only",
			isNewTemplate: false,
			lowSimilarity: true,
			rareCluster:   false,
			wantReason:    "low similarity to known template",
		},
		{
			name:          "rare cluster only",
			isNewTemplate: false,
			lowSimilarity: false,
			rareCluster:   true,
			wantReason:    "rare cluster (few observations)",
		},
		{
			name:          "new template and low similarity",
			isNewTemplate: true,
			lowSimilarity: true,
			rareCluster:   false,
			wantReason:    "new log template discovered; low similarity to known template",
		},
		{
			name:          "new template and rare cluster",
			isNewTemplate: true,
			lowSimilarity: false,
			rareCluster:   true,
			wantReason:    "new log template discovered; rare cluster (few observations)",
		},
		{
			name:          "low similarity and rare cluster",
			isNewTemplate: false,
			lowSimilarity: true,
			rareCluster:   true,
			wantReason:    "low similarity to known template; rare cluster (few observations)",
		},
		{
			// This case is valid for buildReason in isolation, but Analyze never sets
			// both IsNewTemplate and LowSimilarity because those flags are mutually exclusive.
			name:          "all flags set",
			isNewTemplate: true,
			lowSimilarity: true,
			rareCluster:   true,
			wantReason:    "new log template discovered; low similarity to known template; rare cluster (few observations)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AnomalyReport{
				IsNewTemplate: tt.isNewTemplate,
				LowSimilarity: tt.lowSimilarity,
				RareCluster:   tt.rareCluster,
			}
			got := buildReason(r)
			assert.Equal(t, tt.wantReason, got, "buildReason mismatch")
		})
	}
}

func TestAnalyzeEvent(t *testing.T) {
	cfg := DefaultConfig()
	m, err := NewMiner(cfg)
	require.NoError(t, err, "NewMiner should succeed")
	require.NotNil(t, m, "NewMiner should return a non-nil miner")

	evtPlan := AgentEvent{
		Stage:  "plan",
		Fields: map[string]string{"action": "start", "model": "gpt-4"},
	}
	evtFinish := AgentEvent{
		Stage:  "finish",
		Fields: map[string]string{"status": "ok"},
	}

	// This table is intentionally order-dependent because each call mutates miner state.
	tests := []struct {
		name             string
		event            AgentEvent
		wantIsNew        bool
		wantNewCluster   bool
		errorDescription string
	}{
		{
			name:             "first occurrence is flagged as new template",
			event:            evtPlan,
			wantIsNew:        true,
			wantNewCluster:   true,
			errorDescription: "first event",
		},
		{
			name:             "second identical occurrence is not flagged as new",
			event:            evtPlan,
			wantIsNew:        false,
			wantNewCluster:   false,
			errorDescription: "second identical event",
		},
		{
			name:             "distinct event creates its own new template",
			event:            evtFinish,
			wantIsNew:        true,
			wantNewCluster:   true,
			errorDescription: "distinct event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, report, analyzeErr := m.AnalyzeEvent(tt.event)
			require.NoError(t, analyzeErr, "AnalyzeEvent should not fail for %s", tt.errorDescription)
			require.NotNil(t, result, "AnalyzeEvent should return a non-nil result")
			require.NotNil(t, report, "AnalyzeEvent should return a non-nil report")
			assert.Equal(t, tt.wantIsNew, report.IsNewTemplate, "IsNewTemplate mismatch")
			assert.Equal(t, tt.wantNewCluster, report.NewClusterCreated, "NewClusterCreated mismatch")
		})
	}
}
