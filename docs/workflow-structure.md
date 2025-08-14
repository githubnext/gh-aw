# 📋 Workflow Structure

This guide explains how agentic workflows are organized and structured within your repository.

## Directory Structure

Agentic workflows are stored in a unified location:

- **`.github/workflows/`**: Contains both your markdown workflow definitions (source files) and the generated GitHub Actions YAML files (.lock.yml files)
- **`.gitattributes`**: Automatically created/updated to mark `.lock.yml` files as generated code using `linguist-generated=true`

Create markdown files in `.github/workflows/` with the following structure:

```markdown
---
on:
  issues:
    types: [opened]

permissions:
  issues: write

tools:
  github:
    allowed: [add_issue_comment]
---

# Workflow Description

Read the issue #${{ github.event.issue.number }}. Add a comment to the issue listing useful resources and links.
```

## File Organization

Your repository structure will look like this:

```
your-repository/
├── .github/
│   └── workflows/
│       ├── issue-responder.md        # Your source workflow
│       ├── issue-responder.lock.yml  # Generated GitHub Actions file
│       ├── weekly-summary.md         # Another source workflow
│       └── weekly-summary.lock.yml   # Generated GitHub Actions file
├── .gitattributes                    # Marks .lock.yml as generated
└── ... (other repository files)
```

## Workflow File Format

Each workflow consists of:

1. **YAML Frontmatter**: Configuration options wrapped in `---`. See [Frontmatter Options](frontmatter.md) for details.
2. **Markdown Content**: Natural language instructions for the AI

### Example Workflow File

```markdown
---
name: Issue Auto-Responder
on:
  issues:
    types: [opened, labeled]

permissions:
  issues: write
  contents: read

engine: claude

tools:
  github:
    allowed: [get_issue, add_issue_comment, list_issue_comments]

cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules

max-runs: 50
stop-time: "2025-12-31 23:59:59"
ai-reaction: "eyes"
---

# Issue Auto-Responder

When a new issue is opened, analyze the issue content and:

1. Determine if it's a bug report, feature request, or question
2. Add appropriate labels based on the content
3. Provide a helpful initial response with:
   - Acknowledgment of the issue
   - Request for additional information if needed
   - Links to relevant documentation

The issue details are: "${{ needs.task.outputs.text }}"
```

## Expression Security

For security reasons, agentic workflows restrict which GitHub Actions expressions can be used in **markdown content**. This prevents potential security vulnerabilities from unauthorized access to secrets or environment variables.

> **Note**: These restrictions apply only to expressions in the markdown content portion of workflows. The YAML frontmatter can still use secrets and environment variables as needed for workflow configuration (e.g., `env:` and authentication).

### Allowed Expressions

The following GitHub Actions context expressions are permitted in workflow markdown:
#### GitHub Context Expressions

