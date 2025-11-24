---
description: Automatically closes discussions created by github-actions bot that are older than 1 week
on:
  workflow_dispatch:
  schedule:
    - cron: "0 6 */2 * *"  # Every 2 days at 6 AM UTC
permissions:
  contents: read
  actions: read
  discussions: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [default, discussions]
safe-outputs:
  close-discussion:
    max: 100
timeout-minutes: 10
strict: true
steps:
  - name: Fetch open discussions
    id: fetch-discussions
    run: |
      # Use GraphQL to fetch all open discussions in one query
      # Filter to only get discussions created by github-actions[bot]
      # Calculate cutoff date (7 days ago)
      CUTOFF_DATE=$(date -u -d '7 days ago' +%Y-%m-%dT%H:%M:%SZ)
      
      # Fetch discussions with pagination
      DISCUSSIONS_FILE="/tmp/gh-aw/discussions.json"
      echo '[]' > "$DISCUSSIONS_FILE"
      
      CURSOR=""
      HAS_NEXT_PAGE=true
      PAGE_COUNT=0
      
      while [ "$HAS_NEXT_PAGE" = "true" ]; do
        if [ -z "$CURSOR" ]; then
          CURSOR_ARG=""
        else
          CURSOR_ARG=", after: \"$CURSOR\""
        fi
        
        RESULT=$(gh api graphql -f query="
          query {
            repository(owner: \"${{ github.repository_owner }}\", name: \"${{ github.event.repository.name }}\") {
              discussions(first: 100, states: OPEN${CURSOR_ARG}) {
                pageInfo {
                  hasNextPage
                  endCursor
                }
                nodes {
                  number
                  title
                  createdAt
                  author {
                    login
                  }
                }
              }
            }
          }
        ")
        
        # Extract discussions created by github-actions[bot]
        echo "$RESULT" | jq -r --arg cutoff "$CUTOFF_DATE" '
          .data.repository.discussions.nodes 
          | map(select(
              .author != null and 
              .author.login == "github-actions[bot]" and 
              .createdAt < $cutoff
            ))
          | map({number, title, createdAt, author: .author.login})
        ' | jq -s 'add' > /tmp/gh-aw/temp_discussions.json
        
        # Merge with existing discussions
        jq -s 'add | unique_by(.number)' "$DISCUSSIONS_FILE" /tmp/gh-aw/temp_discussions.json > /tmp/gh-aw/merged.json
        mv /tmp/gh-aw/merged.json "$DISCUSSIONS_FILE"
        
        # Check if there are more pages
        HAS_NEXT_PAGE=$(echo "$RESULT" | jq -r '.data.repository.discussions.pageInfo.hasNextPage')
        CURSOR=$(echo "$RESULT" | jq -r '.data.repository.discussions.pageInfo.endCursor')
        
        # Safety check - break after 10 pages (1000 discussions)
        PAGE_COUNT=$((PAGE_COUNT + 1))
        if [ $PAGE_COUNT -ge 10 ]; then
          echo "Reached pagination limit (10 pages)"
          break
        fi
      done
      
      # Output summary for logging
      DISCUSSION_COUNT=$(jq 'length' "$DISCUSSIONS_FILE")
      echo "Found $DISCUSSION_COUNT discussions to close"
      
      # Output the filtered data for the agent
      jq -c '.[] | {number, title, createdAt}' "$DISCUSSIONS_FILE" > /tmp/gh-aw/filtered-discussions.jsonl
    env:
      GH_TOKEN: ${{ github.token }}
---

# Close Old Discussions Created by GitHub Actions Bot

This workflow automatically closes discussions that were created by the `github-actions[bot]` user and are older than 1 week.

**Pre-filtered Discussion Data**: The workflow has already downloaded and filtered discussions using GitHub GraphQL API. The filtered data is available in `/tmp/gh-aw/filtered-discussions.jsonl` with only discussions that:
- Were created by `github-actions[bot]`
- Are older than 7 days (1 week)

## Task

Read the pre-filtered discussion data from `/tmp/gh-aw/filtered-discussions.jsonl` and generate `close_discussion` outputs for each discussion.

**Important**: Do NOT call the GitHub API to list discussions - the data has already been fetched and filtered in the custom step above.

### Instructions

1. **Read the filtered data**: Load `/tmp/gh-aw/filtered-discussions.jsonl`
2. **Generate close outputs**: For each discussion in the file, create a `close_discussion` output:
   - **discussion_number**: The discussion number from the data
   - **body**: "This discussion was automatically closed because it was created by the automated workflow system more than 1 week ago."
   - **reason**: "OUTDATED"

Output format (JSONL):
```jsonl
{"type":"close_discussion","discussion_number":123,"body":"This discussion was automatically closed because it was created by the automated workflow system more than 1 week ago.","reason":"OUTDATED"}
```

### Example

If `/tmp/gh-aw/filtered-discussions.jsonl` contains:
```jsonl
{"number":1234,"title":"Test Discussion","createdAt":"2025-11-10T00:00:00Z"}
{"number":1235,"title":"Another Discussion","createdAt":"2025-11-11T00:00:00Z"}
```

You should output:
```jsonl
{"type":"close_discussion","discussion_number":1234,"body":"This discussion was automatically closed because it was created by the automated workflow system more than 1 week ago.","reason":"OUTDATED"}
{"type":"close_discussion","discussion_number":1235,"body":"This discussion was automatically closed because it was created by the automated workflow system more than 1 week ago.","reason":"OUTDATED"}
```

## Notes

- Maximum closures: Up to 100 discussions per run (configured via safe-outputs max)
- Data is pre-filtered: No need to check author or age - all discussions in the file match criteria
- If the file is empty or doesn't exist: Report that no discussions need to be closed

Begin the task now. Read the filtered data and generate close_discussion outputs.
