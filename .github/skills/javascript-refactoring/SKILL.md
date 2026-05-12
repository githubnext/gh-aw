---
name: javascript-refactoring
description: Instructions for refactoring JavaScript code into separate files
---


# JavaScript Code Refactoring Guide

This guide explains how to refactor JavaScript code into a separate `.cjs` file in the gh-aw repository. Follow these steps when extracting shared functionality or creating new JavaScript modules.

## Overview

The gh-aw project uses CommonJS modules (`.cjs` files) for JavaScript code that runs in GitHub Actions workflows. These files are:
- Embedded in the Go binary using `//go:embed` directives
- Bundled using a custom JavaScript bundler that inlines local `require()` calls
- Executed in GitHub Actions using `actions/github-script@v8`

### Top-Level Script Pattern

Top-level `.cjs` scripts (those that are executed directly in workflows) follow a specific pattern:

**✅ Correct Pattern - Export main, but don't call it:**
```javascript
async function main() {
  // Script logic here
  core.info("Running the script");
}

module.exports = { main };
```

**❌ Incorrect Pattern - Don't call main in the file:**
```javascript
async function main() {
  // Script logic here
  core.info("Running the script");
}

await main(); // ❌ Don't do this!

module.exports = { main };
```

**Why this pattern?**
- The bundler automatically injects `await main()` during inline execution in GitHub Actions
- This allows the script to be both imported (for testing) and executed (in workflows)
- It provides a clean separation between module definition and execution
- It enables better testing by allowing tests to import and call `main()` with mocks

**Examples of top-level scripts:**
- `create_issue.cjs` - Creates GitHub issues
- `add_comment.cjs` - Adds comments to issues/PRs
- `add_labels.cjs` - Adds labels to issues/PRs
- `update_project.cjs` - Updates GitHub Projects

All of these files export `main` but do not call it directly.

## Step 1: Create the New .cjs File

Create your new file in `/home/runner/work/gh-aw/gh-aw/pkg/workflow/js/` with a descriptive name:

**File naming convention:**
- Use snake_case for filenames (e.g., `sanitize_content.cjs`, `load_agent_output.cjs`)
- Use `.cjs` extension (CommonJS module)
- Choose names that clearly describe the module's purpose

**Example file structure:**
```javascript
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Brief description of what this module does
 */

/**
 * Function documentation
 * @param {string} input - Description of parameter
 * @returns {string} Description of return value
 */
function myFunction(input) {
  // Implementation
  return input;
}

// Export the function(s)
module.exports = {
  myFunction,
};
```

**Key points:**
- Include `// @ts-check` for TypeScript checking
- Include `/// <reference types="@actions/github-script" />` for GitHub Actions types
- Use JSDoc comments for documentation
- Export functions using `module.exports = { ... }`
- Do NOT import `@actions/core` or `@actions/github` - these are available globally in GitHub Actions

## Step 2: Add Tests

Create a test file with the same base name plus `.test.cjs`:

**Example: `pkg/workflow/js/my_module.test.cjs`**
```javascript
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global mocks before importing the module
global.core = mockCore;

describe("myFunction", () => {
  beforeEach(() => {
    // Reset mocks before each test
    vi.clearAllMocks();
  });

  it("should handle basic input", async () => {
    // Import the module to test
    const { myFunction } = await import("./my_module.cjs");
    
    const result = myFunction("test input");
    
    expect(result).toBe("expected output");
  });

  it("should handle edge cases", async () => {
    const { myFunction } = await import("./my_module.cjs");
    
    const result = myFunction("");
    
    expect(result).toBe("");
  });
});
```

**Testing guidelines:**
- Use vitest for testing framework
- Mock `core` and `github` globals as needed
- Use dynamic imports (`await import()`) to allow mocking before module load
- Clear mocks in `beforeEach` to ensure test isolation
- Test both success cases and error handling
- Follow existing test patterns in `pkg/workflow/js/*.test.cjs` files

**Run tests:**
```bash
make test-js
```

## Step 3: Add Embedded Variable in Go

Add an `//go:embed` directive and variable in the appropriate Go file:

### For shared utility functions (used by multiple scripts):

Add to **`pkg/workflow/js.go`**:

```go
//go:embed js/my_module.cjs
var myModuleScript string
```

Then add to the `GetJavaScriptSources()` function:

