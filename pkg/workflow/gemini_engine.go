package workflow

import (
	"strconv"
	"strings"
	"time"
)

// GeminiEngine represents the Google Gemini CLI agentic engine
type GeminiEngine struct {
	BaseEngine
}

func NewGeminiEngine() *GeminiEngine {
	return &GeminiEngine{
		BaseEngine: BaseEngine{
			id:                     "gemini",
			displayName:            "Gemini CLI",
			description:            "Uses Google Gemini CLI with GitHub integration and tool support",
			experimental:           false,
			supportsToolsWhitelist: true,
			supportsHTTPTransport:  false, // Gemini CLI does not support custom MCP HTTP servers
		},
	}
}

func (e *GeminiEngine) GetInstallationSteps(engineConfig *EngineConfig) []GitHubActionStep {
	// Gemini CLI doesn't require installation as it uses the Google GitHub Action
	return []GitHubActionStep{}
}

func (e *GeminiEngine) GetExecutionConfig(workflowName string, logFile string, engineConfig *EngineConfig) ExecutionConfig {
	config := ExecutionConfig{
		StepName: "Execute Gemini CLI Action",
		Action:   "google-github-actions/run-gemini-cli@v1",
		Inputs: map[string]string{
			"prompt":         "$(cat /tmp/aw-prompts/prompt.txt)", // Read from the prompt file
			"gemini_api_key": "${{ secrets.GEMINI_API_KEY }}",
		},
		Environment: map[string]string{
			"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
	}

	// Add model configuration via settings if specified
	if engineConfig != nil && engineConfig.Model != "" {
		// Gemini CLI uses settings JSON for model configuration
		settingsJSON := `{"model": "` + engineConfig.Model + `"}`
		config.Inputs["settings"] = settingsJSON
	}

	return config
}

func (e *GeminiEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	// Gemini CLI has built-in GitHub integration, so we don't need external MCP configuration
	// The GitHub tools are handled natively by the Gemini CLI when it has access to GITHUB_TOKEN

	yaml.WriteString("          # Gemini CLI handles GitHub integration natively when GITHUB_TOKEN is available\n")
	yaml.WriteString("          # No additional MCP configuration required for GitHub tools\n")

	// Check if there are custom MCP tools beyond GitHub
	hasCustomMCP := false
	for _, toolName := range mcpTools {
		if toolName != "github" {
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					hasCustomMCP = true
					break
				}
			}
		}
	}

	if hasCustomMCP {
		yaml.WriteString("          # Note: Custom MCP tools are not currently supported by Gemini CLI engine\n")
		yaml.WriteString("          # Consider using claude or opencode engines for custom MCP integrations\n")
	}
}

// ParseLogMetrics implements engine-specific log parsing for Gemini
func (e *GeminiEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics
	var startTime, endTime time.Time
	var maxTokenUsage int

	lines := strings.Split(logContent, "\n")

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Try to parse as streaming JSON first
		jsonMetrics := ExtractJSONMetrics(line, verbose)
		if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 || !jsonMetrics.Timestamp.IsZero() {
			// For Gemini, use maximum token usage approach (similar to non-Codex)
			if jsonMetrics.TokenUsage > maxTokenUsage {
				maxTokenUsage = jsonMetrics.TokenUsage
			}
			if jsonMetrics.EstimatedCost > 0 {
				metrics.EstimatedCost += jsonMetrics.EstimatedCost
			}
			if !jsonMetrics.Timestamp.IsZero() {
				if startTime.IsZero() || jsonMetrics.Timestamp.Before(startTime) {
					startTime = jsonMetrics.Timestamp
				}
				if endTime.IsZero() || jsonMetrics.Timestamp.After(endTime) {
					endTime = jsonMetrics.Timestamp
				}
			}
			continue
		}

		// Fall back to text pattern extraction
		// Extract timestamps for duration calculation
		timestamp := ExtractTimestamp(line)
		if !timestamp.IsZero() {
			if startTime.IsZero() || timestamp.Before(startTime) {
				startTime = timestamp
			}
			if endTime.IsZero() || timestamp.After(endTime) {
				endTime = timestamp
			}
		}

		// Extract Gemini-specific token usage patterns
		if tokenUsage := e.extractGeminiTokenUsage(line); tokenUsage > maxTokenUsage {
			maxTokenUsage = tokenUsage
		}

		// Extract cost information
		if cost := e.extractGeminiCost(line); cost > 0 {
			metrics.EstimatedCost += cost
		}

		// Count errors and warnings
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") {
			metrics.ErrorCount++
		}
		if strings.Contains(lowerLine, "warning") {
			metrics.WarningCount++
		}
	}

	metrics.TokenUsage = maxTokenUsage

	// Calculate duration
	if !startTime.IsZero() && !endTime.IsZero() {
		metrics.Duration = endTime.Sub(startTime)
	}

	return metrics
}

// extractGeminiTokenUsage extracts token usage from Gemini-specific log lines
func (e *GeminiEngine) extractGeminiTokenUsage(line string) int {
	// Look for Gemini-specific token patterns
	patterns := []string{
		`tokens?[:\s]+(\d+)`,
		`token[_\s]count[:\s]+(\d+)`,
		`input[_\s]tokens[:\s]+(\d+)`,
		`output[_\s]tokens[:\s]+(\d+)`,
		`total[_\s]tokens[:\s]+(\d+)`,
	}

	for _, pattern := range patterns {
		if match := ExtractFirstMatch(line, pattern); match != "" {
			if count, err := strconv.Atoi(match); err == nil {
				return count
			}
		}
	}

	return 0
}

// extractGeminiCost extracts cost information from Gemini log lines
func (e *GeminiEngine) extractGeminiCost(line string) float64 {
	// Look for patterns like "cost: $1.23", "price: 0.45", etc.
	patterns := []string{
		`cost[:\s]+\$?(\d+\.?\d*)`,
		`price[:\s]+\$?(\d+\.?\d*)`,
		`\$(\d+\.?\d+)`,
	}

	for _, pattern := range patterns {
		if match := ExtractFirstMatch(line, pattern); match != "" {
			if cost, err := strconv.ParseFloat(match, 64); err == nil {
				return cost
			}
		}
	}

	return 0
}
