package cli

import "github.com/github/gh-aw/pkg/sliceutil"

// DomainBuckets holds allowed and blocked domain lists with accessor methods.
// This struct is embedded by DomainAnalysis and FirewallAnalysis to share
// domain management functionality and eliminate code duplication.
type DomainBuckets struct {
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// GetAllowedDomains returns the list of allowed domains
func (d *DomainBuckets) GetAllowedDomains() []string {
	return d.AllowedDomains
}

// GetBlockedDomains returns the list of blocked domains
func (d *DomainBuckets) GetBlockedDomains() []string {
	return d.BlockedDomains
}

// SetAllowedDomains sets the list of allowed domains
func (d *DomainBuckets) SetAllowedDomains(domains []string) {
	d.AllowedDomains = domains
}

// SetBlockedDomains sets the list of blocked domains
func (d *DomainBuckets) SetBlockedDomains(domains []string) {
	d.BlockedDomains = domains
}

// AnalysisBase is the shared base embedded by DomainAnalysis and FirewallAnalysis.
// It holds the common counters and domain lists that both analysis types share,
// and provides a single AddMetrics implementation for the shared fields.
type AnalysisBase struct {
	DomainBuckets
	TotalRequests   int `json:"total_requests"`
	AllowedRequests int `json:"allowed_requests"`
	BlockedRequests int `json:"blocked_requests"`
}

// addBaseMetrics merges TotalRequests, AllowedRequests, BlockedRequests and domain
// lists from other into a. It is called by DomainAnalysis.AddMetrics and
// FirewallAnalysis.AddMetrics to eliminate the shared accumulation logic.
func (a *AnalysisBase) addBaseMetrics(other *AnalysisBase) {
	a.TotalRequests += other.TotalRequests
	a.AllowedRequests += other.AllowedRequests
	a.BlockedRequests += other.BlockedRequests
	a.BlockedDomains = mergeDomainList(a.BlockedDomains, other.BlockedDomains)
	a.AllowedDomains = mergeDomainList(a.AllowedDomains, other.AllowedDomains)
}

// mergeDomainList returns a sorted, deduplicated union of existing and incoming domain lists.
// If incoming is empty, existing is returned unchanged.
func mergeDomainList(existing, incoming []string) []string {
	if len(incoming) == 0 {
		return existing
	}
	domainSet := make(map[string]struct{}, len(existing)+len(incoming))
	for _, d := range existing {
		domainSet[d] = struct{}{}
	}
	for _, d := range incoming {
		domainSet[d] = struct{}{}
	}
	return sliceutil.SortedKeys(domainSet)
}
