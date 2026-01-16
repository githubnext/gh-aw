---
title: SDK Custom Tools
description: Guide to defining and using custom inline tools in SDK mode workflows.
sidebar:
  order: 726
---

Custom inline tools enable you to define workflow-specific functionality directly in your workflow frontmatter without needing external MCP servers.

## Overview

**Inline tools** are JavaScript functions defined within your workflow configuration that the AI agent can call during execution. Unlike MCP server tools which require separate deployment, inline tools:

- Are defined directly in workflow frontmatter
- Execute in the workflow's runtime environment
- Have access to workflow secrets and context
- Can implement custom business logic
- Are scoped to the specific workflow

## Basic Inline Tool

### Minimal Example

```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: calculate_coverage
        description: "Calculate test coverage percentage"
        implementation: |
          const { execSync } = require('child_process');
          const result = execSync('npm run coverage');
          const match = result.toString().match(/(\d+)%/);
          return { coverage: parseInt(match[1]) };
---
# Coverage Check

Use calculate_coverage tool to check if coverage meets 80% threshold.
```

### Complete Example

```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: validate_code
        description: "Run custom code validation checks"
        parameters:
          type: object
          required: [file_path]
          properties:
            file_path:
              type: string
              description: "Path to file to validate"
            strict:
              type: boolean
              description: "Enable strict mode validation"
              default: false
        implementation: |
          const fs = require('fs');
          const path = require('path');
          
          // Validate input
          if (!fs.existsSync(file_path)) {
            throw new Error(`File not found: ${file_path}`);
          }
          
          // Read and validate
          const content = fs.readFileSync(file_path, 'utf8');
          const issues = [];
          
          // Custom validation logic
          if (content.includes('console.log')) {
            issues.push('Contains console.log statement');
          }
          
          if (strict && content.length > 1000) {
            issues.push('File too long for strict mode');
          }
          
          return {
            valid: issues.length === 0,
            issues: issues,
            file: file_path
          };
---
# Code Validation

Use validate_code tool to check files for common issues.
```

## Tool Definition Structure

### Required Fields

**`name`** (string, required)  
Tool identifier used by the agent. Must be valid JavaScript identifier (alphanumeric, underscores, no spaces).

```yaml
tools:
  inline:
    - name: my_tool  # Valid
    - name: myTool   # Valid
    - name: my-tool  # Invalid (hyphen)
```

**`description`** (string, required)  
Human-readable description explaining what the tool does. The agent uses this to decide when to call the tool.

```yaml
tools:
  inline:
    - name: check_tests
      description: "Run test suite and return pass/fail status with details"
```

**`implementation`** (string, required)  
JavaScript code that executes when tool is called. Can be multi-line using YAML pipe (`|`) syntax.

```yaml
tools:
  inline:
    - name: example
      implementation: |
        // Your JavaScript code here
        const result = doSomething();
        return result;
```

### Optional Fields

**`parameters`** (object, optional)  
JSON Schema defining the tool's parameters. If omitted, tool accepts no parameters.

```yaml
tools:
  inline:
    - name: greet
      parameters:
        type: object
        required: [name]
        properties:
          name:
            type: string
            description: "Name to greet"
          formal:
            type: boolean
            default: false
      implementation: |
        const greeting = formal ? "Hello" : "Hi";
        return { message: `${greeting}, ${name}!` };
```

**`timeout`** (integer, optional, default: 30)  
Maximum seconds the tool can run before timing out.

```yaml
tools:
  inline:
    - name: slow_operation
      timeout: 120  # 2 minutes
      implementation: |
        // Long-running operation
```

**`async`** (boolean, optional, default: true)  
Whether tool implementation is asynchronous. Most tools should be async.

```yaml
tools:
  inline:
    - name: fetch_data
      async: true
      implementation: |
        const response = await fetch('https://api.example.com');
        return await response.json();
```

## Parameter Schema

### JSON Schema Specification

