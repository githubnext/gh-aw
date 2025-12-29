---
# Pattern: Issue Responder
# Complexity: Beginner
# Use Case: Automatically respond to new issues with helpful information and triage
name: Issue Responder
description: Automatically responds to new issues with helpful information, questions, or actions
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [issues]
safe-outputs:
  create-comment:
    # TODO: Customize max comments if needed
    max: 1
  add-labels:
    max: 3
  update-issue:
    max: 1
strict: true
---

# Issue Responder

Automatically respond to newly created issues with helpful information, triage questions, or automated actions.

## Your Task

When a new issue is opened, analyze it and provide appropriate responses:

### 1. Welcome New Contributors

Check if the issue author is a first-time contributor:

```markdown
# TODO: Customize welcome message for your project
Thanks for opening your first issue in this repository! 🎉

A maintainer will review your issue soon. In the meantime:
- Make sure you've provided all the requested information
- Check if there are similar existing issues
- Review our [contributing guidelines](../CONTRIBUTING.md)
```

### 2. Validate Issue Template

Check if the issue follows the expected template format:

# TODO: Customize these checks for your issue templates
- **Bug reports** should include:
  - Steps to reproduce
  - Expected vs actual behavior
  - Version information
  - Environment details

- **Feature requests** should include:
  - Clear description of the feature
  - Use case explanation
  - Alternatives considered

If required information is missing, politely request it:

```markdown
Thank you for opening this issue! To help us understand and address it quickly, could you please provide:

- [ ] Steps to reproduce the issue
- [ ] Expected behavior
- [ ] Actual behavior
- [ ] Version of the software
- [ ] Operating system and environment

This information will help us investigate the issue more efficiently.
```

### 3. Classify Issue Type

Determine the issue type and apply appropriate labels:

# TODO: Customize classification rules for your project
- **Bug reports**: Contains "error", "crash", "broken", "doesn't work", or "failed"
  - Label: `bug`
  - Action: Request reproduction steps if missing

- **Feature requests**: Title starts with "Feature:" or contains "add", "support", "would like"
  - Label: `enhancement` or `feature-request`
  - Action: Ask about use case and priority

- **Questions**: Title ends with "?" or contains "how to", "how do I"
  - Label: `question`
  - Action: Point to documentation or discussions

- **Documentation**: Mentions docs, README, or documentation improvements
  - Label: `documentation`
  - Action: Acknowledge and ask for specific changes needed

### 4. Check for Duplicates

Search for similar existing issues:

```bash
# Use GitHub search to find similar issues
gh search issues --repo ${{ github.repository }} "keywords from title"
```

If duplicates found, comment:

```markdown
This issue appears similar to #[NUMBER]. Could you check if that issue addresses your concern?

If this is different, please explain how it differs so we can better understand your use case.
```

### 5. Auto-assign Based on Keywords

# TODO: Customize assignment rules for your team
Assign to specific team members based on issue content:

- Issues about **API** → assign to @api-team
- Issues about **UI/UX** → assign to @frontend-team
- Issues about **performance** → assign to @performance-team
- Issues about **security** → assign to @security-team

Use the `update_issue` safe-output to set assignees (if you have write permissions).

### 6. Provide Context and Links

Include helpful links based on issue type:

```markdown
# TODO: Add links relevant to your project
Here are some resources that might help:

- [Documentation](https://docs.example.com)
- [Troubleshooting Guide](https://docs.example.com/troubleshooting)
- [Community Forum](https://community.example.com)
- [Similar Issues](https://github.com/owner/repo/issues?q=is%3Aissue+label%3Abug)
```

## Implementation Steps

1. **Get issue details**:
   ```markdown
   Use `issue_read` with method `get` to retrieve:
   - Issue number: ${{ github.event.issue.number }}
   - Issue title: ${{ github.event.issue.title }}
   - Issue body: ${{ github.event.issue.body }}
   - Author: ${{ github.event.issue.user.login }}
   ```

2. **Check contributor status**:
   ```markdown
   Determine if this is the author's first issue in the repository
   Use GitHub API or check issue author's contribution history
   ```

3. **Analyze content**:
   - Parse the issue body for expected sections
   - Check for required information based on issue type
   - Extract keywords for classification

4. **Take appropriate actions**:
   - Comment with welcome message (for new contributors)
   - Comment with missing information request (if incomplete)
   - Apply labels based on classification
   - Search for duplicates and link if found
   - Assign to appropriate team member (if configured)

## Example Scenarios

