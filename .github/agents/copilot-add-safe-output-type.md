---
name: copilot-add-safe-output-type
description: Adding a New Safe Output Type to GitHub Agentic Workflows
---

# Copilot Agent: Adding a New Safe Output Type to GitHub Agentic Workflows

## Overview

You are tasked with adding a new safe output type to the GitHub Agentic Workflows system. This system processes AI agent outputs as JSONL (JSON Lines) format and validates them through a multi-step pipeline involving TypeScript types, JSON schema validation, and JavaScript collection logic.

## Background: Understanding Safe Output Types

Safe output types are structured data formats that AI agents can emit to perform GitHub actions safely. The system currently supports:

- `create-issue` - Create GitHub issues
- `add-comment` - Add comments to issues/PRs  
- `create-pull-request` - Create pull requests
- `create-pull-request-review-comment` - Add code review comments
- `add-labels` - Add labels to issues/PRs
- `update-issue` - Update existing issues
- `push-to-pull-request-branch` - Push commits to PR branches
- `create-discussion` - Create GitHub discussions
- `missing-tool` - Report missing functionality
- `create-code-scanning-alert` - Create SARIF security alerts
- `upload-asset` - Upload files as assets

## Overview

You are tasked with adding a new safe output type to the GitHub Agentic Workflows system. This system processes AI agent outputs as JSONL (JSON Lines) format and validates them through a multi-step pipeline involving TypeScript types, JSON schema validation, and JavaScript collection logic.

## Background: Understanding Safe Output Types

Safe output types are structured data formats that AI agents can emit to perform GitHub actions safely. The system currently supports:

- `create-issue` - Create GitHub issues
- `add-comment` - Add comments to issues/PRs  
- `create-pull-request` - Create pull requests
- `create-pull-request-review-comment` - Add code review comments
- `add-labels` - Add labels to issues/PRs
- `update-issue` - Update existing issues
- `push-to-pull-request-branch` - Push commits to PR branches
- `create-discussion` - Create GitHub discussions
- `missing-tool` - Report missing functionality
- `create-code-scanning-alert` - Create SARIF security alerts
- `upload-asset` - Upload files as assets

## Implementation Plan

### Step 1: Update JSON Schema

**File**: `schemas/agent-output.json`

1. Add a new object definition in the `$defs` section following the pattern:
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

2. Add your new type to the `SafeOutput` oneOf array:
   ```json
   {"$ref": "#/$defs/YourNewTypeOutput"}
   ```

**Validation Notes**:
- Use `const` for the type field to ensure exact matching
- Include `minLength: 1` for required string fields
- Set `additionalProperties: false` to prevent extra fields
- Use `oneOf` for union types like `[{"type": "number"}, {"type": "string"}]` for flexible numeric fields

### Step 2: Update TypeScript Types

**File**: `pkg/workflow/js/types/safe-outputs.d.ts`

