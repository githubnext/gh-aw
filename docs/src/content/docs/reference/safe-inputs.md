---
title: Safe Inputs
description: Define custom MCP tools inline as JavaScript or shell scripts with secret access, providing lightweight tool creation without external dependencies.
sidebar:
  order: 750
---

:::caution[Experimental Feature]
Safe Inputs is an experimental feature. The API and behavior may change in future releases.
:::

The [`safe-inputs:`](/gh-aw/reference/glossary/#safe-inputs) (validated user input tools) element allows you to define custom [MCP](/gh-aw/reference/glossary/#mcp-model-context-protocol) (Model Context Protocol) tools directly in your workflow [frontmatter](/gh-aw/reference/glossary/#frontmatter) using JavaScript, shell scripts, or Python. These tools are generated at runtime and mounted as an MCP server, giving your agent access to custom functionality with controlled secret access.

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
    timeout: 120                       # Optional: timeout in seconds (default: 60)
```

### Required Fields

- **`description:`** - Human-readable description of what the tool does. This is shown to the agent for tool selection.

### Optional Fields

- **`timeout:`** - Maximum execution time in seconds (default: 60). The tool will be terminated if it exceeds this duration. Applies to shell (`run:`) and Python (`py:`) tools.

### Implementation Options

Choose one implementation method:

- **`script:`** - JavaScript (CommonJS) code
- **`run:`** - Shell script
- **`py:`** - Python script (Python 3.1x)

You can only use one of `script:`, `run:`, or `py:` per tool.

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

### Shared gh CLI Tool

The repository includes a reusable shared workflow (`shared/gh.md`) that provides a general-purpose gh CLI tool:

```yaml wrap
imports:
  - shared/gh.md
```

This imports a `gh` tool that accepts any gh CLI command as arguments:

```yaml
# The agent can use:
gh with args: "pr list --limit 5"
gh with args: "issue view 123"
gh with args: "api repos/{owner}/{repo}"
```

The shared workflow uses `${{ github.token }}` for authentication, providing access based on the workflow's `permissions` configuration.

## Python Tools (`py:`)

Python tools execute using `python3` with inputs provided as a dictionary (similar to JavaScript tools):

```yaml wrap
safe-inputs:
  analyze-data:
    description: "Analyze data with Python"
    inputs:
      numbers:
        type: string
        description: "Comma-separated numbers"
        required: true
    py: |
      import json
      
      # Inputs are available as a dictionary
      numbers_str = inputs.get('numbers', '')
      numbers = [float(x.strip()) for x in numbers_str.split(',') if x.strip()]
      
      # Calculate statistics
      result = {
          "count": len(numbers),
          "sum": sum(numbers),
          "average": sum(numbers) / len(numbers) if numbers else 0
      }
      
      # Print result as JSON to stdout
      print(json.dumps(result))
```

### Accessing Input Parameters

Input parameters are automatically parsed from JSON and available in the `inputs` dictionary:

```python
# Access with .get() and optional default value
name = inputs.get('name', 'default')

# Access directly (may raise KeyError if missing)
required_param = inputs['required_param']

# Check if parameter exists
if 'optional_param' in inputs:
    process(inputs['optional_param'])
```

Parameter names with dashes are accessible as-is:
- `data-file` → `inputs.get('data-file')`
- `my-param` → `inputs.get('my-param')`

### Using Python Libraries

Python 3.10+ is available with standard library modules. For additional packages, you can install them inline:

```yaml wrap
safe-inputs:
  analyze-with-numpy:
    description: "Statistical analysis with NumPy"
    inputs:
      values:
        type: string
        required: true
    py: |
      import json
      import subprocess
      import sys
      
      # Install package if needed (for demonstration)
      # subprocess.check_call([sys.executable, "-m", "pip", "install", "--quiet", "numpy"])
      # import numpy as np
      
      # Get input from inputs dictionary
      values_str = inputs.get('values', '')
      values = [float(x.strip()) for x in values_str.split(',') if x.strip()]
      
      # Calculate statistics
      result = {"mean": sum(values) / len(values) if values else 0}
      print(json.dumps(result))
```

### Returning Results

Python scripts return results by printing JSON to stdout:

```yaml wrap
safe-inputs:
  process-data:
    description: "Process data and return results"
    inputs:
      text:
        type: string
        required: true
    py: |
      import json
      
      # Get input
      text = inputs.get('text', '')
      
      # Process and return result
      result = {
          "original": text,
          "uppercase": text.upper(),
          "length": len(text)
      }
      
      # Print result as JSON
      print(json.dumps(result))
```

### Accessing Environment Variables

Access secrets and environment variables via `os.environ`:

```yaml wrap
safe-inputs:
  fetch-api-data:
    description: "Fetch data from API using Python"
    inputs:
      endpoint:
        type: string
        required: true
    py: |
      import os
      import json
      try:
          from urllib import request
      except ImportError:
          import urllib.request as request
      
      # Get input from inputs dictionary
      endpoint = inputs.get('endpoint', '')
      
      # Get secret from environment
      api_key = os.environ.get('API_KEY', '')
      
      # Make API request
      url = f"https://api.example.com/{endpoint}"
      req = request.Request(url, headers={"Authorization": f"Bearer {api_key}"})
      
      with request.urlopen(req) as response:
          data = json.loads(response.read())
          print(json.dumps(data))
    env:
      API_KEY: "${{ secrets.API_KEY }}"
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

## Timeout Configuration

Each tool can specify a maximum execution time using the `timeout:` field. The default timeout is **60 seconds**.

### Default Timeout

If not specified, tools use a 60-second timeout:

```yaml wrap
safe-inputs:
  quick-task:
    description: "Runs with default 60s timeout"
    run: |
      echo "This will timeout after 60 seconds"
```

### Custom Timeout

Set a custom timeout for long-running operations:

```yaml wrap
safe-inputs:
  slow-processing:
    description: "Process large dataset"
    timeout: 300  # 5 minutes
    py: |
      import time
      import json
      
      # Long-running operation
      time.sleep(120)  # Simulate processing
      print(json.dumps({"status": "complete"}))
```

### Fast Timeout

Use shorter timeouts for quick operations:

```yaml wrap
safe-inputs:
  fast-check:
    description: "Quick health check"
    timeout: 10  # 10 seconds
    run: |
      curl -f https://api.example.com/health
```

### Timeout Enforcement

- **Shell tools (`run:`)**: Process is terminated after timeout
- **Python tools (`py:`)**: Process is terminated after timeout  
- **JavaScript tools (`script:`)**: Timeouts are not enforced (runs in-process)

When a tool exceeds its timeout, the execution is terminated and an error is returned to the agent.

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
| Languages | JavaScript, Shell, Python | Any language | Shell only |
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
