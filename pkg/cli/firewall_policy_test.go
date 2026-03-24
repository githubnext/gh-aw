//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainMatchesRule(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		rule     PolicyRule
		expected bool
	}{
		{
			name: "exact match without port",
			host: "github.com",
			rule: PolicyRule{
				Domains: []string{"github.com"},
			},
			expected: true,
		},
		{
			name: "exact match with port",
			host: "github.com:443",
			rule: PolicyRule{
				Domains: []string{"github.com"},
			},
			expected: true,
		},
		{
			name: "wildcard match - subdomain",
			host: "api.github.com:443",
			rule: PolicyRule{
				Domains: []string{".github.com"},
			},
			expected: true,
		},
		{
			name: "wildcard match - base domain",
			host: "github.com:443",
			rule: PolicyRule{
				Domains: []string{".github.com"},
			},
			expected: true,
		},
		{
			name: "wildcard match - deep subdomain",
			host: "api.v2.github.com:443",
			rule: PolicyRule{
				Domains: []string{".github.com"},
			},
			expected: true,
		},
		{
			name: "no match - different domain",
			host: "evil.com:443",
			rule: PolicyRule{
				Domains: []string{".github.com"},
			},
			expected: false,
		},
		{
			name: "no match - suffix collision",
			host: "notgithub.com:443",
			rule: PolicyRule{
				Domains: []string{".github.com"},
			},
			expected: false,
		},
		{
			name: "case insensitive match",
			host: "API.GitHub.COM:443",
			rule: PolicyRule{
				Domains: []string{".github.com"},
			},
			expected: true,
		},
		{
			name: "multiple domains in rule",
			host: "npmjs.org:443",
			rule: PolicyRule{
				Domains: []string{".github.com", "npmjs.org"},
			},
			expected: true,
		},
		{
			name: "no match - empty domains",
			host: "example.com",
			rule: PolicyRule{
				Domains: []string{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domainMatchesRule(tt.host, tt.rule)
			assert.Equal(t, tt.expected, result, "domainMatchesRule(%q) should return %v", tt.host, tt.expected)
		})
	}
}

func TestFindMatchingRule(t *testing.T) {
	rules := []PolicyRule{
		{
			ID:      "allow-github",
			Order:   1,
			Action:  "allow",
			Domains: []string{".github.com"},
		},
		{
			ID:      "allow-npm",
			Order:   2,
			Action:  "allow",
			Domains: []string{"registry.npmjs.org"},
		},
		{
			ID:      "deny-all",
			Order:   3,
			Action:  "deny",
			Domains: []string{"."},
		},
	}

	t.Run("matches first rule", func(t *testing.T) {
		rule := findMatchingRule("api.github.com:443", rules)
		require.NotNil(t, rule, "Should find a matching rule")
		assert.Equal(t, "allow-github", rule.ID, "Should match allow-github rule")
	})

	t.Run("matches second rule", func(t *testing.T) {
		rule := findMatchingRule("registry.npmjs.org:443", rules)
		require.NotNil(t, rule, "Should find a matching rule")
		assert.Equal(t, "allow-npm", rule.ID, "Should match allow-npm rule")
	})

	t.Run("no match returns nil", func(t *testing.T) {
		// evil.com does not match any of the specific rules, and the deny-all
		// rule has domain "." which won't match via our matching logic
		rule := findMatchingRule("evil.com:443", rules[:2])
		assert.Nil(t, rule, "Should not find a matching rule for unlisted domain")
	})

	t.Run("first matching rule wins", func(t *testing.T) {
		// github.com matches both allow-github (.github.com) and potentially others
		rule := findMatchingRule("github.com:443", rules)
		require.NotNil(t, rule, "Should find a matching rule")
		assert.Equal(t, "allow-github", rule.ID, "First matching rule should win")
	})
}

