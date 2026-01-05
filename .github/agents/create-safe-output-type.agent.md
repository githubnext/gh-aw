---
description: Adding a New Safe Output Type to GitHub Agentic Workflows
infer: false
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

### 7. Create Handler Implementation (`actions/setup/js/your_new_type.cjs`)

Create a handler factory that returns a message processing function. The handler manager will call this factory once during initialization and use the returned function to process each message.

```javascript
// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { generateTemporaryId } = require("./temporary_id.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "your_new_type";

/**
 * Main handler factory for your_new_type
 * Returns a message handler function that processes individual your_new_type messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract and log configuration
  const customOption = config.custom_option || "";
  const maxCount = config.max || 10;
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Custom option: ${customOption}`);
  core.info(`Max count: ${maxCount}`);
  core.info(`Staged mode: ${isStaged}`);

  // Track handler state
  let processedCount = 0;
  const processedItems = [];

  /**
   * Message handler function that processes a single your_new_type message
   * @param {Object} message - The your_new_type message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleYourNewType(message, resolvedTemporaryIds) {
    // Check max count
    if (processedCount >= maxCount) {
      core.warning(`Skipping your_new_type: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Validate required fields
    if (!item.required_field) {
      core.warning("Skipping your_new_type: required_field is missing");
      return {
        success: false,
        error: "required_field is required",
      };
    }

    // Generate temporary ID if not provided
    const temporaryId = item.temporary_id || generateTemporaryId();
    core.info(`Processing your_new_type: required_field=${item.required_field}, temporaryId=${temporaryId}`);

    // Staged mode: collect for preview
    if (isStaged) {
      processedItems.push({
        required_field: item.required_field,
        optional_field: item.optional_field,
        temporaryId,
      });

      return {
        success: true,
        staged: true,
        temporaryId,
      };
    }

    // Process the message
    try {
      // Implement your GitHub API call or custom logic here
      core.info(`Processing your-new-type: ${item.required_field}`);
      
      // Example GitHub API pattern:
      // const result = await github.rest.yourApi.yourMethod({
      //   owner: context.repo.owner,
      //   repo: context.repo.repo,
      //   your_field: item.required_field,
      // });
      
      // Simulate successful processing
      const resultId = 123; // Replace with actual result ID
      const resultUrl = `https://github.com/example/result/${resultId}`;

      core.info(`✓ Successfully processed your-new-type: ${resultUrl}`);

      return {
        success: true,
        temporaryId,
        resultId,
        resultUrl,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`✗ Failed to process your-new-type: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
```

**Key Handler Factory Patterns**:

1. **Factory Function**: `main(config)` is called once during initialization
2. **Closure State**: Variables in factory scope persist across messages (e.g., `processedCount`)
3. **Message Handler**: Factory returns an async function that processes individual messages
4. **Handler Signature**: `async (message, resolvedTemporaryIds) => { success, error?, ... }`
5. **Max Count Enforcement**: Check `processedCount` before processing each message
6. **Staged Mode**: Collect items for preview instead of executing operations
7. **Temporary IDs**: Generate or use provided temporary IDs for cross-referencing
8. **Error Handling**: Return `{ success: false, error }` instead of throwing
9. **Result Object**: Include fields needed for outputs or temporary ID resolution

**Available Helper Modules**:
- `error_helpers.cjs` - `getErrorMessage(error)` for consistent error formatting
- `temporary_id.cjs` - `generateTemporaryId()`, `isTemporaryId()`, `normalizeTemporaryId()`
- `repo_helpers.cjs` - `parseRepoSlug()`, `validateRepo()`, `getDefaultTargetRepo()`
- `sanitize_label_content.cjs` - `sanitizeLabelContent()` for label validation
- `generate_footer.cjs` - `generateFooter()` for AI-generated message footers

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

### 10. Integrate with Handler Manager

Most safe output types are now processed through the **handler manager** architecture, which provides centralized message dispatching, temporary ID resolution, and consistent error handling. The handler manager loads individual message handlers for each enabled safe output type and orchestrates their execution.

#### Architecture Overview

The handler manager (`actions/setup/js/safe_output_handler_manager.cjs`) acts as a dispatcher:
1. Loads configuration from `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` environment variable
2. Initializes handler factories for enabled safe output types
3. Reads and validates agent output messages
4. Dispatches messages to appropriate handlers
5. Manages temporary ID resolution across handlers
6. Collects results and outputs

#### Handler Factory Pattern

Each safe output type implements a **factory function** that returns a message handler:

```javascript
/**
 * Main handler factory for your_new_type
 * Returns a message handler function that processes individual your_new_type messages
 * @param {Object} config - Configuration object from workflow YAML
 * @returns {Promise<Function>} Message handler function
 */
