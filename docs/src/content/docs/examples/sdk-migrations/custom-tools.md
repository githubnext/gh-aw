---
title: Custom Tools Migration
description: Example of migrating workflows with bash scripts to SDK mode using inline tools.
---

This example demonstrates converting a workflow that uses bash scripts for custom validation to SDK mode with inline tools.

## Original CLI Workflow

```yaml
---
title: Code Quality Check
engine: copilot
on:
  pull_request:
    types: [opened, synchronize]
jobs:
  validate:
    steps:
      - name: Run validation
        run: |
          #!/bin/bash
          set -e
          
          # Check for TODOs
          if grep -r "TODO" src/; then
            echo "Found TODO comments"
            exit 1
          fi
          
          # Check file sizes
          find src/ -size +100k -print
          
          # Check for console.log
          grep -r "console.log" src/ || echo "No console.log found"
      
      - name: Review results
        uses: copilot
        with:
          instructions: Review validation results and provide feedback
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---
```

**Limitations:**
- Bash script separate from agent
- Agent cannot directly call validation
- No structured return values
- Difficult to pass parameters

## Migrated SDK Workflow

```yaml
---
title: Code Quality Check
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 8
  tools:
    inline:
      - name: check_todos
        description: "Check for TODO comments in source code"
        parameters:
          type: object
          properties:
            directory:
              type: string
              description: "Directory to scan"
              default: "src"
        implementation: |
          const { execSync } = require('child_process');
          const fs = require('fs');
          const path = require('path');
          
          function scanForTodos(dir) {
            const todos = [];
            const files = fs.readdirSync(dir, { withFileTypes: true });
            
            for (const file of files) {
              const fullPath = path.join(dir, file.name);
              
              if (file.isDirectory()) {
                todos.push(...scanForTodos(fullPath));
              } else if (file.isFile() && /\.(js|ts|jsx|tsx)$/.test(file.name)) {
                const content = fs.readFileSync(fullPath, 'utf8');
                const lines = content.split('\n');
                
                lines.forEach((line, index) => {
                  if (line.includes('TODO')) {
                    todos.push({
                      file: fullPath,
                      line: index + 1,
                      content: line.trim()
                    });
                  }
                });
              }
            }
            
            return todos;
          }
          
          const todos = scanForTodos(directory);
          
          return {
            found: todos.length > 0,
            count: todos.length,
            items: todos
          };
      
      - name: check_file_sizes
        description: "Check for large files in source code"
        parameters:
          type: object
          properties:
            directory:
              type: string
              default: "src"
            max_size_kb:
              type: integer
              description: "Maximum file size in KB"
              default: 100
        implementation: |
          const fs = require('fs');
          const path = require('path');
          
          function findLargeFiles(dir, maxSizeBytes) {
            const largeFiles = [];
            const files = fs.readdirSync(dir, { withFileTypes: true });
            
            for (const file of files) {
              const fullPath = path.join(dir, file.name);
              
              if (file.isDirectory()) {
                largeFiles.push(...findLargeFiles(fullPath, maxSizeBytes));
              } else if (file.isFile()) {
                const stats = fs.statSync(fullPath);
                if (stats.size > maxSizeBytes) {
                  largeFiles.push({
                    file: fullPath,
                    size_bytes: stats.size,
                    size_kb: Math.round(stats.size / 1024)
                  });
                }
              }
            }
            
            return largeFiles;
          }
          
          const maxSizeBytes = max_size_kb * 1024;
          const largeFiles = findLargeFiles(directory, maxSizeBytes);
          
          return {
            found: largeFiles.length > 0,
            count: largeFiles.length,
            files: largeFiles
          };
      
      - name: check_console_logs
        description: "Check for console.log statements in source code"
        parameters:
          type: object
          properties:
            directory:
              type: string
              default: "src"
        implementation: |
          const fs = require('fs');
          const path = require('path');
          
          function findConsoleLogs(dir) {
            const logs = [];
            const files = fs.readdirSync(dir, { withFileTypes: true });
            
            for (const file of files) {
              const fullPath = path.join(dir, file.name);
              
              if (file.isDirectory()) {
                logs.push(...findConsoleLogs(fullPath));
              } else if (file.isFile() && /\.(js|ts|jsx|tsx)$/.test(file.name)) {
                const content = fs.readFileSync(fullPath, 'utf8');
                const lines = content.split('\n');
                
                lines.forEach((line, index) => {
                  if (/console\.(log|warn|error|info)/.test(line)) {
                    logs.push({
                      file: fullPath,
                      line: index + 1,
                      content: line.trim()
                    });
                  }
                });
              }
            }
            
            return logs;
          }
          
          const logs = findConsoleLogs(directory);
          
          return {
            found: logs.length > 0,
            count: logs.length,
            items: logs
          };
on:
  pull_request:
    types: [opened, synchronize]
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---

# Code Quality Check

Perform comprehensive code quality checks on the pull request.

## Step 1: Check for TODOs

Use the check_todos tool to scan for TODO comments in the source code.
Report any TODOs found with file locations and suggest addressing them.

## Step 2: Check File Sizes

Use the check_file_sizes tool to find files larger than 100KB.
Large files may indicate need for refactoring or code splitting.

## Step 3: Check Console Logs

Use the check_console_logs tool to find console.log statements.
These should typically be removed before merging to production.

## Step 4: Provide Summary

Provide a comprehensive summary of all findings with:
- Severity assessment (blocking, recommended, optional)
- Specific recommendations for each issue
- Overall code quality score
```

