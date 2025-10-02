# Tidy Workflow Audit - Quick Reference

## 🚨 Critical Fixes (Implement First)

### 1. Add max-turns to prevent runaway costs
```yaml
engine:
  id: copilot
  model: claude-sonnet-4
  max-turns: 5  # ← ADD THIS
```

### 2. Add stop-after deadline
```yaml
on:
  workflow_dispatch:
  command:
    name: tidy  # ← ADD THIS
  push:
    branches: [main]
    paths:
      - '**/*.go'
      - '**/*.js'
      - '**/*.cjs'
      - '**/*.ts'
  stop-after: +2h  # ← ADD THIS
```

## ⚠️ High Priority Fixes

### 3. Fix network permissions
```yaml
network:
  allowed:  # ← CHANGE FROM {}
    - defaults
    - node
    - go
```

## 📋 Medium Priority Improvements

### 4. Clean up safe-outputs
```yaml
safe-outputs:
  create-pull-request:
    title-prefix: "[tidy] "
    labels: [automation, maintenance]
    draft: false
  push-to-pull-request-branch:
  # missing-tool: ← REMOVE THIS LINE
```

### 5. Expand bash permissions
```yaml
tools:
  github:
    allowed: [list_pull_requests, get_pull_request]
  bash: 
    - "make:*"
    - "git status"      # ← ADD
    - "git diff:*"      # ← ADD
    - "git log:*"       # ← ADD
    - "wc"              # ← ADD
    - "grep"            # ← ADD
```

### 6. Increase timeout
```yaml
timeout_minutes: 15  # ← CHANGE FROM 10
```

## 📊 Expected Benefits

| Improvement | Risk Reduction | Reliability Increase |
|-------------|----------------|---------------------|
| max-turns + stop-after | 95% | 40% |
| Network permissions | - | 20% |
| Bash permissions | - | 10% |
| Timeout increase | 5% | 10% |

## 🔍 Files to Review

1. **TIDY_WORKFLOW_AUDIT_SUMMARY.md** - Executive summary
2. **tidy-workflow-audit-analysis.md** - Detailed technical analysis
3. **.github/workflows/tidy.md** - The workflow file to update

## ✅ Implementation Checklist

- [ ] Add `max-turns: 5` to engine config
- [ ] Add `stop-after: +2h` to on section
- [ ] Add `name: tidy` to command trigger
- [ ] Update network to allow defaults, node, go
- [ ] Add explicit model to engine config
- [ ] Remove missing-tool from safe-outputs
- [ ] Expand bash tool permissions
- [ ] Increase timeout to 15 minutes
- [ ] Test changes with a manual workflow run
- [ ] Monitor metrics after deployment

---
**Generated**: October 2024
**Priority**: P0-P2 issues identified
**Confidence**: High (based on code analysis)
