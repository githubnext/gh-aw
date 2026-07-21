package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var tokenUsageLog = logger.New("cli:token_usage")

// tokenUsageJSONLPath is the relative path within the firewall logs directory
const tokenUsageJSONLPath = "api-proxy-logs/token-usage.jsonl"
const proxyEventsJSONLPath = "api-proxy-logs/events.jsonl"
const agentUsageJSONPath = "agent_usage.json"
const modelMismatchReasonTokenUsageMissing = "TOKEN_USAGE_MISSING"
const modelMismatchReasonModelNotObserved = "REQUESTED_MODEL_NOT_OBSERVED"
const subagentStdioWarning = "partial or incorrect data: sub-agent model requests are inferred from agent-stdio.log; use token_usage.jsonl for reliable token consumption"
const tokenSteeringEventName = "token_steering"
const timeoutSteeringEventName = "timeout_steering"
const awfTokenWarningPrefix = "[AWF TOKEN WARNING]"
const awfTimeWarningPrefix = "[AWF TIME WARNING]"

// analyzeTokenUsage finds and parses the token-usage.jsonl file from a run directory.
func analyzeTokenUsage(runDir string, verbose bool) (*TokenUsageSummary, error) {
	tokenUsageLog.Printf("Analyzing token usage in: %s", runDir)

	filePath := findTokenUsageFile(runDir)
	if filePath != "" {
		fileInfo, _ := os.Stat(filePath)
		if fileInfo != nil {
			console.LogVerbose(verbose, fmt.Sprintf("  Found token usage file: %s (%d bytes)", filepath.Base(filePath), fileInfo.Size()))
		}

		summary, err := parseTokenUsageFile(filePath)
		if err != nil {
			return summary, err
		}
		// When the file exists but contains no entries (e.g. usage artifact has an
		// empty placeholder token_usage.jsonl), fall through to the agent_usage.json
		// fallback rather than returning nil immediately.
		if summary != nil {
			summary.TotalSteeringEvents = countAPIProxySteeringEvents(runDir)
			augmentSubagentModelAttribution(runDir, summary)
			return summary, nil
		}
	}

	agentUsagePath := findAgentUsageFile(runDir)
	if agentUsagePath == "" {
		return nil, nil
	}
	agentFileInfo, _ := os.Stat(agentUsagePath)
	if agentFileInfo != nil {
		console.LogVerbose(verbose, fmt.Sprintf("  Found agent usage file: %s (%d bytes)", filepath.Base(agentUsagePath), agentFileInfo.Size()))
	}

	summary, err := parseAgentUsageFile(agentUsagePath)
	if err != nil || summary == nil {
		return summary, err
	}
	summary.TotalSteeringEvents = countAPIProxySteeringEvents(runDir)
	augmentSubagentModelAttribution(runDir, summary)
	return summary, nil
}
