---
title: Migrate to SDK Mode
description: Comprehensive guide for migrating workflows from CLI mode to SDK mode.
sidebar:
  order: 729
---

This guide helps you migrate existing CLI-based workflows to SDK mode, with decision frameworks, step-by-step instructions, and common migration patterns.

## Should I Migrate to SDK Mode?

### Decision Framework

Use this framework to determine if SDK mode is right for your workflow:

#### Migrate to SDK if you need:

✅ **Multi-turn conversations** - Your workflow requires back-and-forth dialogue with context retention  
✅ **Custom inline tools** - You need workflow-specific logic beyond standard MCP tools  
✅ **Real-time event handling** - You want to monitor and respond to events during execution  
✅ **Multi-agent orchestration** - You need multiple specialized agents working together  
✅ **Programmatic control flow** - You need retry logic, branching, or conditional execution  
✅ **Cost-aware execution** - You want fine-grained budget controls and monitoring  
✅ **Session restoration** - You need to resume workflows after interruption  
✅ **Dynamic tool generation** - You want to create tools based on workflow state

#### Stay with CLI if:

✅ **Simple single-pass workflows** - One-shot task execution is sufficient  
✅ **No conversation context needed** - Each run is independent  
✅ **Standard MCP tools work** - Existing tools meet all requirements  
✅ **Battle-tested stability** - You prioritize proven production reliability  
✅ **Simpler configuration** - You want minimal setup complexity  
✅ **Faster execution** - Simple tasks complete faster in CLI mode

### Migration Complexity Assessment

Estimate migration effort based on your workflow characteristics:

| Workflow Type | Complexity | Estimated Effort |
|---------------|------------|------------------|
| Simple single-job workflow | Low | 30 minutes |
| Multi-job workflow with dependencies | Medium | 2-4 hours |
| Workflow with custom bash scripts | Medium-High | 4-8 hours |
| Complex workflow with external integrations | High | 1-3 days |
| Mission-critical production workflow | Very High | Plan, test, gradual rollout |

## Step-by-Step Migration Process

### Step 1: Evaluate Compatibility

Run the compatibility checker to identify potential issues:

```bash
gh aw check-sdk-compatibility workflow.md
```

**Output example:**
```
✓ Workflow structure compatible
✗ Uses custom bash scripts (consider inline tools)
✓ MCP tools available in SDK mode
⚠ Complex multi-job workflow (test thoroughly)

Compatibility Score: 75/100
Recommendation: Migrate with caution, test extensively
```

### Step 2: Backup Current Workflow

Create a backup before making changes:

```bash
# Backup workflow file
cp workflow.md workflow-cli.md.backup

# Backup lock file
cp workflow.lock.yml workflow-cli.lock.yml.backup
```

### Step 3: Update Frontmatter

Convert CLI engine configuration to SDK mode:

#### Before (CLI mode):
```yaml
---
engine: copilot
tools:
  github:
    allowed: [issue_read, add_issue_comment]
network:
  allowed:
    - defaults
    - "api.example.com"
---
```

#### After (SDK mode):
```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 5
tools:
  github:
    allowed: [issue_read, add_issue_comment]
network:
  allowed:
    - defaults
    - "api.example.com"
---
```

**Key changes:**
- `engine: copilot` → `engine: { id: copilot, mode: sdk }`
- Add `session` configuration
- Other fields remain compatible

### Step 4: Convert Custom Logic

If your workflow uses bash scripts or custom actions, convert to inline tools:

#### Before (CLI with bash):
```yaml
---
engine: copilot
tools:
  bash:
    - validate.sh
---
# Workflow with Custom Validation

Review code and run ./validate.sh for checks.
```

#### After (SDK with inline tool):
```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: validate_code
        description: "Run custom validation checks"
        implementation: |
          const { execSync } = require('child_process');
          const result = execSync('./validate.sh');
          return { 
            valid: result.exitCode === 0,
            output: result.toString()
          };
tools:
  github:
    allowed: [issue_read]
---
# Workflow with Custom Validation

Review code and use validate_code tool for checks.
```

