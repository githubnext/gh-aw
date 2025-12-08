---
description: Adding a New Safe Output Type to GitHub Agentic Workflows
---

# Add New Safe Output Type

This guide covers adding a new safe output type to process AI agent outputs in JSONL format through a validation pipeline (TypeScript types → JSON schema → JavaScript collection).

## Implementation Steps

### 1. Update JSON Schema (`schemas/agent-output.json`)

Add object definition in `$defs` section:
   ```json
   "YourNewTypeOutput": {
     "title": "Your New Type Output",
     "description": "Output for your new functionality",
     "type": "object",
     "properties": {
       "type": {
         "const": "your-new-type"
       },
       "required_field": {
         "type": "string",
         "description": "Description of required field",
         "minLength": 1
       },
       "optional_field": {
         "type": "string", 
         "description": "Description of optional field"
       }
     },
     "required": ["type", "required_field"],
     "additionalProperties": false
   }
   ```

Add to `SafeOutput` oneOf array: `{"$ref": "#/$defs/YourNewTypeOutput"}`

**Validation Notes**: Use `const` for type field, `minLength: 1` for required strings, `additionalProperties: false`, `oneOf` for union types.

### 2. Update TypeScript Types

**File**: `pkg/workflow/js/types/safe-outputs.d.ts`
   ```typescript
   /**
    * JSONL item for [description]
    */
   interface YourNewTypeItem extends BaseSafeOutputItem {
     type: "your-new-type";
     /** Required field description */
     required_field: string;
     /** Optional field description */
     optional_field?: string;
   }
   }

Add to `SafeOutputItem` union type and export list.

**File**: `pkg/workflow/js/types/safe-outputs-config.d.ts` - Add config interface, add to `SpecificSafeOutputConfig` union, export.

### 3. Update Safe Outputs Tools JSON (`pkg/workflow/js/safe_outputs_tools.json`)

Add tool signature to expose to AI agents:

```json
{
  "name": "your_new_type",
  "description": "Brief description of what this tool does (use underscores in name, not hyphens)",
  "inputSchema": {
    "type": "object",
    "required": ["required_field"],
    "properties": {
      "required_field": {
        "type": "string",
        "description": "Description of the required field"
      },
      "optional_field": {
        "type": "string",
        "description": "Description of the optional field"
      },
      "numeric_field": {
        "type": ["number", "string"],
        "description": "Numeric field that accepts both number and string types"
      }
    },
    "additionalProperties": false
  }
}
```

**Guidelines**: Use underscores in tool `name`, match with type field, set `additionalProperties: false`, use `"type": ["number", "string"]` for numeric fields.

**Important**: File is embedded via `//go:embed` - **must rebuild** with `make build` after changes.

### 4. Update MCP Server JavaScript (If Custom Handler Needed) (`pkg/workflow/js/safe_outputs_mcp_server.cjs`)

Most types use the default JSONL handler. Add custom handler only if needed for file operations, git commands, or complex validation:

```javascript
/**
 * Handler for your_new_type safe output
 * @param {Object} args - Arguments passed to the tool
 * @returns {Object} MCP tool response
 */
const yourNewTypeHandler = args => {
  // Perform any custom validation
  if (!args.required_field || typeof args.required_field !== "string") {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: "required_field is required and must be a string",
          }),
        },
      ],
      isError: true,
    };
  }

  // Perform custom operations (e.g., file system operations, git commands)
  try {
    // Your custom logic here
    const result = performCustomOperation(args);
    
    // Write the JSONL entry
    const entry = {
      type: "your_new_type",
      required_field: args.required_field,
      optional_field: args.optional_field,
      // Add any additional fields from custom processing
      result_data: result,
    };
    
    appendSafeOutput(entry);
    
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            success: true,
            message: "Your new type processed successfully",
            result: result,
          }),
        },
      ],
    };
  } catch (error) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: error instanceof Error ? error.message : String(error),
          }),
        },
      ],
      isError: true,
    };
  }
};
```

2. **Attach the handler to the tool** (around line 570-580):

```javascript
// Attach handlers to tools that need them
ALL_TOOLS.forEach(tool => {
  if (tool.name === "create_pull_request") {
    tool.handler = createPullRequestHandler;
  } else if (tool.name === "push_to_pull_request_branch") {
    tool.handler = pushToPullRequestBranchHandler;
  } else if (tool.name === "upload_asset") {
    tool.handler = uploadAssetHandler;
  } else if (tool.name === "your_new_type") {
    tool.handler = yourNewTypeHandler;  // Add your handler here
  }
});
```

