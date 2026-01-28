package workflow

import (
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/parser"
)

// ComplexityTier represents the complexity level of a workflow request
type ComplexityTier string

const (
	// ComplexityBasic indicates a simple workflow with standard configurations
	ComplexityBasic ComplexityTier = "basic"
	// ComplexityIntermediate indicates a moderately complex workflow with some customization
	ComplexityIntermediate ComplexityTier = "intermediate"
	// ComplexityAdvanced indicates a complex workflow with advanced features
	ComplexityAdvanced ComplexityTier = "advanced"
)

// String returns the string representation of the complexity tier
func (c ComplexityTier) String() string {
	return string(c)
}

// IsValid checks if the complexity tier is valid
func (c ComplexityTier) IsValid() bool {
	switch c {
	case ComplexityBasic, ComplexityIntermediate, ComplexityAdvanced:
		return true
	default:
		return false
	}
}

// ComplexityScore represents the result of complexity detection
type ComplexityScore struct {
	// Tier is the detected complexity tier
	Tier ComplexityTier
	// Score is the calculated complexity score
	Score int
	// Indicators lists the detected complexity indicators
	Indicators []string
}

// DetectWorkflowComplexity analyzes workflow content and description to detect complexity tier
// It uses heuristics based on triggers, tools, configuration, and description patterns
func DetectWorkflowComplexity(content string, description string) *ComplexityScore {
	score := 0
	indicators := []string{}

	// Parse frontmatter to analyze configuration
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		// If we can't parse frontmatter, analyze description only
		return analyzeDescriptionComplexity(description)
	}

	frontmatter := result.Frontmatter

	// Analyze triggers (on: section)
	if onSection, exists := frontmatter["on"]; exists {
		triggerScore, triggerIndicators := analyzeTriggers(onSection)
		score += triggerScore
		indicators = append(indicators, triggerIndicators...)
	}

	// Analyze tools configuration
	if toolsSection, exists := frontmatter["tools"]; exists {
		toolScore, toolIndicators := analyzeTools(toolsSection)
		score += toolScore
		indicators = append(indicators, toolIndicators...)
	}

	// Analyze network permissions
	if networkSection, exists := frontmatter["network"]; exists {
		networkScore, networkIndicators := analyzeNetwork(networkSection)
		score += networkScore
		indicators = append(indicators, networkIndicators...)
	}

	// Analyze sandbox configuration
	if sandboxSection, exists := frontmatter["sandbox"]; exists {
		sandboxScore, sandboxIndicators := analyzeSandbox(sandboxSection)
		score += sandboxScore
		indicators = append(indicators, sandboxIndicators...)
	}

	// Analyze jobs configuration
	if jobsSection, exists := frontmatter["jobs"]; exists {
		jobScore, jobIndicators := analyzeJobs(jobsSection)
		score += jobScore
		indicators = append(indicators, jobIndicators...)
	}

	// Analyze description patterns
	descScore, descIndicators := analyzeDescription(description)
	score += descScore
	indicators = append(indicators, descIndicators...)

	// Determine tier based on score thresholds
	tier := determineTier(score)

	return &ComplexityScore{
		Tier:       tier,
		Score:      score,
		Indicators: indicators,
	}
}

// analyzeTriggers analyzes the 'on:' section to detect trigger complexity
func analyzeTriggers(onSection any) (int, []string) {
	score := 0
	indicators := []string{}

	switch on := onSection.(type) {
	case string:
		// Single trigger as string (e.g., on: push)
		if isStandardEvent(on) {
			// Standard single trigger - no complexity added
		} else {
			score += 1
			indicators = append(indicators, "custom trigger type")
		}
	case []any:
		// Multiple triggers as array
		if len(on) == 1 {
			// Single trigger in array form
		} else if len(on) <= 3 {
			score += 2
			indicators = append(indicators, "multiple triggers")
		} else {
			score += 3
			indicators = append(indicators, "many triggers (4+)")
		}
	case map[string]any:
		// Complex trigger configuration
		triggerCount := len(on)
		if triggerCount == 1 {
			// Single trigger with config
			score += 1
			indicators = append(indicators, "trigger with configuration")
		} else if triggerCount <= 3 {
			score += 3
			indicators = append(indicators, "multiple triggers with configuration")
		} else {
			score += 4
			indicators = append(indicators, "many configured triggers (4+)")
		}

		// Check for advanced trigger features
		if hasScheduleTrigger(on) {
			score += 1
			indicators = append(indicators, "scheduled trigger")
		}
		if hasWorkflowCallTrigger(on) {
			score += 2
			indicators = append(indicators, "reusable workflow")
		}
	}

	return score, indicators
}

