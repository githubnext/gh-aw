//go:build !integration

package agentdrain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// anomalyScore computes the expected normalized anomaly score from the individual flag weights,
// mirroring the scoring logic in Analyze. Using the exported constants (AnomalyWeightNew,
// AnomalyWeightLow, AnomalyWeightRare, AnomalyMaxScore) gives a compile-time sync guarantee:
// if production weights change, this helper diverges at compile time, not silently at runtime.
func anomalyScore(isNew, lowSim, rare bool) float64 {
	var score float64
	if isNew {
		score += AnomalyWeightNew
	}
	if lowSim {
		score += AnomalyWeightLow
	}
	if rare {
		score += AnomalyWeightRare
	}
	if score > AnomalyMaxScore {
		score = AnomalyMaxScore
	}
	return score / AnomalyMaxScore
}

func TestAnomalyDetector_Analyze(t *testing.T) {
	tests := []struct {
		name              string
		simThreshold      float64
		rareThreshold     int
		result            *MatchResult
		isNew             bool
		cluster           *Cluster
		wantIsNewTemplate bool
		wantLowSimilarity bool
		wantRareCluster   bool
		wantScore         float64
		wantReason        string
	}{
		{
			// isNew=true → IsNewTemplate; size=1 ≤ rareThreshold=2 → RareCluster.
			// score = (1.0 + 0.3) / 2.0 = 0.65
			name:              "new template creates cluster and is also rare",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 1.0},
			isNew:             true,
			cluster:           &Cluster{ID: 1, Template: []string{"stage=plan"}, Size: 1},
			wantIsNewTemplate: true,
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
			wantLowSimilarity: false,
			wantRareCluster:   true,
			wantScore:         0.15,
			wantReason:        "rare cluster (few observations)",
		},
		{
			// size == rareThreshold boundary: 2 <= 2 is true.
			// score = (rare-cluster weight 0.3) / maxScore 2.0 = 0.15
			name:              "cluster size exactly at rare threshold is rare",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.9},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 2},
			wantIsNewTemplate: false,
			wantLowSimilarity: false,
			wantRareCluster:   true,
			wantScore:         0.15,
			wantReason:        "rare cluster (few observations)",
		},
		{
			// size == rareThreshold+1 boundary: 3 <= 2 is false.
			name:              "cluster size just above rare threshold is not rare",
			simThreshold:      0.4,
			rareThreshold:     2,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.9},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 3},
			wantIsNewTemplate: false,
			wantLowSimilarity: false,
			wantRareCluster:   false,
			wantScore:         0.0,
			wantReason:        "no anomaly detected",
		},
		{
			// rareThreshold=0 boundary: size=0 satisfies 0 <= 0.
			// score = (rare-cluster weight 0.3) / maxScore 2.0 = 0.15
			name:              "zero rare threshold marks zero-sized cluster as rare",
			simThreshold:      0.4,
			rareThreshold:     0,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.9},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 0},
			wantIsNewTemplate: false,
			wantLowSimilarity: false,
			wantRareCluster:   true,
			wantScore:         0.15,
			wantReason:        "rare cluster (few observations)",
		},
		{
			// rareThreshold=0 boundary: size-one cluster does not satisfy 1 <= 0.
			name:              "zero rare threshold does not mark size one cluster as rare",
			simThreshold:      0.4,
			rareThreshold:     0,
			result:            &MatchResult{ClusterID: 1, Similarity: 0.9},
			isNew:             false,
			cluster:           &Cluster{ID: 1, Template: []string{"a"}, Size: 1},
			wantIsNewTemplate: false,
			wantLowSimilarity: false,
			wantRareCluster:   false,
			wantScore:         0.0,
			wantReason:        "no anomaly detected",
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
			wantLowSimilarity: false,
			wantRareCluster:   true,
			wantScore:         0.65,
			wantReason:        "new log template discovered; rare cluster (few observations)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, err := NewAnomalyDetector(tt.simThreshold, tt.rareThreshold)
			require.NoError(t, err, "NewAnomalyDetector should succeed with valid thresholds")
			require.NotNil(t, d, "NewAnomalyDetector should return a non-nil detector")

			report := d.Analyze(tt.result, tt.isNew, tt.cluster)

			require.NotNil(t, report, "Analyze should always return a non-nil report")
			assert.Equal(t, tt.wantIsNewTemplate, report.IsNewTemplate, "IsNewTemplate mismatch")
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
		wantErr       string
	}{
		{
			name:          "zero thresholds are preserved",
			simThreshold:  0.0,
			rareThreshold: 0,
		},
		{
			name:          "negative similarity threshold is rejected",
			simThreshold:  -0.1,
			rareThreshold: 1,
			wantErr:       "simThreshold must be in [0,1]",
		},
		{
			name:          "upper-bound similarity threshold is preserved",
			simThreshold:  1.0,
			rareThreshold: 5,
		},
		{
			name:          "similarity threshold above one is rejected",
			simThreshold:  1.1,
			rareThreshold: 1,
			wantErr:       "simThreshold must be in [0,1]",
		},
		{
			name:          "negative rare cluster threshold is rejected",
			simThreshold:  0.4,
			rareThreshold: -1,
			wantErr:       "rareClusterThreshold must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			detector, err := NewAnomalyDetector(tt.simThreshold, tt.rareThreshold)
			if tt.wantErr != "" {
				require.Error(t, err, "NewAnomalyDetector should reject invalid thresholds")
				assert.Contains(t, err.Error(), tt.wantErr, "error should describe invalid threshold")
				require.Nil(t, detector, "NewAnomalyDetector should return nil detector on validation error")
				return
			}

			require.NoError(t, err, "NewAnomalyDetector should succeed with valid thresholds")
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
			t.Parallel()
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

	// Sub-tests are sequential: each step mutates the shared miner state and depends
	// on the state from the previous step. Do NOT add t.Parallel() to any of these
	// sub-tests — doing so would cause a data race on the shared Miner.

	t.Run("first occurrence trains template and is flagged new", func(t *testing.T) {
		resultFirst, reportFirst, errFirst := m.AnalyzeEvent(evtPlan)
		require.NoError(t, errFirst, "AnalyzeEvent should not fail for first event")
		require.NotNil(t, resultFirst, "AnalyzeEvent should return a non-nil result")
		require.NotNil(t, reportFirst, "AnalyzeEvent should return a non-nil report")
		// Both assertions below are gates for steps 2+3: a failure here stops the test
		// before exercising state that is now invalid.
		require.True(t, reportFirst.IsNewTemplate, "IsNewTemplate mismatch for first event")
		require.InDelta(t, anomalyScore(true, false, true), reportFirst.AnomalyScore, 1e-9, "AnomalyScore mismatch for first event")
		assert.Equal(t, "new log template discovered; rare cluster (few observations)", reportFirst.Reason, "Reason mismatch for first event")
	})

	t.Run("second identical occurrence reuses trained template", func(t *testing.T) {
		resultSecond, reportSecond, errSecond := m.AnalyzeEvent(evtPlan)
		require.NoError(t, errSecond, "AnalyzeEvent should not fail for second identical event")
		require.NotNil(t, resultSecond, "AnalyzeEvent should return a non-nil result")
		require.NotNil(t, reportSecond, "AnalyzeEvent should return a non-nil report")
		assert.False(t, reportSecond.IsNewTemplate, "IsNewTemplate mismatch for second identical event")
		assert.InDelta(t, anomalyScore(false, false, true), reportSecond.AnomalyScore, 1e-9, "AnomalyScore mismatch for second identical event")
		assert.Equal(t, "rare cluster (few observations)", reportSecond.Reason, "Reason mismatch for second identical event")
	})

	t.Run("distinct event creates separate new template", func(t *testing.T) {
		resultDistinct, reportDistinct, errDistinct := m.AnalyzeEvent(evtFinish)
		require.NoError(t, errDistinct, "AnalyzeEvent should not fail for distinct event")
		require.NotNil(t, resultDistinct, "AnalyzeEvent should return a non-nil result")
		require.NotNil(t, reportDistinct, "AnalyzeEvent should return a non-nil report")
		assert.True(t, reportDistinct.IsNewTemplate, "IsNewTemplate mismatch for distinct event")
		assert.InDelta(t, anomalyScore(true, false, true), reportDistinct.AnomalyScore, 1e-9, "AnomalyScore mismatch for distinct event")
		assert.Equal(t, "new log template discovered; rare cluster (few observations)", reportDistinct.Reason, "Reason mismatch for distinct event")
	})
}

// TestAnalyzeEvent_Variants covers edge-case event shapes: empty stage and nil/empty fields.
func TestAnalyzeEvent_Variants(t *testing.T) {
	tests := []struct {
		name       string
		evt        AgentEvent
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "empty stage with fields succeeds",
			evt: AgentEvent{
				Stage:  "",
				Fields: map[string]string{"action": "start"},
			},
		},
		{
			name: "stage with nil fields succeeds",
			evt: AgentEvent{
				Stage:  "plan",
				Fields: nil,
			},
		},
		{
			name:       "empty stage and nil fields returns error",
			evt:        AgentEvent{Stage: "", Fields: nil},
			wantErr:    true,
			wantErrMsg: "empty event after masking",
		},
		{
			name:       "empty stage and empty fields returns error",
			evt:        AgentEvent{Stage: "", Fields: map[string]string{}},
			wantErr:    true,
			wantErrMsg: "empty event after masking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := DefaultConfig()
			m, err := NewMiner(cfg)
			require.NoError(t, err, "NewMiner should succeed")

			result, report, err := m.AnalyzeEvent(tt.evt)
			if tt.wantErr {
				require.Error(t, err, "AnalyzeEvent should return an error")
				assert.Contains(t, err.Error(), tt.wantErrMsg, "error message mismatch")
				assert.Nil(t, result, "AnalyzeEvent should return nil result on error")
				assert.Nil(t, report, "AnalyzeEvent should return nil report on error")
				return
			}
			require.NoError(t, err, "AnalyzeEvent should not return an error")
			require.NotNil(t, result, "AnalyzeEvent should return a non-nil result")
			require.NotNil(t, report, "AnalyzeEvent should return a non-nil report")
		})
	}
}

