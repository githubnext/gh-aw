package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSafeOutputsAllowedDomains(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "empty allowed domains",
			config:  &SafeOutputsConfig{AllowedDomains: []string{}},
			wantErr: false,
		},
		{
			name: "valid plain domains",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"github.com",
					"api.github.com",
					"example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "valid wildcard domains",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"*.github.com",
					"*.example.org",
				},
			},
			wantErr: false,
		},
		{
			name: "mixed valid domains",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"github.com",
					"*.githubusercontent.com",
					"api.example.com",
					"*.test.org",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - empty domain",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{""},
			},
			wantErr: true,
			errMsg:  "domain cannot be empty",
		},
		{
			name: "invalid - wildcard only",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"*"},
			},
			wantErr: true,
			errMsg:  "wildcard-only domain '*' is not allowed",
		},
		{
			name: "invalid - multiple wildcards",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"*.*.github.com"},
			},
			wantErr: true,
			errMsg:  "contains multiple wildcards",
		},
		{
			name: "invalid - wildcard in middle",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"github.*.com"},
			},
			wantErr: true,
			errMsg:  "wildcard in invalid position",
		},
		{
			name: "invalid - wildcard at end",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"github.*"},
			},
			wantErr: true,
			errMsg:  "wildcard in invalid position",
		},
		{
			name: "invalid - trailing dot",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"github.com."},
			},
			wantErr: true,
			errMsg:  "cannot end with a dot",
		},
		{
			name: "invalid - leading dot",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{".github.com"},
			},
			wantErr: true,
			errMsg:  "cannot start with a dot",
		},
		{
			name: "invalid - consecutive dots",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"github..com"},
			},
			wantErr: true,
			errMsg:  "cannot contain consecutive dots",
		},
		{
			name: "invalid - special characters",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"github@example.com"},
			},
			wantErr: true,
			errMsg:  "contains invalid character",
		},
		{
			name: "invalid - spaces",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"github .com"},
			},
			wantErr: true,
			errMsg:  "contains invalid character",
		},
		{
			name: "invalid - wildcard without base domain",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{"*."},
			},
			wantErr: true,
			errMsg:  "must have a domain after",
		},
		{
			name: "invalid - multiple domains in first entry",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"github.com",
					"*.example.com",
					"invalid domain",
				},
			},
			wantErr: true,
			errMsg:  "safe-outputs.allowed-domains[2]",
		},
		{
			name: "valid - complex subdomain",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"very.long.subdomain.example.com",
					"*.multi.level.example.org",
				},
			},
			wantErr: false,
		},
		{
			name: "valid - domains with numbers and hyphens",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"api-v2.github.com",
					"test123.example.com",
					"*.cdn-example.org",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsAllowedDomains(tt.config)
			if tt.wantErr {
				assert.Error(t, err, "Expected an error but got none")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateDomainPattern(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
		errMsg  string
	}{
		// Valid plain domains
		{
			name:    "valid - simple domain",
			domain:  "github.com",
			wantErr: false,
		},
		{
			name:    "valid - subdomain",
			domain:  "api.github.com",
			wantErr: false,
		},
		{
			name:    "valid - multiple subdomains",
			domain:  "api.v2.github.com",
			wantErr: false,
		},
		{
			name:    "valid - domain with numbers",
			domain:  "test123.example.com",
			wantErr: false,
		},
		{
			name:    "valid - domain with hyphens",
			domain:  "my-api.example-site.com",
			wantErr: false,
		},

		// Valid wildcard domains
		{
			name:    "valid - wildcard subdomain",
			domain:  "*.github.com",
			wantErr: false,
		},
		{
			name:    "valid - wildcard with multiple levels",
			domain:  "*.api.example.com",
			wantErr: false,
		},

		// Invalid patterns
		{
			name:    "invalid - empty",
			domain:  "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "invalid - wildcard only",
			domain:  "*",
			wantErr: true,
			errMsg:  "wildcard-only",
		},
		{
			name:    "invalid - multiple wildcards",
			domain:  "*.*.github.com",
			wantErr: true,
			errMsg:  "multiple wildcards",
		},
		{
			name:    "invalid - wildcard in middle",
			domain:  "api.*.github.com",
			wantErr: true,
			errMsg:  "invalid position",
		},
		{
			name:    "invalid - wildcard at end",
			domain:  "github.*",
			wantErr: true,
			errMsg:  "invalid position",
		},
		{
			name:    "invalid - trailing dot",
			domain:  "github.com.",
			wantErr: true,
			errMsg:  "cannot end with a dot",
		},
		{
			name:    "invalid - leading dot",
			domain:  ".github.com",
			wantErr: true,
			errMsg:  "cannot start with a dot",
		},
		{
			name:    "invalid - consecutive dots",
			domain:  "github..com",
			wantErr: true,
			errMsg:  "consecutive dots",
		},
		{
			name:    "invalid - underscore",
			domain:  "github_api.com",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "invalid - special character @",
			domain:  "user@github.com",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "invalid - space",
			domain:  "github .com",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "invalid - wildcard without domain",
			domain:  "*.",
			wantErr: true,
			errMsg:  "must have a domain after",
		},
		{
			name:    "invalid - wildcard with dot after",
			domain:  "*..",
			wantErr: true,
			errMsg:  "invalid format",
		},

		// Edge cases
		{
			name:    "valid - single character domain (theoretical)",
			domain:  "a.b",
			wantErr: false,
		},
		{
			name:    "valid - long subdomain",
			domain:  "very-long-subdomain-name-with-many-hyphens.example.com",
			wantErr: false,
		},
		{
			name:    "valid - many levels",
			domain:  "a.b.c.d.e.f.example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDomainPattern(tt.domain)
			if tt.wantErr {
				assert.Error(t, err, "Expected an error for domain: %s", tt.domain)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no error for domain: %s, but got: %v", tt.domain, err)
			}
		})
	}
}