## Migration Benefits

### Before (CLI with Bash)

❌ Bash script runs separately from agent  
❌ Agent cannot dynamically call validation  
❌ No structured return values  
❌ Difficult to pass parameters  
❌ Hard to reuse validation logic  
❌ Limited error handling

### After (SDK with Inline Tools)

✅ Agent can call tools when needed  
✅ Structured return values  
✅ Parameterized tools  
✅ Reusable across workflows  
✅ Better error handling  
✅ Type-safe parameters

## Key Improvements

### 1. Structured Return Values

**Before (Bash):**
```bash
grep -r "TODO" src/
# Returns: src/file.js:// TODO: Fix this
```

**After (Inline Tool):**
```json
{
  "found": true,
  "count": 3,
  "items": [
    {
      "file": "src/file.js",
      "line": 42,
      "content": "// TODO: Fix this"
    }
  ]
}
```

### 2. Parameterization

Tools accept parameters for flexibility:

```yaml
parameters:
  type: object
  properties:
    directory:
      type: string
      default: "src"
    max_size_kb:
      type: integer
      default: 100
```

Agent can adjust parameters based on context.

### 3. Error Handling

```javascript
try {
  const result = scanFiles(directory);
  return { success: true, result };
} catch (error) {
  return {
    success: false,
    error: error.message,
    recoverable: true
  };
}
```

### 4. Dynamic Invocation

Agent decides when and how to call tools:

```markdown
## Step 1: Check for TODOs

Use check_todos tool to scan source code.
If TODOs found, categorize by priority.
```

Agent can skip checks or adjust order based on context.

## Migration Effort

**Time:** 4-6 hours  
**Complexity:** Medium-High  
**Testing:** Extensive testing recommended

## Testing Strategy

### 1. Unit Test Each Tool

```javascript
// Test check_todos tool
const result = check_todos({ directory: 'test-fixtures' });
assert(result.found === true);
assert(result.count === 2);
```

### 2. Test with Various Inputs

```bash
# Test with different directories
gh aw run quality-check.md --input '{"directory": "src"}'
gh aw run quality-check.md --input '{"directory": "lib"}'

# Test with different thresholds
gh aw run quality-check.md --input '{"max_size_kb": 50}'
```

### 3. Compare with Original

```bash
# Run original bash script
./validate.sh > cli-output.txt

# Run SDK workflow
gh aw run quality-check.md > sdk-output.txt

# Compare results
diff cli-output.txt sdk-output.txt
```

## Common Patterns

### Pattern 1: Shell Command Wrapper

Wrap existing shell commands:

