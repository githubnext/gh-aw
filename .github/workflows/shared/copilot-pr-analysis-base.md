---
tools:
  github:
    toolsets: [default]
  edit:
  bash: ["*"]

imports:
  - shared/jqschema.md
  - shared/reporting.md
  - shared/copilot-pr-data-fetch.md
---

## Copilot PR Analysis Base

Pre-fetched Copilot PR data is available at `/tmp/gh-aw/pr-data/copilot-prs.json` (last 30 days, up to 1000 PRs from `copilot/*` branches).

### Historical Data with repo-memory

Each analysis workflow should store historical results in `repo-memory` for trend tracking.
Recommended repo-memory configuration (add inline to your workflow's frontmatter):

```yaml
tools:
  repo-memory:
    branch-name: memory/(your-analysis-name)
    description: "Historical (analysis type) results"
    file-glob: ["*.json", "*.jsonl", "*.csv", "*.md"]
    max-file-size: 102400  # 100KB
```

> **Warning**: File glob patterns are matched against the **relative file path** from the artifact directory, not the branch path. Use bare extension patterns like `*.json` — never include the branch name (e.g. `memory/(your-analysis-name)/*.json` is incorrect and will silently skip all files).

### Common jq Queries

```bash
# Count total PRs
jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json

# PRs from last 7 days
jq '[.[] | select(.createdAt >= "'"$(date -d '7 days ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-7d '+%Y-%m-%dT%H:%M:%SZ')"'")]' /tmp/gh-aw/pr-data/copilot-prs.json

# Merged vs closed stats
jq 'group_by(.state) | map({state: .[0].state, count: length})' /tmp/gh-aw/pr-data/copilot-prs.json
```
