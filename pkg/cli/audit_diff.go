package cli

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
)

var auditDiffLog = auditLog

// volumeChangeThresholdPercent is the minimum percentage increase to flag as a volume change.
// >100% increase means the request count more than doubled.
const volumeChangeThresholdPercent = 100.0

// DomainDiffEntry represents the diff for a single domain between two runs
type DomainDiffEntry struct {
	Domain       string `json:"domain"`
	Status       string `json:"status"`                  // "new", "removed", "status_changed", "volume_changed"
	Run1Allowed  int    `json:"run1_allowed"`            // Allowed requests in run 1
	Run1Blocked  int    `json:"run1_blocked"`            // Blocked requests in run 1
	Run2Allowed  int    `json:"run2_allowed"`            // Allowed requests in run 2
	Run2Blocked  int    `json:"run2_blocked"`            // Blocked requests in run 2
	Run1Status   string `json:"run1_status,omitempty"`   // "allowed", "denied", or "" for new domains
	Run2Status   string `json:"run2_status,omitempty"`   // "allowed", "denied", or "" for removed domains
	VolumeChange string `json:"volume_change,omitempty"` // e.g. "+287%" or "-50%"
	IsAnomaly    bool   `json:"is_anomaly,omitempty"`    // Flagged as anomalous (new denied, status flip to allowed)
	AnomalyNote  string `json:"anomaly_note,omitempty"`  // Human-readable anomaly explanation
}

// FirewallDiff represents the complete diff between two runs' firewall behavior
type FirewallDiff struct {
	Run1ID         int64               `json:"run1_id"`
	Run2ID         int64               `json:"run2_id"`
	NewDomains     []DomainDiffEntry   `json:"new_domains,omitempty"`
	RemovedDomains []DomainDiffEntry   `json:"removed_domains,omitempty"`
	StatusChanges  []DomainDiffEntry   `json:"status_changes,omitempty"`
	VolumeChanges  []DomainDiffEntry   `json:"volume_changes,omitempty"`
	Summary        FirewallDiffSummary `json:"summary"`
}

// FirewallDiffSummary provides a quick overview of the diff
type FirewallDiffSummary struct {
	NewDomainCount     int  `json:"new_domain_count"`
	RemovedDomainCount int  `json:"removed_domain_count"`
	StatusChangeCount  int  `json:"status_change_count"`
	VolumeChangeCount  int  `json:"volume_change_count"`
	HasAnomalies       bool `json:"has_anomalies"`
	AnomalyCount       int  `json:"anomaly_count"`
}