func TestLoadPolicyManifest(t *testing.T) {
	t.Run("valid manifest", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "policy-manifest.json")

		manifest := PolicyManifest{
			Version:     1,
			GeneratedAt: "2026-01-01T00:00:00Z",
			Rules: []PolicyRule{
				{ID: "rule-b", Order: 2, Action: "deny", Domains: []string{".evil.com"}, Description: "Block evil"},
				{ID: "rule-a", Order: 1, Action: "allow", Domains: []string{".github.com"}, Description: "Allow GitHub"},
			},
			SSLBumpEnabled: false,
			DLPEnabled:     false,
		}

		data, err := json.Marshal(manifest)
		require.NoError(t, err, "Should marshal manifest")
		require.NoError(t, os.WriteFile(manifestPath, data, 0644), "Should write manifest file")

		loaded, err := loadPolicyManifest(manifestPath)
		require.NoError(t, err, "Should load manifest without error")
		require.NotNil(t, loaded, "Loaded manifest should not be nil")

		assert.Equal(t, 1, loaded.Version, "Version should be 1")
		assert.Len(t, loaded.Rules, 2, "Should have 2 rules")
		// Rules should be sorted by order
		assert.Equal(t, "rule-a", loaded.Rules[0].ID, "First rule should be rule-a (order 1)")
		assert.Equal(t, "rule-b", loaded.Rules[1].ID, "Second rule should be rule-b (order 2)")
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loadPolicyManifest("/nonexistent/path/policy-manifest.json")
		assert.Error(t, err, "Should return error for missing file")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "policy-manifest.json")
		require.NoError(t, os.WriteFile(manifestPath, []byte("not json"), 0644))

		_, err := loadPolicyManifest(manifestPath)
		assert.Error(t, err, "Should return error for invalid JSON")
	})
}

func TestParseAuditJSONL(t *testing.T) {
	t.Run("valid JSONL", func(t *testing.T) {
		dir := t.TempDir()
		jsonlPath := filepath.Join(dir, "audit.jsonl")

		lines := `{"ts":1761074374.646,"client":"172.30.0.20","host":"api.github.com:443","dest":"140.82.114.22:443","method":"CONNECT","status":200,"decision":"TCP_TUNNEL","url":"api.github.com:443"}
{"ts":1761074375.100,"client":"172.30.0.20","host":"evil.com:443","dest":"1.2.3.4:443","method":"CONNECT","status":403,"decision":"NONE_NONE","url":"evil.com:443"}
{"ts":1761074376.200,"client":"172.30.0.20","host":"registry.npmjs.org:443","dest":"104.16.0.1:443","method":"CONNECT","status":200,"decision":"TCP_TUNNEL","url":"registry.npmjs.org:443"}
`
		require.NoError(t, os.WriteFile(jsonlPath, []byte(lines), 0644))

		entries, err := parseAuditJSONL(jsonlPath)
		require.NoError(t, err, "Should parse JSONL without error")
		assert.Len(t, entries, 3, "Should parse 3 entries")

		// Check first entry
		assert.InDelta(t, 1761074374.646, entries[0].Timestamp, 0.001, "Timestamp should match")
		assert.Equal(t, "api.github.com:443", entries[0].Host, "Host should match")
		assert.Equal(t, 200, entries[0].Status, "Status should match")

		// Check denied entry
		assert.Equal(t, "evil.com:443", entries[1].Host, "Second host should match")
		assert.Equal(t, 403, entries[1].Status, "Second status should be 403")
	})

	t.Run("empty lines skipped", func(t *testing.T) {
		dir := t.TempDir()
		jsonlPath := filepath.Join(dir, "audit.jsonl")

		lines := `{"ts":1.0,"host":"a.com:443","status":200}

{"ts":2.0,"host":"b.com:443","status":200}
`
		require.NoError(t, os.WriteFile(jsonlPath, []byte(lines), 0644))

		entries, err := parseAuditJSONL(jsonlPath)
		require.NoError(t, err, "Should parse JSONL without error")
		assert.Len(t, entries, 2, "Should parse 2 entries, skipping empty line")
	})

	t.Run("malformed lines skipped", func(t *testing.T) {
		dir := t.TempDir()
		jsonlPath := filepath.Join(dir, "audit.jsonl")

		lines := `{"ts":1.0,"host":"valid.com:443","status":200}
not valid json
{"ts":2.0,"host":"also-valid.com:443","status":200}
`
		require.NoError(t, os.WriteFile(jsonlPath, []byte(lines), 0644))

		entries, err := parseAuditJSONL(jsonlPath)
		require.NoError(t, err, "Should parse JSONL without error despite malformed line")
		assert.Len(t, entries, 2, "Should parse 2 valid entries")
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := parseAuditJSONL("/nonexistent/path/audit.jsonl")
		assert.Error(t, err, "Should return error for missing file")
	})
}

