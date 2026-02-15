---
title: Reaction Without Status Comments
description: Example workflow showing how to decouple ai-reaction emoji from status comments
---

This example demonstrates the new `status-comment` field that allows you to have reaction emojis without posting "started/completed" status comments.

## Use Case

You want visual feedback on issues (via reaction emoji) but don't want to clutter the conversation with workflow status updates.

## Example Workflow

```markdown
---
name: reaction-only-workflow
description: Uses ai-reaction without status comments
on:
  reaction: eyes           # Adds 👀 reaction
  status-comment: false    # No "started/completed" comments
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

### Reaction Only (New Feature)

```yaml
on:
  reaction: eyes
  status-comment: false
```

**Result**: 👀 reaction only, no comments

---

### Default Behavior (Backward Compatible)

```yaml
on:
  reaction: eyes
  # status-comment not specified
```

**Result**: 👀 reaction + status comments (preserves existing behavior)

---

### Explicit Status Comments

```yaml
on:
  reaction: eyes
  status-comment: true
```

**Result**: 👀 reaction + status comments (same as default)

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
  # neither reaction nor status-comment specified
  issues:
    types: [opened]
```

**Result**: No reaction, no status comments (silent execution)

## When to Use

- **Use `status-comment: false`** when you want:
  - Visual acknowledgment (reaction) without noisy status updates
  - Cleaner issue/PR conversations
  - Multiple workflows on the same trigger (avoid comment spam)

- **Use default behavior** (reaction with comments) when you want:
  - Full transparency about workflow execution
  - Links to workflow runs for debugging
  - Status updates in the conversation thread