### Scenario 1: Well-Formed Bug Report

**Issue**: "Bug: API returns 500 error when creating users"  
**Content**: Includes steps to reproduce, expected/actual behavior, version

**Response**:
```markdown
Thank you for the detailed bug report! 🐛

I've labeled this as a `bug` and assigned it to our API team for investigation.
They'll review it soon and get back to you.

In the meantime, if you discover any additional information, please add it to this issue.
```

**Actions**:
- Label: `bug`, `area/api`
- Assign: @api-team
- No additional information needed

### Scenario 2: Incomplete Bug Report

**Issue**: "It doesn't work"  
**Content**: No steps to reproduce, no version information

**Response**:
```markdown
Thank you for opening this issue! To help us understand and fix the problem, could you please provide:

**Steps to Reproduce:**
1. Step one
2. Step two
3. ...

**Expected Behavior:**
What should happen?

**Actual Behavior:**
What actually happens?

**Version Information:**
- Software version:
- Operating system:
- Browser (if applicable):

This information will help us investigate the issue more effectively. Thanks! 🙏
```

**Actions**:
- Label: `needs-more-info`
- No assignment yet

### Scenario 3: Feature Request

**Issue**: "Feature: Add dark mode support"  
**Content**: Clear description of desired feature

**Response**:
```markdown
Thanks for this feature request! 🌟

To help us prioritize this, could you share:

1. **Use Case**: How would this feature benefit you or your team?
2. **Priority**: Is this blocking your workflow or a nice-to-have improvement?
3. **Alternatives**: Are there any workarounds you're currently using?

We'll review this and add it to our roadmap discussion!
```

**Actions**:
- Label: `enhancement`, `feature-request`
- Link to roadmap discussion (if exists)

## Customization Guide

### Modify Response Templates

Edit the response messages to match your project's tone and requirements:

```markdown
# TODO: Update these templates
WELCOME_MESSAGE = "Thanks for your contribution! ..."
MISSING_INFO_MESSAGE = "Please provide more details..."
DUPLICATE_FOUND_MESSAGE = "This seems similar to #..."
```

### Add Custom Classifications

Add new issue types specific to your project:

```markdown
# TODO: Add your custom issue types
- **Performance issue**: mentions "slow", "performance", "lag"
  - Label: `performance`
  - Action: Request profiling data

- **Security issue**: mentions "security", "vulnerability", "CVE"
  - Label: `security`
  - Action: Alert security team immediately
  - Mark as private/confidential if needed
```

### Configure Team Assignment

Set up auto-assignment rules:

```markdown
# TODO: Configure your team structure
TEAM_ASSIGNMENTS = {
  "api": ["@user1", "@user2"],
  "frontend": ["@user3", "@user4"],
  "docs": ["@user5"],
}
```

## Related Examples

- **Label triggers**: `examples/label-trigger-simple.md` - React to labeled issues
- **Auto-labeler**: `examples/patterns/auto-labeler.md` - Automated labeling
- **Production examples**:
  - `.github/workflows/issue-classifier.md` - Advanced issue classification
  - `.github/workflows/issue-triage-agent.md` - Comprehensive triage workflow

## Tips

- **Be welcoming**: First impressions matter for new contributors
- **Be specific**: Tell users exactly what information you need
- **Be helpful**: Provide links to documentation and resources
- **Be timely**: Automated responses show active maintenance
- **Be respectful**: Always thank users for taking time to report issues

## Advanced Features

### Smart Duplicate Detection

Use semantic similarity to find duplicates:

```python
# Use embeddings or TF-IDF to compare issue titles/descriptions
# Flag potential duplicates above similarity threshold
```

### Priority Inference

Automatically set priority based on multiple signals:

```markdown
High priority if:
- Contains "production", "critical", "urgent"
- Many users affected (check for +1 reactions on similar issues)
- Security-related
```

### Auto-close Invalid Issues

For issues that clearly violate guidelines:

```markdown
# Only if issue is spam, off-topic, or severely incomplete
This issue doesn't follow our guidelines and has been closed automatically.
Please reopen with the required information if this was a mistake.
```

**⚠️ Use with caution**: Auto-closing can frustrate legitimate users.

## Security Considerations

- This workflow only reads issue content and creates comments
- Uses `strict: true` for enhanced security
- No network access required
- All operations validated through safe-outputs

---

**Pattern Info**:
- Complexity: Beginner
- Trigger: Issues opened
- Safe Outputs: create_comment, add_labels, update_issue
- Tools: GitHub (issues)
