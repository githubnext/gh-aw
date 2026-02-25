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
    file-glob: ["memory/(your-analysis-name)/*.json", "memory/(your-analysis-name)/*.jsonl",
                "memory/(your-analysis-name)/*.csv", "memory/(your-analysis-name)/*.md"]
    max-file-size: 102400  # 100KB
```

### Common jq Queries

```bash
# Count total PRs
jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json

# PRs from last 7 days
jq '[.[] | select(.createdAt >= "'"$(date -d '7 days ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-7d '+%Y-%m-%dT%H:%M:%SZ')"'")]' /tmp/gh-aw/pr-data/copilot-prs.json

# Merged vs closed stats
jq 'group_by(.state) | map({state: .[0].state, count: length})' /tmp/gh-aw/pr-data/copilot-prs.json
```
