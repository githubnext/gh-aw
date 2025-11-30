---
title: Safe Inputs
description: Define custom MCP tools inline as JavaScript or shell scripts with secret access, providing lightweight tool creation without external dependencies.
sidebar:
  order: 750
---

The `safe-inputs:` element allows you to define custom MCP (Model Context Protocol) tools directly in your workflow frontmatter using JavaScript or shell scripts. These tools are generated at runtime and mounted as an MCP server, giving your agent access to custom functionality with controlled secret access.

## Quick Start

```yaml wrap
safe-inputs:
  greet-user:
    description: "Greet a user by name"
    inputs:
      name:
        type: string
        required: true
    script: |
      return { message: `Hello, ${name}!` };
```

The agent can now call `greet-user` with a `name` parameter.

## Tool Definition

Each safe-input tool requires a unique name and configuration:

```yaml wrap
safe-inputs:
  tool-name:
    description: "What the tool does"  # Required
    inputs:                            # Optional parameters
      param1:
        type: string
        required: true
        description: "Parameter description"
      param2:
        type: number
        default: 10
    script: |                          # JavaScript implementation
      // Your code here
    env:                               # Environment variables
      API_KEY: "${{ secrets.API_KEY }}"
```

### Required Fields

- **`description:`** - Human-readable description of what the tool does. This is shown to the agent for tool selection.

### Implementation Options

Choose one implementation method:

- **`script:`** - JavaScript (CommonJS) code
- **`run:`** - Shell script

You cannot use both `script:` and `run:` in the same tool.

## JavaScript Tools (`script:`)

JavaScript tools are automatically wrapped in an async function with destructured inputs. Write simple code without worrying about exports:

```yaml wrap
safe-inputs:
  calculate-sum:
    description: "Add two numbers"
    inputs:
      a:
        type: number
        required: true
      b:
        type: number
        required: true
    script: |
      const result = a + b;
      return { sum: result };
```

### Generated Code Structure

Your script is wrapped automatically:

```javascript
async function execute(inputs) {
  const { a, b } = inputs || {};
  
  // Your code here
  const result = a + b;
  return { sum: result };
}
module.exports = { execute };
```

### Accessing Environment Variables

Access secrets via `process.env`:

```yaml wrap
safe-inputs:
  fetch-data:
    description: "Fetch data from API"
    inputs:
      endpoint:
        type: string
        required: true
    script: |
      const apiKey = process.env.API_KEY;
      const response = await fetch(`https://api.example.com/${endpoint}`, {
        headers: { Authorization: `Bearer ${apiKey}` }
      });
      return await response.json();
    env:
      API_KEY: "${{ secrets.API_KEY }}"
```

### Async Operations

Scripts are async by default. Use `await` freely:

```yaml wrap
safe-inputs:
  slow-operation:
    description: "Perform async operation"
    script: |
      await new Promise(resolve => setTimeout(resolve, 1000));
      return { status: "completed" };
```

## Shell Tools (`run:`)

Shell scripts execute in bash with input parameters as environment variables:

```yaml wrap
safe-inputs:
  list-prs:
    description: "List pull requests"
    inputs:
      repo:
        type: string
        required: true
      state:
        type: string
        default: "open"
    run: |
      gh pr list --repo "$INPUT_REPO" --state "$INPUT_STATE" --json number,title
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
```

### Input Variable Naming

Input parameters are converted to environment variables:
- `repo` → `INPUT_REPO`
- `state` → `INPUT_STATE`
- `my-param` → `INPUT_MY_PARAM`

### Using gh CLI

Shell scripts can use the GitHub CLI when `GH_TOKEN` is provided:

```yaml wrap
safe-inputs:
  search-issues:
    description: "Search issues in a repository"
    inputs:
      query:
        type: string
        required: true
    run: |
      gh issue list --search "$INPUT_QUERY" --json number,title,state
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
```

## Input Parameters

Define typed parameters with validation:

```yaml wrap
safe-inputs:
  example-tool:
    description: "Example with all input options"
    inputs:
      required-param:
        type: string
        required: true
        description: "This parameter is required"
      optional-param:
        type: number
        default: 42
        description: "This has a default value"
      choice-param:
        type: string
        enum: ["option1", "option2", "option3"]
        description: "Limited to specific values"
```

### Supported Types

- `string` - Text values
- `number` - Numeric values
- `boolean` - True/false values
- `array` - List of values
- `object` - Structured data

### Validation Options

- `required: true` - Parameter must be provided
- `default: value` - Default if not provided
- `enum: [...]` - Restrict to specific values
- `description: "..."` - Help text for the agent

## Environment Variables (`env:`)

Pass secrets and configuration to tools:

```yaml wrap
safe-inputs:
  secure-tool:
    description: "Tool with multiple secrets"
    script: |
      const { API_KEY, API_SECRET } = process.env;
      // Use secrets...
    env:
      API_KEY: "${{ secrets.SERVICE_API_KEY }}"
      API_SECRET: "${{ secrets.SERVICE_API_SECRET }}"
      CUSTOM_VAR: "static-value"
