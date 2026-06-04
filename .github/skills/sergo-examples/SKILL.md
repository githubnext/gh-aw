---
name: sergo-examples
description: Optional Sergo examples for cache formats and reporting templates.
---

Use this skill only when Sergo needs concrete format examples.

## Example: cached tools snapshot

```json
{
  "last_updated": "2026-01-15T12:00:00Z",
  "tools": [
    {"name": "tool-name-1", "description": "..."},
    {"name": "tool-name-2", "description": "..."}
  ]
}
```

## Example: strategy history entries

```json
{"date": "2026-01-14", "strategy": "symbol-analysis", "tools": ["find-symbol", "get-definition"], "findings": 3, "tasks_created": 2, "success_score": 8}
{"date": "2026-01-13", "strategy": "type-inspection", "tools": ["get-hover", "get-type"], "findings": 5, "tasks_created": 3, "success_score": 9}
```

## Example: task template

```markdown
### Task [N]: [Short Title]

**Issue Type**: [Symbol Analysis / Type Inspection / etc.]
**Problem**: [Clear description]
**Location(s)**: [file paths and line references]
**Impact**: Severity, affected files, risk
**Recommendation**: [Specific fix]
**Validation**: Existing tests, Serena verification, related-pattern check, docs if needed
**Estimated Effort**: [Small/Medium/Large]
```

## Example: discussion structure

```markdown
# 🔬 Sergo Report: [Strategy Name]

**Date**: [YYYY-MM-DD]
**Strategy**: [Name]
**Success Score**: [X/10]

## Executive Summary
## 🛠️ Serena Tools Update
## 📊 Strategy Selection
## 🔍 Analysis Execution
## 📋 Detailed Findings
## ✅ Improvement Tasks Generated
## 📈 Success Metrics
## 📊 Historical Context
## 🎯 Recommendations
## 🔄 Next Run Preview
```