// analyzeTools analyzes the tools configuration to detect tool usage complexity
func analyzeTools(toolsSection any) (int, []string) {
	score := 0
	indicators := []string{}

	toolsMap, ok := toolsSection.(map[string]any)
	if !ok {
		return 0, nil
	}

	// Count all tools including standard ones
	toolCount := len(toolsMap)
	mcpCount := 0

	for toolName, toolConfig := range toolsMap {
		// Check if it's an MCP server configuration
		if configMap, ok := toolConfig.(map[string]any); ok {
			if _, hasMode := configMap["mode"]; hasMode {
				mcpCount++
			}
		}

		// Skip github and playwright for the "custom tools" count
		if toolName == "github" || toolName == "playwright" {
			toolCount--
		}
	}

	// Score based on total tool count (including standard tools if customized)
	if toolCount == 0 {
		// No tools or only github/playwright without customization
	} else if toolCount <= 2 {
		score += 2
		indicators = append(indicators, "2-3 tools configured")
	} else if toolCount <= 5 {
		score += 3
		indicators = append(indicators, "4-5 tools configured")
	} else {
		score += 4
		indicators = append(indicators, "6+ tools configured")
	}

	// Additional score for MCP servers (more complex)
	if mcpCount > 0 {
		score += 1
		indicators = append(indicators, "custom MCP servers")
	}

	return score, indicators
}

// analyzeNetwork analyzes network permissions for complexity
func analyzeNetwork(networkSection any) (int, []string) {
	score := 0
	indicators := []string{}

	networkMap, ok := networkSection.(map[string]any)
	if !ok {
		return 0, nil
	}

	// Check for allowed domains
	if allowed, exists := networkMap["allowed"]; exists {
		if allowedList, ok := allowed.([]any); ok && len(allowedList) > 0 {
			score += 1
			indicators = append(indicators, "network domain restrictions")
		}
	}

	// Check for blocked domains
	if blocked, exists := networkMap["blocked"]; exists {
		if blockedList, ok := blocked.([]any); ok && len(blockedList) > 0 {
			score += 1
			indicators = append(indicators, "network domain blocking")
		}
	}

	return score, indicators
}

// analyzeSandbox analyzes sandbox configuration for complexity
func analyzeSandbox(sandboxSection any) (int, []string) {
	score := 0
	indicators := []string{}

	sandboxMap, ok := sandboxSection.(map[string]any)
	if !ok {
		return 0, nil
	}

	// Check for custom mounts
	if mounts, exists := sandboxMap["mounts"]; exists {
		if mountsList, ok := mounts.([]any); ok && len(mountsList) > 0 {
			score += 2
			indicators = append(indicators, "custom sandbox mounts")
		}
	}

	// Check for environment variables
	if env, exists := sandboxMap["env"]; exists {
		if envMap, ok := env.(map[string]any); ok && len(envMap) > 0 {
			score += 1
			indicators = append(indicators, "sandbox environment variables")
		}
	}

	return score, indicators
}