```

Environment variables are:
- Passed securely to the MCP server process
- Available in JavaScript via `process.env`
- Available in shell via `$VAR_NAME`
- Masked in logs when using `${{ secrets.* }}`

## Large Output Handling

When tool output exceeds 500 characters, it's automatically saved to a file:

```json
{
  "status": "output_saved_to_file",
  "file_path": "/tmp/gh-aw/safe-inputs/calls/call_1732831234567_1.txt",
  "file_size_bytes": 2500,
  "file_size_chars": 2500,
  "message": "Output was too large. Read the file for full content.",
  "json_schema_preview": "{\"type\": \"array\", \"length\": 50, ...}"
}
```

The agent receives:
- File path to read the full output
- File size information
- JSON schema preview (if output is valid JSON)

## Importing Safe Inputs

Import tools from shared workflows:

```yaml wrap
imports:
  - shared/github-tools.md
```

**Shared workflow (`shared/github-tools.md`):**

```yaml wrap
---
safe-inputs:
  fetch-pr-data:
    description: "Fetch PR data from GitHub"
    inputs:
      repo:
        type: string
      search:
        type: string
    run: |
      gh pr list --repo "$INPUT_REPO" --search "$INPUT_SEARCH" --json number,title,state
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
---
```

Tools from imported workflows are merged with local definitions. Local tools take precedence on name conflicts.

## Complete Example

A workflow using multiple safe-input tools:

```yaml wrap
---
on: workflow_dispatch
engine: copilot
imports:
  - shared/pr-data-safe-input.md
safe-inputs:
  analyze-text:
    description: "Analyze text and return statistics"
    inputs:
      text:
        type: string
        required: true
    script: |
      const words = text.split(/\s+/).filter(w => w.length > 0);
      const chars = text.length;
      const sentences = text.split(/[.!?]+/).filter(s => s.trim().length > 0);
      return {
        word_count: words.length,
        char_count: chars,
        sentence_count: sentences.length,
        avg_word_length: (chars / words.length).toFixed(2)
      };
  
  format-date:
    description: "Format a date string"
    inputs:
      date:
        type: string
        required: true
      format:
        type: string
        default: "ISO"
        enum: ["ISO", "US", "EU"]
    script: |
      const d = new Date(date);
      switch (format) {
        case "US": return { formatted: d.toLocaleDateString("en-US") };
        case "EU": return { formatted: d.toLocaleDateString("en-GB") };
        default: return { formatted: d.toISOString() };
      }
safe-outputs:
  create-discussion:
    category: "General"
---

# Text Analysis Workflow

Analyze provided text and create a discussion with the results.

Use the `analyze-text` tool to get text statistics.
Use the `fetch-pr-data` tool to get PR information if needed.
```

## Security Considerations

- **Secret Isolation**: Each tool only receives the secrets specified in its `env:` field
- **Process Isolation**: Tools run in separate processes, isolated from the main workflow
- **Output Sanitization**: Large outputs are saved to files to prevent context overflow
- **No Arbitrary Execution**: Only predefined tools are available to the agent

## Comparison with Other Options

| Feature | Safe Inputs | Custom MCP Servers | Bash Tool |
|---------|-------------|-------------------|-----------|
| Setup | Inline in frontmatter | External service | Simple commands |
| Languages | JavaScript, Shell | Any language | Shell only |
| Secret Access | Controlled via `env:` | Full access | Workflow env |
| Isolation | Process-level | Service-level | None |
| Best For | Custom logic | Complex integrations | Simple commands |

## Troubleshooting

### Tool Not Found

Ensure the tool name in `safe-inputs:` matches exactly what the agent calls.

### Script Errors

Check the workflow logs for JavaScript syntax errors. The MCP server logs detailed error messages.

### Secret Not Available

Verify the secret name in `env:` matches a secret in your repository or organization.

### Large Output Issues

If outputs are truncated, the agent should read the file path provided in the response.

## Related Documentation

- [Tools](/gh-aw/reference/tools/) - Other tool configuration options
- [Imports](/gh-aw/reference/imports/) - Importing shared workflows
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Automated post-workflow actions
- [MCPs](/gh-aw/guides/mcps/) - External MCP server integration
- [Custom Safe Output Jobs](/gh-aw/guides/custom-safe-outputs/) - Post-workflow custom jobs