### Step 5: Test in Parallel

Run both CLI and SDK versions side-by-side:

```bash
# Test CLI version
gh aw run workflow-cli.md

# Test SDK version
gh aw run workflow-sdk.md

# Compare results
diff <(gh aw logs workflow-cli) <(gh aw logs workflow-sdk)
```

### Step 6: Monitor and Tune

Use event handlers to monitor performance:

```yaml
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true
    handlers:
      - on: token_usage
        action: log
      
      - on: completion
        action: custom
        config:
          implementation: |
            console.log('SDK Migration Metrics:', {
              turns: event.data.turns,
              tokens: event.data.total_tokens,
              duration: event.data.duration_seconds
            });
```

### Step 7: Gradual Rollout

For production workflows, use a phased approach:

1. **Week 1**: Test in development environment
2. **Week 2**: Deploy to staging with monitoring
3. **Week 3**: Canary deployment (10% of traffic)
4. **Week 4**: Full rollout or rollback based on metrics

## CLI to SDK Mapping

### Configuration Mapping

| CLI Config | SDK Equivalent | Notes |
|------------|----------------|-------|
| `engine: copilot` | `engine: { id: copilot, mode: sdk }` | Basic conversion |
| `tools: {...}` | `tools: {...}` | No change needed |
| `network: {...}` | `network: {...}` | No change needed |
| `permissions: {...}` | `permissions: {...}` | No change needed |
| Custom bash scripts | Inline tools | Requires rewrite |
| Multiple jobs | Session turns | Architecture change |

### Tool Mapping

| CLI Tool | SDK Equivalent | Migration |
|----------|----------------|-----------|
| MCP server tools | Same MCP tools | No change |
| `bash` tool | Inline tools | Convert to JavaScript |
| GitHub Actions | Inline tools or MCP | Convert or use MCP |
| External scripts | Inline tools | Port to JavaScript |

### Feature Mapping

| CLI Feature | SDK Feature | Migration Path |
|-------------|-------------|----------------|
| Single-pass execution | Multi-turn session | Add session config |
| Job dependencies | Sequential agents | Convert jobs to agents |
| Matrix builds | Parallel agents | Convert matrix to agents |
| Conditional steps | Event handlers | Rewrite as handlers |
| Outputs/artifacts | Session state | Use state persistence |

## Common Migration Patterns

### Pattern 1: Simple Single-Job Workflow

**Minimal changes needed**

#### Before (CLI):
```yaml
---
engine: copilot
tools:
  github:
    allowed: [issue_read, add_issue_comment]
---
# Triage Issue

Read issue #{{ issue.number }} and add triage label.
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
tools:
  github:
    allowed: [issue_read, add_issue_comment]
---
# Triage Issue

Read issue #{{ issue.number }} and add triage label.
```

**Changes:**
- Update engine configuration
- Instructions remain the same

### Pattern 2: Multi-Job Workflow

**Convert jobs to sequential agents or multi-turn sessions**

#### Before (CLI):
```yaml
---
engine: copilot
jobs:
  analyze:
    steps:
      - name: Analyze code
        uses: copilot
  
  implement:
    needs: [analyze]
    steps:
      - name: Implement fix
        uses: copilot
  
  review:
    needs: [implement]
    steps:
      - name: Review changes
        uses: copilot
---
# Multi-Stage Workflow
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 15
tools:
  github:
    allowed: [issue_read, create_pull_request]
---
# Multi-Stage Workflow

Stage 1: Analyze code and identify issues
Stage 2: Implement fixes based on analysis
Stage 3: Review changes for quality
```

**Changes:**
- Remove jobs structure
- Use multi-turn session with stages
- Context automatically flows between stages

