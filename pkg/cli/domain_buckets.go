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

	// Merge blocked domain lists
	if len(other.BlockedDomains) > 0 {
		domainSet := make(map[string]struct{}, len(a.BlockedDomains)+len(other.BlockedDomains))
		for _, d := range a.BlockedDomains {
			domainSet[d] = struct{}{}
		}
		for _, d := range other.BlockedDomains {
			domainSet[d] = struct{}{}
		}
		a.BlockedDomains = sliceutil.SortedKeys(domainSet)
	}

	// Merge allowed domain lists
	if len(other.AllowedDomains) > 0 {
		domainSet := make(map[string]struct{}, len(a.AllowedDomains)+len(other.AllowedDomains))
		for _, d := range a.AllowedDomains {
			domainSet[d] = struct{}{}
		}
		for _, d := range other.AllowedDomains {
			domainSet[d] = struct{}{}
		}
		a.AllowedDomains = sliceutil.SortedKeys(domainSet)
	}
}
