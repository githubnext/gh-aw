package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var logAggregationLog = logger.New("cli:log_aggregation")

// LogAnalysis is an interface that both DomainAnalysis and FirewallAnalysis implement
type LogAnalysis interface {
	// GetAllowedDomains returns the list of allowed domains
	GetAllowedDomains() []string
	// GetDeniedDomains returns the list of denied domains
	GetDeniedDomains() []string
	// SetAllowedDomains sets the list of allowed domains
	SetAllowedDomains(domains []string)
	// SetDeniedDomains sets the list of denied domains
	SetDeniedDomains(domains []string)
	// AddMetrics adds metrics from another analysis
	AddMetrics(other LogAnalysis)
}

// LogParser is a function type that parses a single log file
type LogParser[T LogAnalysis] func(logPath string, verbose bool) (T, error)

// aggregateLogFiles is a generic helper that aggregates multiple log files
// It handles file discovery, parsing, domain deduplication, and sorting
func aggregateLogFiles[T LogAnalysis](
	logsDir string,
	globPattern string,
	verbose bool,
	parser LogParser[T],
	newAnalysis func() T,
) (T, error) {
	logAggregationLog.Printf("Aggregating log files: dir=%s, pattern=%s", logsDir, globPattern)
	var zero T

	// Find log files matching the pattern
	files, err := filepath.Glob(filepath.Join(logsDir, globPattern))
	if err != nil {
		logAggregationLog.Printf("Failed to find log files with pattern '%s': %v", globPattern, err)
		return zero, fmt.Errorf("failed to find log files: %w", err)
	}

	if len(files) == 0 {
		logAggregationLog.Printf("No log files found matching pattern '%s' in %s", globPattern, logsDir)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No log files found in %s", logsDir)))
		}
		return zero, nil
	}

	logAggregationLog.Printf("Found %d log files to aggregate", len(files))

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Analyzing %d log files from %s", len(files), logsDir)))
	}

	// Initialize aggregated analysis
	aggregated := newAnalysis()

	// Track unique domains across all files
	allAllowedDomains := make(map[string]bool)
	allDeniedDomains := make(map[string]bool)

	// Parse each file and aggregate results
	for _, file := range files {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Parsing %s", filepath.Base(file))))
		}

		analysis, err := parser(file, verbose)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse %s: %v", filepath.Base(file), err)))
			}
			continue
		}

		// Aggregate metrics
		aggregated.AddMetrics(analysis)

		// Collect unique domains
		for _, domain := range analysis.GetAllowedDomains() {
			allAllowedDomains[domain] = true
		}
		for _, domain := range analysis.GetDeniedDomains() {
			allDeniedDomains[domain] = true
		}
	}

	// Convert maps to sorted slices
	allowedDomains := make([]string, 0, len(allAllowedDomains))
	for domain := range allAllowedDomains {
		allowedDomains = append(allowedDomains, domain)
	}
	sort.Strings(allowedDomains)

	deniedDomains := make([]string, 0, len(allDeniedDomains))
	for domain := range allDeniedDomains {
		deniedDomains = append(deniedDomains, domain)
	}
	sort.Strings(deniedDomains)

	// Set the sorted domain lists
	aggregated.SetAllowedDomains(allowedDomains)
	aggregated.SetDeniedDomains(deniedDomains)

	logAggregationLog.Printf("Aggregation complete: processed %d files, found %d allowed and %d denied domains",
		len(files), len(allowedDomains), len(deniedDomains))

	return aggregated, nil
}
