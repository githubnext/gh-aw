---
title: Simple Workflow Migration
description: Example of migrating a simple single-job workflow from CLI to SDK mode.
---

This example demonstrates migrating a basic issue triage workflow from CLI to SDK mode.

## Original CLI Workflow

```yaml
---
title: Issue Triage
engine: copilot
on:
  issues:
    types: [opened]
tools:
  github:
    allowed: [issue_read, add_issue_labels]
---

# Triage New Issue

Analyze the newly opened issue and add appropriate labels based on:
- Issue type (bug, feature, documentation, etc.)
- Priority (high, medium, low)
- Component affected

Add relevant labels to help with organization and routing.
```

## Migrated SDK Workflow

```yaml
---
title: Issue Triage
engine:
  id: copilot
  mode: sdk
  session:
    persistent: false  # Simple one-shot task
    max-turns: 3       # Quick triage
on:
  issues:
    types: [opened]
tools:
  github:
    allowed: [issue_read, add_issue_labels]
---

# Triage New Issue

Analyze the newly opened issue and add appropriate labels based on:
- Issue type (bug, feature, documentation, etc.)
- Priority (high, medium, low)
- Component affected

Add relevant labels to help with organization and routing.
```

## Changes Made

1. **Engine Configuration**
   - Changed from `engine: copilot` to structured format
   - Added `mode: sdk`
   - Added session configuration

2. **Session Settings**
   - `persistent: false` - No need to maintain state for simple triage
   - `max-turns: 3` - Reasonable limit for quick task

3. **Instructions**
   - No changes needed - same instructions work in SDK mode

## Why Migrate?

Even for simple workflows, SDK mode provides:

- Better error handling and recovery
- Real-time monitoring capabilities
- Easier extension to multi-turn if needed later

## Migration Effort

**Time:** 5 minutes  
**Complexity:** Low  
**Testing:** Minimal (same behavior expected)

## Testing

```bash
# Test the migrated workflow
gh aw run issue-triage.md

# Compare with original CLI version
diff <(gh aw logs issue-triage-cli) <(gh aw logs issue-triage-sdk)
```

## Next Steps

This workflow could be enhanced with SDK features:
- Add event handler to log triage decisions
- Implement multi-turn if clarification needed
- Add custom validation tool for label rules