**Default handler**: Normalizes type field, handles large content (>16000 tokens), writes JSONL, returns success.

### 5. Update Collection JavaScript (`pkg/workflow/js/collect_ndjson_output.ts`)

Add validation in main switch statement (~line 700):

```typescript
case "your-new-type":
  // Validate required fields
  if (!item.required_field || typeof item.required_field !== "string") {
    errors.push(`Line ${i + 1}: your-new-type requires a 'required_field' string field`);
    continue;
  }
  
  // Sanitize text content
  item.required_field = sanitizeContent(item.required_field);
  
  // Validate optional fields
  if (item.optional_field !== undefined) {
    if (typeof item.optional_field !== "string") {
      errors.push(`Line ${i + 1}: your-new-type 'optional_field' must be a string`);
      continue;
    }
    item.optional_field = sanitizeContent(item.optional_field);
  }
  break;
```

**Patterns**: Check required fields first, use `sanitizeContent()` for strings, use validation helpers for numbers, continue loop on errors.

### 6. Update Go Filter Function (`pkg/workflow/safe_outputs.go`)

Add to `enabledTools` map in `generateFilteredToolsJSON` (~line 1120):

```go
// generateFilteredToolsJSON filters the ALL_TOOLS array based on enabled safe outputs
// Returns a JSON string containing only the tools that are enabled in the workflow
func generateFilteredToolsJSON(data *WorkflowData) (string, error) {
	if data.SafeOutputs == nil {
		return "[]", nil
	}

	safeOutputsLog.Print("Generating filtered tools JSON for workflow")

	// Load the full tools JSON
	allToolsJSON := GetSafeOutputsToolsJSON()

	// Parse the JSON to get all tools
	var allTools []map[string]any
	if err := json.Unmarshal([]byte(allToolsJSON), &allTools); err != nil {
		return "", fmt.Errorf("failed to parse safe outputs tools JSON: %w", err)
	}

	// Create a set of enabled tool names
	enabledTools := make(map[string]bool)

	// Check which safe outputs are enabled and add their corresponding tool names
	if data.SafeOutputs.CreateIssues != nil {
		enabledTools["create_issue"] = true
	}
	// ... existing checks ...
	if data.SafeOutputs.YourNewType != nil {
		enabledTools["your_new_type"] = true  // Add your new type here
	}

	// Filter tools to only include enabled ones
	var filteredTools []map[string]any
	for _, tool := range allTools {
		toolName, ok := tool["name"].(string)
		if !ok {
			continue
		}
		if enabledTools[toolName] {
			filteredTools = append(filteredTools, tool)
		}
	}

	// Serialize filtered tools to JSON
	filteredJSON, err := json.Marshal(filteredTools)
	if err != nil {
		return "", fmt.Errorf("failed to marshal filtered tools: %w", err)
	}

	return string(filteredJSON), nil
}
```

**Flow**: Workflow config → parse to struct → filter tools → write JSON → MCP server exposes to agents.

### 7. Create JavaScript Implementation (`pkg/workflow/js/your_new_type.cjs`)

