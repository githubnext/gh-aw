# Tidy Workflow Audit Analysis

## Executive Summary

This document provides a comprehensive analysis of the `.github/workflows/tidy.md` agentic workflow, identifying potential improvements based on code review, workflow structure analysis, and alignment with best practices documented in the repository.

**Key Findings:**
1. Workflow uses legacy `command:` trigger instead of modern command patterns
2. Safe-outputs configuration includes an undefined tool `missing-tool:` without proper configuration
3. Network configuration is overly restrictive (`network: {}`) which may limit necessary operations
4. Engine configuration lacks explicit model and version specifications
5. Workflow instructions could be more robust regarding error handling
6. Missing timeout and turn limits which are recommended best practices

## Detailed Analysis

### 1. Workflow Trigger Configuration

**Current State:**
```yaml
on:
  workflow_dispatch:
  command:
  push:
    branches: [main]
    paths:
      - '**/*.go'
      - '**/*.js'
      - '**/*.cjs'
      - '**/*.ts'
```

**Issues:**
- The `command:` trigger is present but not properly configured with a `name` field
- This creates a workflow that responds to `/tidy` mentions but the configuration is incomplete

**Recommendation:**
```yaml
on:
  workflow_dispatch:
  command:
    name: tidy
  push:
    branches: [main]
    paths:
      - '**/*.go'
      - '**/*.js'
      - '**/*.cjs'
      - '**/*.ts'
```

**Impact:** Medium - The workflow may not properly respond to /tidy mentions, requiring manual triggers instead

### 2. Safe-Outputs Configuration

**Current State:**
```yaml
safe-outputs:
  create-pull-request:
    title-prefix: "[tidy] "
    labels: [automation, maintenance]
    draft: false
  push-to-pull-request-branch:
  missing-tool:
```

**Issues:**
- `missing-tool:` is listed as a safe-output but this is a reporting tool, not an output creation tool
- Including it here suggests a misunderstanding of its purpose
- `missing-tool` is used to report missing functionality to developers, not to create GitHub artifacts

**Recommendation:**
Remove `missing-tool:` from safe-outputs:
```yaml
safe-outputs:
  create-pull-request:
    title-prefix: "[tidy] "
    labels: [automation, maintenance]
    draft: false
  push-to-pull-request-branch:
```

**Impact:** Low - Doesn't break functionality but creates confusion and unnecessary MCP server configuration

### 3. Network Configuration

**Current State:**
```yaml
network: {}
```

**Issues:**
- Completely blocks all network access
- This may prevent the agent from accessing necessary resources like:
  - Package registries for checking versions
  - Documentation sites
  - External tools or APIs that might be useful for code quality checks

**Recommendation:**
```yaml
network:
  allowed:
    - defaults
    - node
    - go
```

**Rationale:**
- `defaults` provides basic infrastructure (certificates, JSON schema, Ubuntu packages)
- `node` allows access to NPM ecosystem for JavaScript tooling
- `go` allows access to Go ecosystem for Go tooling

**Impact:** Medium - Current restrictive policy may limit agent effectiveness

### 4. Engine Configuration

**Current State:**
```yaml
engine: copilot
```

**Issues:**
- Uses simple string format instead of extended object format
- No explicit model specification
- No max-turns limit to prevent runaway costs
- No version specification

**Recommendation:**
```yaml
engine:
  id: copilot
  model: claude-sonnet-4
  max-turns: 5
```

