package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var redactedDomainsLog = logger.New("cli:redacted_domains")

// RedactedDomainsAnalysis represents analysis of domains that were redacted during URL sanitization.
// The redacted-urls.log file is created during content sanitization when URLs from untrusted domains
// are encountered. This helps track which domains agents attempted to access but were blocked.
type RedactedDomainsAnalysis struct {
	// TotalDomains is the total number of unique domains found in the redacted log
	TotalDomains int `json:"total_domains" console:"header:Total Domains"`
	// Domains is a sorted list of unique domain names that were redacted
	Domains []string `json:"domains" console:"title:Redacted Domains,omitempty"`
}

// RedactedDomainsLogSummary contains aggregated redacted domains data across all runs
type RedactedDomainsLogSummary struct {
	TotalDomains int                                 `json:"total_domains" console:"header:Total Domains"`
	Domains      []string                            `json:"domains" console:"title:Redacted Domains,omitempty"`
	ByWorkflow   map[string]*RedactedDomainsAnalysis `json:"by_workflow,omitempty" console:"-"`
}

// parseRedactedDomainsLog parses the redacted-urls.log file and returns analysis.
// The file contains one domain per line.
func parseRedactedDomainsLog(logPath string, verbose bool) (*RedactedDomainsAnalysis, error) {
	redactedDomainsLog.Printf("Parsing redacted domains log: %s", logPath)

	file, err := os.Open(logPath)
	if err != nil {
		redactedDomainsLog.Printf("Failed to open redacted domains log: %v", err)
		return nil, fmt.Errorf("failed to open redacted domains log: %w", err)
	}
	defer file.Close()

	domainsSet := make(map[string]struct {
	})

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		domainsSet[line] = struct {
		}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading redacted domains log: %w", err)
	}

	// Convert set to sorted slice
	domains := sliceutil.SortedKeys(domainsSet)

	analysis := &RedactedDomainsAnalysis{
		TotalDomains: len(domains),
		Domains:      domains,
	}

	if redactedDomainsLog.Enabled() {
		redactedDomainsLog.Printf("Redacted domains log parsed: total=%d domains", len(domains))
	}

	return analysis, nil
}

// analyzeRedactedDomains analyzes redacted domains logs in a run directory.
// The redacted-urls.log file is typically stored in the agent_outputs artifact directory.
func analyzeRedactedDomains(runDir string, verbose bool) (*RedactedDomainsAnalysis, error) {
	redactedDomainsLog.Printf("Analyzing redacted domains in: %s", runDir)

	if path, message := analyzeRedactedDomainsKnownPath(runDir); path != "" {
		redactedDomainsLog.Printf("Found redacted-urls.log at %s: %s", message, path)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(message))
		}
		return parseRedactedDomainsLog(path, verbose)
	}

	foundPath := analyzeRedactedDomainsWalk(runDir)
	if foundPath != "" {
		redactedDomainsLog.Printf("Found redacted-urls.log via recursive search: %s", foundPath)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found redacted-urls.log at "+foundPath))
		}
		return parseRedactedDomainsLog(foundPath, verbose)
	}

	redactedDomainsLog.Print("No redacted-urls.log found")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No redacted-urls.log found in "+runDir))
	}
	return nil, nil
}

func analyzeRedactedDomainsKnownPath(runDir string) (string, string) {
	candidates := []struct {
		path    string
		message string
	}{
		{filepath.Join(runDir, "redacted-urls.log"), "Found redacted-urls.log in run directory"},
		{filepath.Join(runDir, "agent_outputs", "redacted-urls.log"), "Found redacted-urls.log in agent_outputs directory"},
		{filepath.Join(runDir, "agent_outputs", "tmp", "gh-aw", "redacted-urls.log"), "Found redacted-urls.log at full artifact path"},
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate.path); err == nil {
			return candidate.path, candidate.message
		}
	}
	return "", ""
}

func analyzeRedactedDomainsWalk(runDir string) string {
	var foundPath string
	if walkErr := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			redactedDomainsLog.Printf("walk error at %s: %v", path, err)
			return nil
		}
		if info == nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "redacted-urls.log" {
			foundPath = path
			return errWalkStop
		}
		return nil
	}); walkErr != nil && !errors.Is(walkErr, errWalkStop) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("filesystem error walking %s: %v", runDir, walkErr)))
	}
	return foundPath
}
