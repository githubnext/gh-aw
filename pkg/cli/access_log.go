package cli

import (
	"bufio"
	"fmt"
	neturl "net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var accessLogLog = logger.New("cli:access_log")

// AccessLogEntry represents a parsed squid access log entry
type AccessLogEntry struct {
	Timestamp string
	Duration  string
	ClientIP  string
	Status    string
	Size      string
	Method    string
	URL       string
	User      string
	Hierarchy string
	Type      string
}

// DomainAnalysis represents analysis of domains from access logs
type DomainAnalysis struct {
	DomainBuckets
	TotalRequests int `json:"total_requests"`
	AllowedCount  int `json:"allowed_count"`
	DeniedCount   int `json:"denied_count"`
}

// AddMetrics adds metrics from another analysis
func (d *DomainAnalysis) AddMetrics(other LogAnalysis) {
	if otherDomain, ok := other.(*DomainAnalysis); ok {
		d.TotalRequests += otherDomain.TotalRequests
		d.AllowedCount += otherDomain.AllowedCount
		d.DeniedCount += otherDomain.DeniedCount
	}
}

// parseSquidAccessLog parses a squid access log file and extracts domain information
func parseSquidAccessLog(logPath string, verbose bool) (*DomainAnalysis, error) {
	accessLogLog.Printf("Parsing squid access log: %s", logPath)

	file, err := os.Open(logPath)
	if err != nil {
		accessLogLog.Printf("Failed to open access log %s: %v", logPath, err)
		return nil, fmt.Errorf("failed to open access log: %w", err)
	}
	defer file.Close()

	analysis := &DomainAnalysis{
		DomainBuckets: DomainBuckets{
			AllowedDomains: []string{},
			DeniedDomains:  []string{},
		},
	}

	allowedDomainsSet := make(map[string]bool)
	deniedDomainsSet := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, err := parseSquidLogLine(line)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse log line: %v", err)))
			}
			continue
		}

		analysis.TotalRequests++

		// Extract domain from URL
		domain := extractDomainFromURL(entry.URL)
		if domain == "" {
			continue
		}

		// Determine if request was allowed or denied based on status code
		// Squid typically returns:
		// - 200, 206, 304: Allowed/successful
		// - 403: Forbidden (denied by ACL)
		// - 407: Proxy authentication required
		// - 502, 503: Connection/upstream errors
		statusCode := entry.Status
		isAllowed := statusCode == "TCP_HIT/200" || statusCode == "TCP_MISS/200" ||
			statusCode == "TCP_REFRESH_MODIFIED/200" || statusCode == "TCP_IMS_HIT/304" ||
			strings.Contains(statusCode, "/200") || strings.Contains(statusCode, "/206") ||
			strings.Contains(statusCode, "/304")

		if isAllowed {
			analysis.AllowedCount++
			if !allowedDomainsSet[domain] {
				allowedDomainsSet[domain] = true
				analysis.AllowedDomains = append(analysis.AllowedDomains, domain)
			}
		} else {
			analysis.DeniedCount++
			if !deniedDomainsSet[domain] {
				deniedDomainsSet[domain] = true
				analysis.DeniedDomains = append(analysis.DeniedDomains, domain)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading access log: %w", err)
	}

	// Sort domains for consistent output
	sort.Strings(analysis.AllowedDomains)
	sort.Strings(analysis.DeniedDomains)

	accessLogLog.Printf("Parsed access log: total_requests=%d, allowed=%d, denied=%d, unique_allowed_domains=%d, unique_denied_domains=%d",
		analysis.TotalRequests, analysis.AllowedCount, analysis.DeniedCount, len(analysis.AllowedDomains), len(analysis.DeniedDomains))

	return analysis, nil
}

// parseSquidLogLine parses a single squid access log line
// Squid log format: timestamp duration client status size method url user hierarchy type
func parseSquidLogLine(line string) (*AccessLogEntry, error) {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return nil, fmt.Errorf("invalid log line format: expected at least 10 fields, got %d", len(fields))
	}

	return &AccessLogEntry{
		Timestamp: fields[0],
		Duration:  fields[1],
		ClientIP:  fields[2],
		Status:    fields[3],
		Size:      fields[4],
		Method:    fields[5],
		URL:       fields[6],
		User:      fields[7],
		Hierarchy: fields[8],
		Type:      fields[9],
	}, nil
}

// extractDomainFromURL extracts the domain from a URL
func extractDomainFromURL(url string) string {
	// Handle different URL formats
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		// Parse full URL
		parsedURL, err := neturl.Parse(url)
		if err != nil {
			return ""
		}
		return parsedURL.Hostname()
	}

	// Handle CONNECT requests (domain:port format)
	if strings.Contains(url, ":") {
		parts := strings.Split(url, ":")
		if len(parts) >= 2 {
			return parts[0]
		}
	}

	// Handle direct domain
	return url
}

// analyzeAccessLogs analyzes access logs in a run directory
func analyzeAccessLogs(runDir string, verbose bool) (*DomainAnalysis, error) {
	accessLogLog.Printf("Analyzing access logs in: %s", runDir)

	// Check for access log files in access.log directory
	accessLogsDir := filepath.Join(runDir, "access.log")
	if _, err := os.Stat(accessLogsDir); err == nil {
		accessLogLog.Printf("Found access logs directory: %s", accessLogsDir)
		return analyzeMultipleAccessLogs(accessLogsDir, verbose)
	}

	// No access logs found
	accessLogLog.Printf("No access logs directory found in: %s", runDir)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No access logs found in %s", runDir)))
	}
	return nil, nil
}

// analyzeMultipleAccessLogs analyzes multiple separate access log files
func analyzeMultipleAccessLogs(accessLogsDir string, verbose bool) (*DomainAnalysis, error) {
	return aggregateLogFiles(
		accessLogsDir,
		"access-*.log",
		verbose,
		parseSquidAccessLog,
		func() *DomainAnalysis {
			return &DomainAnalysis{
				DomainBuckets: DomainBuckets{
					AllowedDomains: []string{},
					DeniedDomains:  []string{},
				},
			}
		},
	)
}

// formatDomainWithEcosystem formats a domain with its ecosystem identifier if found
