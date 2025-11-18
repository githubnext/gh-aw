---
tools:
  bash:
    - "gh pr list *"
    - "gh api *"
    - "jq *"
    - "/tmp/gh-aw/jqschema.sh"
    - "mkdir *"
    - "date *"

steps:
  - name: Fetch Copilot PR data
    env:
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Create output directory
      mkdir -p /tmp/gh-aw/pr-data

      # Calculate date 30 days ago
      DATE_30_DAYS_AGO=$(date -d '30 days ago' '+%Y-%m-%d' 2>/dev/null || date -v-30d '+%Y-%m-%d')

      # Search for PRs from copilot/* branches in the last 30 days using gh CLI
      # Using branch prefix search (head:copilot/) instead of author for reliability
      echo "Fetching Copilot PRs from the last 30 days..."
      gh pr list --repo ${{ github.repository }} \
        --search "head:copilot/ created:>=${DATE_30_DAYS_AGO}" \
        --state all \
        --json number,title,author,headRefName,createdAt,state,url,body,labels,updatedAt,closedAt,mergedAt \
        --limit 1000 \
        > /tmp/gh-aw/pr-data/copilot-prs.json

      # Generate schema for reference
      /tmp/gh-aw/jqschema.sh < /tmp/gh-aw/pr-data/copilot-prs.json > /tmp/gh-aw/pr-data/copilot-prs-schema.json

      echo "PR data saved to /tmp/gh-aw/pr-data/copilot-prs.json"
      echo "Schema saved to /tmp/gh-aw/pr-data/copilot-prs-schema.json"
      echo "Total PRs found: $(jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json)"
---

<!--
## Copilot PR Data Fetch

This shared component fetches pull request data for GitHub Copilot agent-created PRs from the last 30 days.

### What It Does

1. Creates the output directory at `/tmp/gh-aw/pr-data/`
2. Calculates the date 30 days ago (cross-platform compatible)
3. Fetches all PRs from branches starting with `copilot/` using `gh pr list`
4. Saves the full PR data to `/tmp/gh-aw/pr-data/copilot-prs.json`
5. Generates a schema of the data structure at `/tmp/gh-aw/pr-data/copilot-prs-schema.json`

### Output Files

- **`/tmp/gh-aw/pr-data/copilot-prs.json`**: Full PR data including number, title, author, branch name, timestamps, state, URL, body, labels, etc.
- **`/tmp/gh-aw/pr-data/copilot-prs-schema.json`**: JSON schema showing the structure of the PR data

### Usage

Import this component in your workflow:

```yaml
imports:
  - shared/copilot-pr-data-fetch.md
  - shared/jqschema.md  # Required for schema generation
```

Then access the pre-fetched data in your workflow prompt:

```bash
# Get PRs from the last 24 hours
TODAY="$(date -d '24 hours ago' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -v-24H '+%Y-%m-%dT%H:%M:%SZ')"
jq --arg today "$TODAY" '[.[] | select(.createdAt >= $today)]' /tmp/gh-aw/pr-data/copilot-prs.json

# Count total PRs
jq 'length' /tmp/gh-aw/pr-data/copilot-prs.json

# Get PR numbers
jq '[.[].number]' /tmp/gh-aw/pr-data/copilot-prs.json
```

### Requirements

- Requires `jqschema.md` to be imported for schema generation
- Uses `gh pr list` with the `--search "head:copilot/"` pattern for reliable Copilot PR detection
- Cross-platform date calculation (works on both GNU and BSD date commands)

### Why Branch-Based Search?

GitHub Copilot creates branches with the `copilot/` prefix, making branch-based search more reliable than author-based search which may miss PRs due to author name variations.
-->
