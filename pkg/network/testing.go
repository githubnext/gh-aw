package network

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ProxyConfig represents the proxy configuration for network testing
type ProxyConfig struct {
	Host           string
	Port           string
	AllowedDomains []string
}

// NetworkTestResult represents the result of a network connectivity test
type NetworkTestResult struct {
	URL          string
	Allowed      bool
	Success      bool
	StatusCode   int
	Error        error
	ResponseTime time.Duration
}

// NetworkTester provides network connectivity testing functionality
type NetworkTester struct {
	proxy   *ProxyConfig
	client  *http.Client
	timeout time.Duration
}

// NewNetworkTester creates a new network tester with the specified proxy configuration
func NewNetworkTester(proxy *ProxyConfig, timeout time.Duration) *NetworkTester {
	var client *http.Client

	if proxy != nil {
		proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s", proxy.Host, proxy.Port))
		if err != nil {
			// Fallback to direct connection if proxy URL is invalid
			client = &http.Client{Timeout: timeout}
		} else {
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			client = &http.Client{
				Transport: transport,
				Timeout:   timeout,
			}
		}
	} else {
		client = &http.Client{Timeout: timeout}
	}

	return &NetworkTester{
		proxy:   proxy,
		client:  client,
		timeout: timeout,
	}
}

// TestURL tests connectivity to a specific URL
func (nt *NetworkTester) TestURL(testURL string) NetworkTestResult {
	start := time.Now()

	result := NetworkTestResult{
		URL:     testURL,
		Allowed: nt.isURLAllowed(testURL),
	}

	ctx, cancel := context.WithTimeout(context.Background(), nt.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		result.Error = err
		result.ResponseTime = time.Since(start)
		return result
	}

	resp, err := nt.client.Do(req)
	if err != nil {
		result.Error = err
		result.ResponseTime = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.Success = true
	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(start)

	return result
}

// TestMultipleURLs tests connectivity to multiple URLs
func (nt *NetworkTester) TestMultipleURLs(urls []string) []NetworkTestResult {
	results := make([]NetworkTestResult, 0, len(urls))

	for _, testURL := range urls {
		result := nt.TestURL(testURL)
		results = append(results, result)
	}

	return results
}

// isURLAllowed checks if a URL is in the allowed domains list
func (nt *NetworkTester) isURLAllowed(testURL string) bool {
	if nt.proxy == nil || len(nt.proxy.AllowedDomains) == 0 {
		return true
	}

	parsedURL, err := url.Parse(testURL)
	if err != nil {
		return false
	}

	hostname := parsedURL.Hostname()

	for _, domain := range nt.proxy.AllowedDomains {
		if hostname == domain || strings.HasSuffix(hostname, "."+domain) {
			return true
		}
	}

	return false
}

// LoadAllowedDomainsFromFile loads the allowed domains list from a file
func LoadAllowedDomainsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open domains file: %w", err)
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			domains = append(domains, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading domains file: %w", err)
	}

	return domains, nil
}

// ValidateProxyConfiguration validates the proxy configuration
func ValidateProxyConfiguration(proxy *ProxyConfig) error {
	if proxy == nil {
		return fmt.Errorf("proxy configuration is nil")
	}

	if proxy.Host == "" {
		return fmt.Errorf("proxy host is required")
	}

	if proxy.Port == "" {
		return fmt.Errorf("proxy port is required")
	}

	return nil
}

// TestProxyConnectivity tests if the proxy is accessible
func TestProxyConnectivity(proxy *ProxyConfig, timeout time.Duration) error {
	if err := ValidateProxyConfiguration(proxy); err != nil {
		return err
	}

	proxyURL := fmt.Sprintf("http://%s:%s", proxy.Host, proxy.Port)

	// Try to connect to the proxy
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(proxyURL)
	if err != nil {
		return fmt.Errorf("failed to connect to proxy %s: %w", proxyURL, err)
	}
	defer resp.Body.Close()

	return nil
}

// AnalyzeNetworkResults analyzes test results and provides insights
func AnalyzeNetworkResults(results []NetworkTestResult) NetworkAnalysis {
	analysis := NetworkAnalysis{
		TotalTests:      len(results),
		AllowedTests:    0,
		BlockedTests:    0,
		SuccessfulTests: 0,
		FailedTests:     0,
		Results:         results,
	}

	for _, result := range results {
		if result.Allowed {
			analysis.AllowedTests++
		} else {
			analysis.BlockedTests++
		}

		if result.Success {
			analysis.SuccessfulTests++
		} else {
			analysis.FailedTests++
		}
	}

	return analysis
}

// NetworkAnalysis provides summary statistics for network tests
type NetworkAnalysis struct {
	TotalTests      int
	AllowedTests    int
	BlockedTests    int
	SuccessfulTests int
	FailedTests     int
	Results         []NetworkTestResult
}

// PrintAnalysis prints a formatted analysis report
func (na *NetworkAnalysis) PrintAnalysis(writer io.Writer) {
	fmt.Fprintf(writer, "\n=== Network Permission Test Analysis ===\n")
	fmt.Fprintf(writer, "Total Tests: %d\n", na.TotalTests)
	fmt.Fprintf(writer, "Allowed Domains: %d\n", na.AllowedTests)
	fmt.Fprintf(writer, "Blocked Domains: %d\n", na.BlockedTests)
	fmt.Fprintf(writer, "Successful Connections: %d\n", na.SuccessfulTests)
	fmt.Fprintf(writer, "Failed Connections: %d\n", na.FailedTests)

	fmt.Fprintf(writer, "\n=== Detailed Results ===\n")
	for _, result := range na.Results {
		status := "❌ BLOCKED"
		if result.Success {
			if result.Allowed {
				status = "✅ ALLOWED & CONNECTED"
			} else {
				status = "⚠️  UNEXPECTED SUCCESS"
			}
		} else if result.Allowed {
			status = "⚠️  ALLOWED BUT FAILED"
		}

		fmt.Fprintf(writer, "%s - %s", status, result.URL)
		if result.StatusCode > 0 {
			fmt.Fprintf(writer, " (HTTP %d)", result.StatusCode)
		}
		if result.Error != nil {
			fmt.Fprintf(writer, " - Error: %v", result.Error)
		}
		fmt.Fprintf(writer, " [%v]\n", result.ResponseTime)
	}
}
