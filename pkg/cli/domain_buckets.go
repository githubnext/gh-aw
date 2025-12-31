package cli

// DomainBuckets holds allowed and denied domain lists with accessor methods.
// This struct is embedded by DomainAnalysis and FirewallAnalysis to share
// domain management functionality and eliminate code duplication.
type DomainBuckets struct {
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	DeniedDomains  []string `json:"denied_domains,omitempty"`
}

// GetAllowedDomains returns the list of allowed domains
func (d *DomainBuckets) GetAllowedDomains() []string {
	return d.AllowedDomains
}

// GetDeniedDomains returns the list of denied domains
func (d *DomainBuckets) GetDeniedDomains() []string {
	return d.DeniedDomains
}

// SetAllowedDomains sets the list of allowed domains
func (d *DomainBuckets) SetAllowedDomains(domains []string) {
	d.AllowedDomains = domains
}

// SetDeniedDomains sets the list of denied domains
func (d *DomainBuckets) SetDeniedDomains(domains []string) {
	d.DeniedDomains = domains
}
