// This file provides command-line interface functionality for gh-aw.
// This file (logs_parsing_firewall.go) contains functionality for parsing
// and analyzing firewall logs from workflow runs.
//
// Key responsibilities:
//   - Locating firewall logs in various directory structures
//   - Running JavaScript firewall log parser
//   - Generating markdown summaries of firewall activity

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsParsingFirewallLog = logger.New("cli:logs_parsing_firewall")

const firewallNodeScriptPrefix = `
const fs = require('fs');
const path = require('path');

// Mock @actions/core for the parser
const core = {
	summary: {
		addRaw: function(content) {
			this._content = content;
			return this;
		},
		write: function() {
			console.log(this._content);
		},
		_content: ''
	},
	setFailed: function(message) {
		console.error('FAILED:', message);
		process.exit(1);
	},
	info: function(message) {
		// Silent in CLI mode
	}
};

// Set up environment
// We'll use a custom workflow name that points to our temp directory
process.env.GITHUB_WORKFLOW = 'temp-workflow';

// Override require to provide our mock
const originalRequire = require;
require = function(name) {
	if (name === '@actions/core') {
		return core;
	}
	return originalRequire.apply(this, arguments);
};

// Monkey-patch the main function to use our logs directory
const originalMain = function() {
  const fs = require("fs");
  const path = require("path");

  try {
    // Use our custom logs directory instead of /tmp/gh-aw/squid-logs-*
    const squidLogsDir = '%s';

    if (!fs.existsSync(squidLogsDir)) {
      core.info('No firewall logs directory found at: ' + squidLogsDir);
      return;
    }

    // Find all .log files
    const files = fs.readdirSync(squidLogsDir).filter(file => file.endsWith(".log"));

    if (files.length === 0) {
      core.info('No firewall log files found in: ' + squidLogsDir);
      return;
    }

    core.info('Found ' + files.length + ' firewall log file(s)');

    // Parse all log files and aggregate results
    let totalRequests = 0;
    let allowedRequests = 0;
    let blockedRequests = 0;
    const allowedDomains = new Set();
    const blockedDomains = new Set();
    const requestsByDomain = new Map();

    for (const file of files) {
      const filePath = path.join(squidLogsDir, file);
      core.info('Parsing firewall log: ' + file);

      const content = fs.readFileSync(filePath, "utf8");
      const lines = content.split("\n").filter(line => line.trim());

      for (const line of lines) {
        const entry = parseFirewallLogLine(line);
        if (!entry) {
          continue;
        }

        totalRequests++;
        const isAllowed = isRequestAllowed(entry.decision, entry.status);
        const domainKey =
          entry.domain !== "-" ? entry.domain : entry.destIpPort !== "-" ? entry.destIpPort : "-";

        if (isAllowed) {
          allowedRequests++;
          allowedDomains.add(domainKey);
        } else {
          blockedRequests++;
          blockedDomains.add(domainKey);
        }

        if (!requestsByDomain.has(domainKey)) {
          requestsByDomain.set(domainKey, { allowed: 0, blocked: 0 });
        }
        const domainStats = requestsByDomain.get(domainKey);
        if (isAllowed) {
          domainStats.allowed++;
        } else {
          domainStats.blocked++;
        }
      }
    }

    const summary = generateFirewallSummary({
      totalRequests,
      allowedRequests,
      blockedRequests,
      allowedDomains: Array.from(allowedDomains).sort(),
      blockedDomains: Array.from(blockedDomains).sort(),
      requestsByDomain,
    });

    core.summary.addRaw(summary).write();
    core.info("Firewall log summary generated successfully");
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
};
`

const firewallNodeScriptSuffix = `

// Replace main() call with our custom version
originalMain();
`

func loadFirewallParserScript(verbose bool) (string, bool) {
	jsScript := workflow.GetLogParserScript("parse_firewall_logs")
	if jsScript != "" {
		return jsScript, true
	}

	logsParsingFirewallLog.Print("Failed to get firewall log parser script")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Failed to get firewall log parser script"))
	}

	return "", false
}

