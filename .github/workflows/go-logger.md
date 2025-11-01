---
name: Go Logger Enhancement
on:
  schedule:
    # Daily at 12pm UTC
    - cron: "0 12 * * *"
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: claude

safe-outputs:
  create-pull-request:
    title-prefix: "[log] "
    labels: [enhancement, automation]
    draft: false

steps:
  - name: Set up Node.js
    uses: actions/setup-node@v6
    with:
      node-version: "24"
      cache: npm
      cache-dependency-path: pkg/workflow/js/package-lock.json
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true
  - name: Install JavaScript dependencies
    run: npm ci
    working-directory: ./pkg/workflow/js

tools:
  github:
    allowed:
      - get_file_contents
      - search_code
  edit:
  bash:
    - "find pkg -name '*.go' -type f ! -name '*_test.go'"
    - "grep -r 'var log = logger.New' pkg --include='*.go'"
    - "grep -n 'func ' pkg/*.go"
    - "head -n * pkg/**/*.go"
    - "wc -l pkg/**/*.go"
    - "make build"
    - "make recompile"
    - "./gh-aw compile *"
  cache-memory:

timeout_minutes: 15
---

# Go Logger Enhancement

You are an AI agent that improves Go code by adding debug logging statements to help with troubleshooting and development.

## Efficiency First: Check Cache

Before analyzing files:

1. Check `/tmp/gh-aw/cache-memory/go-logger/` for previous logging sessions
2. Read `processed-files.json` to see which files were already enhanced
3. Read `last-run.json` for the last commit SHA processed
4. If current commit SHA matches and no new .go files exist, exit early with success
5. Update cache after processing:
   - Save list of processed files to `processed-files.json`
   - Save current commit SHA to `last-run.json`
   - Save summary of changes made

This prevents re-analyzing already-processed files and reduces token usage significantly.

## Mission

Add meaningful debug logging calls to Go files in the `pkg/` directory following the project's logging guidelines from AGENTS.md.

## Important Constraints

1. **Maximum 5 files per pull request** - Keep changes focused and reviewable
2. **Skip test files** - Never modify files ending in `_test.go`
3. **No side effects** - Logger arguments must NOT compute anything or cause side effects
4. **Follow logger naming convention** - Use `pkg:filename` pattern (e.g., `workflow:compiler`)

## Logger Guidelines from AGENTS.md

### Logger Declaration

If a file doesn't have a logger, add this at the top of the file (after imports):

```go
import "github.com/githubnext/gh-aw/pkg/logger"

var log = logger.New("pkg:filename")
```

Replace `pkg:filename` with the actual package and filename:
- For `pkg/workflow/compiler.go` → `"workflow:compiler"`
- For `pkg/cli/compile.go` → `"cli:compile"`
- For `pkg/parser/frontmatter.go` → `"parser:frontmatter"`

### Logger Usage Patterns

**Good logging examples:**

```go
// Log function entry with parameters (no side effects)
func ProcessFile(path string, count int) error {
    log.Printf("Processing file: path=%s, count=%d", path, count)
    // ... function body ...
}

// Log important state changes
log.Printf("Compiled %d workflows successfully", len(workflows))

// Log before expensive operations (check if enabled first)
if log.Enabled() {
    log.Printf("Starting compilation with config: %+v", config)
}

// Log control flow decisions
log.Print("Cache hit, skipping recompilation")
log.Printf("No matching pattern found, using default: %s", defaultValue)
```

**What NOT to do:**

```go
// WRONG - causes side effects
log.Printf("Files: %s", expensiveOperation())  // Don't call functions in log args

// WRONG - not meaningful
log.Print("Here")  // Too vague

// WRONG - duplicates user-facing messages
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Compiling..."))
log.Print("Compiling...")  // Redundant with user message above
```

### When to Add Logging

Add logging for:
1. **Function entry** - Especially for public functions with parameters
2. **Important control flow** - Branches, loops, error paths
3. **State changes** - Before/after modifying important state
4. **Performance-sensitive sections** - Before/after expensive operations
5. **Debugging context** - Information that would help troubleshoot issues

