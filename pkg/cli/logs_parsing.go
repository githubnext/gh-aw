// Package cli provides command-line interface functionality for gh-aw.
// This file (logs_parsing.go) contains functions for parsing and analyzing
// workflow logs from various AI engines.
//
// Key responsibilities:
//   - Parsing engine-specific log formats (Claude, Copilot, Codex, Custom)
//   - Extracting engine configuration from aw_info.json
//   - Locating agent log files and output artifacts
//   - Parsing firewall logs and generating summaries
//   - Running JavaScript parsers to generate markdown reports
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/cli/fileutil"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var logsParsingLog = logger.New("cli:logs_parsing")

// parseAwInfo reads and parses aw_info.json file, returning the parsed data
// Handles cases where aw_info.json is a file or a directory containing the actual file
func parseAwInfo(infoFilePath string, verbose bool) (*AwInfo, error) {
	// Sanitize the path to prevent path traversal attacks
	cleanPath := filepath.Clean(infoFilePath)
	logsParsingLog.Printf("Parsing aw_info.json from: %s", cleanPath)
	var data []byte
	var err error

	// Check if the path exists and determine if it's a file or directory
	stat, statErr := os.Stat(cleanPath)
	if statErr != nil {
		logsParsingLog.Printf("Failed to stat aw_info.json: %v", statErr)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to stat aw_info.json: %v", statErr)))
		}
		return nil, statErr
	}

	if stat.IsDir() {
		// It's a directory - look for nested aw_info.json
		nestedPath := filepath.Join(cleanPath, "aw_info.json")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("aw_info.json is a directory, trying nested file: %s", nestedPath)))
		}
		data, err = os.ReadFile(nestedPath)
	} else {
		// It's a regular file
		data, err = os.ReadFile(cleanPath)
	}

	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read aw_info.json: %v", err)))
		}
		return nil, err
	}

	var info AwInfo
	if err := json.Unmarshal(data, &info); err != nil {
		logsParsingLog.Printf("Failed to unmarshal aw_info.json: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse aw_info.json: %v", err)))
		}
		return nil, err
	}

	logsParsingLog.Printf("Successfully parsed aw_info.json with engine_id: %s", info.EngineID)
	return &info, nil
}

// extractEngineFromAwInfo reads aw_info.json and returns the appropriate engine
// Handles cases where aw_info.json is a file or a directory containing the actual file
func extractEngineFromAwInfo(infoFilePath string, verbose bool) workflow.CodingAgentEngine {
	logsParsingLog.Printf("Extracting engine from aw_info.json: %s", infoFilePath)
	info, err := parseAwInfo(infoFilePath, verbose)
	if err != nil {
		return nil
	}

	if info.EngineID == "" {
		logsParsingLog.Print("No engine_id found in aw_info.json")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No engine_id found in aw_info.json"))
		}
		return nil
	}

	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine(info.EngineID)
	if err != nil {
		logsParsingLog.Printf("Unknown engine: %s", info.EngineID)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Unknown engine in aw_info.json: %s", info.EngineID)))
		}
		return nil
	}

	logsParsingLog.Printf("Successfully extracted engine: %s", engine.GetID())
	return engine
}

// parseLogFileWithEngine parses a log file using a specific engine or falls back to auto-detection
func parseLogFileWithEngine(filePath string, detectedEngine workflow.CodingAgentEngine, isGitHubCopilotAgent bool, verbose bool) (LogMetrics, error) {
	logsParsingLog.Printf("Parsing log file: %s, isGitHubCopilotAgent=%v", filePath, isGitHubCopilotAgent)
	// Read the entire log file at once to avoid JSON parsing issues from chunked reading
	content, err := os.ReadFile(filePath)
	if err != nil {
		logsParsingLog.Printf("Failed to read log file: %v", err)
		return LogMetrics{}, fmt.Errorf("error reading log file: %w", err)
	}

	logContent := string(content)
	logsParsingLog.Printf("Read %d bytes from log file", len(logContent))

	// If this is a GitHub Copilot agent run, use the specialized parser
	if isGitHubCopilotAgent {
		logsParsingLog.Print("Using GitHub Copilot agent parser")
		return ParseCopilotAgentLogMetrics(logContent, verbose), nil
	}

	// If we have a detected engine from aw_info.json, use it directly
	if detectedEngine != nil {
		logsParsingLog.Printf("Using detected engine: %s", detectedEngine.GetID())
		return detectedEngine.ParseLogMetrics(logContent, verbose), nil
	}

	// No aw_info.json metadata available - use fallback parser with common error patterns
	logsParsingLog.Print("No engine detected, using fallback parser with common error patterns")
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No aw_info.json found, using fallback parser"))
	}

	// Use empty metrics for fallback case
	var metrics LogMetrics

	return metrics, nil
}

