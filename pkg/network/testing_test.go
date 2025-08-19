package network

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewNetworkTester(t *testing.T) {
	tests := []struct {
		name    string
		proxy   *ProxyConfig
		timeout time.Duration
	}{
		{
			name:    "with nil proxy",
			proxy:   nil,
			timeout: 30 * time.Second,
		},
		{
			name: "with valid proxy",
			proxy: &ProxyConfig{
				Host: "localhost",
				Port: "3128",
			},
			timeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tester := NewNetworkTester(tt.proxy, tt.timeout)
			if tester == nil {
				t.Error("NewNetworkTester returned nil")
				return
			}
			if tester.timeout != tt.timeout {
				t.Errorf("Expected timeout %v, got %v", tt.timeout, tester.timeout)
			}
			if tt.proxy != nil && tester.proxy == nil {
				t.Error("Expected proxy configuration to be preserved")
			}
		})
	}
}

func TestIsURLAllowed(t *testing.T) {
	tester := &NetworkTester{
		proxy: &ProxyConfig{
			AllowedDomains: []string{"example.com", "api.trusted.com"},
		},
	}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "allowed domain",
			url:      "https://example.com/path",
			expected: true,
		},
		{
			name:     "allowed subdomain",
			url:      "https://api.example.com/data",
			expected: true,
		},
		{
			name:     "second allowed domain",
			url:      "https://api.trusted.com/endpoint",
			expected: true,
		},
		{
			name:     "blocked domain",
			url:      "https://malicious.com/evil",
			expected: false,
		},
		{
			name:     "similar but different domain",
			url:      "https://notexample.com/fake",
			expected: false,
		},
		{
			name:     "invalid url",
			url:      "not-a-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tester.isURLAllowed(tt.url)
			if result != tt.expected {
				t.Errorf("isURLAllowed(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsURLAllowedNoProxy(t *testing.T) {
	tester := &NetworkTester{
		proxy: nil,
	}

	// Without proxy configuration, all URLs should be allowed
	result := tester.isURLAllowed("https://any-domain.com")
	if !result {
		t.Error("Expected all URLs to be allowed when no proxy configuration")
	}
}

func TestIsURLAllowedEmptyDomains(t *testing.T) {
	tester := &NetworkTester{
		proxy: &ProxyConfig{
			AllowedDomains: []string{},
		},
	}

	// With empty allowed domains, all URLs should be allowed
	result := tester.isURLAllowed("https://any-domain.com")
	if !result {
		t.Error("Expected all URLs to be allowed when allowed domains is empty")
	}
}

func TestTestURL(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	tester := NewNetworkTester(nil, 5*time.Second)

	result := tester.TestURL(server.URL)

	if !result.Success {
		t.Errorf("Expected successful connection, got error: %v", result.Error)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}
	if result.URL != server.URL {
		t.Errorf("Expected URL %s, got %s", server.URL, result.URL)
	}
}

func TestTestURLTimeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use very short timeout
	tester := NewNetworkTester(nil, 100*time.Millisecond)

	result := tester.TestURL(server.URL)

	if result.Success {
		t.Error("Expected timeout error, but request succeeded")
	}
	if result.Error == nil {
		t.Error("Expected error due to timeout, but got nil")
	}
}

func TestTestMultipleURLs(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tester := NewNetworkTester(nil, 5*time.Second)

	urls := []string{server.URL, server.URL + "/path"}
	results := tester.TestMultipleURLs(urls)

	if len(results) != len(urls) {
		t.Errorf("Expected %d results, got %d", len(urls), len(results))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("Expected successful connection for URL %d, got error: %v", i, result.Error)
		}
		if result.URL != urls[i] {
			t.Errorf("Expected URL %s, got %s", urls[i], result.URL)
		}
	}
}

func TestLoadAllowedDomainsFromFile(t *testing.T) {
	// Create a temporary file with domain list
	tmpDir := t.TempDir()
	domainsFile := filepath.Join(tmpDir, "allowed_domains.txt")

	content := `# Test domains file
example.com
api.trusted.com

# Another domain with comment
github.com
`

	err := os.WriteFile(domainsFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	domains, err := LoadAllowedDomainsFromFile(domainsFile)
	if err != nil {
		t.Fatalf("LoadAllowedDomainsFromFile failed: %v", err)
	}

	expected := []string{"example.com", "api.trusted.com", "github.com"}
	if len(domains) != len(expected) {
		t.Errorf("Expected %d domains, got %d", len(expected), len(domains))
	}

	for i, domain := range expected {
		if i >= len(domains) || domains[i] != domain {
			t.Errorf("Expected domain %s at position %d, got %v", domain, i, domains)
		}
	}
}

func TestLoadAllowedDomainsFromFileNotExists(t *testing.T) {
	_, err := LoadAllowedDomainsFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestValidateProxyConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		proxy       *ProxyConfig
		expectError bool
	}{
		{
			name:        "nil proxy",
			proxy:       nil,
			expectError: true,
		},
		{
			name: "empty host",
			proxy: &ProxyConfig{
				Host: "",
				Port: "3128",
			},
			expectError: true,
		},
		{
			name: "empty port",
			proxy: &ProxyConfig{
				Host: "localhost",
				Port: "",
			},
			expectError: true,
		},
		{
			name: "valid configuration",
			proxy: &ProxyConfig{
				Host: "localhost",
				Port: "3128",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProxyConfiguration(tt.proxy)
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestAnalyzeNetworkResults(t *testing.T) {
	results := []NetworkTestResult{
		{
			URL:     "https://example.com",
			Allowed: true,
			Success: true,
		},
		{
			URL:     "https://blocked.com",
			Allowed: false,
			Success: false,
		},
		{
			URL:     "https://allowed-but-failed.com",
			Allowed: true,
			Success: false,
		},
		{
			URL:     "https://unexpected-success.com",
			Allowed: false,
			Success: true,
		},
	}

	analysis := AnalyzeNetworkResults(results)

	if analysis.TotalTests != 4 {
		t.Errorf("Expected 4 total tests, got %d", analysis.TotalTests)
	}
	if analysis.AllowedTests != 2 {
		t.Errorf("Expected 2 allowed tests, got %d", analysis.AllowedTests)
	}
	if analysis.BlockedTests != 2 {
		t.Errorf("Expected 2 blocked tests, got %d", analysis.BlockedTests)
	}
	if analysis.SuccessfulTests != 2 {
		t.Errorf("Expected 2 successful tests, got %d", analysis.SuccessfulTests)
	}
	if analysis.FailedTests != 2 {
		t.Errorf("Expected 2 failed tests, got %d", analysis.FailedTests)
	}
}