```go
func GetJavaScriptSources() map[string]string {
	return map[string]string{
		"sanitize_content.cjs":       sanitizeContentScript,
		"sanitize_label_content.cjs": sanitizeLabelContentScript,
		"sanitize_workflow_name.cjs": sanitizeWorkflowNameScript,
		"load_agent_output.cjs":      loadAgentOutputScript,
		"staged_preview.cjs":         stagedPreviewScript,
		"is_truthy.cjs":              isTruthyScript,
		"my_module.cjs":              myModuleScript,  // Add this line
	}
}
```

### For main scripts (top-level scripts that use bundling):

Add to **`pkg/workflow/scripts.go`**:

```go
//go:embed js/my_script.cjs
var myScriptSource string
```

Then create a getter function with bundling:

```go
var (
	myScript     string
	myScriptOnce sync.Once
)

// getMyScript returns the bundled my_script script
// Bundling is performed on first access and cached for subsequent calls
func getMyScript() string {
	myScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(myScriptSource, sources, "")
		if err != nil {
			scriptsLog.Printf("Bundling failed for my_script, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			myScript = myScriptSource
		} else {
			myScript = bundled
		}
	})
	return myScript
}
```

**Important:** 
- Variables in `js.go` are for **shared utilities** that get bundled into other scripts
- Variables in `scripts.go` are for **main scripts** that use the bundler to inline dependencies
- Use `sync.Once` pattern for lazy bundling in `scripts.go`
- The bundler will inline all local `require()` calls at runtime

## Step 4: Register in the Bundler (if creating a shared utility)

If you're creating a shared utility that will be used by other scripts via `require()`, it's automatically available through the `GetJavaScriptSources()` map (Step 3).

**The bundler will:**
1. Detect `require('./my_module.cjs')` in any script
2. Look up the file in the `GetJavaScriptSources()` map
3. Inline the required module's content
4. Remove the `require()` statement
5. Deduplicate if the same module is required multiple times

**No additional bundler registration needed** - just ensure the file is in the `GetJavaScriptSources()` map.

## Step 5: Use Local Require in Other JavaScript Files

To use your new module in other JavaScript files, use CommonJS `require()`:

**Example usage in another `.cjs` file:**
```javascript
// @ts-check
/// <reference types="@actions/github-script" />

const { myFunction } = require("./my_module.cjs");

async function main() {
  const result = myFunction("some input");
  core.info(`Result: ${result}`);
}

module.exports = { main };
```

**Important:** Top-level scripts should export `main` but **NOT** call it directly. The bundler injects `await main()` during inline execution in GitHub Actions.

**Require guidelines:**
- Use relative paths starting with `./`
- Include the `.cjs` extension
- Use destructuring to import specific functions
- The bundler will inline the required module at compile time

**Multiple requires example:**
```javascript
const { sanitizeContent } = require("./sanitize_content.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
```

## Complete Example: Creating a New Utility Module

Let's walk through creating a new `format_timestamp.cjs` utility:

### 1. Create the file: `pkg/workflow/js/format_timestamp.cjs`

```javascript
// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Formats a timestamp to ISO 8601 format
 * @param {Date|string|number} timestamp - Timestamp to format
 * @returns {string} ISO 8601 formatted timestamp
 */
function formatTimestamp(timestamp) {
  const date = timestamp instanceof Date ? timestamp : new Date(timestamp);
  return date.toISOString();
}

/**
 * Formats a timestamp to a human-readable string
 * @param {Date|string|number} timestamp - Timestamp to format
 * @returns {string} Human-readable timestamp
 */
function formatTimestampHuman(timestamp) {
  const date = timestamp instanceof Date ? timestamp : new Date(timestamp);
  return date.toLocaleString('en-US', { 
    dateStyle: 'medium', 
    timeStyle: 'short' 
  });
}

module.exports = {
  formatTimestamp,
  formatTimestampHuman,
};
```

### 2. Create tests: `pkg/workflow/js/format_timestamp.test.cjs`

