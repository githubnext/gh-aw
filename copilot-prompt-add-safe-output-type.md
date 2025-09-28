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

### Step 3: Update Collection JavaScript

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

### Step 4: Create JavaScript Implementation

**File**: `pkg/workflow/js/your_new_type.cjs`

Create the main implementation file:

```javascript
async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
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

### Step 5: Create Test File

**File**: `pkg/workflow/js/your_new_type.test.cjs`

Create comprehensive tests:

```javascript
import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    // Mock your GitHub API calls here
  },
};

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
};

// Set up global mocks
globalThis.core = mockCore;
globalThis.github = mockGithub;
globalThis.context = mockContext;
globalThis.require = require;

describe("your_new_type.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Clear environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED;
  });

  const loadScript = () => {
    const scriptPath = new URL("./your_new_type.cjs", import.meta.url).pathname;
    return fs.readFileSync(scriptPath, "utf8");
  };

  it("should handle empty agent output", async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = "";
    
    const script = loadScript();
    await eval(`(async () => { ${script} })()`);
    
    expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
  });

  it("should process valid your-new-type items", async () => {
    const testOutput = {
      items: [
        {
          type: "your-new-type",
          required_field: "test value",
          optional_field: "optional value"
        }
      ],
      errors: []
    };
    
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(testOutput);
    
    const script = loadScript();
    await eval(`(async () => { ${script} })()`);
    
    expect(mockCore.info).toHaveBeenCalledWith("Found 1 your-new-type item(s)");
    // Add more specific assertions based on your implementation
  });

  it("should handle staged mode correctly", async () => {
    const testOutput = {
      items: [
        {
          type: "your-new-type", 
          required_field: "test value"
        }
      ],
      errors: []
    };
    
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(testOutput);
    process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED = "true";
    
    const script = loadScript();
    await eval(`(async () => { ${script} })()`);
    
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should handle missing items gracefully", async () => {
    const testOutput = {
      items: [
        {
          type: "create-issue",
          title: "Test",
          body: "Test body"
        }
      ],
      errors: []
    };
    
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(testOutput);
    
    const script = loadScript();
    await eval(`(async () => { ${script} })()`);
    
    expect(mockCore.info).toHaveBeenCalledWith("No your-new-type items found in agent output");
  });
});
```

**Testing Guidelines**:
- Test empty/missing input scenarios
- Test valid input processing
- Test staged mode behavior  
- Test error handling
- Use vitest framework with proper mocking
- Follow existing test patterns in the codebase

### Step 6: Update Collection Tests

**File**: `pkg/workflow/js/collect_ndjson_output.test.cjs`

Add test cases for your new type:

```javascript
it("should handle your-new-type validation", async () => {
  const testFile = "/tmp/test-ndjson-output.txt";
  const ndjsonContent = `{"type": "your-new-type", "required_field": "test value", "optional_field": "optional"}`;

  fs.writeFileSync(testFile, ndjsonContent);
  process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
  process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"your-new-type": true}';

  await eval(`(async () => { ${collectScript} })()`);

  const setOutputCalls = mockCore.setOutput.mock.calls;
  const outputCall = setOutputCalls.find(call => call[0] === "output");
  expect(outputCall).toBeDefined();

  const parsedOutput = JSON.parse(outputCall[1]);
  expect(parsedOutput.items).toHaveLength(1);
  expect(parsedOutput.items[0].type).toBe("your-new-type");
  expect(parsedOutput.items[0].required_field).toBe("test value");
  expect(parsedOutput.errors).toHaveLength(0);
});

it("should reject your-new-type items with missing required fields", async () => {
  const testFile = "/tmp/test-ndjson-output.txt";
  const ndjsonContent = `{"type": "your-new-type", "optional_field": "optional"}`;

  fs.writeFileSync(testFile, ndjsonContent);
  process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
  process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"your-new-type": true}';

  await eval(`(async () => { ${collectScript} })()`);

  const setOutputCalls = mockCore.setOutput.mock.calls;
  const outputCall = setOutputCalls.find(call => call[0] === "output");
  expect(outputCall).toBeDefined();

  const parsedOutput = JSON.parse(outputCall[1]);
  expect(parsedOutput.items).toHaveLength(0);
  expect(parsedOutput.errors).toHaveLength(1);
  expect(parsedOutput.errors[0]).toContain("requires a 'required_field' string field");
});
```

### Step 7: Create Test Agentic Workflows

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
timeout_minutes: 5
---

# Test Your New Type

Test the new safe output type functionality.

Create a your-new-type output with:
- required_field: "Hello World"  
- optional_field: "This is optional"

Output as JSONL format.
```

### Step 8: Build and Test

1. **Compile TypeScript**: `make js`
2. **Format code**: `make fmt-cjs`
3. **Run linting**: `make lint-cjs`
4. **Run tests**: `make test-unit`
5. **Compile workflows**: `make recompile`
6. **Full validation**: `make agent-finish`

### Step 9: Manual Validation

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