// findAgentOutputFile searches for a file named agent_output.json within the logDir tree.
// Returns the first path found (depth-first) and a boolean indicating success.
func findAgentOutputFile(logDir string) (string, bool) {
	var foundPath string
	_ = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), constants.AgentOutputArtifactName) {
			foundPath = path
			return errors.New("stop") // sentinel to stop walking early
		}
		return nil
	})
	if foundPath == "" {
		return "", false
	}
	return foundPath, true
}

// findAgentLogFile searches for agent logs within the logDir.
// It uses engine.GetLogFileForParsing() to determine which log file to use:
//   - If GetLogFileForParsing() returns a non-empty value that doesn't point to agent-stdio.log,
//     look for files in the "agent_output" artifact directory (before flattening)
//     or in the flattened location (after flattening)
//   - Otherwise, look for the "agent-stdio.log" artifact file
//
// Returns the first path found and a boolean indicating success.
func findAgentLogFile(logDir string, engine workflow.CodingAgentEngine) (string, bool) {
	// Use GetLogFileForParsing to determine which log file to use
	logFileForParsing := engine.GetLogFileForParsing()

	// If the engine specifies a log file that isn't the default agent-stdio.log,
	// look in the agent_output artifact directory or flattened location
	if logFileForParsing != "" && logFileForParsing != defaultAgentStdioLogPath {
		// Check for agent_output directory (artifact, before flattening)
		agentOutputDir := filepath.Join(logDir, "agent_output")
		if fileutil.DirExists(agentOutputDir) {
			// Find the first file in this directory
			var foundFile string
			_ = filepath.Walk(agentOutputDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info == nil {
					return nil
				}
				if !info.IsDir() && foundFile == "" {
					foundFile = path
					return errors.New("stop") // sentinel to stop walking early
				}
				return nil
			})
			if foundFile != "" {
				return foundFile, true
			}
		}

		// Check for flattened location (after flattening)
		// The engine's log path is absolute (e.g., /tmp/gh-aw/sandbox/agent/logs/)
		// After flattening, it's at logDir/sandbox/agent/logs/
		// Strip /tmp/gh-aw/ prefix to get the relative path
		const tmpGhAwPrefix = "/tmp/gh-aw/"
		if strings.HasPrefix(logFileForParsing, tmpGhAwPrefix) {
			relPath := strings.TrimPrefix(logFileForParsing, tmpGhAwPrefix)
			flattenedDir := filepath.Join(logDir, relPath)
			logsParsingLog.Printf("Checking flattened location for logs: %s", flattenedDir)
			if fileutil.DirExists(flattenedDir) {
				// Find the first .log file in this directory
				var foundFile string
				_ = filepath.Walk(flattenedDir, func(path string, info os.FileInfo, err error) error {
					if err != nil || info == nil {
						return nil
					}
					if !info.IsDir() && strings.HasSuffix(info.Name(), ".log") && foundFile == "" {
						foundFile = path
						logsParsingLog.Printf("Found session log file: %s", path)
						return errors.New("stop") // sentinel to stop walking early
					}
					return nil
				})
				if foundFile != "" {
					return foundFile, true
				}
			}
		}

		// Fallback: search recursively in logDir for session*.log or process*.log files
		// This handles cases where the artifact structure is different than expected
		// Note: Copilot changed from session-*.log to process-*.log naming convention
		logsParsingLog.Printf("Searching recursively in %s for session*.log or process*.log files", logDir)
		var foundFile string
		_ = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			// Look for session*.log or process*.log files
			fileName := info.Name()
			if !info.IsDir() && (strings.HasPrefix(fileName, "session") || strings.HasPrefix(fileName, "process")) && strings.HasSuffix(fileName, ".log") && foundFile == "" {
				foundFile = path
				logsParsingLog.Printf("Found Copilot log file via recursive search: %s", path)
				return errors.New("stop") // sentinel to stop walking early
			}
			return nil
		})
		if foundFile != "" {
			return foundFile, true
		}
	}

	// Default to agent-stdio.log
	agentStdioLog := filepath.Join(logDir, "agent-stdio.log")
	if fileutil.FileExists(agentStdioLog) {
		return agentStdioLog, true
	}

	// Also check for nested agent-stdio.log in case it's in a subdirectory
	var foundPath string
	_ = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "agent-stdio.log" {
			foundPath = path
			return errors.New("stop") // sentinel to stop walking early
		}
		return nil
	})
	if foundPath != "" {
		return foundPath, true
	}

	return "", false
}

