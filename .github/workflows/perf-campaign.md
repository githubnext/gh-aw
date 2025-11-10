---
on:
  schedule:
    - cron: "0 9 * * 1"  # Every Monday at 9am
  workflow_dispatch:

engine: copilot

permissions:
  contents: read
  issues: write
  repository-projects: write

safe-outputs:
  create-issue:
    max: 5
  update-project:
    max: 10

tools:
  github:
    mode: remote
    toolsets: [default]
---

# Performance Improvement Campaign - Q1 2025

You are managing a performance improvement campaign for Q1 2025. Your job is to:

1. **Ensure the campaign project exists**: Look for a project board named "Performance Q1 2025"
   - If it doesn't exist, create it using `update-project` with:
     - project: "Performance Q1 2025"
     - description: "Campaign to improve application performance by 30% in Q1 2025"
     - create_if_missing: true
   - The system will automatically generate a campaign ID (like `performance-q1-2025-a3f2b4c8`)

2. **Scan the repository for performance issues**:
   - Use the GitHub MCP to search for:
     - TODO comments mentioning "performance", "slow", "optimize"
     - Files with "FIXME: performance" comments
     - Issues labeled with "performance" or "slow"
   
3. **Create tracking issues** for each performance concern found:
   - Title: Brief description of the performance issue
   - Body: Include:
     - File location and code context
     - Why this is a performance concern
     - Suggested optimization approach
     - Estimated impact (high/medium/low)
   - Labels: "performance", "campaign-q1-2025"

4. **Add issues to the campaign board**:
   - For each created issue, use `update-project` to add it to the board:
     - project: "Performance Q1 2025"
     - content_type: "issue"
     - content_number: (the issue number you just created)
     - fields:
       - Status: "To Do"
       - Priority: (based on estimated impact: "High", "Medium", or "Low")
       - Effort: (estimate: "S" for < 4h, "M" for 4-8h, "L" for > 8h)
   - The campaign ID label will be automatically added

## Example Safe Outputs

**Create the campaign project (first run):**
```json
{
  "type": "update-project",
  "project": "Performance Q1 2025",
  "description": "Campaign to improve application performance by 30% in Q1 2025",
  "create_if_missing": true
}
```

**Create a performance tracking issue:**
```json
{
  "type": "create-issue",
  "title": "Optimize database query in user search",
  "body": "**File**: `pkg/db/users.go:45`\n\n**Issue**: Full table scan on users table during search\n\n**Optimization**: Add index on `username` and `email` columns\n\n**Impact**: High - affects 80% of user searches",
  "labels": ["performance", "campaign-q1-2025", "database"]
}
```

**Add issue to campaign board:**
```json
{
  "type": "update-project",
  "project": "Performance Q1 2025",
  "content_type": "issue",
  "content_number": 123,
  "fields": {
    "Status": "To Do",
    "Priority": "High",
    "Effort": "M"
  }
}
```

## Notes

- Focus on actionable performance improvements with measurable impact
- Prioritize issues that affect user-facing features
- Group related optimizations together in issue descriptions
- The campaign ID is automatically generated and tracked in the project description
- Issues get labeled with `campaign:[id]` automatically for easy filtering