### Pattern 3: Workflow with Custom Scripts

**Convert bash scripts to inline tools**

#### Before (CLI):
```yaml
---
engine: copilot
tools:
  bash:
    - scripts/validate.sh
    - scripts/analyze.sh
---
# Code Review with Validation

Review code using validate.sh and analyze.sh scripts.
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: validate_code
        description: "Validate code meets standards"
        implementation: |
          const { execSync } = require('child_process');
          try {
            const result = execSync('./scripts/validate.sh', {
              encoding: 'utf8'
            });
            return { valid: true, output: result };
          } catch (error) {
            return { valid: false, error: error.message };
          }
      
      - name: analyze_code
        description: "Analyze code quality"
        implementation: |
          const { execSync } = require('child_process');
          const result = execSync('./scripts/analyze.sh', {
            encoding: 'utf8'
          });
          return { analysis: result };
tools:
  github:
    allowed: [issue_read]
---
# Code Review with Validation

Use validate_code and analyze_code tools for code review.
```

**Changes:**
- Convert shell scripts to inline JavaScript tools
- Wrap script execution in JavaScript
- Provide structured return values

### Pattern 4: Conditional Execution

**Use event handlers for dynamic control**

#### Before (CLI):
```yaml
---
engine: copilot
jobs:
  check:
    steps:
      - name: Check condition
        id: check
        uses: copilot
  
  action:
    needs: [check]
    if: steps.check.outputs.proceed == 'true'
    steps:
      - name: Perform action
        uses: copilot
---
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
  tools:
    inline:
      - name: check_condition
        description: "Check if action should proceed"
        implementation: |
          // Custom condition logic
          const shouldProceed = checkCondition();
          return { proceed: shouldProceed };
---
# Conditional Workflow

First, use check_condition tool to determine if action needed.
If proceed is true, perform the action.
```

**Changes:**
- Replace conditional jobs with conditional instructions
- Use inline tools for condition checking
- Let agent decide based on tool results

### Pattern 5: Matrix/Parallel Execution

**Convert to parallel agents**

#### Before (CLI):
```yaml
---
engine: copilot
jobs:
  test:
    strategy:
      matrix:
        node-version: [16, 18, 20]
    steps:
      - name: Test on Node ${{ matrix.node-version }}
        uses: copilot
---
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  agents:
    - name: test_node_16
      role: "Test on Node 16"
      parallel: true
    
    - name: test_node_18
      role: "Test on Node 18"
      parallel: true
    
    - name: test_node_20
      role: "Test on Node 20"
      parallel: true
    
    - name: aggregate
      role: "Aggregate test results"
      session:
        restore: merge
  
  coordination:
    strategy: hybrid
---
# Parallel Testing

Test across Node 16, 18, and 20 in parallel, then aggregate results.
```

**Changes:**
- Replace matrix strategy with parallel agents
- Add aggregator agent for results
- Use session merge for consolidation

### Pattern 6: Complex Multi-Stage

**Progressive enhancement with context retention**

#### Before (CLI - not possible):
```yaml
# Cannot implement progressive multi-stage with full context
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 20
  budget:
    max-tokens: 100000
  events:
    handlers:
      - on: token_usage
        action: log
        threshold: 20000
tools:
  github:
    allowed: [issue_read, pull_request_read, create_pull_request]
---
# Progressive Code Review

Stage 1: Initial PR review (turns 1-4)
- Quick scan for obvious issues
- Identify areas needing deep dive

Stage 2: Deep analysis (turns 5-10)
- Detailed review of flagged areas
- Security and performance analysis

Stage 3: Follow-up (turns 11-15)
- Address developer questions
- Re-review after changes

Stage 4: Final approval (turns 16-20)
- Validate all issues resolved
- Provide final approval or rejection
```

**Changes:**
- Enable long-running session
- Implement staged approach with context
- Monitor budget throughout process

## Migration Examples

### Example 1: Issue Triage

