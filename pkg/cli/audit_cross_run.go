package cli

import (
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

var auditCrossRunLog = logger.New("cli:audit_cross_run")

// maxAuditReportRuns is the upper bound on runs to analyze in a single report
// to bound download time and memory usage.
const maxAuditReportRuns = 50

// CrossRunFirewallReport represents aggregated firewall analysis across multiple workflow runs.
// Unlike FirewallAnalysis (single run), this provides a union of domains across runs
// with per-run breakdown and overall statistics.
type CrossRunFirewallReport struct {
	RunsAnalyzed    int                       `json:"runs_analyzed"`
	RunsWithData    int                       `json:"runs_with_data"`
	RunsWithoutData int                       `json:"runs_without_data"`
	Summary         CrossRunSummary           `json:"summary"`
	DomainInventory []DomainInventoryEntry    `json:"domain_inventory"`
	PerRunBreakdown []PerRunFirewallBreakdown `json:"per_run_breakdown"`
}

// CrossRunSummary provides top-level statistics across all analyzed runs.
type CrossRunSummary struct {
	TotalRequests   int     `json:"total_requests"`
	TotalAllowed    int     `json:"total_allowed"`
	TotalBlocked    int     `json:"total_blocked"`
	OverallDenyRate float64 `json:"overall_deny_rate"` // 0.0–1.0
	UniqueDomains   int     `json:"unique_domains"`
}

// DomainInventoryEntry describes a single domain seen across multiple runs.
type DomainInventoryEntry struct {
	Domain        string            `json:"domain"`
	SeenInRuns    int               `json:"seen_in_runs"`
	TotalAllowed  int               `json:"total_allowed"`
	TotalBlocked  int               `json:"total_blocked"`
	OverallStatus string            `json:"overall_status"` // "allowed", "denied", "mixed"
	PerRunStatus  []DomainRunStatus `json:"per_run_status"`
}

// DomainRunStatus records the status of a domain in a single run.
type DomainRunStatus struct {
	RunID   int64  `json:"run_id"`
	Status  string `json:"status"` // "allowed", "denied", "mixed", "absent"
	Allowed int    `json:"allowed"`
	Blocked int    `json:"blocked"`
}

// PerRunFirewallBreakdown is a summary row for a single run within the cross-run report.
type PerRunFirewallBreakdown struct {
	RunID         int64   `json:"run_id"`
	WorkflowName  string  `json:"workflow_name"`
	Conclusion    string  `json:"conclusion"`
	TotalRequests int     `json:"total_requests"`
	Allowed       int     `json:"allowed"`
	Blocked       int     `json:"blocked"`
	DenyRate      float64 `json:"deny_rate"` // 0.0–1.0
	UniqueDomains int     `json:"unique_domains"`
	HasData       bool    `json:"has_data"`
}

// crossRunInput bundles per-run data needed for aggregation.
type crossRunInput struct {
	RunID            int64
	WorkflowName     string
	Conclusion       string
	FirewallAnalysis *FirewallAnalysis
}

// buildCrossRunFirewallReport aggregates firewall data from multiple runs into a
// CrossRunFirewallReport.
func buildCrossRunFirewallReport(inputs []crossRunInput) *CrossRunFirewallReport {
	auditCrossRunLog.Printf("Building cross-run firewall report: %d inputs", len(inputs))

	report := &CrossRunFirewallReport{
		RunsAnalyzed: len(inputs),
	}

	// Aggregate per-domain data across all runs
	type domainAgg struct {
		totalAllowed int
		totalBlocked int
		perRun       []DomainRunStatus
	}
	domainMap := make(map[string]*domainAgg)

	// Ordered list of run IDs for deterministic per-run status
	runIDs := make([]int64, 0, len(inputs))
	for _, in := range inputs {
		runIDs = append(runIDs, in.RunID)
	}

	for _, in := range inputs {
		breakdown := PerRunFirewallBreakdown{
			RunID:        in.RunID,
			WorkflowName: in.WorkflowName,
			Conclusion:   in.Conclusion,
		}

		if in.FirewallAnalysis != nil {
			report.RunsWithData++
			breakdown.HasData = true
			breakdown.TotalRequests = in.FirewallAnalysis.TotalRequests
			breakdown.Allowed = in.FirewallAnalysis.AllowedRequests
			breakdown.Blocked = in.FirewallAnalysis.BlockedRequests
			if breakdown.TotalRequests > 0 {
				breakdown.DenyRate = float64(breakdown.Blocked) / float64(breakdown.TotalRequests)
			}
			breakdown.UniqueDomains = len(in.FirewallAnalysis.RequestsByDomain)

			report.Summary.TotalRequests += breakdown.TotalRequests
			report.Summary.TotalAllowed += breakdown.Allowed
			report.Summary.TotalBlocked += breakdown.Blocked

			for domain, stats := range in.FirewallAnalysis.RequestsByDomain {
				agg, exists := domainMap[domain]
				if !exists {
					agg = &domainAgg{}
					domainMap[domain] = agg
				}
				agg.totalAllowed += stats.Allowed
				agg.totalBlocked += stats.Blocked
				agg.perRun = append(agg.perRun, DomainRunStatus{
					RunID:   in.RunID,
					Status:  domainStatus(stats),
					Allowed: stats.Allowed,
					Blocked: stats.Blocked,
				})
			}
		} else {
			report.RunsWithoutData++
		}

		report.PerRunBreakdown = append(report.PerRunBreakdown, breakdown)
	}

	// Compute overall deny rate
	if report.Summary.TotalRequests > 0 {
		report.Summary.OverallDenyRate = float64(report.Summary.TotalBlocked) / float64(report.Summary.TotalRequests)
	}

	// Build domain inventory sorted by domain name
	sortedDomains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		sortedDomains = append(sortedDomains, domain)
	}
	sort.Strings(sortedDomains)

	report.Summary.UniqueDomains = len(sortedDomains)

	// Build a set of run IDs that have data for each domain to fill in "absent"
	for _, domain := range sortedDomains {
		agg := domainMap[domain]
		presentRuns := make(map[int64]bool, len(agg.perRun))
		for _, prs := range agg.perRun {
			presentRuns[prs.RunID] = true
		}

		// Build full per-run status including "absent" for runs without this domain
		fullPerRun := make([]DomainRunStatus, 0, len(runIDs))
		for _, rid := range runIDs {
			if presentRuns[rid] {
				for _, prs := range agg.perRun {
					if prs.RunID == rid {
						fullPerRun = append(fullPerRun, prs)
						break
					}
				}
			} else {
				fullPerRun = append(fullPerRun, DomainRunStatus{
					RunID:  rid,
					Status: "absent",
				})
			}
		}

		entry := DomainInventoryEntry{
			Domain:        domain,
			SeenInRuns:    len(agg.perRun),
			TotalAllowed:  agg.totalAllowed,
			TotalBlocked:  agg.totalBlocked,
			OverallStatus: domainStatus(DomainRequestStats{Allowed: agg.totalAllowed, Blocked: agg.totalBlocked}),
			PerRunStatus:  fullPerRun,
		}
		report.DomainInventory = append(report.DomainInventory, entry)
	}

	auditCrossRunLog.Printf("Cross-run report built: runs=%d, with_data=%d, unique_domains=%d, total_requests=%d",
		report.RunsAnalyzed, report.RunsWithData, report.Summary.UniqueDomains, report.Summary.TotalRequests)

	return report
}
