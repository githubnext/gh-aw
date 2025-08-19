package network

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/network"
)

// TestNetworkIntegration tests the complete network testing workflow
func TestNetworkIntegration(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a test domains file
	domainsFile := filepath.Join(tmpDir, "allowed_domains.txt")
	domainsContent := `example.com
httpbin.org`

	err := os.WriteFile(domainsFile, []byte(domainsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create domains file: %v", err)
	}

	// Load domains
	domains, err := network.LoadAllowedDomainsFromFile(domainsFile)
	if err != nil {
		t.Fatalf("Failed to load domains: %v", err)
	}

	// Create proxy config
	proxyConfig := &network.ProxyConfig{
		Host:           "localhost",
		Port:           "3128",
		AllowedDomains: domains,
	}

	// Create network tester
	tester := network.NewNetworkTester(proxyConfig, 10*time.Second)

	// Test URLs - mix of allowed and blocked
	testURLs := []string{
		"https://example.com",        // Should be allowed
		"https://httpbin.org/json",   // Should be allowed
		"https://github.com",         // Should be blocked
		"https://malicious-site.com", // Should be blocked
	}

	t.Logf("Testing %d URLs with proxy configuration", len(testURLs))

	// Run tests
	results := tester.TestMultipleURLs(testURLs)

	// Analyze results
	analysis := network.AnalyzeNetworkResults(results)

	// Verify we got results for all URLs
	if analysis.TotalTests != len(testURLs) {
		t.Errorf("Expected %d test results, got %d", len(testURLs), analysis.TotalTests)
	}

	// Check that allowed domains are properly identified
	expectedAllowed := 2 // example.com and httpbin.org
	if analysis.AllowedTests != expectedAllowed {
		t.Errorf("Expected %d allowed tests, got %d", expectedAllowed, analysis.AllowedTests)
	}

	// Check that blocked domains are properly identified
	expectedBlocked := 2 // github.com and malicious-site.com
	if analysis.BlockedTests != expectedBlocked {
		t.Errorf("Expected %d blocked tests, got %d", expectedBlocked, analysis.BlockedTests)
	}

	// Log results for debugging
	for i, result := range results {
		t.Logf("Test %d: URL=%s, Allowed=%v, Success=%v, StatusCode=%d, Error=%v",
			i+1, result.URL, result.Allowed, result.Success, result.StatusCode, result.Error)
	}

	t.Logf("Analysis: Total=%d, Allowed=%d, Blocked=%d, Successful=%d, Failed=%d",
		analysis.TotalTests, analysis.AllowedTests, analysis.BlockedTests,
		analysis.SuccessfulTests, analysis.FailedTests)
}

// TestNetworkConfigurationValidation tests the complete validation workflow
func TestNetworkConfigurationValidation(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: Valid domains file
	t.Run("ValidDomainsFile", func(t *testing.T) {
		domainsFile := filepath.Join(tmpDir, "valid_domains.txt")
		content := `# Valid domains file
example.com
api.trusted.com
github.com
`
		err := os.WriteFile(domainsFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		domains, err := network.LoadAllowedDomainsFromFile(domainsFile)
		if err != nil {
			t.Errorf("Validation failed for valid domains file: %v", err)
		}

		expectedCount := 3
		if len(domains) != expectedCount {
			t.Errorf("Expected %d domains, got %d", expectedCount, len(domains))
		}
	})

	// Test 2: Empty domains file
	t.Run("EmptyDomainsFile", func(t *testing.T) {
		domainsFile := filepath.Join(tmpDir, "empty_domains.txt")
		content := `# Empty domains file
# All lines are comments

`
		err := os.WriteFile(domainsFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		domains, err := network.LoadAllowedDomainsFromFile(domainsFile)
		if err != nil {
			t.Errorf("Validation failed for empty domains file: %v", err)
		}

		if len(domains) != 0 {
			t.Errorf("Expected 0 domains for empty file, got %d", len(domains))
		}
	})

	// Test 3: Non-existent file
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := network.LoadAllowedDomainsFromFile("/nonexistent/file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})
}

// TestProxyConfigurationValidation tests proxy configuration validation
func TestProxyConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *network.ProxyConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil configuration",
			config:      nil,
			expectError: true,
			errorMsg:    "proxy configuration is nil",
		},
		{
			name: "missing host",
			config: &network.ProxyConfig{
				Host: "",
				Port: "3128",
			},
			expectError: true,
			errorMsg:    "proxy host is required",
		},
		{
			name: "missing port",
			config: &network.ProxyConfig{
				Host: "localhost",
				Port: "",
			},
			expectError: true,
			errorMsg:    "proxy port is required",
		},
		{
			name: "valid configuration",
			config: &network.ProxyConfig{
				Host:           "localhost",
				Port:           "3128",
				AllowedDomains: []string{"example.com"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := network.ValidateProxyConfiguration(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestEndToEndNetworkWorkflow tests the complete network testing workflow
func TestEndToEndNetworkWorkflow(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create temporary configuration
	tmpDir := t.TempDir()
	domainsFile := filepath.Join(tmpDir, "test_domains.txt")

	// Write test configuration
	content := `# Test network configuration
example.com
httpbin.org`

	err := os.WriteFile(domainsFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test configuration: %v", err)
	}

	// Step 1: Validate configuration
	domains, err := network.LoadAllowedDomainsFromFile(domainsFile)
	if err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	if len(domains) != 2 {
		t.Fatalf("Expected 2 domains, got %d", len(domains))
	}

	// Step 2: Create proxy configuration
	proxyConfig := &network.ProxyConfig{
		Host:           "localhost",
		Port:           "3128",
		AllowedDomains: domains,
	}

	err = network.ValidateProxyConfiguration(proxyConfig)
	if err != nil {
		t.Fatalf("Proxy configuration validation failed: %v", err)
	}

	// Step 3: Create network tester
	tester := network.NewNetworkTester(proxyConfig, 15*time.Second)

	// Step 4: Run comprehensive tests
	testURLs := []string{
		"https://example.com",     // Allowed
		"https://httpbin.org/get", // Allowed
		"https://github.com",      // Blocked
		"https://google.com",      // Blocked
	}

	results := tester.TestMultipleURLs(testURLs)

	// Step 5: Analyze results
	analysis := network.AnalyzeNetworkResults(results)

	// Verify analysis
	if analysis.TotalTests != 4 {
		t.Errorf("Expected 4 total tests, got %d", analysis.TotalTests)
	}

	if analysis.AllowedTests != 2 {
		t.Errorf("Expected 2 allowed tests, got %d", analysis.AllowedTests)
	}

	if analysis.BlockedTests != 2 {
		t.Errorf("Expected 2 blocked tests, got %d", analysis.BlockedTests)
	}

	// Check individual results
	for _, result := range results {
		switch result.URL {
		case "https://example.com", "https://httpbin.org/get":
			if !result.Allowed {
				t.Errorf("URL %s should be allowed", result.URL)
			}
		case "https://github.com", "https://google.com":
			if result.Allowed {
				t.Errorf("URL %s should be blocked", result.URL)
			}
		}
	}

	t.Logf("End-to-end test completed successfully")
	t.Logf("Results: %d total, %d allowed, %d blocked, %d successful, %d failed",
		analysis.TotalTests, analysis.AllowedTests, analysis.BlockedTests,
		analysis.SuccessfulTests, analysis.FailedTests)
}
