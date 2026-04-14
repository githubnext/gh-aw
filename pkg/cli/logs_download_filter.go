// This file provides command-line interface functionality for gh-aw.
// This file (logs_download_filter.go) contains filter predicate functions used
// by DownloadWorkflowLogs to decide whether to include a given run.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
)

// runContainsSafeOutputType checks if a run's agent_output.json contains a specific safe output type
func runContainsSafeOutputType(runDir string, safeOutputType string, verbose bool) (bool, error) {
	logsOrchestratorLog.Printf("Checking run for safe output type: dir=%s, type=%s", runDir, safeOutputType)
	// Normalize the type for comparison (convert dashes to underscores)
	normalizedType := stringutil.NormalizeSafeOutputIdentifier(safeOutputType)

	// Look for agent_output.json in the run directory
	agentOutputPath := filepath.Join(runDir, constants.AgentOutputFilename)

	// Support both new flattened form and old directory form
	if stat, err := os.Stat(agentOutputPath); err != nil || stat.IsDir() {
		// Try old structure
		oldPath := filepath.Join(runDir, constants.AgentOutputArtifactName, constants.AgentOutputArtifactName)
		if _, err := os.Stat(oldPath); err == nil {
			agentOutputPath = oldPath
		} else {
			// No agent_output.json found
			return false, nil
		}
	}

	// Read the file
	content, err := os.ReadFile(agentOutputPath)
	if err != nil {
		// File doesn't exist or can't be read
		return false, nil
	}

	// Parse the JSON
	var safeOutput struct {
		Items []json.RawMessage `json:"items"`
	}

	if err := json.Unmarshal(content, &safeOutput); err != nil {
		return false, fmt.Errorf("failed to parse agent_output.json: %w", err)
	}

	// Check each item for the specified type
	for _, itemRaw := range safeOutput.Items {
		var item struct {
			Type string `json:"type"`
		}

		if err := json.Unmarshal(itemRaw, &item); err != nil {
			continue // Skip malformed items
		}

		// Normalize the item type for comparison
		normalizedItemType := stringutil.NormalizeSafeOutputIdentifier(item.Type)

		if normalizedItemType == normalizedType {
			return true, nil
		}
	}

	return false, nil
}

// runHasDifcFilteredItems checks if a run's gateway logs contain any DIFC_FILTERED events.
// It parses the gateway logs (falling back to rpc-messages.jsonl when gateway.jsonl is absent)
// and returns true when at least one DIFC integrity- or secrecy-filtered event is present.
func runHasDifcFilteredItems(runDir string, verbose bool) (bool, error) {
	logsOrchestratorLog.Printf("Checking run for DIFC filtered items: dir=%s", runDir)

	gatewayMetrics, err := parseGatewayLogs(runDir, verbose)
	if err != nil {
		// No gateway log file present — not an error for workflows without MCP
		return false, nil
	}

	if gatewayMetrics == nil {
		return false, nil
	}

	return gatewayMetrics.TotalFiltered > 0, nil
}