// parseAgentLog parses agent logs and generates a markdown summary
func parseAgentLog(runDir string, engine workflow.CodingAgentEngine, verbose bool) error {
	logsParsingLog.Printf("Parsing agent logs in: %s", runDir)
	// Determine which parser script to use based on the engine
	if engine == nil {
		logsParsingLog.Print("No engine detected, skipping log parsing")
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No engine detected in %s, skipping log parsing", filepath.Base(runDir))))
		return nil
	}

	// Find the agent log file - use engine.GetLogFileForParsing() to determine location
	agentLogPath, found := findAgentLogFile(runDir, engine)
	if !found {
		logsParsingLog.Print("No agent log file found")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No agent logs found in %s, skipping log parsing", filepath.Base(runDir))))
		return nil
	}

	logsParsingLog.Printf("Found agent log file: %s", agentLogPath)

	parserScriptName := engine.GetLogParserScriptId()
	if parserScriptName == "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No log parser available for engine %s in %s, skipping", engine.GetID(), filepath.Base(runDir))))
		return nil
	}

	jsScript := workflow.GetLogParserScript(parserScriptName)
	if jsScript == "" {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to get log parser script %s", parserScriptName)))
		}
		return nil
	}

	// Read the log content
	logContent, err := os.ReadFile(agentLogPath)
	if err != nil {
		return fmt.Errorf("failed to read agent log file: %w", err)
	}

	// Create a temporary directory for running the parser
	tempDir, err := os.MkdirTemp("", "log_parser")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write the log content to a temporary file
	logFile := filepath.Join(tempDir, "agent.log")
	if err := os.WriteFile(logFile, logContent, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	// Write the bootstrap helper to the temp directory
	bootstrapScript := workflow.GetLogParserBootstrap()
	if bootstrapScript != "" {
		bootstrapFile := filepath.Join(tempDir, "log_parser_bootstrap.cjs")
		if err := os.WriteFile(bootstrapFile, []byte(bootstrapScript), 0644); err != nil {
			return fmt.Errorf("failed to write bootstrap file: %w", err)
		}
	}

	// Write the shared helper to the temp directory
	sharedScript := workflow.GetJavaScriptSources()["log_parser_shared.cjs"]
	if sharedScript != "" {
		sharedFile := filepath.Join(tempDir, "log_parser_shared.cjs")
		if err := os.WriteFile(sharedFile, []byte(sharedScript), 0644); err != nil {
			return fmt.Errorf("failed to write shared helper file: %w", err)
		}
	}

	// Create a Node.js script that mimics the GitHub Actions environment
	nodeScript := fmt.Sprintf(`
const fs = require('fs');

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

// Set global core for the parser scripts
global.core = core;

// Set up environment
process.env.GH_AW_AGENT_OUTPUT = '%s';

// Execute the parser script
%s
`, logFile, jsScript)

	// Write the Node.js script
	nodeFile := filepath.Join(tempDir, "parser.js")
	if err := os.WriteFile(nodeFile, []byte(nodeScript), 0644); err != nil {
		return fmt.Errorf("failed to write node script: %w", err)
	}

	// Execute the Node.js script
	cmd := exec.Command("node", "parser.js")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute parser script: %w\nOutput: %s", err, string(output))
	}

	// Write the output to log.md in the run directory
	logMdPath := filepath.Join(runDir, "log.md")
	if err := os.WriteFile(logMdPath, []byte(strings.TrimSpace(string(output))), 0644); err != nil {
		return fmt.Errorf("failed to write log.md: %w", err)
	}

	return nil
}