1. Add your interface following the pattern:
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
   ```

2. Add to the `SafeOutputItem` union type:
   ```typescript
   type SafeOutputItem =
     | CreateIssueItem
     | AddCommentItem
     // ... existing types
     | YourNewTypeItem;
   ```

3. Add to the export list:
   ```typescript
   export {
     // ... existing exports
     YourNewTypeItem,
   };
   ```

**File**: `pkg/workflow/js/types/safe-outputs-config.d.ts`

1. Add configuration interface:
   ```typescript
   /**
    * Configuration for your new type
    */
   interface YourNewTypeConfig extends SafeOutputConfig {
     "custom-option"?: string;
     "another-option"?: boolean;
   }
   ```

2. Add to the `SpecificSafeOutputConfig` union type:
   ```typescript
   type SpecificSafeOutputConfig =
     | CreateIssueConfig
     // ... existing configs
     | YourNewTypeConfig;
   ```

3. Add to exports:
   ```typescript
   export {
     // ... existing exports
     YourNewTypeConfig,
   };
   ```

### Step 3: Update Safe Outputs Tools JSON

**File**: `pkg/workflow/js/safe_outputs_tools.json`

Add a tool signature for your new safe output type to expose it to AI agents through the MCP server. This file defines the tools that AI agents can call.

Add a new tool definition to the array:

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

**Tool Signature Guidelines**:
- Tool `name` must use underscores (e.g., `your_new_type`), matching the type field in the JSONL output
- The `name` field should match the safe output type name with underscores instead of hyphens
- Include a clear `description` explaining what the tool does
- Define an `inputSchema` with all fields the AI agent will provide
- Use `required` array for mandatory fields
- Set `additionalProperties: false` to prevent extra fields
- For numeric fields, use `"type": ["number", "string"]` to allow both types (agents sometimes send strings)
- Use descriptive field names and descriptions to guide AI agents

**Examples from existing tools**:
```json
{
  "name": "create_issue",
  "description": "Create a new GitHub issue",
  "inputSchema": {
    "type": "object",
    "required": ["title", "body"],
    "properties": {
      "title": { "type": "string", "description": "Issue title" },
      "body": { "type": "string", "description": "Issue body/description" },
      "labels": {
        "type": "array",
        "items": { "type": "string" },
        "description": "Issue labels"
      }
    },
    "additionalProperties": false
  }
}
```

### Step 4: Update Collection JavaScript

**File**: `pkg/workflow/js/collect_ndjson_output.ts`

Add validation logic in the main switch statement around line 700+:

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

**Validation Patterns**:
- Always check required fields first with `!item.field || typeof item.field !== "expected_type"`
- Use `sanitizeContent()` for all user-provided string content
- Use `validatePositiveInteger()` helper for numeric fields that must be positive
- Use `validateOptionalPositiveInteger()` for optional numeric fields
- Use `validateIssueOrPRNumber()` for GitHub issue/PR number fields
- Continue the loop on validation errors to process remaining items

### Step 5: Create JavaScript Implementation

**File**: `pkg/workflow/js/your_new_type.cjs`

Create the main implementation file:

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

**Implementation Guidelines**:
- Always start with staged mode check for preview functionality
- Use `core.info`, `core.error`, `core.setFailed` for logging (not console.log)
- Use `core.summary.addRaw().write()` for step summaries in staged mode
- Handle errors gracefully with try/catch blocks
- Use GitHub Actions context variables for repo information
- Follow the existing pattern for environment variable handling

### Step 6: Create Test File

**File**: `pkg/workflow/js/your_new_type.test.cjs`

Create comprehensive tests following existing patterns in the codebase:

**Testing Guidelines**:
- Test empty/missing input scenarios
- Test valid input processing  
- Test staged mode behavior
- Test error handling
- Use vitest framework with proper mocking
- Follow existing test patterns in `.test.cjs` files

### Step 7: Update Collection Tests

**File**: `pkg/workflow/js/collect_ndjson_output.test.cjs`

Add test cases for your new type following existing patterns:
- Test successful validation with valid fields
- Test validation errors with missing required fields  
- Test field type validation
- Follow existing test structure in the file

### Step 8: Create Test Agentic Workflows

Create test workflows for each supported engine to validate the new safe output type:

**Files**: 
- `pkg/cli/workflows/test-claude-your-new-type.md`
- `pkg/cli/workflows/test-codex-your-new-type.md` 
- `pkg/cli/workflows/test-copilot-your-new-type.md`

**Example**: `pkg/cli/workflows/test-claude-your-new-type.md`

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

### Step 9: Create Go Job Builder

**File**: `pkg/workflow/your_new_type.go`

Create the Go job builder using the refactored pattern with `buildSafeOutputJob()` helper:

```go
package workflow

import (
	"fmt"
)

// YourNewTypeConfig holds configuration for your new type from agent output
type YourNewTypeConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	CustomOption         string `yaml:"custom-option,omitempty"`      // Custom configuration option
	AnotherOption        *bool  `yaml:"another-option,omitempty"`     // Another optional configuration
	TargetRepoSlug       string `yaml:"target-repo,omitempty"`        // Target repository for cross-repo operations
}