async function main(config = {}) {
  // 1. Extract and log configuration
  const customOption = config.custom_option || "";
  const maxCount = config.max || 10;
  
  core.info(`Custom option: ${customOption}`);
  core.info(`Max count: ${maxCount}`);
  
  // 2. Initialize handler state (if needed)
  let processedCount = 0;
  const temporaryIdMap = new Map();
  
  // 3. Return the message handler function
  return async function handleYourNewType(message, resolvedTemporaryIds) {
    // Check max count
    if (processedCount >= maxCount) {
      core.warning(`Skipping your_new_type: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }
    
    processedCount++;
    
    // Process the message
    try {
      // Your implementation here
      const result = await processYourNewType(message);
      
      // Return success with any outputs
      return {
        success: true,
        // Add any outputs needed for temporary ID resolution or step outputs
        temporaryId: message.temporary_id,
        // ... other fields
      };
    } catch (error) {
      core.error(`Failed to process your_new_type: ${error.message}`);
      return {
        success: false,
        error: error.message,
      };
    }
  };
}

module.exports = { main };
```

#### Integration Steps

**Step 1: Add Config Type** in `pkg/workflow/frontmatter_types.go` or create a new file (e.g., `pkg/workflow/your_new_type.go`):

```go
// YourNewTypeConfig holds configuration for your new type from agent output
type YourNewTypeConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	CustomOption         string `yaml:"custom-option,omitempty"`
}
```

**Step 2: Add Config Parser** (in the same file as the config type):

```go
// parseYourNewTypeConfig handles your-new-type configuration
func (c *Compiler) parseYourNewTypeConfig(outputMap map[string]any) *YourNewTypeConfig {
	if configData, exists := outputMap["your-new-type"]; exists {
		yourNewTypeConfig := &YourNewTypeConfig{}
		yourNewTypeConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &yourNewTypeConfig.BaseSafeOutputConfig)

			// Parse custom fields
			yourNewTypeConfig.CustomOption = ParseStringFromConfig(configMap, "custom-option")
		}

		return yourNewTypeConfig
	}

	return nil
}
```

**Step 3: Register in Handler Manager** in `actions/setup/js/safe_output_handler_manager.cjs`:

Add your handler to the `HANDLER_MAP`:

```javascript
const HANDLER_MAP = {
  create_issue: "./create_issue.cjs",
  add_comment: "./add_comment.cjs",
  // ... existing handlers ...
  your_new_type: "./your_new_type.cjs",  // Add your handler here
};
```

**Step 4: Add Handler Config to Compiler** in `pkg/workflow/compiler_safe_outputs_config.go`:

Update `addHandlerManagerConfigEnvVar()` to include your config:

```go
if data.SafeOutputs.YourNewType != nil {
	handlerConfig := map[string]any{
		"custom_option": data.SafeOutputs.YourNewType.CustomOption,
		"max":           data.SafeOutputs.YourNewType.Max,
	}
	config["your_new_type"] = handlerConfig
}
```

**Step 5: Add to Consolidated Job Check** in `pkg/workflow/compiler_safe_outputs_job.go`:

Update `hasHandlerManagerTypes` condition:

```go
hasHandlerManagerTypes := data.SafeOutputs.CreateIssues != nil ||
	data.SafeOutputs.AddComments != nil ||
	// ... existing checks ...
	data.SafeOutputs.YourNewType != nil
```

Add permissions:

```go
if data.SafeOutputs.YourNewType != nil {
	permissions.Merge(NewPermissionsContentsReadYourPermissions())
}
```

#### Standalone Step Alternative

If your safe output type requires operations **before or after** message processing (e.g., git checkout, file operations), use a standalone step instead:

```go
// In pkg/workflow/compiler_safe_outputs_specialized.go or a new file
func (c *Compiler) buildYourNewTypeStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.YourNewType

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("your_new_type")

	return SafeOutputStepConfig{
		StepName:      "Execute Your New Type",
		StepID:        "your_new_type",
		ScriptName:    "your_new_type",
		Script:        getYourNewTypeScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}
```

Then integrate in `buildConsolidatedSafeOutputsJob()`:

```go
if data.SafeOutputs.YourNewType != nil {
	stepConfig := c.buildYourNewTypeStepConfig(data, mainJobName, threatDetectionEnabled)
	stepYAML := c.buildConsolidatedSafeOutputStep(data, stepConfig)
	steps = append(steps, stepYAML...)
	safeOutputStepNames = append(safeOutputStepNames, stepConfig.StepID)

	outputs["your_new_type_result"] = "${{ steps.your_new_type.outputs.result }}"
	permissions.Merge(NewPermissionsContentsReadYourPermissions())
}
```

Add to `STANDALONE_STEP_TYPES` in handler manager:

```javascript
const STANDALONE_STEP_TYPES = new Set([
  "assign_to_agent",
  "create_agent_task", 
  "update_project",
  "upload_asset",
  "your_new_type",  // Add if using standalone step
]);
```

#### Key Integration Points

1. **Config Type**: Define in `pkg/workflow/*.go` with `BaseSafeOutputConfig` embedding
2. **Config Parser**: Parse YAML config and extract typed fields
3. **Handler Registration**: Add to `HANDLER_MAP` in handler manager
4. **Handler Config**: Add to `addHandlerManagerConfigEnvVar()` for runtime configuration
5. **Job Integration**: Add to `hasHandlerManagerTypes` check and permissions
6. **Handler Implementation**: Create factory function in `actions/setup/js/your_new_type.cjs`

#### Shared Helpers

**Config Types**:
- `BaseSafeOutputConfig` - Common fields (max, github-token, staged)
- `SafeOutputTargetConfig` - Target repo configuration

**Parsers**:
- `ParseStringFromConfig()` - Parse string field
- `ParseTargetConfig()` - Parse target/target-repo
- `parseBaseSafeOutputConfig()` - Parse base fields

**Handler Helpers** (in `actions/setup/js/`):
- `load_agent_output.cjs` - Load and parse agent output
- `temporary_id.cjs` - Temporary ID generation and resolution
- `repo_helpers.cjs` - Repository parsing and validation
- `error_helpers.cjs` - Error message formatting
- `sanitize_label_content.cjs` - Label sanitization
- `generate_footer.cjs` - AI-generated message footer


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
- [ ] Handler factory function implemented (returns message handler)
- [ ] Handler registered in HANDLER_MAP or STANDALONE_STEP_TYPES
- [ ] Handler config added to `addHandlerManagerConfigEnvVar()`
- [ ] Permissions added to consolidated job
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
12. Handler factory not returning a function (must return async message handler)
13. Forgetting to add handler to HANDLER_MAP in safe_output_handler_manager.cjs
14. Not adding handler config to addHandlerManagerConfigEnvVar() in compiler
15. Missing hasHandlerManagerTypes check for consolidated job integration

## References

- JSON Schema: https://json-schema.org/draft-07/schema
- GitHub Actions Core: https://github.com/actions/toolkit/tree/main/packages/core  
- GitHub REST API: https://docs.github.com/en/rest
- Vitest: https://vitest.dev/
- Handler Manager: `actions/setup/js/safe_output_handler_manager.cjs`
- Existing Handler Implementations: `actions/setup/js/create_issue.cjs`, `actions/setup/js/add_comment.cjs`, etc.
- Compiler Integration: `pkg/workflow/compiler_safe_outputs_*.go`