```yaml
tools:
  inline:
    - name: run_linter
      implementation: |
        const { execSync } = require('child_process');
        try {
          const output = execSync('eslint src/', {
            encoding: 'utf8'
          });
          return { passed: true, output };
        } catch (error) {
          return { 
            passed: false, 
            output: error.stdout,
            errors: parseEslintOutput(error.stdout)
          };
        }
```

### Pattern 2: File System Operations

```yaml
tools:
  inline:
    - name: analyze_structure
      implementation: |
        const fs = require('fs');
        const structure = {
          files: 0,
          directories: 0,
          total_size: 0
        };
        
        function analyze(dir) {
          const items = fs.readdirSync(dir, { withFileTypes: true });
          for (const item of items) {
            if (item.isDirectory()) {
              structure.directories++;
              analyze(path.join(dir, item.name));
            } else {
              structure.files++;
              structure.total_size += fs.statSync(
                path.join(dir, item.name)
              ).size;
            }
          }
        }
        
        analyze('src');
        return structure;
```

### Pattern 3: Data Processing

```yaml
tools:
  inline:
    - name: calculate_coverage
      implementation: |
        const fs = require('fs');
        const coverage = JSON.parse(
          fs.readFileSync('coverage/coverage-summary.json', 'utf8')
        );
        
        const total = coverage.total;
        return {
          statements: total.statements.pct,
          branches: total.branches.pct,
          functions: total.functions.pct,
          lines: total.lines.pct,
          passes: total.lines.pct >= 80
        };
```

## Advanced Enhancements

### Add Caching

```yaml
tools:
  inline:
    - name: check_todos_cached
      implementation: |
        const crypto = require('crypto');
        const cacheKey = crypto
          .createHash('md5')
          .update(directory)
          .digest('hex');
        
        if (global.cache?.[cacheKey]) {
          return { 
            ...global.cache[cacheKey],
            cached: true
          };
        }
        
        const result = scanForTodos(directory);
        global.cache = global.cache || {};
        global.cache[cacheKey] = result;
        
        return result;
```

### Add Progress Reporting

```yaml
tools:
  inline:
    - name: check_with_progress
      implementation: |
        const core = require('@actions/core');
        
        core.info('Starting TODO check...');
        const todos = scanForTodos(directory);
        core.info(`Found ${todos.length} TODOs`);
        
        core.info('Checking file sizes...');
        const largeFiles = checkFileSizes(directory);
        core.info(`Found ${largeFiles.length} large files`);
        
        return { todos, largeFiles };
```

### Add Configurable Rules

```yaml
tools:
  inline:
    - name: check_with_config
      parameters:
        type: object
        properties:
          rules:
            type: object
            properties:
              check_todos:
                type: boolean
                default: true
              max_file_size:
                type: integer
                default: 100
              check_console:
                type: boolean
                default: true
      implementation: |
        const results = {};
        
        if (rules.check_todos) {
          results.todos = scanForTodos('src');
        }
        
        if (rules.max_file_size) {
          results.largeFiles = checkFileSizes('src', rules.max_file_size);
        }
        
        if (rules.check_console) {
          results.consoleLogs = findConsoleLogs('src');
        }
        
        return results;
```

## Troubleshooting

### Issue: Tool Timeouts

```yaml
tools:
  inline:
    - name: slow_check
      timeout: 120  # Increase timeout for slow operations
```

### Issue: File Not Found

```javascript
implementation: |
  const fs = require('fs');
  if (!fs.existsSync(directory)) {
    return {
      error: `Directory not found: ${directory}`,
      found: false,
      count: 0
    };
  }
```

### Issue: Memory Limits

```javascript
implementation: |
  // Process files in batches
  const batchSize = 100;
  const files = getAllFiles(directory);
  
  for (let i = 0; i < files.length; i += batchSize) {
    const batch = files.slice(i, i + batchSize);
    processBatch(batch);
  }
```

## Related Examples

- [Simple Workflow Migration](./simple-workflow/) - Basic migration
- [Multi-Turn Review](./multi-turn-review/) - Context retention
- [Multi-Agent Migration](./multi-agent/) - Parallel coordination
