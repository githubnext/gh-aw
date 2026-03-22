package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var dispatchRepositoryLog = logger.New("workflow:dispatch_repository")

// DispatchRepositoryToolConfig defines a single repository dispatch tool within dispatch_repository
type DispatchRepositoryToolConfig struct {
	Description         string         `yaml:"description,omitempty"`          // Human-readable description
	Workflow            string         `yaml:"workflow"`                       // Target workflow name (for traceability and payload)
	EventType           string         `yaml:"event_type"`                     // repository_dispatch event_type
	Repository          string         `yaml:"repository,omitempty"`           // Single target repository (owner/repo)
	AllowedRepositories []string       `yaml:"allowed_repositories,omitempty"` // Multiple allowed target repositories
	Inputs              map[string]any `yaml:"inputs,omitempty"`               // Input schema (similar to workflow_dispatch inputs)
	Max                 *string        `yaml:"max,omitempty"`                  // Max dispatch executions (templatable int)
	GitHubToken         string         `yaml:"github-token,omitempty"`         // Optional override token
	Staged              bool           `yaml:"staged,omitempty"`               // If true, preview-only mode
}

// DispatchRepositoryConfig holds configuration for dispatching repository_dispatch events
// Uses a map-of-tools pattern where each key defines a named dispatch tool
type DispatchRepositoryConfig struct {
	Tools map[string]*DispatchRepositoryToolConfig // Map of tool name to tool config
}

// parseDispatchRepositoryConfig parses dispatch_repository configuration from the safe-outputs map.
// Accepts both "dispatch_repository" (underscore, preferred) and "dispatch-repository" (dash, alias).
func (c *Compiler) parseDispatchRepositoryConfig(outputMap map[string]any) *DispatchRepositoryConfig {
	dispatchRepositoryLog.Print("Parsing dispatch_repository configuration")

	var configData any
	var exists bool

	// Support both underscore and dash variants
	if configData, exists = outputMap["dispatch_repository"]; !exists {
		if configData, exists = outputMap["dispatch-repository"]; !exists {
			return nil
		}
	}

	configMap, ok := configData.(map[string]any)
	if !ok {
		dispatchRepositoryLog.Print("dispatch_repository value is not a map, skipping")
		return nil
	}

	dispatchRepositoryLog.Printf("Parsing dispatch_repository tools map with %d entries", len(configMap))

	dispatchRepoConfig := &DispatchRepositoryConfig{
		Tools: make(map[string]*DispatchRepositoryToolConfig),
	}

	for toolKey, toolValue := range configMap {
		toolMap, ok := toolValue.(map[string]any)
		if !ok {
			dispatchRepositoryLog.Printf("Skipping tool %q: value is not a map", toolKey)
			continue
		}

		tool := &DispatchRepositoryToolConfig{}

		if desc, ok := toolMap["description"].(string); ok {
			tool.Description = desc
		}

		if workflow, ok := toolMap["workflow"].(string); ok {
			tool.Workflow = workflow
		}

		if eventType, ok := toolMap["event_type"].(string); ok {
			tool.EventType = eventType
		}

		if repo, ok := toolMap["repository"].(string); ok {
			tool.Repository = repo
		}

		// Parse allowed_repositories (list of repos)
		if allowedReposRaw, exists := toolMap["allowed_repositories"]; exists {
			if allowedReposList, ok := allowedReposRaw.([]any); ok {
				for _, r := range allowedReposList {
					if rStr, ok := r.(string); ok {
						tool.AllowedRepositories = append(tool.AllowedRepositories, rStr)
					}
				}
			}
		}

		// Parse inputs (map of input definitions)
		if inputsRaw, exists := toolMap["inputs"]; exists {
			if inputsMap, ok := inputsRaw.(map[string]any); ok {
				tool.Inputs = inputsMap
			}
		}

		// Parse max (templatable int, default 1)
		var baseCfg BaseSafeOutputConfig
		c.parseBaseSafeOutputConfig(toolMap, &baseCfg, 1)
		tool.Max = baseCfg.Max
		tool.GitHubToken = baseCfg.GitHubToken
		tool.Staged = baseCfg.Staged

		// Cap max at 50
		if maxVal := templatableIntValue(tool.Max); maxVal > 50 {
			dispatchRepositoryLog.Printf("Tool %q: max value %d exceeds limit, capping at 50", toolKey, maxVal)
			tool.Max = defaultIntStr(50)
		}

		dispatchRepositoryLog.Printf("Parsed dispatch_repository tool %q: workflow=%s, event_type=%s, max=%v",
			toolKey, tool.Workflow, tool.EventType, tool.Max)

		dispatchRepoConfig.Tools[toolKey] = tool
	}

	if len(dispatchRepoConfig.Tools) == 0 {
		dispatchRepositoryLog.Print("No valid tools found in dispatch_repository config")
		return nil
	}

	return dispatchRepoConfig
}