```javascript
import { describe, it, expect } from "vitest";

describe("formatTimestamp", () => {
  it("should format Date object to ISO 8601", async () => {
    const { formatTimestamp } = await import("./format_timestamp.cjs");
    const date = new Date('2024-01-15T12:30:00Z');
    
    const result = formatTimestamp(date);
    
    expect(result).toBe('2024-01-15T12:30:00.000Z');
  });

  it("should format timestamp number to ISO 8601", async () => {
    const { formatTimestamp } = await import("./format_timestamp.cjs");
    const timestamp = 1705323000000; // Jan 15, 2024 12:30:00 UTC
    
    const result = formatTimestamp(timestamp);
    
    expect(result).toBe('2024-01-15T12:30:00.000Z');
  });
});

describe("formatTimestampHuman", () => {
  it("should format Date object to human-readable string", async () => {
    const { formatTimestampHuman } = await import("./format_timestamp.cjs");
    const date = new Date('2024-01-15T12:30:00Z');
    
    const result = formatTimestampHuman(date);
    
    expect(result).toContain('Jan');
    expect(result).toContain('15');
    expect(result).toContain('2024');
  });
});
```

### 3. Add to `pkg/workflow/js.go`:

```go
//go:embed js/format_timestamp.cjs
var formatTimestampScript string

func GetJavaScriptSources() map[string]string {
	return map[string]string{
		// ... existing entries ...
		"format_timestamp.cjs": formatTimestampScript,
	}
}
```

### 4. Use in another script:

```javascript
// @ts-check
/// <reference types="@actions/github-script" />

const { formatTimestamp } = require("./format_timestamp.cjs");

async function main() {
  const now = new Date();
  core.info(`Current time: ${formatTimestamp(now)}`);
}

module.exports = { main };
```

**Note:** The script exports `main` but does not call it. The bundler will inject `await main()` when the script is executed inline in GitHub Actions.

### 5. Build and test:

```bash
# Format the code
make fmt-cjs

# Run JavaScript tests
make test-js

# Run Go tests (includes bundler tests)
make test-unit

# Build the binary (embeds JavaScript files)
make build
```

## Verification Checklist

Before committing your refactored code:

- [ ] New `.cjs` file created in `pkg/workflow/js/`
- [ ] Tests created in corresponding `.test.cjs` file
- [ ] Tests pass with `make test-js`
- [ ] Embedded variable added in `pkg/workflow/js.go` or `pkg/workflow/scripts.go`
- [ ] If utility: Added to `GetJavaScriptSources()` map
- [ ] If main script: Created bundling getter function with `sync.Once`
- [ ] Local `require()` statements work correctly in other files
- [ ] Code formatted with `make fmt-cjs`
- [ ] Code linted with `make lint-cjs`
- [ ] All Go tests pass with `make test-unit`
- [ ] Build succeeds with `make build`

## Common Patterns

### Pattern 1: Shared Utility Function

Files like `sanitize_content.cjs`, `load_agent_output.cjs` that provide reusable functions:
- Add to `js.go` with `//go:embed`
- Add to `GetJavaScriptSources()` map
- Use via `require()` in other scripts

### Pattern 2: Main Workflow Script

Files like `create_issue.cjs`, `add_labels.cjs` that are top-level scripts:
- Add to `scripts.go` with `//go:embed` as `xxxSource` variable
- Create bundling getter function with `sync.Once` pattern
- These scripts can `require()` utilities from `GetJavaScriptSources()`
- **Must export `main` function but NOT call it** - the bundler injects `await main()` during execution

### Pattern 3: Log Parser

Files like `parse_claude_log.cjs` that parse AI engine logs:
- Add to `js.go` with `//go:embed`
- Add case in `GetLogParserScript()` function
- Used by workflow compilation system

## Troubleshooting

### Issue: "required file not found in sources"

**Cause:** File not added to `GetJavaScriptSources()` map

**Solution:** Add the file to the map in `pkg/workflow/js.go`

### Issue: Tests fail with "core is not defined"

**Cause:** Missing global mocks

**Solution:** Add proper mocks before importing the module:
```javascript
global.core = mockCore;
global.github = mockGithub;
```

### Issue: Bundler fails with circular dependency

**Cause:** File A requires File B which requires File A

**Solution:** Restructure to break the circular dependency, or combine the modules

### Issue: Changes not reflected after rebuild

**Cause:** Go build cache not recognizing embedded file changes

**Solution:** 
```bash
make clean
make build
```

## References

- Bundler implementation: `pkg/workflow/bundler.go`
- JavaScript sources registry: `pkg/workflow/js.go`
- Script bundling: `pkg/workflow/scripts.go`
- Existing test examples: `pkg/workflow/js/*.test.cjs`
- GitHub Actions script documentation: [actions/toolkit](https://github.com/actions/toolkit)