func TestValidateDomainPatternCoverage(t *testing.T) {
	// Test various error paths to ensure comprehensive coverage
	errorCases := []struct {
		domain      string
		description string
	}{
		{"", "empty domain"},
		{"*", "wildcard only"},
		{"*.*", "double wildcard"},
		{"*.*.*.com", "triple wildcard"},
		{"test.*.example.com", "wildcard in middle"},
		{"test.*", "wildcard at end"},
		{"example.com.", "trailing dot"},
		{".example.com", "leading dot"},
		{"example..com", "consecutive dots"},
		{"example@test.com", "@ character"},
		{"example test.com", "space character"},
		{"example_test.com", "underscore"},
		{"*..", "wildcard with double dot"},
		{"*.", "wildcard without base"},
		{"example!.com", "exclamation mark"},
		{"*.*.github.com", "multiple wildcards nested"},
	}

	for _, tc := range errorCases {
		t.Run(tc.description, func(t *testing.T) {
			err := validateDomainPattern(tc.domain)
			assert.Error(t, err, "Domain '%s' (%s) should produce an error", tc.domain, tc.description)
		})
	}

	// Test valid patterns for positive coverage
	validCases := []string{
		"example.com",
		"api.example.com",
		"*.example.com",
		"test-api.example.com",
		"api123.example.com",
		"*.api.example.org",
		"a.b.c.d.example.com",
	}

	for _, domain := range validCases {
		t.Run("valid-"+domain, func(t *testing.T) {
			err := validateDomainPattern(domain)
			assert.NoError(t, err, "Valid domain '%s' should not produce an error", domain)
		})
	}
}

// TestDomainPatternRegex tests the domain pattern regex directly
func TestDomainPatternRegex(t *testing.T) {
	tests := []struct {
		domain  string
		matches bool
	}{
		// Should match
		{"example.com", true},
		{"*.example.com", true},
		{"api.example.com", true},
		{"test-123.example.com", true},

		// Should not match
		{"", false},
		{"example.com.", false},
		{".example.com", false},
		{"example..com", false},
		{"*.*.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			matches := domainPattern.MatchString(tt.domain)
			assert.Equal(t, tt.matches, matches, "Domain pattern regex match for '%s'", tt.domain)
		})
	}
}

// TestValidateSafeOutputsAllowedDomainsIntegration tests validation with realistic workflow configurations
func TestValidateSafeOutputsAllowedDomainsIntegration(t *testing.T) {
	tests := []struct {
		name    string
		config  *SafeOutputsConfig
		wantErr bool
	}{
		{
			name: "typical configuration",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"api.github.com",
					"*.githubusercontent.com",
					"raw.githubusercontent.com",
				},
			},
			wantErr: false,
		},
		{
			name: "multi-repository configuration",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"*.github.com",
					"*.gitlab.com",
					"api.bitbucket.org",
				},
			},
			wantErr: false,
		},
		{
			name: "CDN and API domains",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"cdn.example.com",
					"*.cdn.example.com",
					"api-v2.example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "configuration with error in list",
			config: &SafeOutputsConfig{
				AllowedDomains: []string{
					"api.github.com",
					"*.invalid..com", // Double dot
					"valid.example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsAllowedDomains(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
