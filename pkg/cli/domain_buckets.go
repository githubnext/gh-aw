package cli

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