func TestEnrichWithPolicyRules(t *testing.T) {
	manifest := &PolicyManifest{
		Version:     1,
		GeneratedAt: "2026-01-01T00:00:00Z",
		Rules: []PolicyRule{
			{
				ID:          "allow-github",
				Order:       1,
				Action:      "allow",
				ACLName:     "allowed_domains",
				Domains:     []string{".github.com"},
				Description: "Allow GitHub and subdomains",
			},
			{
				ID:          "allow-npm",
				Order:       2,
				Action:      "allow",
				ACLName:     "npm_domains",
				Domains:     []string{"registry.npmjs.org"},
				Description: "Allow npm registry",
			},
			{
				ID:          "deny-all",
				Order:       3,
				Action:      "deny",
				ACLName:     "blocked_all",
				Domains:     []string{"."},
				Description: "Deny all other traffic",
			},
		},
		SSLBumpEnabled: false,
		DLPEnabled:     false,
	}

	entries := []AuditLogEntry{
		{Timestamp: 1761074374.646, Host: "api.github.com:443", Status: 200},
		{Timestamp: 1761074375.100, Host: "github.com:443", Status: 200},
		{Timestamp: 1761074376.200, Host: "registry.npmjs.org:443", Status: 200},
		{Timestamp: 1761074377.300, Host: "evil.com:443", Status: 403},
	}

	analysis := enrichWithPolicyRules(entries, manifest)
	require.NotNil(t, analysis, "Analysis should not be nil")

	t.Run("summary statistics", func(t *testing.T) {
		assert.Equal(t, 4, analysis.TotalRequests, "Should have 4 total requests")
		assert.Equal(t, 3, analysis.AllowedCount, "Should have 3 allowed requests")
		assert.Equal(t, 1, analysis.DeniedCount, "Should have 1 denied request")
		assert.Equal(t, 4, analysis.UniqueDomains, "Should have 4 unique domains")
	})

	t.Run("policy summary", func(t *testing.T) {
		assert.Contains(t, analysis.PolicySummary, "3 rules", "Policy summary should mention 3 rules")
		assert.Contains(t, analysis.PolicySummary, "SSL Bump disabled", "Should mention SSL Bump status")
		assert.Contains(t, analysis.PolicySummary, "DLP disabled", "Should mention DLP status")
	})

	t.Run("rule hits", func(t *testing.T) {
		require.Len(t, analysis.RuleHits, 3, "Should have 3 rule hit entries")
		// Rules should be in order
		assert.Equal(t, "allow-github", analysis.RuleHits[0].Rule.ID, "First rule should be allow-github")
		assert.Equal(t, 2, analysis.RuleHits[0].Hits, "allow-github should have 2 hits")
		assert.Equal(t, "allow-npm", analysis.RuleHits[1].Rule.ID, "Second rule should be allow-npm")
		assert.Equal(t, 1, analysis.RuleHits[1].Hits, "allow-npm should have 1 hit")
	})

	t.Run("denied requests", func(t *testing.T) {
		// evil.com does not match any specific rule's domains; it gets implicit deny
		require.Len(t, analysis.DeniedRequests, 1, "Should have 1 denied request")
		assert.Equal(t, "evil.com:443", analysis.DeniedRequests[0].Host, "Denied request host should match")
		assert.Equal(t, "deny", analysis.DeniedRequests[0].Action, "Action should be deny")
	})

	t.Run("entries with empty host skipped", func(t *testing.T) {
		emptyEntries := []AuditLogEntry{
			{Timestamp: 1.0, Host: "", Status: 200},
			{Timestamp: 2.0, Host: "-", Status: 200},
			{Timestamp: 3.0, Host: "valid.com:443", Status: 200},
		}
		result := enrichWithPolicyRules(emptyEntries, manifest)
		// valid.com doesn't match any rule, gets implicit deny
		assert.Equal(t, 1, result.TotalRequests, "Only valid entries should be counted")
	})
}