#### Before (CLI):
```yaml
---
engine: copilot
tools:
  github:
    allowed: [issue_read, add_issue_labels]
---
# Triage New Issue

Analyze issue and add appropriate labels.
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    max-turns: 3  # Quick triage
tools:
  github:
    allowed: [issue_read, add_issue_labels]
---
# Triage New Issue

Analyze issue and add appropriate labels.
```

**Migration effort:** 5 minutes  
**Changes:** Minimal, just engine config

### Example 2: PR Review with Tests

#### Before (CLI):
```yaml
---
engine: copilot
jobs:
  review:
    steps:
      - uses: copilot
  
  test:
    needs: [review]
    steps:
      - run: npm test
---
# PR Review and Test

Review PR, then run tests.
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 8
  tools:
    inline:
      - name: run_tests
        description: "Run test suite"
        implementation: |
          const { execSync } = require('child_process');
          try {
            const result = execSync('npm test', { encoding: 'utf8' });
            return { passed: true, output: result };
          } catch (error) {
            return { passed: false, output: error.stdout };
          }
tools:
  github:
    allowed: [pull_request_read, add_pull_request_comment]
---
# PR Review and Test

Step 1: Review the PR code changes
Step 2: Use run_tests tool to execute test suite
Step 3: Comment on PR with review and test results
```

**Migration effort:** 1 hour  
**Changes:** Convert test job to inline tool

### Example 3: Security Audit

#### Before (CLI):
```yaml
---
engine: copilot
jobs:
  scan:
    steps:
      - run: npm audit
      - uses: copilot
---
# Security Audit

Run npm audit and analyze results.
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: security_audit
        description: "Run npm security audit"
        timeout: 120
        implementation: |
          const { execSync } = require('child_process');
          try {
            execSync('npm audit --json > audit.json');
            const audit = require('./audit.json');
            return {
              vulnerabilities: audit.metadata.vulnerabilities,
              total: audit.metadata.vulnerabilities.total
            };
          } catch (error) {
            // npm audit exits 1 if vulnerabilities found
            const audit = require('./audit.json');
            return {
              vulnerabilities: audit.metadata.vulnerabilities,
              total: audit.metadata.vulnerabilities.total
            };
          }
tools:
  github:
    allowed: [issue_read, add_issue_comment]
---
# Security Audit

Use security_audit tool to check for vulnerabilities and report findings.
```

**Migration effort:** 1-2 hours  
**Changes:** Convert npm audit to inline tool

### Example 4: Documentation Generation

#### Before (CLI):
```yaml
---
engine: copilot
jobs:
  analyze:
    steps:
      - uses: copilot
  
  generate:
    needs: [analyze]
    steps:
      - uses: copilot
  
  review:
    needs: [generate]
    steps:
      - uses: copilot
---
# Documentation Pipeline

Analyze → Generate → Review
```

#### After (SDK):
```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 15
  tools:
    inline:
      - name: save_docs
        description: "Save generated documentation"
        parameters:
          type: object
          required: [content, filename]
          properties:
            content:
              type: string
            filename:
              type: string
        implementation: |
          const fs = require('fs');
          fs.writeFileSync(filename, content, 'utf8');
          return { saved: true, path: filename };
tools:
  github:
    allowed: [issue_read, create_pull_request]
---
# Documentation Pipeline

Stage 1: Analyze codebase and identify documentation gaps
Stage 2: Generate comprehensive documentation
Stage 3: Use save_docs tool to write files
Stage 4: Review generated documentation for quality
Stage 5: Create PR with documentation updates
```

**Migration effort:** 2-4 hours  
**Changes:** Multi-job to multi-turn, add file save tool

## Common Pitfalls and Solutions

### Pitfall 1: Assuming Same Execution Model

**Problem:** Expecting SDK to work exactly like CLI

