package agentdrain

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var anomalyLog = logger.New("agentdrain:anomaly")

// AnomalyDetector evaluates match results and produces AnomalyReports.
type AnomalyDetector struct {
	threshold     float64
	rareThreshold int
}

// NewAnomalyDetector creates an AnomalyDetector with the given thresholds.
func NewAnomalyDetector(simThreshold float64, rareClusterThreshold int) *AnomalyDetector {
	return &AnomalyDetector{
		threshold:     simThreshold,
		rareThreshold: rareClusterThreshold,
	}
}

// Analyze produces an AnomalyReport for a match result.
//
//   - isNew indicates the line created a brand-new cluster.
//   - cluster is the cluster that was matched or created.
func (d *AnomalyDetector) Analyze(result *MatchResult, isNew bool, cluster *Cluster) *AnomalyReport {
	report := &AnomalyReport{
		IsNewTemplate:     isNew,
		NewClusterCreated: isNew,
		// LowSimilarity is mutually exclusive with IsNewTemplate: brand-new templates are
		// already classified as anomalies, so we only evaluate similarity for existing ones.
		LowSimilarity: !isNew && result.Similarity < d.threshold,
		RareCluster:   cluster != nil && cluster.Size <= d.rareThreshold,
	}

	// Weighted anomaly score.
	var score float64
	if report.IsNewTemplate {
		score += 1.0
	}
	if report.LowSimilarity {
		score += 0.7
	}
	if report.RareCluster {
		score += 0.3
	}
	// Normalize to [0, 1].
	const maxScore = 2.0
	// Defensive guard: with current mutually exclusive flags the score cannot exceed maxScore,
	// but keep clamping in case future weighting or flag logic changes.
	if score > maxScore {
		score = maxScore
	}
	report.AnomalyScore = score / maxScore

	report.Reason = buildReason(report)
	if anomalyLog.Enabled() {
		anomalyLog.Printf("Anomaly analysis: score=%.2f, isNew=%t, lowSim=%t, rare=%t, reason=%s",
			report.AnomalyScore, report.IsNewTemplate, report.LowSimilarity, report.RareCluster, report.Reason)
	}
	return report
}

// buildReason constructs a human-readable summary of detected anomalies.
func buildReason(r *AnomalyReport) string {
	var parts []string
	if r.IsNewTemplate {
		parts = append(parts, "new log template discovered")
	}
	if r.LowSimilarity {
		parts = append(parts, "low similarity to known template")
	}
	if r.RareCluster {
		parts = append(parts, "rare cluster (few observations)")
	}
	if len(parts) == 0 {
		return "no anomaly detected"
	}
	return strings.Join(parts, "; ")
}
