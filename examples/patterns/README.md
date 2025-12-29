# Workflow Patterns Library

A curated collection of the 10 most common agentic workflow patterns, based on analysis of 126 production workflows. Use these templates as starting points for your own workflows.

## Quick Pattern Selection

**Choose a pattern based on your use case:**

| Pattern | Use When... | Complexity |
|---------|------------|------------|
| [Daily Report](#1-daily-report) | Need scheduled analysis and reporting | Intermediate |
| [PR Reviewer](#2-pr-reviewer) | Want automated PR analysis and feedback | Intermediate |
| [Auto-Labeler](#3-auto-labeler) | Need automatic issue/PR labeling | Beginner |
| [Issue Responder](#4-issue-responder) | Want automated issue triage and response | Beginner |
| [Discussion Bot](#5-discussion-bot) | Need to create/manage discussions | Beginner |
| [Code Scanner](#6-code-scanner) | Want automated code analysis | Advanced |
| [CI Monitor](#7-ci-monitor) | Need CI/CD monitoring and alerts | Intermediate |
| [Workflow Manager](#8-workflow-manager) | Want to generate/update workflows | Advanced |
| [Documentation Bot](#9-documentation-bot) | Need automated doc generation | Intermediate |
| [Integration Bot](#10-integration-bot) | Want to integrate with external tools | Intermediate |

## Pattern Categories

### Scheduled Patterns
Workflows that run on a schedule (daily, weekly, etc.)
- Daily Report
- Code Scanner
- Documentation Bot

### Event-Driven Patterns
Workflows triggered by GitHub events
- PR Reviewer
- Auto-Labeler
- Issue Responder
- CI Monitor

### Interactive Patterns
Workflows that create or manage GitHub content
- Discussion Bot
- Documentation Bot
- Integration Bot

### Management Patterns
Workflows that manage repository infrastructure
- Workflow Manager
- CI Monitor

## 1. Daily Report

**Complexity**: Intermediate  
**Frequency**: Daily/Scheduled  
**Safe Outputs**: create_discussion, upload_asset

Generate comprehensive reports analyzing repository activity, issues, PRs, or code metrics on a schedule.

**Common Use Cases:**
- Daily issues summary with trends and clustering
- Weekly PR merge statistics
- Code quality metrics dashboard
- Performance analysis reports

**Template**: [`daily-report.md`](./daily-report.md)  
**Related Examples**:
- Repository: `.github/workflows/daily-issues-report.md`
- Repository: `.github/workflows/copilot-pr-merged-report.md`

**Key Features:**
- Scheduled execution (daily, weekly, etc.)
- Data collection and analysis with Python
- Visualization generation
- Discussion creation with auto-cleanup

---

## 2. PR Reviewer

**Complexity**: Intermediate  
**Frequency**: On PR events  
**Safe Outputs**: add_comment

Automatically analyze pull requests and provide feedback on code changes, potential issues, or style compliance.

**Common Use Cases:**
- Code review assistance
- PR description validation
- Breaking change detection
- CI/CD monitoring and failure analysis

**Template**: [`pr-reviewer.md`](./pr-reviewer.md)  
**Related Examples**:
- Repository: `.github/workflows/dev-hawk.md`
- Repository: `.github/workflows/breaking-change-checker.md`

**Key Features:**
- Triggered on PR open/update
- Analyzes PR diff and files changed
- Comments on specific issues found
- Can integrate with CI results

---

## 3. Auto-Labeler

**Complexity**: Beginner  
**Frequency**: On issue/PR events  
**Safe Outputs**: add_labels

Automatically apply labels to issues and PRs based on content, file changes, or other criteria.

**Common Use Cases:**
- Label by file path (docs/, src/, tests/)
- Label by keywords in title/description
- Label by PR size or complexity
- Priority/severity labeling

**Template**: [`auto-labeler.md`](./auto-labeler.md)  
**Related Examples**:
- `examples/label-trigger-simple.md`
- `examples/label-trigger-pull-request.md`
- `examples/label-trigger-discussion.md`

**Key Features:**
- Simple trigger configuration
- Keyword and pattern matching
- Multi-label support
- Conditional logic

---

## 4. Issue Responder

**Complexity**: Beginner  
**Frequency**: On issue events  
**Safe Outputs**: create_comment, update_issue

Respond to new issues with helpful information, triage questions, or automated actions.

**Common Use Cases:**
- Welcome new contributors
- Request additional information
- Classify issue type
- Auto-assign based on keywords

**Template**: [`issue-responder.md`](./issue-responder.md)  
**Related Examples**:
- Repository: `.github/workflows/issue-classifier.md`
- Repository: `.github/workflows/issue-triage-agent.md`

**Key Features:**
- Triggered on issue creation
- Template validation
- Intelligent triage
- Auto-assignment logic

---

## 5. Discussion Bot

**Complexity**: Beginner  
**Frequency**: Scheduled or on-demand  
**Safe Outputs**: create_discussion, close_discussion

Create and manage GitHub Discussions for reports, announcements, or community engagement.

**Common Use Cases:**
- Daily/weekly report discussions
- Release announcement threads
- Community Q&A posts
- Auto-cleanup of old discussions

**Template**: [`discussion-bot.md`](./discussion-bot.md)  
**Related Examples**:
- Repository: `.github/workflows/daily-issues-report.md`
- `examples/label-trigger-discussion.md`

**Key Features:**
- Discussion creation with categories
- Auto-cleanup of old discussions
- Asset upload support
- Formatted content generation

---

## 6. Code Scanner

**Complexity**: Advanced  
**Frequency**: On push or scheduled  
**Safe Outputs**: create_issue, create_pull_request

Analyze codebase for patterns, issues, or improvements and report findings.

**Common Use Cases:**
- Detect breaking changes
- Check CLI consistency
- Audit dependencies
- Find code smells or anti-patterns

**Template**: [`code-scanner.md`](./code-scanner.md)  
**Related Examples**:
- Repository: `.github/workflows/breaking-change-checker.md`
- Repository: `.github/workflows/cli-consistency-checker.md`

**Key Features:**
- Deep code analysis with bash tools
- Pattern detection
- Issue creation for findings
- PR generation for fixes

---

## 7. CI Monitor

**Complexity**: Intermediate  
**Frequency**: On workflow_run  
**Safe Outputs**: add_comment, create_issue

Monitor CI/CD pipeline execution and provide alerts or analysis when workflows fail.

**Common Use Cases:**
- Failure root cause analysis
- Test failure reports
- Build time monitoring
- Flaky test detection

**Template**: [`ci-monitor.md`](./ci-monitor.md)  
**Related Examples**:
- Repository: `.github/workflows/dev-hawk.md`
- Repository: `.github/workflows/ci-doctor.md`

**Key Features:**
- workflow_run trigger
- Audit log analysis
- Failure correlation
- Auto-task creation for fixes

---

## 8. Workflow Manager

**Complexity**: Advanced  
**Frequency**: On-demand  
**Safe Outputs**: create_pull_request

Generate or update agentic workflow files based on requirements or patterns.

**Common Use Cases:**
- Generate workflows from templates
- Update workflow configurations
- Migrate legacy workflows
- Batch workflow updates

**Template**: [`workflow-manager.md`](./workflow-manager.md)  
**Related Examples**:
- Repository: `.github/workflows/workflow-generator.md`

**Key Features:**
- Workflow file generation
- Template-based creation
- Validation and compilation
- PR creation with changes

---

## 9. Documentation Bot

**Complexity**: Intermediate  
**Frequency**: Scheduled or on push  
**Safe Outputs**: create_pull_request

Generate or update documentation based on code changes or scheduled analysis.

**Common Use Cases:**
- Update README files
- Generate API documentation
- Sync code and docs
- Create changelog entries

**Template**: [`documentation-bot.md`](./documentation-bot.md)  
**Related Examples**:
- Repository: `.github/workflows/technical-doc-writer.md`
- Repository: `.github/workflows/daily-doc-updater.md`

**Key Features:**
- Documentation analysis
- Content generation
- PR creation with updates
- Link validation

---

## 10. Integration Bot

**Complexity**: Intermediate  
**Frequency**: Event-driven  
**Safe Outputs**: Varies by integration

Integrate with external tools like Slack, Notion, or custom APIs for notifications and data sync.

**Common Use Cases:**
- Slack notifications
- Notion database sync
- Custom webhook integration
- Third-party reporting

**Template**: [`integration-bot.md`](./integration-bot.md)  
**Related Examples**:
- Repository: `.github/workflows/notion-issue-summary.md`

**Key Features:**
- Network configuration for external APIs
- Webhook handling
- Data transformation
- Error handling and retry logic

---

## Getting Started

1. **Choose a pattern** that matches your use case
2. **Copy the template** to your `.github/workflows/` directory
3. **Customize** the TODO sections for your repository
4. **Compile** the workflow: `gh aw compile your-workflow.md`
5. **Test** with workflow_dispatch before enabling automated triggers
6. **Monitor** the workflow runs and iterate

## Best Practices

- **Start simple**: Begin with beginner patterns before tackling advanced ones
- **Test thoroughly**: Use `workflow_dispatch` triggers for initial testing
- **Use strict mode**: Enable `strict: true` for enhanced security
- **Monitor costs**: Watch AI token usage, especially for frequent workflows
- **Document changes**: Add comments explaining customizations
- **Version control**: Track workflow changes in git

## Need Help?

- **Documentation**: [GitHub Agentic Workflows Docs](https://githubnext.github.io/gh-aw/)
- **Examples**: Check `examples/` directory for more examples
- **Community**: Join `#continuous-ai` in [GitHub Next Discord](https://gh.io/next-discord)
- **Issues**: File bugs and feature requests in the repository

---

> **Note**: These patterns are based on analysis of 126 production workflows. Each template includes inline documentation, customization points, and links to related examples.
