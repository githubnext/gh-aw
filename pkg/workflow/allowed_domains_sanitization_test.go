//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// extractQuotedCSV returns the comma-separated domain list embedded inside
// the first pair of double-quotes in line. Used to enable exact-entry checks
// (avoiding substring false-positives like "corp.example.com" matching "copilot.corp.example.com").
func extractQuotedCSV(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return line
	}
	rest := line[start+1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// TestAllowedDomainsFromNetworkConfig tests that GH_AW_ALLOWED_DOMAINS is computed
// from network configuration for sanitization
func TestAllowedDomainsFromNetworkConfig(t *testing.T) {
	tests := []struct {
		name             string
		workflow         string
		expectedDomains  []string // domains that should be in GH_AW_ALLOWED_DOMAINS
		unexpectedDomain string   // domain that should NOT be in GH_AW_ALLOWED_DOMAINS
	}{
		{
			name: "Copilot with network permissions",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
network:
  allowed:
    - example.com
    - test.org
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with network permissions.
`,
			expectedDomains: []string{
				"example.com",
				"test.org",
				// Copilot defaults should also be included
				"api.github.com",
				"github.com",
				"raw.githubusercontent.com",
				"registry.npmjs.org",
			},
			unexpectedDomain: "",
		},
		{
			name: "Claude with network permissions",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
strict: false
network:
  allowed:
    - example.com
    - test.org
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with network permissions.
`,
			expectedDomains: []string{
				"example.com",
				"test.org",
				// Claude now has its own default domains with AWF support
				"api.github.com",
				"anthropic.com",
				"api.anthropic.com",
			},
			// No unexpected domains - Claude has its own defaults
			unexpectedDomain: "",
		},
		{
			name: "Copilot with defaults network mode",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
network: defaults
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with defaults network.
`,
			expectedDomains: []string{
				// Should have Copilot defaults
				"api.github.com",
				"github.com",
				"raw.githubusercontent.com",
				// Note: network: defaults for Copilot doesn't expand ecosystem domains
				// in GetCopilotAllowedDomains - it only merges when network.allowed has values
			},
			unexpectedDomain: "",
		},
		{
			name: "Copilot without network config",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow without network config.
`,
			expectedDomains: []string{
				// Should have Copilot defaults
				"api.github.com",
				"github.com",
				"raw.githubusercontent.com",
				// Note: nil network for Copilot only returns Copilot defaults
			},
			unexpectedDomain: "",
		},
		{
			name: "Claude with ecosystem identifier",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
strict: false
network:
  allowed:
    - python
    - node
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with ecosystem identifiers.
`,
			expectedDomains: []string{
				// Python ecosystem
				"pypi.org",
				"files.pythonhosted.org",
				// Node ecosystem
				"npmjs.org",
				"registry.npmjs.org",
			},
			unexpectedDomain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test
			tmpDir := testutil.TempDir(t, "allowed-domains-test")

			// Create a test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflow), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockStr := string(lockContent)

			// Check if GH_AW_ALLOWED_DOMAINS is set in the Ingest agent output step
			if !strings.Contains(lockStr, "GH_AW_ALLOWED_DOMAINS:") {
				t.Error("Expected GH_AW_ALLOWED_DOMAINS environment variable in lock file")
			}

			// Extract the GH_AW_ALLOWED_DOMAINS value
			lines := strings.Split(lockStr, "\n")
			var domainsLine string
			for _, line := range lines {
				if strings.Contains(line, "GH_AW_ALLOWED_DOMAINS:") {
					domainsLine = line
					break
				}
			}

			if domainsLine == "" {
				t.Fatal("GH_AW_ALLOWED_DOMAINS not found in lock file")
			}

			// Check that expected domains are present
			for _, expectedDomain := range tt.expectedDomains {
				if !strings.Contains(domainsLine, expectedDomain) {
					t.Errorf("Expected domain '%s' not found in GH_AW_ALLOWED_DOMAINS.\nLine: %s", expectedDomain, domainsLine)
				}
			}

			// Check that unexpected domain is NOT present
			if tt.unexpectedDomain != "" {
				if strings.Contains(domainsLine, tt.unexpectedDomain) {
					t.Errorf("Unexpected domain '%s' found in GH_AW_ALLOWED_DOMAINS.\nLine: %s", tt.unexpectedDomain, domainsLine)
				}
			}
		})
	}
}

// TestManualAllowedDomainsUnionWithNetworkConfig tests that manually configured allowed-domains
// unions with network configuration (not overrides it)
func TestManualAllowedDomainsUnionWithNetworkConfig(t *testing.T) {
	tests := []struct {
		name             string
		workflow         string
		expectedDomains  []string
		unexpectedDomain string
	}{
		{
			name: "Manual allowed-domains unions with network config",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
network:
  allowed:
    - example.com
    - python
safe-outputs:
  create-issue:
  allowed-domains:
    - manual-domain.com
    - override.org
---

# Test Workflow

Test that manual allowed-domains unions with network config.
`,
			expectedDomains: []string{
				"manual-domain.com",
				"override.org",
				"example.com", // from network.allowed - still present (union)
			},
			// No domain should be absent
			unexpectedDomain: "",
		},
		{
			name: "Empty allowed-domains uses network config",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
network:
  allowed:
    - example.com
safe-outputs:
  create-issue:
---

# Test Workflow

Test that empty allowed-domains falls back to network config.
`,
			expectedDomains: []string{
				"example.com",
				"api.github.com", // Copilot default
			},
			unexpectedDomain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test
			tmpDir := testutil.TempDir(t, "manual-domains-test")

			// Create a test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflow), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockStr := string(lockContent)

			// Check if GH_AW_ALLOWED_DOMAINS is set
			if !strings.Contains(lockStr, "GH_AW_ALLOWED_DOMAINS:") {
				t.Error("Expected GH_AW_ALLOWED_DOMAINS environment variable in lock file")
			}

			// Extract the GH_AW_ALLOWED_DOMAINS value
			lines := strings.Split(lockStr, "\n")
			var domainsLine string
			for _, line := range lines {
				if strings.Contains(line, "GH_AW_ALLOWED_DOMAINS:") {
					domainsLine = line
					break
				}
			}

			if domainsLine == "" {
				t.Fatal("GH_AW_ALLOWED_DOMAINS not found in lock file")
			}

			// Check that expected domains are present
			for _, expectedDomain := range tt.expectedDomains {
				if !strings.Contains(domainsLine, expectedDomain) {
					t.Errorf("Expected domain '%s' not found in GH_AW_ALLOWED_DOMAINS.\nLine: %s", expectedDomain, domainsLine)
				}
			}

			// Check that unexpected domain is NOT present
			if tt.unexpectedDomain != "" {
				if strings.Contains(domainsLine, tt.unexpectedDomain) {
					t.Errorf("Unexpected domain '%s' found in GH_AW_ALLOWED_DOMAINS.\nLine: %s", tt.unexpectedDomain, domainsLine)
				}
			}
		})
	}
}

// TestComputeAllowedDomainsForSanitization tests the computeAllowedDomainsForSanitization function
func TestComputeAllowedDomainsForSanitization(t *testing.T) {
	tests := []struct {
		name              string
		engineID          string
		apiTarget         string
		networkPerms      *NetworkPermissions
		expectedDomains   []string
		unexpectedDomains []string
	}{
		{
			name:     "Copilot with custom domains",
			engineID: "copilot",
			networkPerms: &NetworkPermissions{
				Allowed: []string{"example.com", "test.org"},
			},
			expectedDomains: []string{
				"example.com",
				"test.org",
				"api.github.com", // Copilot default
				"github.com",     // Copilot default
			},
		},
		{
			name:     "Claude with custom domains",
			engineID: "claude",
			networkPerms: &NetworkPermissions{
				Allowed: []string{"example.com", "test.org"},
			},
			expectedDomains: []string{
				"example.com",
				"test.org",
			},
		},
		{
			name:         "Copilot with nil network",
			engineID:     "copilot",
			networkPerms: nil,
			expectedDomains: []string{
				"api.github.com",            // Copilot default
				"github.com",                // Copilot default
				"raw.githubusercontent.com", // Copilot default
				// Note: When network is nil, GetCopilotAllowedDomains only returns Copilot defaults
				// It does NOT include ecosystem defaults
			},
		},
		{
			name:         "Claude with nil network",
			engineID:     "claude",
			networkPerms: nil,
			expectedDomains: []string{
				"json-schema.org",    // ecosystem default
				"archive.ubuntu.com", // ecosystem default
			},
		},
		{
			name:     "Codex with custom domains",
			engineID: "codex",
			networkPerms: &NetworkPermissions{
				Allowed: []string{"example.com"},
			},
			expectedDomains: []string{
				"example.com",
			},
		},
		{
			name:         "Copilot with GHES api-target includes api and base domains",
			engineID:     "copilot",
			apiTarget:    "api.acme.ghe.com",
			networkPerms: nil,
			expectedDomains: []string{
				"api.acme.ghe.com", // GHES API domain
				"acme.ghe.com",     // GHES base domain (derived from api-target)
				"api.github.com",   // Copilot default
				"github.com",       // Copilot default
			},
		},
		{
			name:         "non-api prefix api-target only adds the configured hostname",
			engineID:     "copilot",
			apiTarget:    "copilot.corp.example.com",
			networkPerms: nil,
			expectedDomains: []string{
				"copilot.corp.example.com", // configured hostname
			},
			unexpectedDomains: []string{
				"corp.example.com", // base hostname should NOT be added for non-api. prefix
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a compiler and workflow data
			compiler := NewCompiler()
			data := &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:        tt.engineID,
					APITarget: tt.apiTarget,
				},
				NetworkPermissions: tt.networkPerms,
			}

			// Call the function
			domainsStr, err := compiler.computeAllowedDomainsForSanitization(data)
			require.NoError(t, err, "computeAllowedDomainsForSanitization should not return an error for valid test data")

			// Verify expected domains are present (substring match is fine here since domain names
			// in a CSV string that are exact entries won't appear as substrings of other entries
			// when checking expected ones – we only need exact match for the negative "not present" check)
			for _, expectedDomain := range tt.expectedDomains {
				if !strings.Contains(domainsStr, expectedDomain) {
					t.Errorf("Expected domain '%s' not found in result: %s", expectedDomain, domainsStr)
				}
			}

			// Verify unexpected domains are absent using exact membership (not substring)
			// to avoid false positives where "corp.example.com" matches "copilot.corp.example.com"
			parts := strings.Split(domainsStr, ",")
			for _, unexpectedDomain := range tt.unexpectedDomains {
				if slices.Contains(parts, unexpectedDomain) {
					t.Errorf("Unexpected domain '%s' found in result: %s", unexpectedDomain, domainsStr)
				}
			}
		})
	}
}

// TestAPITargetDomainsInCompiledWorkflow is a regression test verifying that when engine.api-target
// is configured, both --allow-domains (AWF firewall flag) and GH_AW_ALLOWED_DOMAINS (sanitization
// env var) in the compiled lock file contain the api-target hostname and its derived base hostname.
func TestAPITargetDomainsInCompiledWorkflow(t *testing.T) {
	tests := []struct {
		name              string
		workflow          string
		expectedDomains   []string
		unexpectedDomains []string
	}{
		{
			name: "GHES api-target adds api and base domains to allow-domains and GH_AW_ALLOWED_DOMAINS",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: copilot
  api-target: api.acme.ghe.com
strict: false
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with GHES api-target.
`,
			expectedDomains: []string{
				"api.acme.ghe.com", // GHES API domain
				"acme.ghe.com",     // GHES base domain derived from api-target
				"api.github.com",   // Copilot default
				"github.com",       // Copilot default
			},
		},
		{
			name: "non-api prefix api-target only adds the configured hostname",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: copilot
  api-target: copilot.corp.example.com
strict: false
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with non-api prefix api-target.
`,
			expectedDomains: []string{
				"copilot.corp.example.com", // configured hostname
			},
			unexpectedDomains: []string{
				"corp.example.com", // base hostname should NOT be added for non-api. prefix
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "api-target-domains-test")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflow), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockStr := string(lockContent)

			// Check --allow-domains in AWF command contains expected domains
			allowDomainsIdx := strings.Index(lockStr, "--allow-domains")
			if allowDomainsIdx < 0 {
				t.Fatal("--allow-domains flag not found in compiled lock file")
			}
			// Extract the line with --allow-domains for more targeted checking
			allowDomainsEnd := strings.Index(lockStr[allowDomainsIdx:], "\n")
			if allowDomainsEnd < 0 {
				allowDomainsEnd = len(lockStr) - allowDomainsIdx
			}
			allowDomainsLine := lockStr[allowDomainsIdx : allowDomainsIdx+allowDomainsEnd]

			for _, domain := range tt.expectedDomains {
				if !strings.Contains(allowDomainsLine, domain) {
					t.Errorf("Expected domain %q not found in --allow-domains.\nLine: %s", domain, allowDomainsLine)
				}
			}
			// Use exact CSV membership for "not present" checks to avoid false positives
			// (e.g. "corp.example.com" would substring-match "copilot.corp.example.com")
			allowedDomainsCSV := extractQuotedCSV(allowDomainsLine)
			allowedParts := strings.Split(allowedDomainsCSV, ",")
			for _, domain := range tt.unexpectedDomains {
				if slices.Contains(allowedParts, domain) {
					t.Errorf("Unexpected domain %q found in --allow-domains.\nLine: %s", domain, allowDomainsLine)
				}
			}

			// Check GH_AW_ALLOWED_DOMAINS env var contains expected domains
			lines := strings.Split(lockStr, "\n")
			var domainsLine string
			for _, line := range lines {
				if strings.Contains(line, "GH_AW_ALLOWED_DOMAINS:") {
					domainsLine = line
					break
				}
			}
			if domainsLine == "" {
				t.Fatal("GH_AW_ALLOWED_DOMAINS not found in compiled lock file")
			}

			for _, domain := range tt.expectedDomains {
				if !strings.Contains(domainsLine, domain) {
					t.Errorf("Expected domain %q not found in GH_AW_ALLOWED_DOMAINS.\nLine: %s", domain, domainsLine)
				}
			}
			// Use exact CSV membership for "not present" checks
			allowedDomainsEnvCSV := extractQuotedCSV(domainsLine)
			allowedEnvParts := strings.Split(allowedDomainsEnvCSV, ",")
			for _, domain := range tt.unexpectedDomains {
				if slices.Contains(allowedEnvParts, domain) {
					t.Errorf("Unexpected domain %q found in GH_AW_ALLOWED_DOMAINS.\nLine: %s", domain, domainsLine)
				}
			}
		})
	}
}

// TestGitHubCopilotBaseURLInCompiledWorkflow verifies that when GITHUB_COPILOT_BASE_URL is set
// in engine.env (without an explicit engine.api-target), the compiled lock file contains
// --copilot-api-target and includes the extracted hostname in both --allow-domains and
// GH_AW_ALLOWED_DOMAINS — matching the OPENAI_BASE_URL/ANTHROPIC_BASE_URL pattern for other engines.
func TestGitHubCopilotBaseURLInCompiledWorkflow(t *testing.T) {
	workflow := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: copilot
  env:
    GITHUB_COPILOT_BASE_URL: "https://copilot-proxy.corp.example.com"
strict: false
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with GITHUB_COPILOT_BASE_URL in engine.env.
`

	tmpDir := testutil.TempDir(t, "copilot-base-url-test")
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflow), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	// --copilot-api-target should be derived from the env var
	if !strings.Contains(lockStr, "--copilot-api-target copilot-proxy.corp.example.com") {
		t.Error("Expected --copilot-api-target to be derived from GITHUB_COPILOT_BASE_URL")
	}

	// Extracted hostname should appear in --allow-domains
	allowDomainsIdx := strings.Index(lockStr, "--allow-domains")
	if allowDomainsIdx < 0 {
		t.Fatal("--allow-domains flag not found in compiled lock file")
	}
	allowDomainsEnd := strings.Index(lockStr[allowDomainsIdx:], "\n")
	if allowDomainsEnd < 0 {
		allowDomainsEnd = len(lockStr) - allowDomainsIdx
	}
	allowDomainsLine := lockStr[allowDomainsIdx : allowDomainsIdx+allowDomainsEnd]
	if !strings.Contains(allowDomainsLine, "copilot-proxy.corp.example.com") {
		t.Errorf("Expected hostname from GITHUB_COPILOT_BASE_URL in --allow-domains.\nLine: %s", allowDomainsLine)
	}

	// Extracted hostname should appear in GH_AW_ALLOWED_DOMAINS
	lines := strings.Split(lockStr, "\n")
	var domainsLine string
	for _, line := range lines {
		if strings.Contains(line, "GH_AW_ALLOWED_DOMAINS:") {
			domainsLine = line
			break
		}
	}
	if domainsLine == "" {
		t.Fatal("GH_AW_ALLOWED_DOMAINS not found in compiled lock file")
	}
	if !strings.Contains(domainsLine, "copilot-proxy.corp.example.com") {
		t.Errorf("Expected hostname from GITHUB_COPILOT_BASE_URL in GH_AW_ALLOWED_DOMAINS.\nLine: %s", domainsLine)
	}
}

// TestAPITargetDomainsInThreatDetectionStep is a regression test verifying that when engine.api-target
// is configured, the threat detection AWF invocation in the compiled lock file also receives
// --copilot-api-target and includes the GHE domains in its --allow-domains list.
// Regression test for: Threat detection AWF run missing --copilot-api-target on data residency.
func TestAPITargetDomainsInThreatDetectionStep(t *testing.T) {
	workflow := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine:
  id: copilot
  api-target: api.contoso-aw.ghe.com
strict: false
safe-outputs:
  create-issue:
---

# Test Workflow

Test workflow with GHE data residency api-target and threat detection.
`

	tmpDir := testutil.TempDir(t, "api-target-threat-detection-test")
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflow), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(testFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	// Verify --copilot-api-target appears at least twice:
	// once for the main agent AWF run and once for the threat detection AWF run.
	apiTargetCount := strings.Count(lockStr, "--copilot-api-target api.contoso-aw.ghe.com")
	if apiTargetCount < 2 {
		t.Errorf("Expected --copilot-api-target to appear in both the main agent and threat detection AWF invocations (at least 2 times), but found %d occurrence(s).", apiTargetCount)
	}

	// Find all --allow-domains occurrences and verify each contains the GHE domains.
	// api.contoso-aw.ghe.com triggers base-domain derivation, so both the API domain
	// and the base domain (contoso-aw.ghe.com) must appear in each AWF invocation.
	requiredDomains := []string{"api.contoso-aw.ghe.com", "contoso-aw.ghe.com"}
	remaining := lockStr
	occurrenceIdx := 0
	for {
		idx := strings.Index(remaining, "--allow-domains")
		if idx < 0 {
			break
		}
		occurrenceIdx++
		lineEnd := strings.Index(remaining[idx:], "\n")
		if lineEnd < 0 {
			lineEnd = len(remaining) - idx
		}
		line := remaining[idx : idx+lineEnd]
		for _, domain := range requiredDomains {
			if !strings.Contains(line, domain) {
				t.Errorf("--allow-domains occurrence #%d is missing GHE domain %q.\nLine: %s", occurrenceIdx, domain, line)
			}
		}
		remaining = remaining[idx+lineEnd:]
	}

	if occurrenceIdx < 2 {
		t.Errorf("Expected at least 2 --allow-domains occurrences (main agent + threat detection), found %d", occurrenceIdx)
	}
}

// TestAllowedDomainsUnionWithNetworkConfig tests that safe-outputs.allowed-domains
// is unioned with network.allowed and always includes localhost and github.com
func TestAllowedDomainsUnionWithNetworkConfig(t *testing.T) {
	tests := []struct {
		name            string
		workflow        string
		expectedDomains []string
	}{
		{
			name: "allowed-domains unioned with Copilot defaults and network config",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
engine: copilot
strict: false
network:
  allowed:
    - example.com
safe-outputs:
  create-issue:
  allowed-domains:
    - extra-domain.com
---

# Test Workflow

Test allowed-domains union with network config.
`,
			expectedDomains: []string{
				"extra-domain.com", // from allowed-domains
				"example.com",      // from network.allowed
				"api.github.com",   // Copilot default
				"localhost",        // always included
				"github.com",       // always included
			},
		},
		{
			name: "allowed-domains supports ecosystem identifiers",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
engine: copilot
strict: false
safe-outputs:
  create-issue:
  allowed-domains:
    - dev-tools
    - python
---

# Test Workflow

Test allowed-domains with ecosystem identifiers.
`,
			expectedDomains: []string{
				"codecov.io", // from dev-tools ecosystem
				"snyk.io",    // from dev-tools ecosystem
				"pypi.org",   // from python ecosystem
				"localhost",  // always included
				"github.com", // always included
			},
		},
		{
			name: "allowed-domains does not override network config",
			workflow: `---
on: push
permissions:
  contents: read
  issues: read
engine: copilot
strict: false
network:
  allowed:
    - network-domain.com
safe-outputs:
  create-issue:
  allowed-domains:
    - url-domain.com
---

# Test Workflow

Test that allowed-domains does not override network config.
`,
			expectedDomains: []string{
				"url-domain.com",     // from allowed-domains
				"network-domain.com", // from network.allowed - still present (union)
				"api.github.com",     // Copilot default
				"localhost",          // always included
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "allowed-domains-test")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflow), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockStr := string(lockContent)

			if !strings.Contains(lockStr, "GH_AW_ALLOWED_DOMAINS:") {
				t.Error("Expected GH_AW_ALLOWED_DOMAINS environment variable in lock file")
			}

			lines := strings.Split(lockStr, "\n")
			var domainsLine string
			for _, line := range lines {
				if strings.Contains(line, "GH_AW_ALLOWED_DOMAINS:") {
					domainsLine = line
					break
				}
			}

			if domainsLine == "" {
				t.Fatal("GH_AW_ALLOWED_DOMAINS not found in lock file")
			}

			for _, expectedDomain := range tt.expectedDomains {
				if !strings.Contains(domainsLine, expectedDomain) {
					t.Errorf("Expected domain %q not found in GH_AW_ALLOWED_DOMAINS.\nLine: %s", expectedDomain, domainsLine)
				}
			}
		})
	}
}