Inline tool parameters use [JSON Schema](https://json-schema.org/) format:

```yaml
parameters:
  type: object
  required: [field1, field2]
  properties:
    field1:
      type: string
      description: "Description for the agent"
      minLength: 1
      maxLength: 100
    field2:
      type: integer
      description: "Numeric value"
      minimum: 0
      maximum: 100
    field3:
      type: boolean
      default: true
    field4:
      type: array
      items:
        type: string
      minItems: 1
```

### Supported Types

**String:**
```yaml
file_path:
  type: string
  description: "Path to file"
  pattern: "^[a-zA-Z0-9/_-]+\\.js$"
  minLength: 1
  maxLength: 255
```

**Integer/Number:**
```yaml
threshold:
  type: integer
  description: "Threshold value"
  minimum: 0
  maximum: 100
  default: 50
```

**Boolean:**
```yaml
strict:
  type: boolean
  description: "Enable strict mode"
  default: false
```

**Array:**
```yaml
files:
  type: array
  description: "List of files"
  items:
    type: string
  minItems: 1
  maxItems: 10
```

**Object:**
```yaml
config:
  type: object
  description: "Configuration object"
  required: [host, port]
  properties:
    host:
      type: string
    port:
      type: integer
```

**Enum:**
```yaml
level:
  type: string
  description: "Log level"
  enum: [debug, info, warn, error]
  default: info
```

### Validation

Parameters are automatically validated against the schema before tool execution:

```yaml
tools:
  inline:
    - name: process_file
      parameters:
        type: object
        required: [path]
        properties:
          path:
            type: string
            pattern: "^/.*\\.txt$"
      implementation: |
        // Only called if path is valid and ends with .txt
        return processFile(path);
```

## Implementation Guidelines

### Return Values

Tools must return JSON-serializable objects:

```yaml
tools:
  inline:
    # ✅ Good: Returns structured data
    - name: good_tool
      implementation: |
        return {
          status: "success",
          data: { value: 42 },
          metadata: { timestamp: new Date().toISOString() }
        };
    
    # ❌ Bad: Returns function (not serializable)
    - name: bad_tool
      implementation: |
        return function() { return 42; };
```

### Error Handling

Use try-catch for robust error handling:

```yaml
tools:
  inline:
    - name: safe_tool
      implementation: |
        try {
          const result = riskyOperation();
          return { success: true, result };
        } catch (error) {
          return {
            success: false,
            error: error.message,
            stack: error.stack
          };
        }
```

### Available APIs

Inline tools run in a Node.js environment with access to:

**Built-in Modules:**
```javascript
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const crypto = require('crypto');
const https = require('https');
```

**GitHub Actions Toolkit:**
```javascript
const core = require('@actions/core');
const github = require('@actions/github');

// Use toolkit functions
core.info('Processing...');
const token = core.getInput('github-token');
const octokit = github.getOctokit(token);
```

**Workflow Context:**
```javascript
// Access workflow variables
const repoName = process.env.GITHUB_REPOSITORY;
const runId = process.env.GITHUB_RUN_ID;

// Access secrets
const apiKey = process.env.API_KEY;
```

### Async/Await

Use async/await for asynchronous operations:

```yaml
tools:
  inline:
    - name: fetch_data
      async: true
      implementation: |
        const https = require('https');
        
        const data = await new Promise((resolve, reject) => {
          https.get('https://api.example.com/data', (res) => {
            let body = '';
            res.on('data', chunk => body += chunk);
            res.on('end', () => resolve(JSON.parse(body)));
          }).on('error', reject);
        });
        
        return { data };
```

## Common Tool Patterns

### Pattern 1: File Processing

```yaml
tools:
  inline:
    - name: analyze_file
      parameters:
        type: object
        required: [file_path]
        properties:
          file_path:
            type: string
      implementation: |
        const fs = require('fs');
        const path = require('path');
        
        if (!fs.existsSync(file_path)) {
          throw new Error(`File not found: ${file_path}`);
        }
        
        const content = fs.readFileSync(file_path, 'utf8');
        const lines = content.split('\n');
        const extension = path.extname(file_path);
        
        return {
          lines: lines.length,
          size: content.length,
          extension: extension,
          empty_lines: lines.filter(l => !l.trim()).length
        };
```

### Pattern 2: Command Execution

```yaml
tools:
  inline:
    - name: run_tests
      parameters:
        type: object
        properties:
          suite:
            type: string
            enum: [unit, integration, e2e]
            default: unit
      implementation: |
        const { execSync } = require('child_process');
        
        try {
          const output = execSync(`npm run test:${suite}`, {
            encoding: 'utf8',
            maxBuffer: 10 * 1024 * 1024  // 10MB
          });
          
          // Parse test results
          const match = output.match(/(\d+) passed, (\d+) failed/);
          
          return {
            success: true,
            passed: parseInt(match[1]),
            failed: parseInt(match[2]),
            output: output
          };
        } catch (error) {
          return {
            success: false,
            error: error.message,
            output: error.stdout?.toString()
          };
        }
```

### Pattern 3: API Calls

```yaml
tools:
  inline:
    - name: fetch_metrics
      parameters:
        type: object
        required: [repo]
        properties:
          repo:
            type: string
            pattern: "^[^/]+/[^/]+$"
      implementation: |
        const https = require('https');
        const [owner, repoName] = repo.split('/');
        
        const options = {
          hostname: 'api.github.com',
          path: `/repos/${owner}/${repoName}`,
          headers: {
            'User-Agent': 'gh-aw',
            'Authorization': `token ${process.env.GITHUB_TOKEN}`
          }
        };
        
        const data = await new Promise((resolve, reject) => {
          https.get(options, (res) => {
            let body = '';
            res.on('data', chunk => body += chunk);
            res.on('end', () => resolve(JSON.parse(body)));
          }).on('error', reject);
        });
        
        return {
          stars: data.stargazers_count,
          forks: data.forks_count,
          open_issues: data.open_issues_count
        };
```

### Pattern 4: Data Validation

```yaml
tools:
  inline:
    - name: validate_config
      parameters:
        type: object
        required: [config_path]
        properties:
          config_path:
            type: string
      implementation: |
        const fs = require('fs');
        const yaml = require('js-yaml');
        
        const content = fs.readFileSync(config_path, 'utf8');
        const config = yaml.load(content);
        
        const errors = [];
        
        // Validate required fields
        if (!config.version) {
          errors.push('Missing version field');
        }
        
        if (!config.name) {
          errors.push('Missing name field');
        }
        
        // Validate version format
        if (config.version && !/^\d+\.\d+\.\d+$/.test(config.version)) {
          errors.push('Invalid version format (expected x.y.z)');
        }
        
        return {
          valid: errors.length === 0,
          errors: errors,
          config: config
        };
```

### Pattern 5: Aggregation

```yaml
tools:
  inline:
    - name: aggregate_results
      parameters:
        type: object
        required: [files]
        properties:
          files:
            type: array
            items:
              type: string
      implementation: |
        const fs = require('fs');
        
        const results = files.map(file => {
          const content = fs.readFileSync(file, 'utf8');
          return JSON.parse(content);
        });
        
        // Aggregate metrics
        const total = results.reduce((acc, r) => {
          acc.passed += r.passed || 0;
          acc.failed += r.failed || 0;
          return acc;
        }, { passed: 0, failed: 0 });
        
        return {
          total_files: files.length,
          aggregate: total,
          success_rate: (total.passed / (total.passed + total.failed) * 100).toFixed(2)
        };
```

## Security Considerations

### Input Validation

**Always validate and sanitize inputs:**

```yaml
tools:
  inline:
    - name: secure_tool
      implementation: |
        // ✅ Good: Validate and sanitize
        if (!file_path || typeof file_path !== 'string') {
          throw new Error('Invalid file_path');
        }
        
        // Prevent path traversal
        const sanitized = path.normalize(file_path).replace(/^(\.\.[\/\\])+/, '');
        
        // Restrict to workspace
        const workspace = process.env.GITHUB_WORKSPACE;
        const fullPath = path.join(workspace, sanitized);
        
        if (!fullPath.startsWith(workspace)) {
          throw new Error('Path traversal detected');
        }
```

### Command Injection Prevention

**Avoid shell injection:**

```yaml
tools:
  inline:
    - name: unsafe_tool
      implementation: |
        // ❌ Bad: Shell injection vulnerability
        execSync(`grep ${pattern} ${file}`);
    
    - name: safe_tool
      implementation: |
        // ✅ Good: Use array form or escape properly
        const { spawn } = require('child_process');
        const child = spawn('grep', [pattern, file]);
```

### Secret Handling

**Access secrets securely:**

```yaml
tools:
  inline:
    - name: api_call
      implementation: |
        // ✅ Good: Use environment variables
        const apiKey = process.env.API_KEY;
        
        if (!apiKey) {
          throw new Error('API_KEY not configured');
        }
        
        // ❌ Bad: Never log secrets
        // console.log(`Using key: ${apiKey}`);
        
        // ✅ Good: Don't include in returned data
        return { 
          success: true,
          // Don't return apiKey
        };
```

### Resource Limits

**Implement timeouts and size limits:**

```yaml
tools:
  inline:
    - name: bounded_tool
      timeout: 60  # Tool-level timeout
      implementation: |
        const MAX_FILE_SIZE = 10 * 1024 * 1024;  // 10MB
        const MAX_ITEMS = 1000;
        
        const stats = fs.statSync(file_path);
        if (stats.size > MAX_FILE_SIZE) {
          throw new Error('File too large');
        }
        
        const items = processFile(file_path);
        if (items.length > MAX_ITEMS) {
          throw new Error('Too many items');
        }
        
        return { items: items.slice(0, MAX_ITEMS) };
```

### Sandboxing

Inline tools are subject to workflow sandbox restrictions:

```yaml
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: network_tool
        implementation: |
          // Subject to network permissions
          const response = await fetch('https://api.example.com');

network:
  allowed:
    - defaults
    - "api.example.com"  # Must be explicitly allowed
```

## Tool Testing

### Testing Strategy

Test inline tools before deployment:

```yaml
tools:
  inline:
    - name: test_coverage
      implementation: |
        // Include test mode
        const TEST_MODE = process.env.TEST_MODE === 'true';
        
        if (TEST_MODE) {
          return {
            coverage: 85.5,
            test_data: true
          };
        }
        
        // Real implementation
        const result = execSync('npm run coverage');
        return parseCoverage(result);
```

### Debugging

Add logging for debugging:

```yaml
tools:
  inline:
    - name: debug_tool
      implementation: |
        const core = require('@actions/core');
        
        core.info(`Processing ${file_path}`);
        core.debug(`Parameters: ${JSON.stringify(parameters)}`);
        
        try {
          const result = process();
          core.info('Processing complete');
          return result;
        } catch (error) {
          core.error(`Failed: ${error.message}`);
          throw error;
        }
```

### Unit Testing

Test tool logic separately:

```javascript
// tools/validate_code.js
module.exports = function validateCode(file_path, strict) {
  const fs = require('fs');
  const content = fs.readFileSync(file_path, 'utf8');
  const issues = [];
  
  if (content.includes('console.log')) {
    issues.push('Contains console.log');
  }
  
  return {
    valid: issues.length === 0,
    issues: issues
  };
};

// tools/validate_code.test.js
const validateCode = require('./validate_code');

test('detects console.log', () => {
  const result = validateCode('test-file.js', false);
  expect(result.valid).toBe(false);
  expect(result.issues).toContain('Contains console.log');
});
```

Then import in workflow:

```yaml
tools:
  inline:
    - name: validate_code
      implementation: |
        const validate = require('./tools/validate_code');
        return validate(file_path, strict);
```

## Best Practices

### 1. Keep Tools Focused

```yaml
# ✅ Good: Single responsibility
tools:
  inline:
    - name: check_lint
    - name: check_format
    - name: check_tests

# ❌ Bad: Too many responsibilities
tools:
  inline:
    - name: check_everything
```

### 2. Provide Clear Descriptions

```yaml
# ✅ Good: Specific, actionable description
tools:
  inline:
    - name: validate_pr
      description: "Validate PR has: title, description, labels, and linked issue"

# ❌ Bad: Vague description
tools:
  inline:
    - name: validate_pr
      description: "Check PR"
```

### 3. Use Type-Safe Parameters

```yaml
# ✅ Good: Explicit types and validation
parameters:
  type: object
  required: [threshold]
  properties:
    threshold:
      type: integer
      minimum: 0
      maximum: 100

# ❌ Bad: Any type accepted
parameters:
  type: object
  properties:
    threshold: {}
```

### 4. Handle Errors Gracefully

```yaml
implementation: |
  try {
    return { success: true, result: doWork() };
  } catch (error) {
    return {
      success: false,
      error: error.message,
      recoverable: true
    };
  }
```

### 5. Document Complex Logic

```yaml
implementation: |
  // Calculate test coverage using nyc
  // Returns percentage (0-100) based on line coverage
  // Throws error if coverage command fails
  
  const { execSync } = require('child_process');
  const output = execSync('nyc report --reporter=text-summary');
  const match = output.toString().match(/Lines\s+:\s+([\d.]+)%/);
  
  return {
    coverage: parseFloat(match[1]),
    output: output.toString()
  };
```

## Examples

### Example 1: License Validation

```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: check_license
        description: "Verify all source files have license headers"
        parameters:
          type: object
          properties:
            directory:
              type: string
              default: "./src"
        implementation: |
          const fs = require('fs');
          const path = require('path');
          const glob = require('glob');
          
          const files = glob.sync(`${directory}/**/*.{js,ts}`, {
            ignore: ['**/node_modules/**', '**/dist/**']
          });
          
          const missing = files.filter(file => {
            const content = fs.readFileSync(file, 'utf8');
            return !content.includes('Copyright');
          });
          
          return {
            total_files: files.length,
            missing_license: missing.length,
            files_without_license: missing,
            compliant: missing.length === 0
          };
---
# License Compliance Check

Use check_license tool to verify all source files have license headers.
```

### Example 2: Dependency Audit

```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: audit_dependencies
        description: "Audit npm dependencies for security vulnerabilities"
        timeout: 120
        implementation: |
          const { execSync } = require('child_process');
          
          try {
            execSync('npm audit --json > audit-result.json');
            const result = JSON.parse(fs.readFileSync('audit-result.json', 'utf8'));
            
            return {
              total_vulnerabilities: result.metadata.vulnerabilities.total,
              critical: result.metadata.vulnerabilities.critical,
              high: result.metadata.vulnerabilities.high,
              moderate: result.metadata.vulnerabilities.moderate,
              low: result.metadata.vulnerabilities.low,
              safe: result.metadata.vulnerabilities.total === 0
            };
          } catch (error) {
            // npm audit exits with 1 if vulnerabilities found
            const result = JSON.parse(fs.readFileSync('audit-result.json', 'utf8'));
            return {
              total_vulnerabilities: result.metadata.vulnerabilities.total,
              details: result
            };
          }
---
# Security Audit

Use audit_dependencies to check for security vulnerabilities.
```

### Example 3: Code Metrics

```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: calculate_metrics
        description: "Calculate code complexity and quality metrics"
        parameters:
          type: object
          required: [file_path]
          properties:
            file_path:
              type: string
        implementation: |
          const fs = require('fs');
          const content = fs.readFileSync(file_path, 'utf8');
          const lines = content.split('\n');
          
          // Calculate metrics
          const metrics = {
            lines_of_code: lines.length,
            blank_lines: lines.filter(l => !l.trim()).length,
            comment_lines: lines.filter(l => l.trim().startsWith('//')).length,
            max_line_length: Math.max(...lines.map(l => l.length)),
            functions: (content.match(/function\s+\w+/g) || []).length,
            classes: (content.match(/class\s+\w+/g) || []).length
          };
          
          // Calculate derived metrics
          metrics.code_lines = metrics.lines_of_code - metrics.blank_lines - metrics.comment_lines;
          metrics.comment_ratio = (metrics.comment_lines / metrics.code_lines * 100).toFixed(2);
          
          return metrics;
---
# Code Metrics Analysis

Use calculate_metrics to analyze code quality indicators.
```

## Related Documentation

- [SDK Engine Reference](/gh-aw/reference/engines-sdk/) - Complete SDK configuration
- [Session Management](/gh-aw/guides/sdk-sessions/) - Multi-turn conversations
- [Security Guide](/gh-aw/guides/security/) - Security best practices
- [Tools Reference](/gh-aw/reference/tools/) - Standard MCP tools