// parseFirewallLogs runs the JavaScript firewall log parser and writes markdown to firewall.md
func parseFirewallLogs(runDir string, verbose bool) error {
	logsParsingLog.Printf("Parsing firewall logs in: %s", runDir)
	// Get the firewall log parser script
	jsScript := workflow.GetLogParserScript("parse_firewall_logs")
	if jsScript == "" {
		logsParsingLog.Print("Failed to get firewall log parser script")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Failed to get firewall log parser script"))
		}
		return nil
	}

	// Check if squid logs directory exists in the run directory
	// The logs could be in workflow-logs subdirectory or directly in the run directory
	squidLogsDir := filepath.Join(runDir, "squid-logs")

	// Also check for squid logs in workflow-logs directory
	workflowLogsSquidDir := filepath.Join(runDir, "workflow-logs", "squid-logs")

	// Determine which directory to use
	var logsDir string
	if fileutil.DirExists(squidLogsDir) {
		logsDir = squidLogsDir
		logsParsingLog.Printf("Found firewall logs in squid-logs directory")
	} else if fileutil.DirExists(workflowLogsSquidDir) {
		logsDir = workflowLogsSquidDir
		logsParsingLog.Printf("Found firewall logs in workflow-logs/squid-logs directory")
	} else {
		logsParsingLog.Print("No firewall logs found, skipping parsing")
		// No firewall logs found - this is not an error, just skip parsing
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No firewall logs found in %s, skipping firewall log parsing", filepath.Base(runDir))))
		}
		return nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found firewall logs in %s", logsDir)))
	}

	// Create a temporary directory for running the parser
	tempDir, err := os.MkdirTemp("", "firewall_log_parser")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a Node.js script that mimics the GitHub Actions environment
	// The firewall parser expects logs in /tmp/gh-aw/squid-logs-{workflow}/
	// We'll set GITHUB_WORKFLOW to a value that makes the parser look in our temp directory
	nodeScript := fmt.Sprintf(`
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

        // Determine if request was allowed or blocked
        const isAllowed = isRequestAllowed(entry.decision, entry.status);

        if (isAllowed) {
          allowedRequests++;
          allowedDomains.add(entry.domain);
        } else {
          blockedRequests++;
          blockedDomains.add(entry.domain);
        }

        // Track request count per domain
        if (!requestsByDomain.has(entry.domain)) {
          requestsByDomain.set(entry.domain, { allowed: 0, blocked: 0 });
        }
        const domainStats = requestsByDomain.get(entry.domain);
        if (isAllowed) {
          domainStats.allowed++;
        } else {
          domainStats.blocked++;
        }
      }
    }

    // Generate step summary
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

// Execute the parser script to get helper functions
%s

// Replace main() call with our custom version
originalMain();
`, logsDir, jsScript)

	// Write the Node.js script
	nodeFile := filepath.Join(tempDir, "parser.js")
	if err := os.WriteFile(nodeFile, []byte(nodeScript), 0644); err != nil {
		return fmt.Errorf("failed to write node script: %w", err)
	}

	// Execute the Node.js script
	cmd := exec.Command("node", "parser.js")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute firewall parser script: %w\nOutput: %s", err, string(output))
	}

	// Write the output to firewall.md in the run directory
	firewallMdPath := filepath.Join(runDir, "firewall.md")
	if err := os.WriteFile(firewallMdPath, []byte(strings.TrimSpace(string(output))), 0644); err != nil {
		return fmt.Errorf("failed to write firewall.md: %w", err)
	}

	return nil
}
