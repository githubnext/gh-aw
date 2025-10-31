package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var agentToolsLog = logger.New("parser:agent_tools")

// AgentToolMapping holds the result of mapping GitHub Copilot agent tools to agentic workflow tools
type AgentToolMapping struct {
	Tools         map[string]any // Mapped tools in agentic workflow format
	UnknownTools  []string       // Tools that couldn't be mapped
	MappedTools   []string       // Tools that were successfully mapped
}

// MapAgentToolsToWorkflowTools converts GitHub Copilot agent tool names to agentic workflow tool format
// Returns a map suitable for merging with workflow tools and lists of mapped/unknown tools
func MapAgentToolsToWorkflowTools(agentTools []string) *AgentToolMapping {
	result := &AgentToolMapping{
		Tools:        make(map[string]any),
		UnknownTools: []string{},
		MappedTools:  []string{},
	}

	if len(agentTools) == 0 {
		return result
	}

	agentToolsLog.Printf("Mapping %d agent tools to workflow format", len(agentTools))

	// Track which tool categories we've seen
	hasEditTools := false
	hasGitHubTools := false
	hasBashTools := false

	// GitHub tool names to add
	githubToolNames := make(map[string]bool)

	for _, tool := range agentTools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}

		switch tool {
		// File editing tools -> edit
		case "createFile", "editFiles", "deleteFiles":
			hasEditTools = true
			result.MappedTools = append(result.MappedTools, tool)
			agentToolsLog.Printf("Mapped '%s' to 'edit' tool", tool)

		// GitHub repository tools -> github
		case "search", "codeSearch":
			hasGitHubTools = true
			if tool == "search" {
				githubToolNames["search_code"] = true
			} else {
				githubToolNames["search_code"] = true // codeSearch also maps to search_code
			}
			result.MappedTools = append(result.MappedTools, tool)
			agentToolsLog.Printf("Mapped '%s' to GitHub tool", tool)

		case "getFile", "listFiles":
			hasGitHubTools = true
			githubToolNames["get_file_contents"] = true
			result.MappedTools = append(result.MappedTools, tool)
			agentToolsLog.Printf("Mapped '%s' to GitHub tool", tool)

		// Shell command tools -> bash
		case "runCommand":
			hasBashTools = true
			result.MappedTools = append(result.MappedTools, tool)
			agentToolsLog.Printf("Mapped '%s' to 'bash' tool", tool)

		default:
			// Unknown tool
			result.UnknownTools = append(result.UnknownTools, tool)
			agentToolsLog.Printf("Unknown agent tool: '%s'", tool)
		}
	}

	// Build the tools map
	if hasEditTools {
		result.Tools["edit"] = true
	}

	if hasGitHubTools {
		// Convert github tool names to allowed list
		allowedTools := make([]string, 0, len(githubToolNames))
		for toolName := range githubToolNames {
			allowedTools = append(allowedTools, toolName)
		}
		
		result.Tools["github"] = map[string]any{
			"allowed": allowedTools,
		}
	}

	if hasBashTools {
		result.Tools["bash"] = true
	}

	agentToolsLog.Printf("Tool mapping complete: %d mapped, %d unknown", len(result.MappedTools), len(result.UnknownTools))

	return result
}

// ExtractToolsFromAgentFrontmatter extracts and maps tools from agent file frontmatter
func ExtractToolsFromAgentFrontmatter(frontmatter map[string]any) (string, []string, error) {
	toolsField, exists := frontmatter["tools"]
	if !exists {
		return "", nil, nil
	}

	agentToolsLog.Print("Found tools field in agent file frontmatter")

	// Convert to array of strings
	var agentTools []string
	switch v := toolsField.(type) {
	case []any:
		for _, item := range v {
			if str, ok := item.(string); ok {
				agentTools = append(agentTools, str)
			}
		}
	case []string:
		agentTools = v
	default:
		return "", nil, fmt.Errorf("agent tools field must be an array of strings, got %T", toolsField)
	}

	if len(agentTools) == 0 {
		return "", nil, nil
	}

	// Map the agent tools to workflow tools
	mapping := MapAgentToolsToWorkflowTools(agentTools)

	// Serialize to JSON for merging
	toolsJSON := "{}"
	if len(mapping.Tools) > 0 {
		bytes, err := json.Marshal(mapping.Tools)
		if err != nil {
			return "", nil, fmt.Errorf("failed to serialize mapped tools: %w", err)
		}
		toolsJSON = string(bytes)
	}

	return toolsJSON, mapping.UnknownTools, nil
}