// TestAnalyze_NilResult ensures that Analyze does not panic when result is nil
// and instead returns a zero-value report with the "no anomaly detected" reason.
func TestAnalyze_NilResult(t *testing.T) {
	t.Parallel()
	d, err := NewAnomalyDetector(0.4, 2)
	require.NoError(t, err, "NewAnomalyDetector should succeed")

	// Nil result must not panic; the nil-guard in Analyze returns a safe report.
	report := d.Analyze(nil, false, nil)
	require.NotNil(t, report, "Analyze should return a non-nil report even for nil result")
	assert.Equal(t, "no anomaly detected", report.Reason, "nil result should produce no-anomaly reason")
	assert.InDelta(t, 0.0, report.AnomalyScore, 1e-9, "nil result should produce zero anomaly score")
	assert.False(t, report.IsNewTemplate, "nil result should not set IsNewTemplate")
	assert.False(t, report.LowSimilarity, "nil result should not set LowSimilarity")
	assert.False(t, report.RareCluster, "nil result should not set RareCluster")

	// Confirm that isNew=true and a non-nil cluster are silently ignored when result is nil.
	// The nil-guard short-circuits before reading those arguments; callers should not rely on
	// them being honoured when result is absent.
	cluster := &Cluster{ID: 1, Template: []string{"x"}, Size: 1}
	reportWithArgs := d.Analyze(nil, true, cluster)
	require.NotNil(t, reportWithArgs, "Analyze should return a non-nil report for nil result with non-nil args")
	assert.Equal(t, "no anomaly detected", reportWithArgs.Reason, "isNew and cluster must be ignored when result is nil")
	assert.InDelta(t, 0.0, reportWithArgs.AnomalyScore, 1e-9, "AnomalyScore must be zero when result is nil")
	assert.False(t, reportWithArgs.IsNewTemplate, "IsNewTemplate must be false when result is nil")
}