```javascript
async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const outputContent = process.env.GH_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all your-new-type items
  const items = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "your-new-type");
  if (items.length === 0) {
    core.info("No your-new-type items found in agent output");
    return;
  }

  core.info(`Found ${items.length} your-new-type item(s)`);

  // If in staged mode, emit step summary instead of performing actions
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Your New Type Preview\n\n";
    summaryContent += "The following actions would be performed if staged mode was disabled:\n\n";

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      summaryContent += `### Action ${i + 1}\n`;
      summaryContent += `**Required Field**: ${item.required_field}\n`;
      if (item.optional_field) {
        summaryContent += `**Optional Field**: ${item.optional_field}\n`;
      }
      summaryContent += "\n";
    }

    core.summary.addRaw(summaryContent).write();
    return;
  }

  // Process each item
  for (const item of items) {
    try {
      // Implement your actual logic here
      core.info(`Processing your-new-type: ${item.required_field}`);
      
      // Example GitHub API call pattern:
      // const result = await github.rest.yourApi.yourMethod({
      //   owner: context.repo.owner,
      //   repo: context.repo.repo,
      //   your_field: item.required_field,
      // });
      
      core.info("Successfully processed your-new-type item");
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to process your-new-type: ${errorMessage}`);
      core.setFailed(`Failed to process your-new-type: ${errorMessage}`);
      return;
    }
  }
}

// Call the main function
await main();
```

**Guidelines**: Check staged mode, use `core.*` methods (not console.log), use `core.summary` for previews, handle errors with try/catch.

### 8. Create Tests

**File**: `pkg/workflow/js/your_new_type.test.cjs` - Test empty input, valid processing, staged mode, errors. Use vitest.

**File**: `pkg/workflow/js/collect_ndjson_output.test.cjs` - Test validation with valid/invalid fields.

### 9. Create Test Workflows

Create for each engine (claude/codex/copilot) in `pkg/cli/workflows/`:

**Example**: `test-claude-your-new-type.md`

```markdown
---
on: workflow_dispatch
permissions:
  contents: read
  actions: read
engine: claude
safe-outputs:
  your-new-type:
    max: 3
    custom-option: "test"
timeout-minutes: 5
---

# Test Your New Type

Test the new safe output type functionality.

Create a your-new-type output with:
- required_field: "Hello World"  
- optional_field: "This is optional"

Output as JSONL format.
```

### 10. Create Go Job Builder (`pkg/workflow/your_new_type.go`)

Use `buildSafeOutputJob()` helper and shared config types:

```go
package workflow

import (
	"fmt"
)

// YourNewTypeConfig holds configuration for your new type from agent output
// Embed shared config types for common fields to reduce duplication
type YourNewTypeConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"` // Provides Target and TargetRepoSlug fields
	CustomOption           string           `yaml:"custom-option,omitempty"`  // Custom configuration option
	AnotherOption          *bool            `yaml:"another-option,omitempty"` // Another optional configuration
}

// buildCreateOutputYourNewTypeJob creates the your_new_type job using the shared builder
func (c *Compiler) buildCreateOutputYourNewTypeJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.YourNewType == nil {
		return nil, fmt.Errorf("safe-outputs.your-new-type configuration is required")
	}

	config := data.SafeOutputs.YourNewType

	// Build custom environment variables specific to your-new-type
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	
	// Add your custom configuration options as environment variables
	if config.CustomOption != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CUSTOM_OPTION: %q\n", config.CustomOption))
	}
	
	if config.AnotherOption != nil && *config.AnotherOption {
		customEnvVars = append(customEnvVars, "          GH_AW_ANOTHER_OPTION: \"true\"\n")
	}

	// Use shared env var builders for common fields
	customEnvVars = append(customEnvVars, BuildTargetEnvVar("GH_AW_YOUR_NEW_TYPE_TARGET", config.Target)...)
	customEnvVars = append(customEnvVars, BuildMaxCountEnvVar("GH_AW_YOUR_NEW_TYPE_MAX_COUNT", config.Max)...)

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		config.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if config.GitHubToken != "" {
		token = config.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"result_id":  "${{ steps.your_new_type.outputs.result_id }}",
		"result_url": "${{ steps.your_new_type.outputs.result_url }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "your_new_type",
		StepName:       "Execute Your New Type",
		StepID:         "your_new_type",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getYourNewTypeScript(),
		Permissions:    NewPermissionsContentsReadYourPermissions(),  // Adjust permissions as needed
		Outputs:        outputs,
		Token:          token,
		TargetRepoSlug: config.TargetRepoSlug,
	})
}

// parseYourNewTypeConfig handles your-new-type configuration using shared parsers
func (c *Compiler) parseYourNewTypeConfig(outputMap map[string]any) *YourNewTypeConfig {
	if configData, exists := outputMap["your-new-type"]; exists {
		yourNewTypeConfig := &YourNewTypeConfig{}
		yourNewTypeConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &yourNewTypeConfig.BaseSafeOutputConfig)

			// Parse target config using shared helper (handles target and target-repo)
			targetConfig, isInvalid := ParseTargetConfig(configMap)
			if isInvalid {
				// target-repo validation error (wildcard not allowed)
				return nil
			}
			yourNewTypeConfig.SafeOutputTargetConfig = targetConfig

			// Parse custom-option using generic string parser
			yourNewTypeConfig.CustomOption = ParseStringFromConfig(configMap, "custom-option")

			// Parse another-option (boolean example - no shared helper for booleans yet)
			if anotherOption, exists := configMap["another-option"]; exists {
				if anotherOptionBool, ok := anotherOption.(bool); ok {
					yourNewTypeConfig.AnotherOption = &anotherOptionBool
				}
			}
		}

		return yourNewTypeConfig
	}

	return nil
}

