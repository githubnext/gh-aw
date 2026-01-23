---
on:
  schedule:
    - cron: '0 */4 * * *'  # Every 4 hours
  workflow_dispatch:
engine:
  id: copilot
  model: gpt-5-mini
imports:
  - shared/mcp/chroma.md
permissions:
  contents: read
  issues: read
tools:
  github:
    mode: local
    read-only: true
    toolsets: [issues]
---

# Chroma Issue Indexer

This workflow indexes issues from the repository into a Chroma vector database for semantic search and duplicate detection.

## Task

Index the 100 most recent issues from the repository into the Chroma vector database:

1. **Fetch Issues**:
   - Use the GitHub MCP server tools to list the most recent 100 issues
   - Include both open and closed issues
   - Get issue number, title, body, state, created date, and author

2. **Create/Update Chroma Collection**:
   - Create a collection named "issues" if it doesn't exist (use `chroma_create_collection`)
   - Use an appropriate embedding function for semantic search

3. **Index Issues**:
   - For each issue, add it to the Chroma collection (use `chroma_add_documents`)
   - Use ID format: `issue-{issue_number}`
   - Document content should be: `{title}\n\n{body}` (title and body combined)
   - Include metadata:
     - `number`: Issue number
     - `title`: Issue title
     - `state`: Issue state (OPEN or CLOSED)
     - `author`: Issue author username
     - `created_at`: Issue creation date
     - `url`: Issue URL

4. **Report Progress**:
   - Log how many issues were indexed
   - Note any issues that couldn't be indexed (e.g., empty body)
   - Report the total number of issues in the collection

## Important Notes

- Process issues in batches if needed to avoid rate limits
- Skip issues that have already been indexed (check if ID exists)
- For issues with empty bodies, use just the title as content
- The collection persists in `/tmp/gh-aw/cache-memory-chroma/` across runs
- This helps other workflows search for similar issues using semantic search

<!--
# Chroma Issue Indexer Workflow

An automated workflow that indexes repository issues into a Chroma vector database for semantic search capabilities.

## Features

- **Scheduled Execution**: Runs every 4 hours to keep the index up-to-date
- **Batch Indexing**: Indexes the 100 most recent issues per run
- **Persistent Storage**: Uses cache-memory with Chroma for persistent vector database
- **Semantic Search Ready**: Enables other workflows to search for similar issues
- **Duplicate Detection**: Helps identify duplicate or related issues

## How It Works

1. **Schedule**: Runs automatically every 4 hours via cron schedule
2. **Fetch Issues**: Uses GitHub MCP server to get the latest 100 issues
3. **Index**: Adds each issue to the Chroma "issues" collection with:
   - Vector embeddings for semantic search
   - Metadata (number, title, state, author, dates)
   - Combined title and body as searchable content
4. **Persist**: Stores the vector database in `/tmp/gh-aw/cache-memory-chroma/`
5. **Share**: Makes the indexed issues available for other workflows

## Configuration

```yaml
on:
  schedule:
    - cron: '0 */4 * * *'  # Every 4 hours
  workflow_dispatch:
engine:
  id: copilot
  model: gpt-5-mini
imports:
  - shared/mcp/chroma.md
permissions:
  contents: read
  issues: read
```

## Usage

The indexed issues can be queried by other workflows that import `shared/mcp/chroma.md`:

```yaml
# Search for similar issues
chroma_query_documents(
  collection_name="issues",
  query="My issue description",
  limit=5
)
```

## Benefits

- **Duplicate Detection**: Find similar issues before creating new ones
- **Issue Triage**: Quickly find related issues for context
- **Search Enhancement**: Semantic search beyond keyword matching
- **Historical Context**: Maintain searchable issue history

## Maintenance

- Indexes automatically every 4 hours
- Can be manually triggered via workflow_dispatch
- Stores data persistently across runs
- No manual cleanup needed - cache managed by GitHub Actions
-->
