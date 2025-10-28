package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWildcardNetworkPermissionsClaudeEngine verifies that wildcard patterns
// are correctly included in the Python network hook for Claude engine
func TestWildcardNetworkPermissionsClaudeEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	yamlContent := `---
on: push
engine: claude
network:
  allowed:
    - "github.com"
    - "*.example.com"
    - "api.trusted.com"
---

# Test Wildcard Network Permissions

Test that wildcard patterns work correctly.`

	tmpDir, err := os.MkdirTemp("", "wildcard-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(filePath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(filePath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(filePath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	result := string(lockContent)

	// Verify the compiled workflow contains the wildcard pattern in ALLOWED_DOMAINS
	if !strings.Contains(result, `"*.example.com"`) {
		t.Error("Compiled workflow should contain wildcard pattern *.example.com in ALLOWED_DOMAINS")
	}

	// Verify the wildcard conversion logic is present
	if !strings.Contains(result, `pattern.replace('.', r'\.').replace('*', '.*')`) {
		t.Error("Compiled workflow should contain wildcard-to-regex conversion logic")
	}

	// Verify the is_domain_allowed function exists
	if !strings.Contains(result, "def is_domain_allowed") {
		t.Error("Compiled workflow should contain is_domain_allowed function")
	}
}

// TestWildcardNetworkPermissionsCopilotEngine verifies that wildcard patterns
// are correctly passed to AWF for Copilot engine
func TestWildcardNetworkPermissionsCopilotEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	yamlContent := `---
on: push
engine: copilot
network:
  firewall: true
  allowed:
    - "github.com"
    - "*.example.com"
    - "api.trusted.com"
tools:
  edit:
  github:
---

# Test Wildcard Network Permissions for Copilot

Test that wildcard patterns are passed to AWF.`

	tmpDir, err := os.MkdirTemp("", "wildcard-copilot-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(filePath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(filePath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(filePath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	result := string(lockContent)

	// Verify AWF is configured with the wildcard pattern
	if !strings.Contains(result, "--allow-domains") {
		t.Error("Compiled workflow should contain --allow-domains flag for AWF")
	}

	// Verify the wildcard pattern is in the allow-domains list
	// The pattern should be in the comma-separated list passed to AWF
	if !strings.Contains(result, "*.example.com") {
		t.Error("Compiled workflow should contain wildcard pattern *.example.com in AWF --allow-domains")
	}
}

// TestWildcardDomainMatching documents the expected wildcard matching behavior
func TestWildcardDomainMatching(t *testing.T) {
	testCases := []struct {
		pattern  string
		domain   string
		expected bool
		reason   string
	}{
		{
			pattern:  "*.example.com",
			domain:   "api.example.com",
			expected: true,
			reason:   "Wildcard should match single-level subdomain",
		},
		{
			pattern:  "*.example.com",
			domain:   "nested.api.example.com",
			expected: true,
			reason:   "Wildcard should match multi-level subdomain",
		},
		{
			pattern:  "*.example.com",
			domain:   "example.com",
			expected: false,
			reason:   "Wildcard should NOT match base domain",
		},
		{
			pattern:  "*.example.com",
			domain:   "notexample.com",
			expected: false,
			reason:   "Wildcard should NOT match different domain with same suffix",
		},
		{
			pattern:  "api.example.com",
			domain:   "api.example.com",
			expected: true,
			reason:   "Exact match should work",
		},
		{
			pattern:  "api.example.com",
			domain:   "other.example.com",
			expected: false,
			reason:   "Exact match should not match different subdomain",
		},
	}

	// This test documents the EXPECTED behavior based on the security guide
	// which states: "*.example.com matches any subdomain including nested ones"
	//
	// For Claude engine: This is implemented via Python regex conversion
	// For Copilot engine: This should be implemented by AWF (gh-aw-firewall)
	//
	// The Python implementation in engine_network_hooks.go line 72:
	//   regex = pattern.replace('.', r'\.').replace('*', '.*')
	//   if re.match(f'^{regex}$', domain):
	//       return True

	t.Log("Expected wildcard domain matching behavior:")
	for _, tc := range testCases {
		t.Logf("  Pattern: %s, Domain: %s => Expected: %v (%s)",
			tc.pattern, tc.domain, tc.expected, tc.reason)
	}
}

// TestWildcardSecurityGuideAccuracy verifies that the security guide
// documents domain matching behavior correctly for different engines
func TestWildcardSecurityGuideAccuracy(t *testing.T) {
	// Read the security guide
	securityGuide, err := os.ReadFile("../../docs/src/content/docs/guides/security.md")
	if err != nil {
		t.Skipf("Skipping test - security guide not found: %v", err)
		return
	}

	content := string(securityGuide)

	// Verify the security guide documents domain matching behavior
	if !strings.Contains(content, "Domain Matching Behavior") {
		t.Error("Security guide should document domain matching behavior in best practices")
	}

	if !strings.Contains(content, "*.example.com") {
		t.Error("Security guide should include wildcard example (*.example.com)")
	}

	// Verify the guide distinguishes between Claude and Copilot
	if !strings.Contains(content, "Claude engine") {
		t.Error("Security guide should document Claude engine behavior")
	}

	if !strings.Contains(content, "Copilot engine") {
		t.Error("Security guide should document Copilot engine behavior")
	}

	// Verify the guide explains AWF's automatic subdomain matching
	if !strings.Contains(content, "automatically matches subdomains") {
		t.Error("Security guide should explain AWF's automatic subdomain matching")
	}

	// The security guide now correctly distinguishes between engines:
	// - Claude engine: Supports wildcard syntax
	// - Copilot engine with AWF: Does NOT support wildcards, uses automatic subdomain matching
	//
	// For implementation verification, see:
	// - Claude engine: TestWildcardNetworkPermissionsClaudeEngine
	// - Copilot engine: TestWildcardNetworkPermissionsCopilotEngine (with AWF warning)
}

// TestAWFWildcardWarning verifies that a warning is emitted when wildcards
// are used with Copilot engine and AWF firewall
func TestAWFWildcardWarning(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	yamlContent := `---
on: push
engine: copilot
network:
  firewall: true
  allowed:
    - "github.com"
    - "*.example.com"
    - "*.trusted.com"
tools:
  edit:
  github:
---

# Test AWF Wildcard Warning

Test that wildcards trigger a warning with Copilot/AWF.`

	tmpDir, err := os.MkdirTemp("", "awf-wildcard-warning-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.md")
	err = os.WriteFile(filePath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(filePath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Verify that a warning was emitted
	if compiler.GetWarningCount() == 0 {
		t.Error("Expected a warning about wildcards with AWF, but no warnings were emitted")
	}

	// Note: The warning message is:
	// "AWF does not support wildcard syntax (found: *.example.com, *.trusted.com).
	//  AWF automatically matches subdomains - use base domain instead
	//  (e.g., 'example.com' matches 'api.example.com')."
}
