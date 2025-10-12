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
	"github.com/githubnext/gh-aw/pkg/workflow"
)

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
	AllowedDomains []string
	DeniedDomains  []string
	TotalRequests  int
	AllowedCount   int
	DeniedCount    int
}

// parseSquidAccessLog parses a squid access log file and extracts domain information
func parseSquidAccessLog(logPath string, verbose bool) (*DomainAnalysis, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open access log: %w", err)
	}
	defer file.Close()

	analysis := &DomainAnalysis{
		AllowedDomains: []string{},
		DeniedDomains:  []string{},
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
	// Check for access log files in access.log directory
	accessLogsDir := filepath.Join(runDir, "access.log")
	if _, err := os.Stat(accessLogsDir); err == nil {
		return analyzeMultipleAccessLogs(accessLogsDir, verbose)
	}

	// No access logs found
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No access logs found in %s", runDir)))
	}
	return nil, nil
}

// analyzeMultipleAccessLogs analyzes multiple separate access log files
func analyzeMultipleAccessLogs(accessLogsDir string, verbose bool) (*DomainAnalysis, error) {
	files, err := filepath.Glob(filepath.Join(accessLogsDir, "access-*.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to find access log files: %w", err)
	}

	if len(files) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No access log files found in %s", accessLogsDir)))
		}
		return nil, nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Analyzing %d access log files from %s", len(files), accessLogsDir)))
	}

	// Aggregate analysis from all files
	aggregatedAnalysis := &DomainAnalysis{
		AllowedDomains: []string{},
		DeniedDomains:  []string{},
	}

	allAllowedDomains := make(map[string]bool)
	allDeniedDomains := make(map[string]bool)

	for _, file := range files {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Parsing %s", filepath.Base(file))))
		}

		analysis, err := parseSquidAccessLog(file, verbose)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		// Aggregate the metrics
		aggregatedAnalysis.TotalRequests += analysis.TotalRequests
		aggregatedAnalysis.AllowedCount += analysis.AllowedCount
		aggregatedAnalysis.DeniedCount += analysis.DeniedCount

		// Collect unique domains
		for _, domain := range analysis.AllowedDomains {
			allAllowedDomains[domain] = true
		}
		for _, domain := range analysis.DeniedDomains {
			allDeniedDomains[domain] = true
		}
	}

	// Convert maps to sorted slices
	for domain := range allAllowedDomains {
		aggregatedAnalysis.AllowedDomains = append(aggregatedAnalysis.AllowedDomains, domain)
	}
	for domain := range allDeniedDomains {
		aggregatedAnalysis.DeniedDomains = append(aggregatedAnalysis.DeniedDomains, domain)
	}

	sort.Strings(aggregatedAnalysis.AllowedDomains)
	sort.Strings(aggregatedAnalysis.DeniedDomains)

	return aggregatedAnalysis, nil
}

// formatDomainWithEcosystem formats a domain with its ecosystem identifier if found
// formatDomainWithEcosystem formats a domain with its ecosystem information
// Deprecated: This function is kept for backward compatibility but is no longer used.
//
//nolint:unused // Kept for potential future use or backward compatibility
func formatDomainWithEcosystem(domain string) string {
	ecosystem := workflow.GetDomainEcosystem(domain)
	if ecosystem != "" {
		return fmt.Sprintf("%s (%s)", domain, ecosystem)
	}
	return domain
}

// displayAccessLogAnalysis displays analysis of access logs from all runs with improved formatting
// displayAccessLogAnalysis displays access log analysis across all runs
// Deprecated: This function is kept for backward compatibility but is no longer used.
// Use buildAccessLogSummary and displayAccessLogFromData instead.
//
//nolint:unused // Kept for potential future use or backward compatibility
func displayAccessLogAnalysis(processedRuns []ProcessedRun, verbose bool) {
	if len(processedRuns) == 0 {
		return
	}

	// Collect all access analyses
	var analyses []*DomainAnalysis
	runsWithAccess := 0
	for _, pr := range processedRuns {
		if pr.AccessAnalysis != nil {
			analyses = append(analyses, pr.AccessAnalysis)
			runsWithAccess++
		}
	}

	if len(analyses) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No access logs found"))
		return
	}

	// Aggregate statistics
	totalRequests := 0
	totalAllowed := 0
	totalDenied := 0
	allAllowedDomains := make(map[string]bool)
	allDeniedDomains := make(map[string]bool)

	for _, analysis := range analyses {
		totalRequests += analysis.TotalRequests
		totalAllowed += analysis.AllowedCount
		totalDenied += analysis.DeniedCount

		for _, domain := range analysis.AllowedDomains {
			allAllowedDomains[domain] = true
		}
		for _, domain := range analysis.DeniedDomains {
			allDeniedDomains[domain] = true
		}
	}

	fmt.Fprintln(os.Stderr, "")

	// Display allowed domains with better formatting
	if len(allAllowedDomains) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✅ Allowed Domains (%d):", len(allAllowedDomains))))
		allowedList := make([]string, 0, len(allAllowedDomains))
		for domain := range allAllowedDomains {
			allowedList = append(allowedList, domain)
		}
		sort.Strings(allowedList)
		for _, domain := range allowedList {
			fmt.Fprintln(os.Stderr, console.FormatListItem(formatDomainWithEcosystem(domain)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Display denied domains with better formatting
	if len(allDeniedDomains) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("❌ Denied Domains (%d):", len(allDeniedDomains))))
		deniedList := make([]string, 0, len(allDeniedDomains))
		for domain := range allDeniedDomains {
			deniedList = append(deniedList, domain)
		}
		sort.Strings(deniedList)
		for _, domain := range deniedList {
			fmt.Fprintln(os.Stderr, console.FormatListItem(formatDomainWithEcosystem(domain)))
		}
		fmt.Fprintln(os.Stderr, "")
	}

	if verbose && len(analyses) > 1 {
		// Show per-run breakdown with improved formatting
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("📋 Per-run breakdown:"))
		for _, pr := range processedRuns {
			if pr.AccessAnalysis != nil {
				analysis := pr.AccessAnalysis
				fmt.Fprintf(os.Stderr, "   %s Run %d: %d requests (%d allowed, %d denied)\n",
					console.FormatListItem(""),
					pr.Run.DatabaseID, analysis.TotalRequests, analysis.AllowedCount, analysis.DeniedCount)
			}
		}
		fmt.Fprintln(os.Stderr, "")
	}
}