- `${{ github.event.after }}` - The SHA of the most recent commit on the ref after the push
- `${{ github.event.before }}` - The SHA of the most recent commit on the ref before the push
- `${{ github.event.check_run.id }}` - The ID of the check run that triggered the workflow
- `${{ github.event.check_suite.id }}` - The ID of the check suite that triggered the workflow
- `${{ github.event.comment.id }}` - The ID of the comment that triggered the workflow
- `${{ github.event.deployment.id }}` - The ID of the deployment that triggered the workflow
- `${{ github.event.deployment_status.id }}` - The ID of the deployment status that triggered the workflow
- `${{ github.event.head_commit.id }}` - The ID of the head commit for the push event
- `${{ github.event.installation.id }}` - The ID of the GitHub App installation
- `${{ github.event.issue.number }}` - The number of the issue that triggered the workflow
- `${{ github.event.label.id }}` - The ID of the label that triggered the workflow
- `${{ github.event.milestone.id }}` - The ID of the milestone that triggered the workflow
- `${{ github.event.organization.id }}` - The ID of the organization that triggered the workflow
- `${{ github.event.page.id }}` - The ID of the page build that triggered the workflow
- `${{ github.event.project.id }}` - The ID of the project that triggered the workflow
- `${{ github.event.project_card.id }}` - The ID of the project card that triggered the workflow
- `${{ github.event.project_column.id }}` - The ID of the project column that triggered the workflow
- `${{ github.event.pull_request.number }}` - The number of the pull request that triggered the workflow
- `${{ github.event.release.assets[0].id }}` - The ID of the first asset in a release
- `${{ github.event.release.id }}` - The ID of the release that triggered the workflow
- `${{ github.event.release.tag_name }}` - The tag name of the release that triggered the workflow
- `${{ github.event.repository.id }}` - The ID of the repository that triggered the workflow
- `${{ github.event.review.id }}` - The ID of the pull request review that triggered the workflow
- `${{ github.event.review_comment.id }}` - The ID of the review comment that triggered the workflow
- `${{ github.event.sender.id }}` - The ID of the user who triggered the workflow
- `${{ github.event.workflow_run.id }}` - The ID of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.conclusion }}` - The conclusion of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.html_url }}` - The URL of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.head_sha }}` - The head SHA of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.run_number }}` - The run number of the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.event }}` - The event that triggered the workflow run that triggered the current workflow
- `${{ github.event.workflow_run.status }}` - The status of the workflow run that triggered the current workflow
- `${{ github.actor }}` - The username of the user who triggered the workflow
- `${{ github.job }}` - Job ID of the current workflow run
- `${{ github.owner }}` - The owner of the repository (user or organization name)
- `${{ github.repository }}` - The owner and repository name (e.g., `octocat/Hello-World`)
- `${{ github.run_id }}` - A unique number for each workflow run within a repository
- `${{ github.run_number }}` - A unique number for each run of a particular workflow in a repository
- `${{ github.server_url }}` - Base URL of the server, e.g. https://github.com
- `${{ github.workflow }}` - The name of the workflow
- `${{ github.workspace }}` - The default working directory on the runner for steps

#### Special Pattern Expressions
- `${{ needs.* }}` - Any outputs from previous jobs (e.g., `${{ needs.task.outputs.text }}`)
- `${{ steps.* }}` - Any outputs from previous steps in the same job
- `${{ github.event.inputs.* }}` - Any workflow inputs when triggered by workflow_dispatch (e.g., `${{ github.event.inputs.name }}`)

### Prohibited Expressions

All other expressions are dissallowed.

### Security Rationale

This restriction prevents:
- **Secret leakage**: Prevents accidentally exposing secrets in AI prompts or logs
- **Environment variable exposure**: Protects sensitive configuration from being accessed
- **Code injection**: Prevents complex expressions that could execute unintended code
- **Expression injection**: Prevents malicious expressions from being injected into AI prompts
- **Prompt hijacking**: Stops unauthorized modification of workflow instructions through expression values
- **Cross-prompt information attacks (XPIA)**: Blocks attempts to leak information between different workflow executions

### Validation

Expression safety is validated during compilation with `gh aw compile`. If unauthorized expressions are found, you'll see an error like:

```
error: unauthorized expressions: [secrets.TOKEN, env.MY_VAR]. 
allowed: [github.repository, github.actor, github.workflow, ...]
```

### Example Valid Usage

```markdown
# Valid expressions
Repository: ${{ github.repository }}
Triggered by: ${{ github.actor }}  
Issue number: ${{ github.event.issue.number }}
Previous output: ${{ needs.task.outputs.text }}
User input: ${{ github.event.inputs.environment }}
Workflow run conclusion: ${{ github.event.workflow_run.conclusion }}

# Invalid expressions (will cause compilation error)
Token: ${{ secrets.GITHUB_TOKEN }}
Environment: ${{ env.MY_VAR }}
Complex: ${{ toJson(github.workflow) }}
```

## Generated Files

When you run `gh aw compile`, the system:

1. **Reads** your `.md` files from `.github/workflows/`
2. **Processes** the frontmatter and markdown content
3. **Generates** corresponding `.lock.yml` GitHub Actions workflow files
4. **Updates** `.gitattributes` to mark generated files

### Lock File Characteristics

- **Automatic Generation**: Never edit `.lock.yml` files manually
- **Complete Workflows**: Contains full GitHub Actions YAML
- **Security**: Includes proper permissions and secret handling
- **MCP Integration**: Sets up Model Context Protocol servers (see [MCP Guide](mcps.md))
- **Artifact Collection**: Automatically saves logs and outputs

## Best Practices

### File Naming

- Use descriptive names: `issue-responder.md`, `pr-reviewer.md`
- Follow kebab-case convention: `weekly-summary.md`
- Avoid spaces and special characters

### Version Control

- **Commit source files**: Always commit `.md` files
- **Commit generated files**: Also commit `.lock.yml` files for transparency
- **Ignore patterns**: Consider `.gitignore` entries if needed:

```gitignore
# Temporary workflow files (if any)
.github/workflows/*.tmp
```

## Integration with GitHub Actions

Generated workflows integrate seamlessly with GitHub Actions:

- **Standard triggers**: All GitHub Actions `on:` events supported
- **Permissions model**: Full GitHub Actions permissions syntax
- **Secrets access**: Automatic handling of required secrets
- **Artifact storage**: Logs and outputs saved as artifacts
- **Concurrency control**: Built-in safeguards against parallel runs

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Frontmatter Options](frontmatter.md) - Configuration options for workflows
- [MCPs](mcps.md) - Model Context Protocol configuration
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [Include Directives](include-directives.md) - Modularizing workflows with includes
- [Secrets Management](secrets.md) - Managing secrets and environment variables