// getYourNewTypeScript returns the JavaScript implementation
func getYourNewTypeScript() string {
	return embedJavaScript("your_new_type.cjs")
}
```

**Key Points**:

1. **Use `buildSafeOutputJob()` helper** - Handles pre-steps, GitHub Script step with artifact download, post-steps, job creation

2. **Embed shared config types** from `safe_output_builder.go`:
   - `SafeOutputTargetConfig` - Target and TargetRepoSlug fields
   - `SafeOutputFilterConfig` - RequiredLabels and RequiredTitlePrefix
   - `SafeOutputDiscussionFilterConfig` - Adds RequiredCategory
   - `CloseJobConfig` - Target + filter for close operations
   - `ListJobConfig` - Target + allowed list

3. **Use parsing helpers**: `ParseTargetConfig()`, `ParseFilterConfig()`, `ParseCloseJobConfig()`, `ParseListJobConfig()`, `ParseStringFromConfig()`, etc.

4. **Use env var builders**: `BuildTargetEnvVar()`, `BuildRequiredLabelsEnvVar()`, `BuildCloseJobEnvVars()`, `BuildListJobEnvVars()`, etc.

5. **SafeOutputJobConfig struct** fields: `JobName`, `StepName`, `StepID`, `MainJobName`, `CustomEnvVars`, `Script`, `Permissions`, `Outputs`, `Token`, `PreSteps`, `PostSteps`, `Condition`, `Needs`

6. **Integration**: Add field to `SafeOutputsConfig` in `config.go`, call parser in `extractSafeOutputsConfig()` in `safe_outputs.go`, call builder in `compiler_jobs.go`

**Close Operations Example**:
```go
type CloseYourTypeConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	CloseJobConfig       `yaml:",inline"`
}
closeConfig, isInvalid := ParseCloseJobConfig(configMap)
customEnvVars = append(customEnvVars, BuildCloseJobEnvVars("GH_AW_CLOSE_YOUR_TYPE", config.CloseJobConfig)...)
```

**List Operations Example**:
```go
type AddYourTypeConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	ListJobConfig        `yaml:",inline"`
}
listConfig, isInvalid := ParseListJobConfig(configMap, "allowed")
customEnvVars = append(customEnvVars, BuildListJobEnvVars("GH_AW_ADD_YOUR_TYPE", config.ListJobConfig, config.Max)...)
```

**Permission Helpers**: Use from `pkg/workflow/permissions.go` - `NewPermissionsContentsRead()`, `NewPermissionsContentsReadIssuesWrite()`, etc.

### 11. Build and Test

```bash
make js fmt-cjs lint-cjs test-unit recompile agent-finish
```

### 12. Manual Validation

Test workflow with staged/non-staged modes, error handling, JSON schema validation, all engines.

## Success Criteria

- [ ] JSON schema validates correctly
- [ ] TypeScript types compile
- [ ] Tools JSON includes tool signature  
- [ ] MCP server handles type (custom handler if needed)
- [ ] Go filter includes type in `generateFilteredToolsJSON`
- [ ] Collection validates fields
- [ ] JavaScript implementation handles all cases
- [ ] Tests pass with good coverage
- [ ] Workflows compile
- [ ] Manual testing confirms functionality

## Common Pitfalls

1. Inconsistent naming across files (kebab-case/camelCase/underscores)
2. Missing tools.json update (agents can't call without it)
3. Missing Go filter update (MCP won't expose tool)
4. Missing field validation/sanitization
5. Not adding to union types
6. Not exporting interfaces
7. Test coverage gaps
8. Schema syntax violations
9. GitHub API error handling
10. Missing staged mode implementation
11. Forgetting `make build` after modifying embedded files

## References

- JSON Schema: https://json-schema.org/draft-07/schema
- GitHub Actions Core: https://github.com/actions/toolkit/tree/main/packages/core  
- GitHub REST API: https://docs.github.com/en/rest
- Vitest: https://vitest.dev/
- Existing implementations: `pkg/workflow/js/*.cjs`
