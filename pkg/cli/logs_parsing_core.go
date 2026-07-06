// This file provides command-line interface functionality for gh-aw.
// This file (logs_parsing_core.go) contains core log parsing functions
// for extracting engine configuration from workflow logs.
//
// Key responsibilities:
//   - Parsing aw_info.json to extract engine configuration
//   - Extracting engine ID from lock file gh-aw-metadata as a fallback
//   - Registering the errWalkStop sentinel used by walk-based helpers in this package

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsParsingCoreLog = logger.New("cli:logs_parsing_core")
var lockFileEngineIDCache sync.Map

// errWalkStop is a sentinel returned from filepath.Walk callbacks to stop traversal early.
// It is shared across all walk-based file-search functions in this package.
var errWalkStop = errors.New("stop")

// parseAwInfo reads and parses aw_info.json file, returning the parsed data
// Handles cases where aw_info.json is a file or a directory containing the actual file
func parseAwInfo(infoFilePath string, verbose bool) (*AwInfo, error) {
	// Sanitize the path to prevent path traversal attacks
	cleanPath := filepath.Clean(infoFilePath)
	logsParsingCoreLog.Printf("Parsing aw_info.json from: %s", cleanPath)
	var data []byte
	var err error

	// Check if the path exists and determine if it's a file or directory
	stat, statErr := os.Stat(cleanPath)
	if statErr != nil {
		logsParsingCoreLog.Printf("Failed to stat aw_info.json: %v", statErr)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to stat aw_info.json: %v", statErr)))
		}
		return nil, statErr
	}

	if stat.IsDir() {
		// It's a directory - look for nested aw_info.json
		nestedPath := filepath.Join(cleanPath, "aw_info.json")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("aw_info.json is a directory, trying nested file: "+nestedPath))
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
		logsParsingCoreLog.Printf("Failed to unmarshal aw_info.json: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse aw_info.json: %v", err)))
		}
		return nil, err
	}

	logsParsingCoreLog.Printf("Successfully parsed aw_info.json with engine_id: %s", info.EngineID)
	return &info, nil
}

// extractEngineFromAwInfo reads aw_info.json and returns the appropriate engine
// Handles cases where aw_info.json is a file or a directory containing the actual file
func extractEngineFromAwInfo(infoFilePath string, verbose bool) workflow.CodingAgentEngine {
	logsParsingCoreLog.Printf("Extracting engine from aw_info.json: %s", infoFilePath)
	info, err := parseAwInfo(infoFilePath, verbose)
	if err != nil {
		return nil
	}

	if info.EngineID == "" {
		logsParsingCoreLog.Print("No engine_id found in aw_info.json")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No engine_id found in aw_info.json"))
		}
		return nil
	}

	registry := workflow.GetGlobalEngineRegistry()
	engine, err := registry.GetEngine(info.EngineID)
	if err != nil {
		logsParsingCoreLog.Printf("Unknown engine: %s", info.EngineID)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Unknown engine in aw_info.json: "+info.EngineID))
		}
		return nil
	}

	logsParsingCoreLog.Printf("Successfully extracted engine: %s", engine.GetID())
	return engine
}

// extractEngineIDFromLockFile reads the local lock file for the given workflow and
// returns the agent_id embedded in its gh-aw-metadata comment.  This provides a
// fallback engine source when aw_info.json is absent (e.g. for older runs that
// pre-date the aw_info.json artifact).
//
// workflowPath is the workflow file path as returned by the GitHub API (e.g.
// ".github/workflows/daily-cli-tools-tester.lock.yml").  Plain ".yml" paths are
// also accepted and are automatically resolved to the ".lock.yml" variant.
// Returns "" when the path is empty, the lock file cannot be read, or no
// agent_id is present in the metadata.
func extractEngineIDFromLockFile(workflowPath string, verbose bool) string {
	if workflowPath == "" {
		return ""
	}

	lockPath := resolveToLockFilePath(workflowPath)
	if lockPath == "" {
		logsParsingCoreLog.Printf("Cannot resolve lock file path from: %s", workflowPath)
		return ""
	}

	if !filepath.IsAbs(lockPath) {
		repoRoot, err := gitutil.FindGitRoot()
		if err == nil {
			lockPath = filepath.Join(repoRoot, filepath.FromSlash(lockPath))
		}
	}

	lockPath = filepath.Clean(lockPath)
	if cachedEngineID, ok := lockFileEngineIDCache.Load(lockPath); ok {
		engineID, _ := cachedEngineID.(string)
		if engineID != "" {
			logsParsingCoreLog.Printf("Using cached engine_id=%s from lock file %s", engineID, lockPath)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Detected engine from lock file metadata: "+engineID))
			}
		}
		return engineID
	}

	content, err := os.ReadFile(lockPath)
	if err != nil {
		lockFileEngineIDCache.Store(lockPath, "")
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			cwd = "<unknown>"
		}
		logsParsingCoreLog.Printf("Cannot read lock file %s (cwd: %s): %v", lockPath, cwd, err)
		return ""
	}

	metadata, _, err := workflow.ExtractMetadataFromLockFile(string(content))
	if err != nil || metadata == nil || metadata.AgentID == "" {
		lockFileEngineIDCache.Store(lockPath, "")
		logsParsingCoreLog.Printf("No agent_id in lock file metadata: %s", lockPath)
		return ""
	}

	lockFileEngineIDCache.Store(lockPath, metadata.AgentID)
	logsParsingCoreLog.Printf("Extracted engine_id=%s from lock file %s", metadata.AgentID, lockPath)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Detected engine from lock file metadata: "+metadata.AgentID))
	}
	return metadata.AgentID
}

// resolveToLockFilePath converts a workflow file path to its lock file path.
// Handles .lock.yml paths (returned as-is), plain .yml paths (converted by
// appending ".lock" before the ".yml" suffix), and .md source paths.
// Returns "" for unrecognised extensions.
func resolveToLockFilePath(workflowPath string) string {
	switch {
	case strings.HasSuffix(workflowPath, ".lock.yml"):
		return workflowPath
	case strings.HasSuffix(workflowPath, ".md"):
		return strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	case strings.HasSuffix(workflowPath, ".yml"):
		return strings.TrimSuffix(workflowPath, ".yml") + ".lock.yml"
	case strings.HasSuffix(workflowPath, ".yaml"):
		return strings.TrimSuffix(workflowPath, ".yaml") + ".lock.yml"
	default:
		return ""
	}
}
