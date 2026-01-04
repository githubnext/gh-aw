# Auto-Close Parent Issues Workflow

## Overview

This GitHub Actions workflow automatically closes parent issues when all of their sub-issues are closed. It recursively walks up the issue hierarchy tree, closing each parent issue when all its children are completed.

## How It Works

1. **Trigger**: Activates whenever any issue in the repository is closed
2. **Check Parents**: Identifies any parent issues that track the closed issue
3. **Verify Sub-Issues**: For each parent, checks if ALL sub-issues are now closed
4. **Close Parent**: If all sub-issues are closed, automatically closes the parent issue
5. **Recurse**: Walks up the tree to check grandparent issues and continues closing as needed
6. **Comment**: Adds an explanatory comment to each auto-closed issue

## Features

### Comprehensive Logging

The workflow provides detailed logging at each step:
- Issue numbers and titles
- Current state of each issue
- Sub-issue status counts (open vs closed)
- Hierarchical depth indicators
- Success and error messages
- Pagination progress for large issue sets

### Recursive Tree Walking

The workflow walks up the entire issue hierarchy:
- Closes parent issues when all sub-issues are complete
- Automatically checks grandparent issues
- Continues recursively until reaching the top of the tree
- Respects already-closed issues (won't reprocess)

### Scalability

Designed to handle large issue hierarchies:
- **Pagination support**: Fetches all sub-issues in batches of 100
- **No limit on sub-issues**: Can process parents with 1000+ sub-issues
- **Safety limits**: Maximum 5,000 sub-issues per parent to prevent timeouts
- **Efficient processing**: Only fetches necessary data for each level

### Safety Features

- Only closes issues when **all** sub-issues are closed
- Skips issues that are already closed
- Provides detailed audit trail in logs
- Adds explanatory comments to closed issues
- Uses GraphQL API for reliable relationship data

## GraphQL API Usage

The workflow uses GitHub's GraphQL API to query issue relationships:

- `trackedInIssues`: Issues that track this issue (parents)
- `trackedIssues`: Issues tracked by this issue (sub-issues)

## Example Scenario

```text
Parent Issue #100 (Open)
├── Sub-Issue #101 (Closed)
├── Sub-Issue #102 (Closed)
└── Sub-Issue #103 (Open)  ← Just closed!
```

**Result**: Parent Issue #100 automatically closes with a comment explaining why.

### Nested Example

```text
Grandparent Issue #200 (Open)
└── Parent Issue #100 (Open)
    ├── Sub-Issue #101 (Closed)
    ├── Sub-Issue #102 (Closed)
    └── Sub-Issue #103 (Open)  ← Just closed!
```

**Result**: 
1. Parent Issue #100 closes (all sub-issues complete)
2. Workflow checks Grandparent Issue #200
3. If all sub-issues of #200 are now closed, it closes too
4. Process continues up the tree

## Permissions

The workflow requires:
- `issues: write` - To close issues and add comments

## Configuration

No configuration needed! The workflow works automatically once added to your repository.

## Logging Examples

```text
=================================================
Auto-Close Parent Issues Workflow
=================================================
Triggered by: Issue #103 was closed
Repository: owner/repo

📊 Fetching issue #103 with relationship data...
✓ Fetched issue #103: "Implement feature X"
  State: CLOSED (COMPLETED)
  Parent issues: 1
  Sub-issues: 0

🔍 Found 1 parent issue(s) to check:
  - #100: "Epic: Feature X" [OPEN]

==================================================
Processing Parent Issue (Depth: 0)
==================================================
Issue: #100 "Epic: Feature X"
Current State: OPEN

🔍 Checking sub-issues of #100 "Epic: Feature X"...
  Total sub-issues: 3
    - #101: "Task 1" [CLOSED]
    - #102: "Task 2" [CLOSED]
    - #103: "Task 3" [CLOSED]
  Summary: 3 closed, 0 open

✅ All sub-issues of #100 are closed!
🔒 Closing issue #100...
✓ Successfully closed issue #100
✓ Added closure comment to issue #100
```

## Troubleshooting

### Issue Not Closing

If a parent issue doesn't close automatically:
1. Check if the parent-child relationship is properly established (use GitHub's sub-issue feature)
2. Verify all sub-issues are actually closed (not just one)
3. Check the workflow run logs for detailed error messages

### Handling Large Numbers of Sub-Issues

The workflow uses **pagination** to handle parents with many sub-issues:
- Fetches sub-issues in batches of 100
- Automatically continues fetching until all sub-issues are retrieved
- Can handle up to **5,000 sub-issues** per parent (50 pages × 100 per page)
- Logs progress for each page fetched

**Parent Issue Limits:**
- Fetches up to 10 parent issues per issue (GitHub's typical limit)

## Technical Details

- **Runtime**: ubuntu-latest
- **Action**: github-script@v8.0.0
- **Language**: JavaScript
- **API**: GitHub GraphQL API v4

## File Location

`.github/workflows/auto-close-parent-issues.yml`