func TestAnalyze_FlagMutualExclusivity(t *testing.T) {
	d, err := NewAnomalyDetector(0.4, 2)
	require.NoError(t, err, "NewAnomalyDetector should succeed")

	tests := []struct {
		name     string
		isNew    bool
		result   *MatchResult
		cluster  *Cluster
		wantLow  bool
		wantRare bool
	}{
		{
			name:     "new template remains exclusive from low similarity",
			isNew:    true,
			result:   &MatchResult{ClusterID: 1, Similarity: 0.0},
			cluster:  &Cluster{ID: 1, Template: []string{"a"}, Size: 1},
			wantLow:  false,
			wantRare: true,
		},
		{
			name:     "existing template can have low similarity",
			isNew:    false,
			result:   &MatchResult{ClusterID: 1, Similarity: 0.2},
			cluster:  &Cluster{ID: 1, Template: []string{"a"}, Size: 1},
			wantLow:  true,
			wantRare: true,
		},
		{
			// nil cluster is intentionally supported by Analyze and must not set RareCluster.
			name:     "existing template with high similarity and nil cluster",
			isNew:    false,
			result:   &MatchResult{ClusterID: 1, Similarity: 0.9},
			cluster:  nil,
			wantLow:  false,
			wantRare: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := d.Analyze(tt.result, tt.isNew, tt.cluster)
			require.NotNil(t, report, "Analyze should always return a non-nil report")
			assert.Equal(t, tt.wantLow, report.LowSimilarity, "LowSimilarity mismatch")
			assert.Equal(t, tt.wantRare, report.RareCluster, "RareCluster mismatch")
			assert.False(t, report.IsNewTemplate && report.LowSimilarity, "IsNewTemplate and LowSimilarity must be mutually exclusive")
			assert.GreaterOrEqual(t, report.AnomalyScore, 0.0, "AnomalyScore must stay within [0,1]")
			assert.LessOrEqual(t, report.AnomalyScore, 1.0, "AnomalyScore must stay within [0,1]")
		})
	}
}
