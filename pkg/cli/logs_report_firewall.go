package cli

import (
	"slices"
)

// AccessLogSummary contains aggregated access log analysis
type AccessLogSummary struct {
	TotalRequests  int                        `json:"total_requests" console:"header:Total Requests"`
	AllowedCount   int                        `json:"allowed_count" console:"header:Allowed"`
	BlockedCount   int                        `json:"blocked_count" console:"header:Blocked"`
	AllowedDomains []string                   `json:"allowed_domains" console:"-"`
	BlockedDomains []string                   `json:"blocked_domains" console:"-"`
	ByWorkflow     map[string]*DomainAnalysis `json:"by_workflow,omitempty" console:"-"`
}

// FirewallLogSummary contains aggregated firewall log data
type FirewallLogSummary struct {
	TotalRequests    int                           `json:"total_requests" console:"header:Total Requests"`
	AllowedRequests  int                           `json:"allowed_requests" console:"header:Allowed"`
	BlockedRequests  int                           `json:"blocked_requests" console:"header:Blocked"`
	AllowedDomains   []string                      `json:"allowed_domains" console:"-"`
	BlockedDomains   []string                      `json:"blocked_domains" console:"-"`
	RequestsByDomain map[string]DomainRequestStats `json:"requests_by_domain,omitempty" console:"-"`
	ByWorkflow       map[string]*FirewallAnalysis  `json:"by_workflow,omitempty" console:"-"`
}

// domainAggregation holds the result of aggregating domain statistics
type domainAggregation struct {
	allAllowedDomains map[string]bool
	allBlockedDomains map[string]bool
	totalRequests     int
	allowedCount      int
	blockedCount      int
}

// aggregateDomainStats aggregates domain statistics across runs
// This is a shared helper for both access log and firewall log summaries
func aggregateDomainStats(processedRuns []ProcessedRun, getAnalysis func(*ProcessedRun) (allowedDomains, blockedDomains []string, totalRequests, allowedCount, blockedCount int, exists bool)) *domainAggregation {
	agg := &domainAggregation{
		allAllowedDomains: make(map[string]bool),
		allBlockedDomains: make(map[string]bool),
	}

	for i := range processedRuns {
		allowedDomains, blockedDomains, totalRequests, allowedCount, blockedCount, exists := getAnalysis(&processedRuns[i])
		if !exists {
			continue
		}

		agg.totalRequests += totalRequests
		agg.allowedCount += allowedCount
		agg.blockedCount += blockedCount

		for _, domain := range allowedDomains {
			agg.allAllowedDomains[domain] = true
		}
		for _, domain := range blockedDomains {
			agg.allBlockedDomains[domain] = true
		}
	}

	return agg
}

// convertDomainsToSortedSlices converts domain maps to sorted slices
func convertDomainsToSortedSlices(allowedMap, blockedMap map[string]bool) (allowed, blocked []string) {
	for domain := range allowedMap {
		allowed = append(allowed, domain)
	}
	slices.Sort(allowed)

	for domain := range blockedMap {
		blocked = append(blocked, domain)
	}
	slices.Sort(blocked)

	return allowed, blocked
}

// buildAccessLogSummary aggregates access log data across all runs
func buildAccessLogSummary(processedRuns []ProcessedRun) *AccessLogSummary {
	byWorkflow := make(map[string]*DomainAnalysis)

	// Use shared aggregation helper
	agg := aggregateDomainStats(processedRuns, func(pr *ProcessedRun) ([]string, []string, int, int, int, bool) {
		if pr.AccessAnalysis == nil {
			return nil, nil, 0, 0, 0, false
		}
		byWorkflow[pr.Run.WorkflowName] = pr.AccessAnalysis
		return pr.AccessAnalysis.AllowedDomains,
			pr.AccessAnalysis.BlockedDomains,
			pr.AccessAnalysis.TotalRequests,
			pr.AccessAnalysis.AllowedCount,
			pr.AccessAnalysis.BlockedCount,
			true
	})

	if agg.totalRequests == 0 {
		return nil
	}

	allowedDomains, blockedDomains := convertDomainsToSortedSlices(agg.allAllowedDomains, agg.allBlockedDomains)

	return &AccessLogSummary{
		TotalRequests:  agg.totalRequests,
		AllowedCount:   agg.allowedCount,
		BlockedCount:   agg.blockedCount,
		AllowedDomains: allowedDomains,
		BlockedDomains: blockedDomains,
		ByWorkflow:     byWorkflow,
	}
}

// buildFirewallLogSummary aggregates firewall log data across all runs
func buildFirewallLogSummary(processedRuns []ProcessedRun) *FirewallLogSummary {
	allRequestsByDomain := make(map[string]DomainRequestStats)
	byWorkflow := make(map[string]*FirewallAnalysis)

	// Use shared aggregation helper
	agg := aggregateDomainStats(processedRuns, func(pr *ProcessedRun) ([]string, []string, int, int, int, bool) {
		if pr.FirewallAnalysis == nil {
			return nil, nil, 0, 0, 0, false
		}
		byWorkflow[pr.Run.WorkflowName] = pr.FirewallAnalysis

		// Aggregate request stats by domain (firewall-specific)
		for domain, stats := range pr.FirewallAnalysis.RequestsByDomain {
			existing := allRequestsByDomain[domain]
			existing.Allowed += stats.Allowed
			existing.Blocked += stats.Blocked
			allRequestsByDomain[domain] = existing
		}

		return pr.FirewallAnalysis.AllowedDomains,
			pr.FirewallAnalysis.BlockedDomains,
			pr.FirewallAnalysis.TotalRequests,
			pr.FirewallAnalysis.AllowedRequests,
			pr.FirewallAnalysis.BlockedRequests,
			true
	})

	if agg.totalRequests == 0 {
		return nil
	}

	allowedDomains, blockedDomains := convertDomainsToSortedSlices(agg.allAllowedDomains, agg.allBlockedDomains)

	return &FirewallLogSummary{
		TotalRequests:    agg.totalRequests,
		AllowedRequests:  agg.allowedCount,
		BlockedRequests:  agg.blockedCount,
		AllowedDomains:   allowedDomains,
		BlockedDomains:   blockedDomains,
		RequestsByDomain: allRequestsByDomain,
		ByWorkflow:       byWorkflow,
	}
}

// buildRedactedDomainsSummary aggregates redacted domains data across all runs
func buildRedactedDomainsSummary(processedRuns []ProcessedRun) *RedactedDomainsLogSummary {
	allDomainsSet := make(map[string]bool)
	byWorkflow := make(map[string]*RedactedDomainsAnalysis)
	hasData := false

	for _, pr := range processedRuns {
		if pr.RedactedDomainsAnalysis == nil {
			continue
		}
		hasData = true
		byWorkflow[pr.Run.WorkflowName] = pr.RedactedDomainsAnalysis

		// Collect all unique domains
		for _, domain := range pr.RedactedDomainsAnalysis.Domains {
			allDomainsSet[domain] = true
		}
	}

	if !hasData {
		return nil
	}

	// Convert set to sorted slice
	var allDomains []string
	for domain := range allDomainsSet {
		allDomains = append(allDomains, domain)
	}
	slices.Sort(allDomains)

	return &RedactedDomainsLogSummary{
		TotalDomains: len(allDomains),
		Domains:      allDomains,
		ByWorkflow:   byWorkflow,
	}
}
