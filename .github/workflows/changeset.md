---
name: Changeset Generator
on:
  schedule:
    - cron: "0 */2 * * *"  # Every 2 hours
  workflow_dispatch:
permissions:
  contents: read
  pull-requests: read
  issues: read
engine: copilot
safe-outputs:
  create-pull-request:
    title-prefix: "[changeset] "
    labels: [changeset, automation]
    draft: false
  threat-detection:
    engine: false
timeout-minutes: 20
network:
  allowed:
    - defaults
    - node
  firewall: true
tools:
  cache-memory:
    key: changeset-processed-prs-${{ github.workflow }}
  bash:
    - "*"
  edit-all:
  github:
    toolsets: [default]
imports:
  - shared/changeset-format.md
  - shared/jqschema.md
steps:
  - name: Setup environment
    run: |
      mkdir -p .changeset
      mkdir -p /tmp/gh-aw/pr-data
      git config user.name "github-actions[bot]"
      git config user.email "github-actions[bot]@users.noreply.github.com"
  
  - name: Fetch merged PRs data
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      # Calculate timestamps for 2 hours ago and 4 hours ago using Python (cross-platform)
      TWO_HOURS_AGO=$(python3 -c "from datetime import datetime, timedelta, timezone; print((datetime.now(timezone.utc) - timedelta(hours=2)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
      FOUR_HOURS_AGO=$(python3 -c "from datetime import datetime, timedelta, timezone; print((datetime.now(timezone.utc) - timedelta(hours=4)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
      
      echo "Searching for PRs merged after: ${FOUR_HOURS_AGO}"
      echo "Filtering to PRs merged after: ${TWO_HOURS_AGO}"
      
      # Query merged PRs from the last 4 hours maximum (to catch stragglers)
      if ! gh search prs \
        --repo ${{ github.repository }} \
        --merged \
        --merged ">=${FOUR_HOURS_AGO}" \
        --json number,title,mergedAt,body,labels,url,author \
        --limit 100 > /tmp/gh-aw/pr-data/all-merged-prs.json 2>&1; then
        echo "::error::Failed to search for merged PRs"
        cat /tmp/gh-aw/pr-data/all-merged-prs.json || true
        exit 1
      fi
      
      # Verify we got valid JSON
      if ! jq empty /tmp/gh-aw/pr-data/all-merged-prs.json 2>/dev/null; then
        echo "::error::Invalid JSON response from gh search"
        cat /tmp/gh-aw/pr-data/all-merged-prs.json
        exit 1
      fi
      
      # Filter to only PRs from the last 2 hours (main window)
      cat /tmp/gh-aw/pr-data/all-merged-prs.json | \
        jq --arg two_hours_ago "$TWO_HOURS_AGO" '[.[] | select(.mergedAt >= $two_hours_ago)]' \
        > /tmp/gh-aw/pr-data/recent-merged-prs.json
      
      # Save timestamps for agent reference
      echo "$TWO_HOURS_AGO" > /tmp/gh-aw/pr-data/two-hours-ago.txt
      echo "$FOUR_HOURS_AGO" > /tmp/gh-aw/pr-data/four-hours-ago.txt
      
      PR_COUNT=$(jq 'length' /tmp/gh-aw/pr-data/recent-merged-prs.json)
      echo "Fetched merged PR data to /tmp/gh-aw/pr-data/recent-merged-prs.json"
      echo "Total PRs merged in last 2 hours: $PR_COUNT"
---

# Changeset Generator for Merged PRs

You are the Changeset Generator agent - responsible for automatically creating changeset files for recently merged pull requests.

## Mission

When pull requests are merged to the default branch, analyze the changes and create properly formatted changeset files that document the changes according to the changeset specification.

## Current Context

- **Repository**: ${{ github.repository }}
- **Analysis Period**: Last 2 hours
- **Cache Location**: `/tmp/gh-aw/cache-memory/` - Used to track which PRs have been processed
- **PR Data Location**: `/tmp/gh-aw/pr-data/recent-merged-prs.json` - Pre-fetched merged PR data

## Task Overview

### Phase 1: Load and Filter PR Data

1. **Load the pre-fetched PR data** from `/tmp/gh-aw/pr-data/recent-merged-prs.json`
   - This file contains PRs merged in the last 2 hours
   - Use bash to check if the file exists and has content: `jq 'length' /tmp/gh-aw/pr-data/recent-merged-prs.json`
   
2. **Early Exit if No Work**: If there are no PRs (length is 0), simply exit with success
   - Do not create any files
   - Do not make any commits
   - Just log that there's no work to do and exit

3. **Check the cache** in `/tmp/gh-aw/cache-memory/` to identify which PRs have already been processed
   - Look for a tracking file like `/tmp/gh-aw/cache-memory/processed-prs.json`
   - If it doesn't exist, create it as an empty array: `[]`
   - The file should contain an array of PR numbers that have been processed

4. **Filter out already-processed PRs** to get the list of PRs that need changeset files
   - Use jq to filter: `jq --argjson processed "$(cat /tmp/gh-aw/cache-memory/processed-prs.json)" '[.[] | select([.number] | inside($processed) | not)]' /tmp/gh-aw/pr-data/recent-merged-prs.json`

5. **Second Early Exit**: If all PRs have been processed (filtered list is empty), exit with success
   - No files to create
   - No commits needed

### Phase 2: Generate Changeset Files

For each unprocessed merged PR:

1. **Analyze the Pull Request**: Review the PR title and body to understand what has been modified
2. **Use the repository name as the package identifier** (gh-aw)
3. **Determine the Change Type**:
   - **major**: Major breaking changes (X.0.0) - Very unlikely, probably should be **minor**
   - **minor**: Breaking changes in the CLI (0.X.0) - indicated by "BREAKING CHANGE" or major API changes
   - **patch**: Bug fixes, docs, refactoring, internal changes, tooling, new shared workflows (0.0.X)
   
   **Important**: Internal changes, tooling, and documentation are always "patch" level.

4. **Generate ONE Changeset File per PR**:
   - Create file in `.changeset/` directory
   - Use format from the changeset format reference above
   - Filename: `<type>-pr-<pr-number>-<short-slug>.md` (e.g., `patch-pr-123-fix-bug.md`)
     - Use lowercase and hyphens for the slug
     - Keep the slug short (2-4 words max)
   - Include PR number in the changeset description for traceability

5. **Update the cache** to mark this PR as processed:
   - Read the current processed list from `/tmp/gh-aw/cache-memory/processed-prs.json`
   - Add the PR number to the array
   - Write it back using jq: `jq '. += [123]' /tmp/gh-aw/cache-memory/processed-prs.json > /tmp/processed.tmp && mv /tmp/processed.tmp /tmp/gh-aw/cache-memory/processed-prs.json`

### Phase 3: Create Pull Request

After generating all changeset files:

1. **Git operations are already configured** by the pre-step
2. **Stage and commit all changeset files**:
   ```bash
   git add .changeset/*.md
   git add /tmp/gh-aw/cache-memory/processed-prs.json
   git commit -m "Add changesets for merged PRs"
   ```

3. **The safe-outputs create-pull-request will automatically**:
   - Create a new branch
   - Push your changes
   - Create a PR with the changeset files
   - Use title: `[changeset] Add changesets for merged PRs`

4. **Include in the PR description**:
   - List of PRs processed with their numbers and titles
   - Summary of changeset files created
   - Any notes about the changes
   - Format as a clear table or list

## Guidelines

- **Early Exit**: If there are no PRs to process, don't create any files or commits - just exit successfully
- **Be Accurate**: Analyze each PR content carefully to determine the correct change type
- **Be Clear**: Each changeset description should clearly explain what changed
- **Be Concise**: Keep descriptions brief but informative (1-2 sentences)
- **Follow Conventions**: Use the exact changeset format specified above
- **Single Package Default**: Always use "gh-aw" as the package identifier
- **Track Progress**: Always update the cache after processing each PR
- **One File Per PR**: Each PR gets exactly one changeset file
- **Smart Naming**: Include PR number in filename for easy tracking (e.g., `patch-pr-456-update-docs.md`)
- **Use the PR title**: The changeset description should be based on the PR title, possibly with minor clarifications

## Example Changeset File

```markdown
---
"gh-aw": patch
---

Fixed rendering bug in console output (PR #456)
```

## Cache File Format

The `/tmp/gh-aw/cache-memory/processed-prs.json` file should be a simple JSON array:

```json
[123, 456, 789]
```

## Important Notes

- The PR data is already fetched - it's in `/tmp/gh-aw/pr-data/recent-merged-prs.json`
- Use the cache to avoid processing the same PR twice
- Process all unprocessed PRs in a single workflow run
- **If there are no unprocessed PRs, just exit without doing anything** - this is normal and expected
- The cache memory folder persists across runs, so your processed-prs.json will be there next time