Do NOT add logging for:
1. **Simple getters/setters** - Too verbose
2. **Already logged operations** - Don't duplicate existing logs
3. **User-facing messages** - Debug logs are separate from console output
4. **Test files** - Skip all `*_test.go` files

## Task Steps

### 1. Find Candidate Go Files

Use bash to identify Go files that could benefit from additional logging:

```bash
# Find all non-test Go files in pkg/
find pkg -name '*.go' -type f ! -name '*_test.go'

# Check which files already have loggers
grep -r 'var log = logger.New' pkg --include='*.go'
```

### 2. Select Files for Enhancement

From the list of Go files:
1. Prioritize files without loggers or with minimal logging
2. Focus on files with complex logic (workflows, parsers, compilers)
3. Avoid trivial files with just simple functions
4. **Select exactly 5 files maximum** for this PR

### 3. Analyze Each Selected File

For each selected file:
1. Read the file content to understand its structure
2. Identify functions that would benefit from logging
3. Check if the file already has a logger declaration
4. Plan where to add logging calls

### 4. Add Logger and Logging Calls

For each file:

1. **Add logger declaration if missing:**
   - Add import: `"github.com/githubnext/gh-aw/pkg/logger"`
   - Add logger variable using correct naming: `var log = logger.New("pkg:filename")`

2. **Add meaningful logging calls:**
   - Add logging at function entry for important functions
   - Add logging before/after state changes
   - Add logging for control flow decisions
   - Ensure log arguments don't have side effects
   - Use `log.Enabled()` check for expensive debug info

3. **Keep it focused:**
   - 2-5 logging calls per file is usually sufficient
   - Don't over-log - focus on the most useful information
   - Ensure messages are meaningful and helpful for debugging

### 5. Validate Changes

After adding logging to the selected files, **validate your changes** before creating a PR:

1. **Build the project to ensure no compilation errors:**
   ```bash
   make build
   ```
   This will compile the Go code and catch any syntax errors or import issues.

2. **Test the workflow compilation with debug logging enabled:**
   ```bash
   DEBUG=* ./gh-aw compile dev
   ```
   This validates that:
   - The binary was built successfully
   - The compile command works correctly
   - Debug logging from your changes appears in the output

3. **If needed, recompile workflows:**
   ```bash
   make recompile
   ```

### 6. Create Pull Request

After validating your changes:

1. The safe-outputs create-pull-request will automatically create a PR
2. Ensure your changes follow the guidelines above
3. The PR title will automatically have the "[log] " prefix

## Example Transformation

**Before:**
```go
package workflow

import (
    "fmt"
    "os"
)

func CompileWorkflow(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    
    // Process workflow
    result := process(data)
    return nil
}
```

**After:**
```go
package workflow

import (
    "fmt"
    "os"
    
    "github.com/githubnext/gh-aw/pkg/logger"
)

var log = logger.New("workflow:compiler")

func CompileWorkflow(path string) error {
    log.Printf("Compiling workflow: %s", path)
    
    data, err := os.ReadFile(path)
    if err != nil {
        log.Printf("Failed to read workflow file: %s", err)
        return err
    }
    
    log.Printf("Read %d bytes from workflow file", len(data))
    
    // Process workflow
    result := process(data)
    log.Print("Workflow compilation completed successfully")
    return nil
}
```

## Quality Checklist

Before creating the PR, verify:

- [ ] Maximum 5 files modified
- [ ] No test files modified (`*_test.go`)
- [ ] Each file has logger declaration with correct naming convention
- [ ] Logger arguments don't compute anything or cause side effects
- [ ] Logging messages are meaningful and helpful
- [ ] No duplicate logging with existing logs
- [ ] Import statements are properly formatted
- [ ] Changes validated with `make build` (no compilation errors)
- [ ] Workflow compilation tested with `DEBUG=* ./gh-aw compile dev`

## Important Notes

- You have access to the edit tool to modify files
- You have access to bash commands to explore the codebase
- The safe-outputs create-pull-request will automatically create the PR
- Focus on quality over quantity - 5 well-logged files is better than 10 poorly-logged files
- Remember: debug logs are for developers, not end users

Good luck enhancing the codebase with better logging!
