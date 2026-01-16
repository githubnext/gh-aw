---
title: SDK Migration Examples
description: Complete examples for migrating workflows from CLI mode to SDK mode, demonstrating various migration patterns and best practices.
---

SDK mode migration examples demonstrate how to convert existing CLI-based workflows to take advantage of SDK features like multi-turn conversations, custom inline tools, and multi-agent orchestration.

## Featured Examples

### [Simple Workflow Migration](/gh-aw/examples/sdk-migrations/simple-workflow/)

Demonstrates migrating a basic single-job issue triage workflow from CLI to SDK mode. Shows minimal changes needed for simple workflows that don't require advanced SDK features. **Migration effort:** 5 minutes, **Complexity:** Low.

### [Multi-Turn Code Review](/gh-aw/examples/sdk-migrations/multi-turn-review/)

Shows how to migrate a multi-stage code review process to leverage SDK's context retention across review stages. Converts separate jobs into cohesive multi-turn conversation. **Migration effort:** 2-3 hours, **Complexity:** Medium.

### [Custom Tools Migration](/gh-aw/examples/sdk-migrations/custom-tools/)

Converts workflows that use bash scripts for custom validation to SDK mode with inline JavaScript tools. Demonstrates structured return values, parameterization, and better error handling. **Migration effort:** 4-6 hours, **Complexity:** Medium-High.

### [Multi-Agent Orchestration](/gh-aw/examples/sdk-migrations/multi-agent/)

Migrates parallel workflow jobs to SDK multi-agent architecture with specialized agents running concurrently. Shows context merging, budget allocation, and coordination strategies. **Migration effort:** 1-2 days, **Complexity:** High.

## Getting Started

Before migrating workflows to SDK mode:

### 1. Evaluate Compatibility

Use the decision framework in the [Migration Guide](/gh-aw/guides/migrate-to-sdk/) to determine if SDK mode is appropriate for your workflow.

**Migrate if you need:**
- Multi-turn conversations with context retention
- Custom inline tools with workflow-specific logic
- Real-time event handling and streaming
- Multi-agent orchestration

**Stay with CLI if:**
- Simple single-pass workflows are sufficient
- Standard MCP tools meet all requirements
- You prefer battle-tested stability

### 2. Understand SDK Features

Review the SDK documentation to understand available features:

- [SDK Engine Reference](/gh-aw/reference/engines-sdk/) - Configuration options and architecture
- [Session Management](/gh-aw/guides/sdk-sessions/) - Multi-turn conversation patterns
- [Custom Tools](/gh-aw/guides/sdk-custom-tools/) - Inline tool development
- [Event Handling](/gh-aw/guides/sdk-events/) - Real-time monitoring
- [Multi-Agent Guide](/gh-aw/guides/sdk-multi-agent/) - Agent coordination

### 3. Follow Migration Process

The [Migration Guide](/gh-aw/guides/migrate-to-sdk/) provides a step-by-step process:

1. Run compatibility checker
2. Backup current workflow
3. Update frontmatter configuration
4. Convert custom logic to inline tools
5. Test in parallel with CLI version
6. Monitor and tune performance

## Migration Complexity Guide

| Workflow Characteristics | Complexity | Recommended Example |
|-------------------------|------------|---------------------|
| Single job, standard tools | Low | [Simple Workflow](/gh-aw/examples/sdk-migrations/simple-workflow/) |
| Multiple jobs, context sharing | Medium | [Multi-Turn Review](/gh-aw/examples/sdk-migrations/multi-turn-review/) |
| Custom bash scripts | Medium-High | [Custom Tools](/gh-aw/examples/sdk-migrations/custom-tools/) |
| Parallel jobs, coordination | High | [Multi-Agent](/gh-aw/examples/sdk-migrations/multi-agent/) |

## Common Migration Patterns

### Pattern 1: CLI to SDK Basic

Convert engine configuration from simple to structured format:

```yaml
# Before (CLI)
engine: copilot

# After (SDK)
engine:
  id: copilot
  mode: sdk
```

### Pattern 2: Jobs to Multi-Turn

Convert multiple jobs into multi-turn conversation:

```yaml
# Before: Separate jobs
jobs:
  analyze:
    steps: [...]
  implement:
    needs: [analyze]
    steps: [...]

# After: Multi-turn session
engine:
  mode: sdk
  session:
    persistent: true
    max-turns: 10
```

### Pattern 3: Bash to Inline Tools

Convert bash scripts to JavaScript inline tools:

```yaml
# Before: Bash script
jobs:
  validate:
    steps:
      - run: ./validate.sh

# After: Inline tool
engine:
  mode: sdk
  tools:
    inline:
      - name: validate
        implementation: |
          const { execSync } = require('child_process');
          return execSync('./validate.sh');
```

### Pattern 4: Parallel Jobs to Multi-Agent

Convert parallel jobs to coordinated agents:

```yaml
# Before: Parallel jobs
jobs:
  security: [...]
  performance: [...]
  quality: [...]

# After: Multi-agent
engine:
  mode: sdk
  agents:
    - name: security_expert
      parallel: true
    - name: performance_expert
      parallel: true
    - name: quality_expert
      parallel: true
  coordination:
    strategy: parallel
```

## Best Practices

### 1. Start Simple

Begin with simple workflows to gain familiarity with SDK mode before tackling complex migrations.

### 2. Test Thoroughly

Always test migrated workflows in development before deploying to production:

```bash
# Run both versions in parallel
gh aw run workflow-cli.md
gh aw run workflow-sdk.md

# Compare results
gh aw logs workflow-cli
gh aw logs workflow-sdk
```

### 3. Monitor Token Usage

SDK workflows typically use more tokens due to context retention:

```yaml
engine:
  budget:
    max-tokens: 50000
  events:
    handlers:
      - on: token_usage
        action: log
```

### 4. Implement Progressive Rollout

For production workflows:
1. Test in development
2. Deploy to staging
3. Canary deployment (10% traffic)
4. Full rollout or rollback

### 5. Document Changes

Update workflow documentation to reflect SDK-specific features and behavior changes.

## Troubleshooting

### Issue: Workflow Behaves Differently

**Solution:** SDK multi-turn execution differs from CLI single-pass. Review instructions for turn-appropriate language.

### Issue: Increased Token Usage

**Solution:** Implement budget controls and monitor usage:

```yaml
engine:
  budget:
    max-tokens: 50000
    warn-threshold: 40000
```

### Issue: Migration Too Complex

**Solution:** Consider if CLI mode is more appropriate for this workflow. Not all workflows benefit from SDK features.

## Additional Resources

- [Migration Guide](/gh-aw/guides/migrate-to-sdk/) - Complete migration documentation
- [SDK Engine Reference](/gh-aw/reference/engines-sdk/) - Configuration details
- [CLI Command Patterns](/gh-aw/reference/command-triggers/) - Understanding CLI mode

## Community Examples

Share your migration examples and learn from others in the [GitHub Discussions](https://github.com/githubnext/gh-aw/discussions).

## Need Help?

- **Documentation issues:** File an issue on [GitHub](https://github.com/githubnext/gh-aw/issues)
- **Migration questions:** Ask in [GitHub Discussions](https://github.com/githubnext/gh-aw/discussions)
- **Bug reports:** Create an issue with "SDK migration" label
