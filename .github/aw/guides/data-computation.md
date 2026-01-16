# Data Computation Best Practices

This guide covers patterns for processing, transforming, and computing data within agentic workflows.

## Overview

Agentic workflows often need to process data from various sources (GitHub API, external APIs, files, etc.) and perform computations before making decisions or generating outputs. This guide provides patterns for effective data computation.

## Core Principles

### 1. Use Bash for Command-Line Tools

Leverage command-line tools available in the environment:

```yaml
tools:
  bash: ["jq", "git", "curl"]  # Or omit for default all-commands access
```

**Common tools**:
- `jq` - JSON processing and transformation
- `git` - Repository operations and history analysis
- `curl` - HTTP requests (with network permissions)
- `grep`, `sed`, `awk` - Text processing
- `python`, `node`, `go` - Scripting for complex logic

### 2. Leverage Language Servers for Code Analysis

Use Serena language servers for understanding code structure:

```yaml
tools:
  serena: ["typescript", "go"]
```

**Capabilities**:
- Find definitions and references
- Navigate code structure
- Analyze dependencies
- Understand type information

### 3. GitHub API for Repository Data

Use GitHub toolsets for querying repository information:

```yaml
tools:
  github:
    toolsets: [default]
```

**Available data**:
- Issues and pull requests
- Commits and branches
- Repository metadata
- Code search results
- Workflow runs

## Common Data Processing Patterns

### JSON Processing with jq

**Use case**: Parse and transform JSON from APIs or GitHub data

```bash
# Extract specific fields
echo "$json_data" | jq '.items[] | {title, number, state}'

# Filter and count
echo "$json_data" | jq '[.[] | select(.state == "open")] | length'

# Transform structure
echo "$json_data" | jq 'map({id: .number, text: .title})'
```

**In workflow prompt**:
```markdown
Use `jq` to:
1. Parse the GitHub API response
2. Extract issue titles and numbers
3. Filter for issues labeled "bug"
4. Count the results
```

### Git Repository Analysis

**Use case**: Analyze commit history, contributors, file changes

```bash
# Get recent commits
git log --oneline --since="1 week ago"

# Find files changed in a commit
git diff-tree --no-commit-id --name-only -r <commit-sha>

# Count commits by author
git shortlog -sn --since="1 month ago"

# Get file history
git log --follow --oneline -- path/to/file
```

**In workflow prompt**:
```markdown
Analyze the repository:
1. Get the last 20 commits using `git log`
2. Identify the most frequently modified files
3. Determine the primary contributors
```

### Text Processing

**Use case**: Extract patterns, transform text, count occurrences

```bash
# Find TODO comments
grep -r "TODO:" --include="*.go" .

# Count lines of code
find . -name "*.py" -exec wc -l {} + | awk '{sum+=$1} END {print sum}'

# Extract URLs
grep -oP 'https?://[^\s]+' file.txt
```

### Web Data Fetching

**Use case**: Fetch and process external data

```yaml
tools:
  web-fetch: {}
network:
  allowed:
    - "api.example.com"
```

**In workflow prompt**:
```markdown
1. Fetch the API endpoint using web-fetch
2. Parse the JSON response
3. Extract relevant data fields
4. Compare with repository data
```

## Computation Patterns

### Aggregation and Counting

**Pattern**: Collect data and compute statistics

```markdown
Your task:
1. List all open issues using the GitHub tool
2. Group by label using jq
3. Count issues per label
4. Sort by count descending
5. Report the top 5 labels
```

### Filtering and Selection

**Pattern**: Filter data based on criteria

```markdown
Your task:
1. Get all pull requests from the last week
2. Filter for PRs with merge conflicts
3. Filter for PRs without reviews
4. Select the top 3 by age
5. Report PR numbers and titles
```

### Transformation and Mapping

**Pattern**: Convert data from one format to another

```markdown
Your task:
1. Fetch issue data from GitHub
2. Transform to CSV format with jq
3. Include columns: number, title, author, created_at
4. Sort by creation date
5. Save to a file
```

### Correlation and Analysis

**Pattern**: Find relationships between data points

```markdown
Your task:
1. Get commit history for the last month
2. Get issue closures for the same period
3. Correlate commits mentioning issue numbers
4. Calculate time-to-close for each issue
5. Identify patterns in resolution time
```

## Data Caching

For workflows with repeated computations or large context reuse:

```yaml
cache-memory:
  enabled: true
```

**Benefits**:
- Faster execution for repeated queries
- Reduced API calls
- Maintain context across invocations

**Use cases**:
- Daily report workflows that query the same data
- Analysis workflows that build on previous results
- Workflows that maintain state across runs

## Performance Considerations

### 1. Minimize API Calls

**Inefficient**:
```markdown
For each issue:
  1. Fetch issue details
  2. Fetch comments
  3. Process data
```

**Efficient**:
```markdown
1. Fetch all issues in one query
2. Batch process the results
3. Use jq for filtering and transformation
```

### 2. Use Pagination for Large Datasets

```markdown
1. Fetch the first page of results
2. Process incrementally
3. Stop when enough data is collected
4. Don't fetch all pages unnecessarily
```

### 3. Prefer Structured Queries

```markdown
Use GitHub search API with filters:
  - `is:issue state:open label:bug`
  - `is:pr created:>2024-01-01 sort:updated`
  
Instead of:
  - Fetching all issues and filtering locally
```

## Error Handling

### Validate Data Before Processing

```markdown
1. Fetch the data
2. Verify it's valid JSON using jq
3. Check for required fields
4. Handle missing or null values
5. Proceed with computation
```

### Graceful Degradation

```markdown
If API call fails:
  1. Log the error
  2. Use cached data if available
  3. Report partial results
  4. Indicate data freshness
```

## Example: Issue Statistics Workflow

```markdown
You are an AI agent that computes weekly issue statistics.

## Your Task

1. **Fetch Data**
   - Get all issues opened in the last 7 days
   - Get all issues closed in the last 7 days
   - Get issue comments from the last 7 days

2. **Compute Statistics**
   - Count issues opened vs closed
   - Calculate average time-to-close
   - Identify most active labels
   - Find most commented issues

3. **Transform Results**
   - Create a summary table with jq
   - Format as markdown
   - Include trends (↑↓) compared to previous week

4. **Generate Report**
   - Create a weekly report issue
   - Include all statistics
   - Add charts if data supports it
```

## Summary

- Use command-line tools (jq, git, grep) for data processing
- Leverage language servers for code analysis
- Use GitHub API efficiently with filters and pagination
- Cache results for repeated computations
- Validate data before processing
- Handle errors gracefully
- Minimize API calls by batching operations
- Transform data at the source when possible