// generateDispatchRepositoryTool generates an MCP tool definition for a specific dispatch_repository tool.
// The tool will be named after the tool key (normalized to underscores) and accept
// the tool's declared inputs as parameters.
func generateDispatchRepositoryTool(toolKey string, toolConfig *DispatchRepositoryToolConfig) map[string]any {
	dispatchRepositoryLog.Printf("Generating dispatch_repository tool: key=%s", toolKey)

	// Normalize tool key to use underscores
	toolName := stringutil.NormalizeSafeOutputIdentifier(toolKey)

	description := toolConfig.Description
	if description == "" {
		description = "Dispatch a repository_dispatch event"
		if toolConfig.EventType != "" {
			description += " with event_type: " + toolConfig.EventType
		}
	}

	if toolConfig.Workflow != "" {
		description += " (targets workflow: " + toolConfig.Workflow + ")"
	}

	// Build input schema from the tool's inputs definition
	properties := make(map[string]any)
	required := []string{}

	for inputName, inputDef := range toolConfig.Inputs {
		inputDefMap, ok := inputDef.(map[string]any)
		if !ok {
			continue
		}

		inputType := "string"
		inputDescription := "Input parameter '" + inputName + "'"
		inputRequired := false

		if desc, ok := inputDefMap["description"].(string); ok && desc != "" {
			inputDescription = desc
		}
		if req, ok := inputDefMap["required"].(bool); ok {
			inputRequired = req
		}

		// Map input types to JSON Schema types
		if typeStr, ok := inputDefMap["type"].(string); ok {
			switch typeStr {
			case "number":
				inputType = "number"
			case "boolean":
				inputType = "boolean"
			case "choice":
				inputType = "string"
				if options, ok := inputDefMap["options"].([]any); ok && len(options) > 0 {
					prop := map[string]any{
						"type":        inputType,
						"description": inputDescription,
						"enum":        options,
					}
					if defaultVal, ok := inputDefMap["default"]; ok {
						prop["default"] = defaultVal
					}
					properties[inputName] = prop
					if inputRequired {
						required = append(required, inputName)
					}
					continue
				}
			case "environment":
				inputType = "string"
			}
		}

		prop := map[string]any{
			"type":        inputType,
			"description": inputDescription,
		}
		if defaultVal, ok := inputDefMap["default"]; ok {
			prop["default"] = defaultVal
		}
		properties[inputName] = prop

		if inputRequired {
			required = append(required, inputName)
		}
	}

	tool := map[string]any{
		"name":                      toolName,
		"description":               description,
		"_dispatch_repository_tool": toolKey, // Internal metadata for handler routing
		"inputSchema": map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		},
	}

	if len(required) > 0 {
		tool["inputSchema"].(map[string]any)["required"] = required
	}

	dispatchRepositoryLog.Printf("Generated dispatch_repository tool: name=%s, properties=%d", toolName, len(properties))
	return tool
}