func reportMissingFirewallLogs(runDir string, verbose bool) error {
	logsParsingFirewallLog.Print("No firewall logs found, skipping parsing")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No firewall logs found in %s, skipping firewall log parsing", filepath.Base(runDir))))
	}
	return nil
}

func createFirewallParserTempDir() (string, error) {
	tempDir, err := os.MkdirTemp("", "firewall_log_parser")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	return tempDir, nil
}

func buildFirewallNodeScript(logsDir string, jsScript string) string {
	return fmt.Sprintf(firewallNodeScriptPrefix, logsDir) +
		"\n// Execute the parser script to get helper functions\n" +
		jsScript +
		firewallNodeScriptSuffix
}

func writeFirewallNodeScript(tempDir string, nodeScript string) (string, error) {
	nodeFile := filepath.Join(tempDir, "parser.js")
	if err := os.WriteFile(nodeFile, []byte(nodeScript), constants.FilePermPublic); err != nil {
		return "", fmt.Errorf("failed to write node script: %w", err)
	}
	return nodeFile, nil
}

func executeFirewallNodeScript(tempDir string, nodeFile string) ([]byte, error) {
	// #nosec G204 -- nodeFile is an absolute path to a script written by this process to tempDir;
	// exec.Command with separate args (not shell execution) prevents shell injection.
	cmd := exec.Command("node", nodeFile)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute firewall parser script: %w\nOutput: %s", err, string(output))
	}
	return output, nil
}

func writeFirewallMarkdown(runDir string, output []byte) error {
	firewallMdPath := filepath.Join(runDir, "firewall.md")
	if err := os.WriteFile(firewallMdPath, []byte(strings.TrimSpace(string(output))), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write firewall.md: %w", err)
	}
	return nil
}

// parseFirewallLogs runs the JavaScript firewall log parser and writes markdown to firewall.md
func parseFirewallLogs(runDir string, verbose bool) error {
	logsParsingFirewallLog.Printf("Parsing firewall logs in: %s", runDir)
	jsScript, ok := loadFirewallParserScript(verbose)
	if !ok {
		return nil
	}

	logsDir, err := findFirewallLogsDir(runDir)
	if err != nil {
		return err
	}
	if logsDir == "" {
		return reportMissingFirewallLogs(runDir, verbose)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Found firewall logs in "+logsDir))
	}

	tempDir, err := createFirewallParserTempDir()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	nodeFile, err := writeFirewallNodeScript(tempDir, buildFirewallNodeScript(logsDir, jsScript))
	if err != nil {
		return err
	}
	output, err := executeFirewallNodeScript(tempDir, nodeFile)
	if err != nil {
		return err
	}
	return writeFirewallMarkdown(runDir, output)
}

func dirHasMatchingFiles(dir string, globPattern string) (bool, error) {
	if !fileutil.DirExists(dir) {
		return false, nil
	}

	files, err := filepath.Glob(filepath.Join(dir, globPattern))
	if err != nil {
		return false, fmt.Errorf("failed to find firewall log files in %s: %w", dir, err)
	}

	return len(files) > 0, nil
}

func findFirewallLogsDir(runDir string) (string, error) {
	for _, candidate := range []struct {
		path        string
		description string
	}{
		{
			path:        filepath.Join(runDir, "sandbox", "firewall", "logs", "squid-logs"),
			description: "sandbox/firewall/logs/squid-logs",
		},
		{
			path:        filepath.Join(runDir, "sandbox", "firewall", "logs"),
			description: "sandbox/firewall/logs",
		},
		{
			path:        filepath.Join(runDir, "squid-logs"),
			description: "squid-logs",
		},
		{
			path:        filepath.Join(runDir, "workflow-logs", "squid-logs"),
			description: "workflow-logs/squid-logs",
		},
	} {
		hasLogs, err := dirHasMatchingFiles(candidate.path, "*.log")
		if err != nil {
			return "", err
		}
		if !hasLogs {
			continue
		}
		logsParsingFirewallLog.Printf("Found firewall logs in %s directory", candidate.description)
		return candidate.path, nil
	}

	return "", nil
}