func TestDetectFirewallAuditArtifacts(t *testing.T) {
	t.Run("sandbox/firewall/audit path", func(t *testing.T) {
		dir := t.TempDir()
		auditDir := filepath.Join(dir, "sandbox", "firewall", "audit")
		require.NoError(t, os.MkdirAll(auditDir, 0755))

		manifestPath := filepath.Join(auditDir, "policy-manifest.json")
		auditPath := filepath.Join(auditDir, "audit.jsonl")
		require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0644))
		require.NoError(t, os.WriteFile(auditPath, []byte(""), 0644))

		foundManifest, foundAudit := detectFirewallAuditArtifacts(dir)
		assert.Equal(t, manifestPath, foundManifest, "Should find policy manifest")
		assert.Equal(t, auditPath, foundAudit, "Should find audit JSONL")
	})

	t.Run("firewall-audit-logs directory", func(t *testing.T) {
		dir := t.TempDir()
		auditDir := filepath.Join(dir, "firewall-audit-logs")
		require.NoError(t, os.MkdirAll(auditDir, 0755))

		manifestPath := filepath.Join(auditDir, "policy-manifest.json")
		auditPath := filepath.Join(auditDir, "audit.jsonl")
		require.NoError(t, os.WriteFile(manifestPath, []byte("{}"), 0644))
		require.NoError(t, os.WriteFile(auditPath, []byte(""), 0644))

		foundManifest, foundAudit := detectFirewallAuditArtifacts(dir)
		assert.Equal(t, manifestPath, foundManifest, "Should find policy manifest in firewall-audit-logs")
		assert.Equal(t, auditPath, foundAudit, "Should find audit JSONL in firewall-audit-logs")
	})

	t.Run("no artifacts", func(t *testing.T) {
		dir := t.TempDir()
		foundManifest, foundAudit := detectFirewallAuditArtifacts(dir)
		assert.Empty(t, foundManifest, "Should not find manifest")
		assert.Empty(t, foundAudit, "Should not find audit JSONL")
	})
}

func TestAnalyzeFirewallPolicy(t *testing.T) {
	t.Run("full enrichment", func(t *testing.T) {
		dir := t.TempDir()
		auditDir := filepath.Join(dir, "sandbox", "firewall", "audit")
		require.NoError(t, os.MkdirAll(auditDir, 0755))

		// Write policy manifest
		manifest := PolicyManifest{
			Version:     1,
			GeneratedAt: "2026-01-01T00:00:00Z",
			Rules: []PolicyRule{
				{ID: "allow-github", Order: 1, Action: "allow", Domains: []string{".github.com"}, Description: "Allow GitHub"},
				{ID: "deny-blocked", Order: 2, Action: "deny", Domains: []string{".evil.com"}, Description: "Block evil domains"},
			},
		}
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(auditDir, "policy-manifest.json"), manifestData, 0644))

		// Write audit JSONL
		jsonl := `{"ts":1.0,"host":"api.github.com:443","status":200}
{"ts":2.0,"host":"evil.com:443","status":403}
`
		require.NoError(t, os.WriteFile(filepath.Join(auditDir, "audit.jsonl"), []byte(jsonl), 0644))

		analysis, err := analyzeFirewallPolicy(dir, false)
		require.NoError(t, err, "Should analyze without error")
		require.NotNil(t, analysis, "Analysis should not be nil")

		assert.Equal(t, 2, analysis.TotalRequests, "Should have 2 total requests")
		assert.Equal(t, 1, analysis.AllowedCount, "Should have 1 allowed request")
		assert.Equal(t, 1, analysis.DeniedCount, "Should have 1 denied request")
		assert.Len(t, analysis.DeniedRequests, 1, "Should have 1 denied request detail")
	})

	t.Run("manifest only - no audit.jsonl", func(t *testing.T) {
		dir := t.TempDir()
		auditDir := filepath.Join(dir, "sandbox", "firewall", "audit")
		require.NoError(t, os.MkdirAll(auditDir, 0755))

		manifest := PolicyManifest{
			Version:        1,
			Rules:          []PolicyRule{{ID: "r1", Order: 1, Action: "allow", Domains: []string{".example.com"}}},
			SSLBumpEnabled: true,
		}
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(auditDir, "policy-manifest.json"), manifestData, 0644))

		analysis, err := analyzeFirewallPolicy(dir, false)
		require.NoError(t, err, "Should not error with manifest only")
		require.NotNil(t, analysis, "Should return analysis with manifest-only data")
		assert.Contains(t, analysis.PolicySummary, "SSL Bump enabled", "Should reflect SSL Bump enabled")
	})

	t.Run("no artifacts returns nil", func(t *testing.T) {
		dir := t.TempDir()
		analysis, err := analyzeFirewallPolicy(dir, false)
		require.NoError(t, err, "Should not error when no artifacts found")
		assert.Nil(t, analysis, "Should return nil when no artifacts found")
	})
}

func TestFormatUnixTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		ts       float64
		expected string
	}{
		{
			name:     "valid timestamp",
			ts:       1761074374.646,
			expected: "19:19:34",
		},
		{
			name:     "zero timestamp",
			ts:       0,
			expected: "-",
		},
		{
			name:     "negative timestamp",
			ts:       -1.0,
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUnixTimestamp(tt.ts)
			assert.Equal(t, tt.expected, result, "formatUnixTimestamp(%v)", tt.ts)
		})
	}
}
