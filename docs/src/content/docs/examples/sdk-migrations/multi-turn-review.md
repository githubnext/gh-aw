---
title: Multi-Turn Review Migration
description: Example of migrating a multi-stage code review workflow to leverage SDK's multi-turn capabilities.
---

This example shows how to migrate a multi-stage code review process from CLI to SDK mode, taking advantage of context retention across review stages.

## Original CLI Workflow

```yaml
---
title: Multi-Stage Code Review
engine: copilot
on:
  pull_request:
    types: [opened, synchronize]
jobs:
  initial-review:
    steps:
      - name: Quick scan
        uses: copilot
        with:
          instructions: |
            Perform a quick initial scan of the PR for obvious issues
  
  deep-review:
    needs: [initial-review]
    steps:
      - name: Detailed analysis
        uses: copilot
        with:
          instructions: |
            Perform detailed code review focusing on:
            - Security vulnerabilities
            - Performance issues
            - Code quality
  
  final-approval:
    needs: [deep-review]
    steps:
      - name: Final check
        uses: copilot
        with:
          instructions: |
            Review all findings and provide final approval or rejection
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---
```

**Limitations:**
- No context sharing between jobs
- Each job starts fresh
- Cannot build on previous findings
- Difficult to maintain conversation thread

## Migrated SDK Workflow

```yaml
---
title: Multi-Stage Code Review
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true      # Maintain context across turns
    max-turns: 15         # Allow for iterative review
    state-size-limit: 100000
  budget:
    max-tokens: 75000     # Reasonable for thorough review
    warn-threshold: 60000
  events:
    streaming: true
    handlers:
      - on: token_usage
        action: log
        threshold: 15000   # Log every 15k tokens
      
      - on: turn_end
        action: custom
        config:
          implementation: |
            const core = require('@actions/core');
            core.info(`Completed turn ${event.data.turn}: ${event.data.duration_ms}ms`);
on:
  pull_request:
    types: [opened, synchronize]
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---

# Multi-Stage Code Review

Perform a comprehensive multi-stage code review with context retention.

## Stage 1: Initial Quick Scan (Turns 1-3)

Perform a quick initial scan of the PR:
- Check for obvious syntax or style issues
- Identify files requiring deep review
- Note any immediate concerns

Document your findings and create a prioritized list of areas for deep review.

## Stage 2: Deep Analysis (Turns 4-9)

Based on the initial scan, perform detailed analysis:

### Security Review
- Check for SQL injection vulnerabilities
- Validate input sanitization
- Review authentication/authorization
- Check for secrets in code

### Performance Review
- Identify inefficient algorithms
- Check for N+1 queries
- Review caching strategies
- Assess resource usage

### Code Quality Review
- Check error handling
- Review code organization
- Assess test coverage
- Verify documentation

## Stage 3: Developer Interaction (Turns 10-13)

After deep analysis:
- Post detailed review comments
- Answer any developer questions
- Provide specific recommendations
- Suggest improvements

## Stage 4: Final Approval (Turns 14-15)

Review the complete analysis and provide:
- Summary of all findings
- Critical issues that must be addressed
- Recommended improvements
- Final approval or request for changes
```

## Key SDK Features Utilized

### 1. Context Retention

```yaml
session:
  persistent: true
```

Each stage builds on previous findings, maintaining full context of:
- Initial scan results
- Deep analysis findings
- Developer responses
- Historical discussion

### 2. Budget Control

```yaml
budget:
  max-tokens: 75000
  warn-threshold: 60000
```

Ensures review completes within reasonable cost while allowing thorough analysis.

### 3. Progress Monitoring

```yaml
events:
  handlers:
    - on: turn_end
      action: custom
```

Track review progress and performance at each stage.

### 4. Flexible Turn Count

```yaml
session:
  max-turns: 15
```

Allows for iterative refinement if developer has questions or makes updates.

## Migration Benefits

### Before (CLI)

❌ Each job starts fresh with no context  
❌ Cannot reference previous findings  
❌ Difficult to maintain conversation flow  
❌ No way to ask follow-up questions  
❌ Separate job outputs hard to combine

### After (SDK)

✅ Full context retained across stages  
✅ Can build on previous findings  
✅ Natural conversation flow  
✅ Can ask and answer questions  
✅ Single coherent review thread

## Example Review Flow

**Turn 1-2:** Initial scan identifies 3 security concerns in `auth.js`

**Turn 3-5:** Deep dive into `auth.js` security issues:
- SQL injection vulnerability on line 45
- Missing input validation on line 67
- Weak password hashing on line 89

**Turn 6-7:** Review performance of `database.js`:
- N+1 query pattern detected
- Missing indexes on frequently queried columns

**Turn 8-9:** Developer asks: "How should I fix the N+1 query?"

**Turn 10-11:** Agent provides specific code example for fix

**Turn 12-13:** Developer implements fix, agent reviews update

**Turn 14-15:** Final approval with summary of all changes

## Migration Effort

**Time:** 2-3 hours  
**Complexity:** Medium  
**Testing:** Thorough testing recommended

## Testing Strategy

1. **Test with various PR sizes:**
   ```bash
   # Small PR (1-2 files)
   gh aw run code-review.md --input small-pr.json
   
   # Medium PR (5-10 files)
   gh aw run code-review.md --input medium-pr.json
   
   # Large PR (20+ files)
   gh aw run code-review.md --input large-pr.json
   ```

2. **Monitor token usage:**
   ```bash
   gh aw logs code-review --filter token_usage
   ```

3. **Verify context retention:**
   - Check that later turns reference earlier findings
   - Confirm no repeated analysis

## Advanced Enhancements

### Add Custom Security Scanner

```yaml
tools:
  inline:
    - name: run_security_scan
      description: "Run custom security scanner"
      timeout: 120
      implementation: |
        const { execSync } = require('child_process');
        const result = execSync('npm audit --json', {
          encoding: 'utf8'
        });
        const audit = JSON.parse(result);
        return {
          vulnerabilities: audit.metadata.vulnerabilities,
          details: audit.advisories
        };
```

### Add Checkpoint Between Stages

```yaml
events:
  handlers:
    - on: turn_end
      action: checkpoint
      filter: "data.turn % 5 === 0"  # Checkpoint every 5 turns
```

### Add Budget Warnings

```yaml
events:
  handlers:
    - on: token_usage
      action: notify
      filter: "data.budget_used_percent >= 80"
      config:
        message: "Review approaching budget limit"
```

## Related Examples

- [Simple Workflow Migration](./simple-workflow/) - Basic migration example
- [Custom Tools Migration](./custom-tools/) - Converting bash scripts
- [Multi-Agent Migration](./multi-agent/) - Parallel agent coordination
