//go:build !integration

package agentdrain

import (
	"testing"
)

func TestAnomalyDetection_NewTemplate(t *testing.T) {
	d := NewAnomalyDetector(0.4, 2)
	c := &Cluster{ID: 1, Template: []string{"stage=plan"}, Size: 1}
	result := &MatchResult{ClusterID: 1, Similarity: 1.0}

	report := d.Analyze(result, true, c)

	if !report.IsNewTemplate {
		t.Error("expected IsNewTemplate=true")
	}
	if !report.NewClusterCreated {
		t.Error("expected NewClusterCreated=true")
	}
	if report.AnomalyScore <= 0 {
		t.Errorf("expected positive anomaly score for new template, got %v", report.AnomalyScore)
	}
}

func TestAnomalyDetection_LowSimilarity(t *testing.T) {
	d := NewAnomalyDetector(0.4, 2)
	// Size=5 means not rare; not new.
	c := &Cluster{ID: 1, Template: []string{"a", "b", "c"}, Size: 5}
	result := &MatchResult{ClusterID: 1, Similarity: 0.2}

	report := d.Analyze(result, false, c)

	if !report.LowSimilarity {
		t.Error("expected LowSimilarity=true for similarity below threshold")
	}
	if report.IsNewTemplate {
		t.Error("expected IsNewTemplate=false")
	}
	if report.AnomalyScore <= 0 {
		t.Errorf("expected positive anomaly score, got %v", report.AnomalyScore)
	}
}

func TestAnomalyDetection_RareCluster(t *testing.T) {
	d := NewAnomalyDetector(0.4, 2)
	c := &Cluster{ID: 1, Template: []string{"a"}, Size: 1}
	result := &MatchResult{ClusterID: 1, Similarity: 0.9}

	report := d.Analyze(result, false, c)

	if !report.RareCluster {
		t.Error("expected RareCluster=true for size=1 with rareThreshold=2")
	}
	if report.AnomalyScore <= 0 {
		t.Errorf("expected positive anomaly score, got %v", report.AnomalyScore)
	}
}

func TestAnomalyDetection_Normal(t *testing.T) {
	d := NewAnomalyDetector(0.4, 2)
	// High size, high similarity, not new.
	c := &Cluster{ID: 1, Template: []string{"a", "b"}, Size: 100}
	result := &MatchResult{ClusterID: 1, Similarity: 0.9}

	report := d.Analyze(result, false, c)

	if report.IsNewTemplate {
		t.Error("expected IsNewTemplate=false")
	}
	if report.LowSimilarity {
		t.Error("expected LowSimilarity=false")
	}
	if report.RareCluster {
		t.Error("expected RareCluster=false")
	}
	if report.AnomalyScore > 0 {
		t.Errorf("expected zero anomaly score for normal event, got %v", report.AnomalyScore)
	}
	if report.Reason != "no anomaly detected" {
		t.Errorf("expected 'no anomaly detected', got %q", report.Reason)
	}
}

func TestAnalyzeEvent(t *testing.T) {
	cfg := DefaultConfig()
	m, err := NewMiner(cfg)
	if err != nil {
		t.Fatalf("NewMiner: %v", err)
	}

	// First occurrence → new template.
	evt := AgentEvent{
		Stage:  "plan",
		Fields: map[string]string{"action": "start", "model": "gpt-4"},
	}
	result, report, err := m.AnalyzeEvent(evt)
	if err != nil {
		t.Fatalf("AnalyzeEvent: %v", err)
	}
	if result == nil {
		t.Fatal("AnalyzeEvent: expected non-nil result")
	}
	if report == nil {
		t.Fatal("AnalyzeEvent: expected non-nil report")
	}
	if !report.IsNewTemplate {
		t.Error("first event should be a new template")
	}

	// Second occurrence of the same event → not new.
	result2, report2, err := m.AnalyzeEvent(evt)
	if err != nil {
		t.Fatalf("AnalyzeEvent (second): %v", err)
	}
	if result2 == nil || report2 == nil {
		t.Fatal("AnalyzeEvent (second): expected non-nil results")
	}
	if report2.IsNewTemplate {
		t.Error("second identical event should not be a new template")
	}
}