// buildCreateOutputYourNewTypeJob creates the your_new_type job using the shared builder
func (c *Compiler) buildCreateOutputYourNewTypeJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.YourNewType == nil {
		return nil, fmt.Errorf("safe-outputs.your-new-type configuration is required")
	}

	// Build custom environment variables specific to your-new-type
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))
	
	// Add your custom configuration options as environment variables
	if data.SafeOutputs.YourNewType.CustomOption != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CUSTOM_OPTION: %q\n", data.SafeOutputs.YourNewType.CustomOption))
	}
	
	if data.SafeOutputs.YourNewType.AnotherOption != nil && *data.SafeOutputs.YourNewType.AnotherOption {
		customEnvVars = append(customEnvVars, "          GH_AW_ANOTHER_OPTION: \"true\"\n")
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.YourNewType.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.YourNewType != nil {
		token = data.SafeOutputs.YourNewType.GitHubToken
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
		TargetRepoSlug: data.SafeOutputs.YourNewType.TargetRepoSlug,
	})
}

// parseYourNewTypeConfig handles your-new-type configuration
func (c *Compiler) parseYourNewTypeConfig(outputMap map[string]any) *YourNewTypeConfig {
	if configData, exists := outputMap["your-new-type"]; exists {
		yourNewTypeConfig := &YourNewTypeConfig{}
		yourNewTypeConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &yourNewTypeConfig.BaseSafeOutputConfig)

			// Parse custom-option
			if customOption, exists := configMap["custom-option"]; exists {
				if customOptionStr, ok := customOption.(string); ok {
					yourNewTypeConfig.CustomOption = customOptionStr
				}
			}

			// Parse another-option
			if anotherOption, exists := configMap["another-option"]; exists {
				if anotherOptionBool, ok := anotherOption.(bool); ok {
					yourNewTypeConfig.AnotherOption = &anotherOptionBool
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			yourNewTypeConfig.TargetRepoSlug = targetRepoSlug
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

**Key Implementation Points**:

1. **Use `buildSafeOutputJob()` helper**: This is the new refactored pattern that eliminates duplicate code. The helper handles:
   - Pre-steps execution (if provided)
   - GitHub Script step creation with artifact download
   - Post-steps execution (if provided)
   - Job creation with standard metadata (timeout, runs-on, permissions)

2. **SafeOutputJobConfig struct**: Pass configuration via this struct:
   - `JobName`: Job identifier (e.g., "your_new_type")
   - `StepName`: Human-readable step name
   - `StepID`: Step identifier for outputs
   - `MainJobName`: Main workflow job for dependencies
   - `CustomEnvVars`: Your specific environment variables
   - `Script`: JavaScript implementation (from `your_new_type.cjs`)
   - `Permissions`: Job permissions using permission helpers
   - `Outputs`: Job outputs mapping
   - `Token`: GitHub token configuration
   - `PreSteps`: Optional steps before main script (e.g., checkout, git config)
   - `PostSteps`: Optional steps after main script (e.g., assignees, reviewers)
   - `Condition`: Optional custom job condition
   - `Needs`: Optional custom job dependencies

3. **No manual Job construction**: The helper creates the Job struct with:
   - Consistent 10-minute timeout
   - Standard runs-on configuration
   - Proper condition rendering
   - Automatic needs configuration

4. **Integration Points**:
   - Add `YourNewType *YourNewTypeConfig` field to `SafeOutputsConfig` struct in `pkg/workflow/config.go`
   - Call `parseYourNewTypeConfig()` in `extractSafeOutputsConfig()` in `pkg/workflow/safe_outputs.go`
   - Call `buildCreateOutputYourNewTypeJob()` in job creation logic in `pkg/workflow/compiler_jobs.go`

**Example with Pre/Post-Steps** (if needed):

```go
// Build pre-steps for special setup (like create-pull-request does for checkout)
var preSteps []string
preSteps = append(preSteps, "      - name: Setup step\n")
preSteps = append(preSteps, "        run: echo 'Setting up'\n")

// Build post-steps for additional actions (like create-issue does for assignees)
var postSteps []string
postSteps = append(postSteps, buildSpecialPostSteps(...)...)

return c.buildSafeOutputJob(data, SafeOutputJobConfig{
	// ... other fields ...
	PreSteps:  preSteps,   // Executed before GitHub Script step
	PostSteps: postSteps,  // Executed after GitHub Script step
})
```

**Adding Custom Conditions** (like update-issue does):

```go
// Build custom job condition
var jobCondition ConditionNode = BuildSafeOutputType("your_new_type")
if needsExtraCondition {
	extraCondition := BuildPropertyAccess("github.event.some.field")
	jobCondition = buildAnd(jobCondition, extraCondition)
}

return c.buildSafeOutputJob(data, SafeOutputJobConfig{
	// ... other fields ...
	Condition: jobCondition,  // Override default condition
})
```

**Permission Helpers**: Use existing helpers from `pkg/workflow/permissions.go`:
- `NewPermissionsContentsRead()` - Read-only content access
- `NewPermissionsContentsReadIssuesWrite()` - Content read + issue write
- `NewPermissionsContentsReadDiscussionsWrite()` - Content read + discussion write
- `NewPermissionsContentsReadIssuesWritePRWrite()` - Content read + issue/PR write
- `NewPermissionsContentsWriteIssuesWritePRWrite()` - Content write + issue/PR write

Or create a new one following the pattern:
```go
func NewPermissionsContentsReadYourPermissions() *Permissions {
	return NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
		PermissionContents:    PermissionRead,
		PermissionYourScope:   PermissionWrite,
	})
}
```

### Step 10: Build and Test

1. **Compile TypeScript**: `make js`
2. **Format code**: `make fmt-cjs`
3. **Run linting**: `make lint-cjs`
4. **Run tests**: `make test-unit`
5. **Compile workflows**: `make recompile`
6. **Full validation**: `make agent-finish`

### Step 11: Manual Validation

1. Create a simple test workflow using your new safe output type
2. Test both staged and non-staged modes
3. Verify error handling with invalid inputs
4. Ensure the JSON schema validation works correctly
5. Test with different engines (claude, codex, copilot)

## Key Success Criteria

- [ ] JSON schema validates your new type correctly
- [ ] TypeScript types compile without errors
- [ ] Collection logic validates fields properly
- [ ] JavaScript implementation handles all cases
- [ ] Tests achieve good coverage
- [ ] All existing tests still pass
- [ ] Workflows compile successfully
- [ ] Manual testing confirms functionality

## Common Pitfalls to Avoid

1. **Inconsistent naming**: Ensure type names match exactly across all files (kebab-case in JSON, camelCase in TypeScript)
2. **Missing validation**: Always validate required fields and sanitize string content
3. **Incorrect union types**: Add your new type to all relevant union types
4. **Missing exports**: Export all new interfaces and types
5. **Test coverage gaps**: Test both success and failure scenarios
6. **Schema violations**: Follow JSON Schema draft-07 syntax strictly
7. **GitHub API misuse**: Use proper error handling for API calls
8. **Staged mode**: Always implement preview functionality for staged mode

## Resources and References

- **JSON Schema**: https://json-schema.org/draft-07/schema
- **GitHub Actions Core**: https://github.com/actions/toolkit/tree/main/packages/core
- **GitHub REST API**: https://docs.github.com/en/rest
- **Vitest Testing**: https://vitest.dev/
- **Existing implementations**: See other `*.cjs` files in `pkg/workflow/js/`

Follow this plan methodically, testing each step before moving to the next. The modular approach ensures you can validate each component independently before integration.