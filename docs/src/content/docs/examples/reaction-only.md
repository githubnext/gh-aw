---
title: Reaction Without Status Comments
description: Example workflow showing how to use ai-reaction and status-comment independently
---

This example demonstrates how `reaction` and `status-comment` work independently - both must be explicitly enabled.

## Use Case

You want visual feedback on issues (via reaction emoji) but don't want to clutter the conversation with workflow status updates.

## Example Workflow

```markdown
---
name: reaction-only-workflow
description: Uses ai-reaction without status comments
on:
  reaction: eyes           # Adds 👀 reaction
  # status-comment not specified = no status comments
  issues:
    types: [opened, labeled]
engine: copilot
safe-outputs:
  create-issue:
    max: 3
---

Analyze this issue and create sub-issues if needed.
```

## Behavior

When this workflow runs:

1. ✅ **Adds 👀 reaction** to the triggering issue (immediate feedback)
2. ❌ **No "Workflow starting..." comment** 
3. ❌ **No "Workflow completed..." comment**
4. ✅ **Agent still executes** and creates issues if needed

## Comparison with Other Configurations

### Reaction Only (No Status Comments)

```yaml
on:
  reaction: eyes
  # status-comment not specified or set to false
```

**Result**: 👀 reaction only, no comments

---

### Reaction + Status Comments

```yaml
on:
  reaction: eyes
  status-comment: true    # Must be explicit
```

**Result**: 👀 reaction + status comments

---

### Status Comments Without Reaction

```yaml
on:
  status-comment: true
  # no reaction specified
```

**Result**: Status comments only, no reaction

---

### No Reaction or Comments

```yaml
on:
  status-comment: true
  # no reaction specified
```

**Result**: Status comments only, no reaction

---

### Neither Enabled

```yaml
on:
  # neither reaction nor status-comment specified
  issues:
    types: [opened]
```

**Result**: No reaction, no status comments (silent execution)

## When to Use

- **Use reaction only** (no `status-comment`) when you want:
  - Visual acknowledgment without noisy status updates
  - Cleaner issue/PR conversations
  - Multiple workflows on the same trigger (avoid comment spam)

- **Use both** (`reaction` + `status-comment: true`) when you want:
  - Full transparency about workflow execution
  - Visual acknowledgment + links to workflow runs for debugging
  - Status updates in the conversation thread
