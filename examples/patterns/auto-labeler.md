---
# Pattern: Auto-Labeler
# Complexity: Beginner
# Use Case: Automatically label issues and PRs based on content, file changes, or keywords
name: Auto Labeler
description: Automatically applies labels to issues and pull requests based on configurable rules
on:
  issues:
    types: [opened, edited]
  pull_request:
    types: [opened, edited, synchronize]
permissions:
  contents: read
  issues: write
  pull-requests: write
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [issues, pull_requests]
safe-outputs:
  add-labels:
    # TODO: Customize max labels per run if needed
    max: 5
strict: true
---

# Auto-Labeler

Automatically apply relevant labels to issues and pull requests based on their content, file changes, and metadata.

## Your Task

Analyze the triggering event and apply appropriate labels based on the following rules:

### For Pull Requests

1. **File-based labeling** - Apply labels based on which files were changed:
   # TODO: Customize these paths and labels for your repository
   - `documentation` - if any files in `docs/` or `*.md` files changed
   - `backend` - if files in `src/`, `pkg/`, or `internal/` changed
   - `frontend` - if files in `ui/`, `components/`, or `*.tsx` changed
   - `tests` - if files in `test/`, `tests/`, or `*_test.*` changed
   - `ci-cd` - if files in `.github/workflows/` changed
   - `dependencies` - if `package.json`, `go.mod`, `requirements.txt`, etc. changed

2. **Size-based labeling** - Apply size labels based on changes:
   # TODO: Adjust these thresholds for your needs
   - `size/XS` - fewer than 10 lines changed
   - `size/S` - 10-99 lines changed
   - `size/M` - 100-499 lines changed
   - `size/L` - 500-999 lines changed
   - `size/XL` - 1000+ lines changed

3. **Content-based labeling** - Check title and description for keywords:
   # TODO: Add keywords relevant to your project
   - `bug` - title contains "fix", "bug", "error", "issue"
   - `feature` - title contains "add", "new", "feature", "implement"
   - `enhancement` - title contains "improve", "enhance", "update", "refactor"
   - `breaking-change` - title contains "breaking" or description mentions breaking changes

### For Issues

1. **Type detection** - Classify issue type based on content:
   # TODO: Customize these keywords for your issue types
   - `bug` - description mentions error, crash, broken, or "doesn't work"
   - `feature-request` - title starts with "Feature:" or contains "would like", "please add"
   - `question` - title ends with "?" or contains "how to", "how do I"
   - `documentation` - mentions docs, readme, or documentation improvements

2. **Priority detection** - Identify urgent issues:
   # TODO: Adjust priority criteria for your workflow
   - `priority-high` - title contains "urgent", "critical", "blocking", or "production"
   - `priority-medium` - title contains "important" or "soon"
   - `priority-low` - title contains "nice to have", "low priority", "enhancement"

3. **Component detection** - Label by affected component:
   # TODO: Add your repository's components
   - `area/api` - mentions API, endpoint, or REST
   - `area/ui` - mentions UI, interface, or frontend
   - `area/auth` - mentions authentication, login, or permissions
   - `area/performance` - mentions slow, performance, or optimization

## Implementation Steps

1. **Get event details**:
   - For PRs: Use `pull_request_read` with method `get` and `get_files` to analyze changes
   - For issues: Get issue details with `issue_read` with method `get`

2. **Analyze content**:
   - Check title and description for keywords (case-insensitive)
   - For PRs, examine the list of changed files
   - For PRs, count total lines changed (additions + deletions)

3. **Apply labels**:
   - Collect all matching labels based on rules above
   - Use the `add_labels` safe-output to apply them
   - Don't apply duplicate labels if they already exist

4. **Be specific**:
   - Only apply labels that clearly match the criteria
   - When in doubt, apply fewer labels rather than more
   - Prefer specific labels over generic ones

## Example Output

For a PR that changes `docs/README.md` and `docs/api.md` (45 lines):
```markdown
Applied labels:
- documentation (files in docs/)
- size/S (45 lines changed)
```

For an issue titled "Bug: Login page crashes on mobile":
```markdown
Applied labels:
- bug (contains "bug" and "crashes")
- area/auth (mentions "login")
- area/ui (mentions "page")
- priority-high (production issue affecting users)
```

## Customization Guide

### Adding New Labels

To add custom labeling rules:

1. **Define the pattern** - What content/files trigger this label?
2. **Add the detection logic** - Check for specific keywords or patterns
3. **Test thoroughly** - Ensure the rule doesn't create false positives
4. **Document it** - Add comments explaining when the label applies

Example: Add a `security` label for security-related issues:
```markdown
- `security` - mentions "vulnerability", "CVE", "exploit", or "security"
```

### Excluding Patterns

To prevent labeling in certain cases:

```markdown
- Skip labeling if PR is a draft
- Skip labeling if issue is already labeled
- Skip labeling if title contains "[no-label]"
```

## Related Examples

- **Simple labeling**: `examples/label-trigger-simple.md` - Basic label trigger example
- **PR labeling**: `examples/label-trigger-pull-request.md` - Advanced PR label handling
- **Discussion labeling**: `examples/label-trigger-discussion.md` - Label-based discussion triggers

## Tips

- **Test incrementally**: Start with a few rules, test them, then add more
- **Use clear label names**: Prefer `area/api` over `api` for clarity
- **Avoid over-labeling**: Too many labels reduce their usefulness
- **Document your labels**: Create a `LABELS.md` file explaining your label taxonomy
- **Monitor accuracy**: Review applied labels regularly and adjust rules

## Security Considerations

- This workflow only reads issue/PR content and applies labels
- It uses `strict: true` for enhanced security
- Network access is not required
- All operations go through safe-outputs with validation

---

**Pattern Info**:
- Complexity: Beginner
- Trigger: Issues and PRs (opened, edited)
- Safe Outputs: add_labels
- Tools: GitHub (issues, pull_requests)