// analyzeJobs analyzes jobs configuration for complexity
func analyzeJobs(jobsSection any) (int, []string) {
	score := 0
	indicators := []string{}

	jobsMap, ok := jobsSection.(map[string]any)
	if !ok {
		return 0, nil
	}

	jobCount := len(jobsMap)
	if jobCount > 1 {
		score += 2
		indicators = append(indicators, "multi-stage workflow")
	}

	// Check for job dependencies (needs: keyword)
	hasDepend := false
	for _, jobConfig := range jobsMap {
		if jobMap, ok := jobConfig.(map[string]any); ok {
			if _, hasNeeds := jobMap["needs"]; hasNeeds {
				hasDepend = true
				break
			}
		}
	}

	if hasDepend {
		score += 2
		indicators = append(indicators, "job dependencies")
	}

	return score, indicators
}

// analyzeDescription analyzes the description text for complexity indicators
func analyzeDescription(description string) (int, []string) {
	score := 0
	indicators := []string{}

	if description == "" {
		return 0, nil
	}

	lower := strings.ToLower(description)

	// Check for technical keywords indicating complexity
	advancedKeywords := []string{
		"performance", "optimization", "scalability", "security",
		"authentication", "authorization", "encryption",
		"microservice", "distributed", "orchestration",
		"compliance", "audit", "monitoring",
	}

	for _, keyword := range advancedKeywords {
		if strings.Contains(lower, keyword) {
			score += 1
			indicators = append(indicators, "advanced requirement: "+keyword)
			break // Count once for advanced keywords
		}
	}

	// Check for conditional logic keywords
	conditionalKeywords := []string{
		"if ", "when ", "conditional", "based on",
		"depending on", "in case", "otherwise",
	}

	for _, keyword := range conditionalKeywords {
		if strings.Contains(lower, keyword) {
			score += 1
			indicators = append(indicators, "conditional logic mentioned")
			break // Count once for conditional keywords
		}
	}

	// Check for file path references (Unix and Windows paths)
	filePathPattern := regexp.MustCompile(`(?i)[\w-]+/[\w-]+\.[\w]+|src/|\.go|\.tsx?|\.jsx?|\.py`)
	if filePathPattern.MatchString(description) {
		score += 1
		indicators = append(indicators, "specific file paths referenced")
	}

	// Check for integration keywords
	integrationKeywords := []string{
		"integrate", "integration", "api", "webhook",
		"external", "third-party", "service",
	}

	for _, keyword := range integrationKeywords {
		if strings.Contains(lower, keyword) {
			score += 1
			indicators = append(indicators, "integration requirements")
			break // Count once for integration keywords
		}
	}

	return score, indicators
}

// analyzeDescriptionComplexity provides basic complexity detection based only on description
// Used as fallback when frontmatter cannot be parsed
func analyzeDescriptionComplexity(description string) *ComplexityScore {
	score, indicators := analyzeDescription(description)
	tier := determineTier(score)

	return &ComplexityScore{
		Tier:       tier,
		Score:      score,
		Indicators: indicators,
	}
}

// determineTier maps a complexity score to a tier
func determineTier(score int) ComplexityTier {
	if score <= 3 {
		return ComplexityBasic
	} else if score <= 7 {
		return ComplexityIntermediate
	} else {
		return ComplexityAdvanced
	}
}

// isStandardEvent checks if a trigger is a standard GitHub event
func isStandardEvent(event string) bool {
	standardEvents := []string{
		"push", "pull_request", "issues", "issue_comment",
		"pull_request_review", "pull_request_review_comment",
		"create", "delete", "fork", "watch", "star",
		"release", "workflow_dispatch",
	}

	eventLower := strings.ToLower(strings.TrimSpace(event))
	for _, std := range standardEvents {
		if eventLower == std {
			return true
		}
	}

	return false
}

// hasScheduleTrigger checks if the trigger configuration includes a schedule
func hasScheduleTrigger(triggers map[string]any) bool {
	_, exists := triggers["schedule"]
	return exists
}

// hasWorkflowCallTrigger checks if the trigger configuration includes workflow_call
func hasWorkflowCallTrigger(triggers map[string]any) bool {
	_, exists := triggers["workflow_call"]
	return exists
}