**Rationale:**
- Explicit model selection ensures consistent behavior
- `max-turns: 5` prevents infinite loops and controls costs (based on repository's best practices documentation)
- Clear configuration makes debugging easier

**Impact:** High - Cost control and predictable behavior

### 5. Workflow Instructions - Error Handling

**Current State:**
The workflow instructions tell the agent to:
- Run tests and only fix related failures
- Be conservative about changes

**Issues:**
- No explicit guidance on what to do if compilation fails
- No guidance on handling git conflicts
- No explicit instruction to abort if tests fail after fixes

**Recommendation:**
Add to workflow instructions after step 6:
```markdown
### 6.5. Handle Test Failures Appropriately
If tests fail and you cannot fix them:
- Use the `missing_tool` tool to report the issue
- Do NOT create a pull request with failing tests
- Provide a clear explanation of what failed and why
```

**Impact:** Medium - Prevents creation of PRs with broken code

### 6. Tool Permissions - Bash Commands

**Current State:**
```yaml
tools:
  github:
    allowed: [list_pull_requests, get_pull_request]
  bash: ["make:*"]
```

**Issues:**
- Only allows `make:*` commands
- Agent might need other git commands to check status
- Missing common shell utilities that might be useful

**Recommendation:**
```yaml
tools:
  github:
    allowed: [list_pull_requests, get_pull_request]
  bash: 
    - "make:*"
    - "git status"
    - "git diff:*"
    - "git log:*"
    - "wc"
    - "grep"
```

**Rationale:**
- `git status` helps agent understand repository state
- `git diff` helps verify changes before committing
- `git log` helps understand recent changes
- `wc` and `grep` are useful for analyzing output

**Impact:** Medium - Improves agent's ability to understand and verify changes

### 7. Timeout Configuration

**Current State:**
```yaml
timeout_minutes: 10
```

**Assessment:**
- 10 minutes might be tight for:
  - Installing dependencies (`make deps-dev`)
  - Running full test suite (`make test`)
  - Waiting for linter to complete
  
**Recommendation:**
```yaml
timeout_minutes: 15
```

**Impact:** Low - Prevents premature timeouts during legitimate operations

### 8. Missing Cost Control: Stop-After

**Current State:**
No `stop-after` deadline specified

**Recommendation:**
Add to the `on:` section:
```yaml
on:
  workflow_dispatch:
  command:
    name: tidy
  push:
    branches: [main]
    paths:
      - '**/*.go'
      - '**/*.js'
      - '**/*.cjs'
      - '**/*.ts'
  stop-after: +2h
```

**Rationale:**
- Provides a hard deadline for workflow execution
- Prevents runaway costs if workflow gets stuck
- 2 hours is generous for a tidy workflow

**Impact:** High - Cost control and prevents resource exhaustion

## Priority Improvements

### P0 (Critical - Should Fix Immediately)
1. **Add max-turns to engine config** - Prevents infinite loops and runaway costs
2. **Add stop-after deadline** - Prevents workflow from running indefinitely

### P1 (High - Should Fix Soon)
3. **Fix command trigger configuration** - Add explicit `name: tidy`
4. **Improve network permissions** - Allow access to necessary package ecosystems
5. **Add explicit model to engine config** - Ensures consistent behavior

### P2 (Medium - Nice to Have)
6. **Remove missing-tool from safe-outputs** - Clean up configuration
7. **Expand bash tool permissions** - Add git status/diff commands
8. **Increase timeout to 15 minutes** - Prevent premature timeouts
9. **Add error handling instructions** - Better guidance for test failures

## Implementation Recommendations

### Quick Wins (Can be implemented immediately)
- Add `name: tidy` to command trigger
- Add `max-turns: 5` to engine config
- Add `stop-after: +2h` to on section
- Remove `missing-tool:` from safe-outputs

### Requires Testing
- Network permission changes (test with actual workflow runs)
- Bash tool permission additions (verify all commands work)
- Timeout increase (monitor actual execution times)

## Monitoring Recommendations

After implementing changes, monitor:
1. **Workflow execution times** - Ensure 15-minute timeout is sufficient
2. **Turn counts** - Verify max-turns=5 is adequate
3. **Network access logs** - Check if allowed domains are sufficient
4. **Success rate** - Track how often workflow successfully creates PRs

## Conclusion

The tidy workflow is functionally sound but has several configuration issues that limit its effectiveness and risk management. The highest priority improvements focus on cost control (max-turns, stop-after) and proper configuration (command trigger, network access).

Implementing these changes will:
- Reduce risk of runaway costs
- Improve workflow reliability
- Enhance agent capabilities
- Make configuration clearer and more maintainable