**Solution:**
- CLI: Single-pass, stateless
- SDK: Multi-turn, stateful
- Design for conversation flow, not job flow

### Pitfall 2: Not Setting Turn Limits

**Problem:** Workflows run indefinitely

**Solution:**
```yaml
engine:
  session:
    max-turns: 10  # Always set a reasonable limit
```

### Pitfall 3: Ignoring Budget Controls

**Problem:** Unexpected high token usage

**Solution:**
```yaml
engine:
  budget:
    max-tokens: 50000  # Set appropriate limits
    warn-threshold: 40000
  events:
    handlers:
      - on: token_usage
        action: log
```

### Pitfall 4: Over-Engineering

**Problem:** Migrating simple workflows that don't need SDK

**Solution:**
- Use the decision framework
- Keep simple workflows in CLI mode
- Only migrate when SDK features needed

### Pitfall 5: Not Testing Thoroughly

**Problem:** Deploying untested SDK workflows

**Solution:**
- Test in development first
- Run parallel CLI/SDK comparison
- Monitor carefully during rollout

### Pitfall 6: Losing Error Context

**Problem:** Converting try-catch logic incorrectly

**Solution:**
```yaml
# Preserve error handling in inline tools
tools:
  inline:
    - name: risky_operation
      implementation: |
        try {
          const result = performOperation();
          return { success: true, result };
        } catch (error) {
          return {
            success: false,
            error: error.message,
            stack: error.stack,
            recoverable: isRecoverable(error)
          };
        }
```

### Pitfall 7: Not Using Event Handlers

**Problem:** Missing SDK's monitoring capabilities

**Solution:**
```yaml
# Add monitoring from the start
engine:
  events:
    streaming: true
    handlers:
      - on: token_usage
        action: log
      - on: error
        action: notify
```

## Rollback Strategy

If SDK migration encounters issues:

### Immediate Rollback

```bash
# Revert to CLI version
mv workflow-cli.md.backup workflow.md
mv workflow-cli.lock.yml.backup workflow.lock.yml

# Rebuild
gh aw compile workflow.md
```

### Conditional Rollback

Use feature flag to toggle between CLI and SDK:

```yaml
---
engine:
  id: copilot
  mode: ${{ env.USE_SDK_MODE == 'true' && 'sdk' || 'cli' }}
---
```

### Canary Rollback

Monitor metrics and rollback based on thresholds:

```yaml
# Monitor success rate
# If < 95%, trigger rollback
```

## Success Checklist

After migration, verify:

- [ ] Workflow completes successfully
- [ ] All features work as expected
- [ ] Token usage is within acceptable range
- [ ] Execution time is reasonable
- [ ] Error handling works correctly
- [ ] Monitoring and alerts configured
- [ ] Documentation updated
- [ ] Team trained on new workflow
- [ ] Rollback plan documented
- [ ] Production testing completed

## Getting Help

If you encounter issues during migration:

1. **Check documentation:**
   - [SDK Engine Reference](/gh-aw/reference/engines-sdk/)
   - [Session Management](/gh-aw/guides/sdk-sessions/)
   - [Custom Tools](/gh-aw/guides/sdk-custom-tools/)
   - [Event Handling](/gh-aw/guides/sdk-events/)

2. **Review examples:** See `docs/src/content/docs/examples/sdk-migrations/`

3. **Community support:** Ask in GitHub Discussions

4. **File issues:** Report bugs on GitHub Issues

## Related Documentation

- [SDK Engine Reference](/gh-aw/reference/engines-sdk/) - Complete SDK configuration
- [Session Management](/gh-aw/guides/sdk-sessions/) - Multi-turn conversations
- [Custom Tools](/gh-aw/guides/sdk-custom-tools/) - Creating inline tools
- [Event Handling](/gh-aw/guides/sdk-events/) - Real-time event processing
- [Multi-Agent Guide](/gh-aw/guides/sdk-multi-agent/) - Coordinating multiple agents