// computeFirewallDiff computes the diff between two FirewallAnalysis results.
// run1 is the "before" (baseline) and run2 is the "after" (comparison target).
// Either analysis may be nil, indicating no firewall data for that run.
func computeFirewallDiff(run1ID, run2ID int64, run1, run2 *FirewallAnalysis) *FirewallDiff {
	diff := &FirewallDiff{
		Run1ID: run1ID,
		Run2ID: run2ID,
	}

	// Handle nil cases
	run1Stats := make(map[string]DomainRequestStats)
	run2Stats := make(map[string]DomainRequestStats)

	if run1 != nil {
		run1Stats = run1.RequestsByDomain
	}
	if run2 != nil {
		run2Stats = run2.RequestsByDomain
	}

	// If both are nil/empty, return empty diff
	if len(run1Stats) == 0 && len(run2Stats) == 0 {
		return diff
	}

	// Collect all domains
	allDomains := make(map[string]bool)
	for domain := range run1Stats {
		allDomains[domain] = true
	}
	for domain := range run2Stats {
		allDomains[domain] = true
	}

	// Sorted domain list for deterministic output
	sortedDomains := make([]string, 0, len(allDomains))
	for domain := range allDomains {
		sortedDomains = append(sortedDomains, domain)
	}
	sort.Strings(sortedDomains)

	anomalyCount := 0

	for _, domain := range sortedDomains {
		stats1, inRun1 := run1Stats[domain]
		stats2, inRun2 := run2Stats[domain]

		if !inRun1 && inRun2 {
			// New domain in run 2
			entry := DomainDiffEntry{
				Domain:      domain,
				Status:      "new",
				Run2Allowed: stats2.Allowed,
				Run2Blocked: stats2.Blocked,
				Run2Status:  domainStatus(stats2),
			}
			// Anomaly: new denied domain
			if stats2.Blocked > 0 {
				entry.IsAnomaly = true
				entry.AnomalyNote = "new denied domain"
				anomalyCount++
			}
			diff.NewDomains = append(diff.NewDomains, entry)
		} else if inRun1 && !inRun2 {
			// Removed domain
			entry := DomainDiffEntry{
				Domain:      domain,
				Status:      "removed",
				Run1Allowed: stats1.Allowed,
				Run1Blocked: stats1.Blocked,
				Run1Status:  domainStatus(stats1),
			}
			diff.RemovedDomains = append(diff.RemovedDomains, entry)
		} else {
			// Domain exists in both runs - check for changes
			status1 := domainStatus(stats1)
			status2 := domainStatus(stats2)

			if status1 != status2 {
				// Status changed
				entry := DomainDiffEntry{
					Domain:      domain,
					Status:      "status_changed",
					Run1Allowed: stats1.Allowed,
					Run1Blocked: stats1.Blocked,
					Run2Allowed: stats2.Allowed,
					Run2Blocked: stats2.Blocked,
					Run1Status:  status1,
					Run2Status:  status2,
				}
				// Anomaly: previously denied, now allowed
				if status1 == "denied" && status2 == "allowed" {
					entry.IsAnomaly = true
					entry.AnomalyNote = "previously denied, now allowed"
					anomalyCount++
				}
				// Anomaly: previously allowed, now denied
				if status1 == "allowed" && status2 == "denied" {
					entry.IsAnomaly = true
					entry.AnomalyNote = "previously allowed, now denied"
					anomalyCount++
				}
				diff.StatusChanges = append(diff.StatusChanges, entry)
			} else {
				// Check for significant volume changes (>100% threshold)
				total1 := stats1.Allowed + stats1.Blocked
				total2 := stats2.Allowed + stats2.Blocked

				if total1 > 0 {
					pctChange := (float64(total2-total1) / float64(total1)) * 100
					if math.Abs(pctChange) > volumeChangeThresholdPercent {
						entry := DomainDiffEntry{
							Domain:       domain,
							Status:       "volume_changed",
							Run1Allowed:  stats1.Allowed,
							Run1Blocked:  stats1.Blocked,
							Run2Allowed:  stats2.Allowed,
							Run2Blocked:  stats2.Blocked,
							Run1Status:   status1,
							Run2Status:   status2,
							VolumeChange: formatVolumeChange(total1, total2),
						}
						diff.VolumeChanges = append(diff.VolumeChanges, entry)
					}
				}
			}
		}
	}

	diff.Summary = FirewallDiffSummary{
		NewDomainCount:     len(diff.NewDomains),
		RemovedDomainCount: len(diff.RemovedDomains),
		StatusChangeCount:  len(diff.StatusChanges),
		VolumeChangeCount:  len(diff.VolumeChanges),
		HasAnomalies:       anomalyCount > 0,
		AnomalyCount:       anomalyCount,
	}

	return diff
}

// domainStatus returns "allowed", "denied", or "mixed" based on request stats
func domainStatus(stats DomainRequestStats) string {
	if stats.Allowed > 0 && stats.Blocked == 0 {
		return "allowed"
	}
	if stats.Blocked > 0 && stats.Allowed == 0 {
		return "denied"
	}
	if stats.Allowed > 0 && stats.Blocked > 0 {
		return "mixed"
	}
	return "unknown"
}

// formatVolumeChange formats the volume change as a human-readable string
func formatVolumeChange(total1, total2 int) string {
	if total1 == 0 {
		return "+∞"
	}
	pctChange := (float64(total2-total1) / float64(total1)) * 100
	if pctChange >= 0 {
		return "+" + formatPercent(pctChange)
	}
	return formatPercent(pctChange)
}

// formatPercent formats a float percentage with no decimal places
func formatPercent(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}

// loadFirewallAnalysisForRun loads or computes the FirewallAnalysis for a given run.
// It first tries to load from a cached RunSummary; otherwise it downloads artifacts
// and analyzes firewall logs from scratch.
func loadFirewallAnalysisForRun(runID int64, outputDir string, owner, repo, hostname string, verbose bool) (*FirewallAnalysis, error) {
	runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", runID))
	if absDir, err := filepath.Abs(runOutputDir); err == nil {
		runOutputDir = absDir
	}

	// Try cached summary first
	if summary, ok := loadRunSummary(runOutputDir, verbose); ok {
		auditDiffLog.Printf("Using cached firewall analysis for run %d", runID)
		return summary.FirewallAnalysis, nil
	}

	// Download artifacts if needed
	if err := downloadRunArtifacts(runID, runOutputDir, verbose, owner, repo, hostname); err != nil {
		if !errors.Is(err, ErrNoArtifacts) {
			return nil, fmt.Errorf("failed to download artifacts for run %d: %w", runID, err)
		}
	}

	// Analyze firewall logs
	analysis, err := analyzeFirewallLogs(runOutputDir, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze firewall logs for run %d: %w", runID, err)
	}

	return analysis, nil
}
