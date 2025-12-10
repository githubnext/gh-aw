---
title: DispatchOps
description: Manually trigger and test agentic workflows with custom inputs using workflow_dispatch
sidebar:
  badge: { text: 'Manual', variant: 'tip' }
---

DispatchOps enables manual workflow execution via the GitHub Actions UI or CLI, perfect for on-demand tasks, testing, and workflows that need human judgment about timing. The `workflow_dispatch` trigger lets you run workflows with custom inputs whenever needed.

Use DispatchOps for research tasks, operational commands, testing workflows during development, debugging production issues, or any task that doesn't fit a schedule or event trigger.

## How Workflow Dispatch Works

Workflows with `workflow_dispatch` can be triggered manually rather than waiting for events like issues, pull requests, or schedules.

### Basic Syntax

Add `workflow_dispatch:` to the `on:` section in your workflow frontmatter:

```yaml
on:
  workflow_dispatch:
```

### With Input Parameters

Define inputs to customize workflow behavior at runtime:

```yaml
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
      priority:
        description: 'Task priority'
        required: false
        type: choice
        options:
          - low
          - medium
          - high
        default: medium
```

**Supported input types:**
- **`string`** - Free-form text input
- **`boolean`** - True/false checkbox
- **`choice`** - Dropdown selection with predefined options
- **`environment`** - Repository environment selector

## Security Model

### Permission Requirements

Manual workflow execution respects the same security model as other triggers:

- **Repository permissions** - User must have write access or higher to trigger workflows
- **Role-based access** - Use the `roles:` field to restrict who can run workflows:

```yaml
on:
  workflow_dispatch:
roles: [admin, maintainer]
```

- **Bot authorization** - Use the `bots:` field to allow specific bot accounts:

```yaml
on:
  workflow_dispatch:
bots: ["dependabot[bot]", "github-actions[bot]"]
```

### Fork Protection

Unlike issue/PR triggers, `workflow_dispatch` only executes in the repository where it's defined—forks cannot trigger workflows in the parent repository. This provides inherent protection against fork-based attacks.

### Environment Approval Gates

Require manual approval before execution using GitHub environment protection rules:

```yaml
on:
  workflow_dispatch:
manual-approval: production
```

Configure approval rules, required reviewers, and wait timers in repository Settings → Environments. See [GitHub's environment documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment) for setup details.

## Running Workflows from GitHub.com

### Via Actions Tab

1. Navigate to your repository on GitHub.com
2. Click the **Actions** tab
3. Select the workflow from the left sidebar
4. Click the **Run workflow** dropdown button
5. Select the branch to run from (default: main)
6. Fill in any required inputs
7. Click the **Run workflow** button

The workflow will execute immediately, and you can watch progress in the Actions tab.

### Finding Runnable Workflows

Only workflows with `workflow_dispatch:` appear in the "Run workflow" dropdown. If your workflow isn't listed:
- Verify `workflow_dispatch:` exists in the `on:` section
- Ensure the workflow has been compiled and pushed to GitHub
- Check that the `.lock.yml` file exists in `.github/workflows/`

## Running Workflows with CLI

The `gh aw run` command provides a faster way to trigger workflows from the command line.

### Basic Usage

```bash
gh aw run workflow-name
```

The command:
1. Finds the workflow by name (e.g., `research` matches `research.md`)
2. Validates it has `workflow_dispatch:` trigger
3. Triggers execution via GitHub Actions API
4. Returns immediately with the run URL

### With Input Parameters

Pass inputs using the `--raw-field` or `-f` flag in `key=value` format:

```bash
gh aw run research --raw-field topic="quantum computing"
```

**Multiple inputs:**
```bash
gh aw run scout \
  --raw-field topic="AI safety research" \
  --raw-field priority=high
```

### Wait for Completion

Monitor workflow execution and wait for results:

```bash
gh aw run research --raw-field topic="AI agents" --wait
```

The `--wait` flag:
- Monitors workflow progress in real-time
- Shows status updates
- Waits for completion before returning
- Exits with success/failure code based on workflow result

### Branch Selection

Run workflows from specific branches:

```bash
gh aw run research --ref feature-branch
```

### Running Remote Workflows

Execute workflows from other repositories:

```bash
gh aw run workflow-name --repo owner/repository
```

### Verbose Output

See detailed execution information:

```bash
gh aw run research --raw-field topic="AI" --verbose
```

## Declaring and Referencing Inputs

### Declaring Inputs in Frontmatter

Define inputs in the `workflow_dispatch` section with clear descriptions:

```yaml
on:
  workflow_dispatch:
    inputs:
      analysis_depth:
        description: 'How deep should the analysis go?'
        required: true
        type: choice
        options:
          - surface
          - detailed
          - comprehensive
        default: detailed
      
      include_examples:
        description: 'Include code examples in the report'
        required: false
        type: boolean
        default: true
      
      max_results:
        description: 'Maximum number of results to return'
        required: false
        type: string
        default: '10'
```

**Best practices:**
- Use descriptive `description` text to guide users
- Set sensible `default` values for optional inputs
- Use `choice` type to constrain options and prevent invalid values
- Mark truly required inputs with `required: true`

### Referencing Inputs in Markdown

Access input values using GitHub Actions expression syntax:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
      depth:
        description: 'Analysis depth'
        type: choice
        options:
          - brief
          - detailed
        default: brief
permissions:
  contents: read
safe-outputs:
  create-discussion:
---

# Research Assistant

Research the following topic: "${{ github.event.inputs.topic }}"

Analysis depth requested: ${{ github.event.inputs.depth }}

Provide a ${{ github.event.inputs.depth }} analysis with key findings and recommendations.
```

**Expression syntax:**
- Use `${{ github.event.inputs.INPUT_NAME }}` to reference input values
- Inputs are available throughout the entire workflow markdown
- Values are interpolated at workflow compile time into the GitHub Actions YAML

### Conditional Logic Based on Inputs

Use Handlebars conditionals to change behavior based on input values:

```markdown
{{#if (eq github.event.inputs.include_code "true")}}
Include actual code snippets in your analysis.
{{else}}
Describe code patterns without including actual code.
{{/if}}

{{#if (eq github.event.inputs.priority "high")}}
URGENT: Prioritize speed over completeness.
{{/if}}
```

## Development Pattern: Branch Testing

A common pattern is developing workflows in a branch before merging to main:

### Pattern 1: Push to Main, Dispatch from Branch

**When to use:** Testing workflow changes without affecting production triggers.

**Steps:**

1. **Develop in a feature branch:**
   ```bash
   git checkout -b feature/improve-research-workflow
   # Edit .github/workflows/research.md
   ```

2. **Add workflow_dispatch if not present:**
   ```yaml
   on:
     schedule:
       - cron: daily at 09:00
     workflow_dispatch:  # Add this for testing
   ```

3. **Compile and commit to branch:**
   ```bash
   gh aw compile research
   git add .github/workflows/research.md .github/workflows/research.lock.yml
   git commit -m "Add manual dispatch for testing"
   git push origin feature/improve-research-workflow
   ```

4. **Push workflow to main for testing:**
   ```bash
   # Temporarily merge or cherry-pick to main
   git checkout main
   git cherry-pick <commit-sha>
   git push origin main
   ```

5. **Test by dispatching from your branch:**
   ```bash
   # Switch back to your branch
   git checkout feature/improve-research-workflow
   
   # Run the workflow that's now on main
   gh aw run research --ref feature/improve-research-workflow
   ```

6. **Observe behavior in your branch context:**
   - The workflow runs with your branch's code
   - Safe outputs (issues, PRs, comments) are created
   - Any repository reads use your branch's state

7. **Iterate and refine:**
   - Make changes to the workflow on your branch
   - Re-compile: `gh aw compile research`
   - Push workflow updates to main
   - Test again with dispatch

8. **Clean up when done:**
   - Create PR from your branch to main
   - Remove temporary commits from main if needed

### Pattern 2: Trial Mode for Isolated Testing

**When to use:** Testing without affecting any production repository.

Instead of pushing to main, use trial mode:

```bash
# Test in isolated trial repository
gh aw trial ./research.md --raw-field topic="test query"
```

See the [TrialOps guide](/gh-aw/guides/trialops/) for complete trial testing patterns.

### Pattern 3: Development Workflow

**Complete development cycle:**

```bash
# 1. Create feature branch
git checkout -b feature/new-workflow

# 2. Create/edit workflow with workflow_dispatch
cat > .github/workflows/my-workflow.md <<EOF
---
on:
  workflow_dispatch:
    inputs:
      test_input:
        description: 'Test parameter'
        required: true
permissions:
  contents: read
---
# My Workflow
Test input: \${{ github.event.inputs.test_input }}
EOF

# 3. Compile
gh aw compile my-workflow

# 4. Test locally with trial mode first
gh aw trial ./my-workflow.md --raw-field test_input="hello"

# 5. Once working, commit to branch
git add .github/workflows/
git commit -m "Add new workflow"
git push origin feature/new-workflow

# 6. For testing in real repo, temporarily push to main
git checkout main
git merge feature/new-workflow
git push origin main

# 7. Switch back and test with dispatch
git checkout feature/new-workflow
gh aw run my-workflow --raw-field test_input="production test"

# 8. Watch results, iterate if needed

# 9. Create PR when satisfied
gh pr create --title "Add new workflow" --body "Testing complete"
```

## Common Use Cases

### On-Demand Research

```yaml
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
```

Run when needed:
```bash
gh aw run research --raw-field topic="AI safety best practices"
```

### Manual Operations

```yaml
on:
  workflow_dispatch:
    inputs:
      operation:
        description: 'Operation to perform'
        type: choice
        options:
          - cleanup
          - sync
          - audit
```

Execute specific tasks:
```bash
gh aw run operations --raw-field operation=audit
```

### Testing and Debugging

```yaml
on:
  issues:
    types: [opened]
  workflow_dispatch:  # Add for testing without creating real issues
    inputs:
      test_issue_url:
        description: 'Test issue URL'
        required: false
```

Test issue handling:
```bash
gh aw run triage --raw-field test_issue_url="https://github.com/org/repo/issues/123"
```

### Scheduled Workflow Testing

```yaml
on:
  schedule:
    - cron: daily at 09:00
  workflow_dispatch:  # Test before waiting for schedule
```

Test scheduled workflows immediately:
```bash
gh aw run daily-report
```

## Troubleshooting

### Workflow Not Listed in GitHub UI

**Problem:** Workflow doesn't appear in "Run workflow" dropdown.

**Solutions:**
- Verify `workflow_dispatch:` is in the `on:` section
- Compile the workflow: `gh aw compile workflow-name`
- Push both `.md` and `.lock.yml` files to GitHub
- Check the compiled `.lock.yml` has `workflow_dispatch` trigger
- Refresh the Actions page (may take a few seconds)

### "Workflow Not Found" Error

**Problem:** `gh aw run` can't find the workflow.

**Solutions:**
- Use the workflow filename without `.md`: `research` not `research.md`
- Ensure workflow exists in `.github/workflows/`
- Check if workflow has been compiled
- Try explicit path: `gh aw run .github/workflows/research.md`

### "Workflow Cannot Be Run" Error

**Problem:** Workflow found but can't be executed.

**Solutions:**
- Add `workflow_dispatch:` to the workflow's `on:` section
- Recompile after adding: `gh aw compile workflow-name`
- Verify the `.lock.yml` includes workflow_dispatch
- Push changes to GitHub before running

### Permission Denied

**Problem:** User lacks permissions to trigger workflow.

**Solutions:**
- Verify you have write access to the repository
- Check the `roles:` field in workflow frontmatter
- Confirm you're not in the excluded list
- For organization repos, verify your org role

### Inputs Not Appearing

**Problem:** Input fields don't show in GitHub UI.

**Solutions:**
- Check YAML syntax in `workflow_dispatch.inputs`
- Ensure proper indentation (2 spaces)
- Validate input type is one of: `string`, `boolean`, `choice`, `environment`
- Recompile and push the workflow
- Clear browser cache

### Branch Context Issues

**Problem:** Workflow runs with wrong branch context.

**Solutions:**
- Specify branch explicitly: `gh aw run workflow --ref branch-name`
- In GitHub UI, select correct branch in dropdown before running
- Verify workflow exists in the target branch
- Check that branch has the compiled `.lock.yml` file

## Best Practices

### Input Design

**Do:**
- Use descriptive input names: `analysis_depth` not `depth`
- Provide helpful descriptions: "How detailed should the analysis be?"
- Set sensible defaults for optional inputs
- Use `choice` type to constrain options
- Group related inputs logically

**Don't:**
- Create too many inputs (>5 typically overwhelming)
- Use vague descriptions: "Input 1", "Value"
- Make everything required when defaults work
- Use `string` type for values that should be constrained

### Development Workflow

**Do:**
- Always include `workflow_dispatch:` during development
- Test with trial mode first: `gh aw trial workflow.md`
- Use meaningful commit messages when testing iterations
- Document your testing process
- Clean up test branches after merging

**Don't:**
- Remove `workflow_dispatch` from workflows you might need to test
- Test directly in production without trial mode first
- Leave debugging inputs in production workflows
- Push untested workflow changes directly to main

### Security Considerations

**Do:**
- Use `roles:` to restrict sensitive operations
- Set `manual-approval:` for production workflows
- Validate and sanitize string inputs in workflow logic
- Document who should run what workflows
- Review workflow run history regularly

**Don't:**
- Allow unrestricted workflow_dispatch on sensitive operations
- Pass secrets or credentials via inputs
- Trust input values without validation
- Use workflow_dispatch as a replacement for proper CI/CD

### Combining Triggers

Add `workflow_dispatch` to event-triggered workflows for testing:

```yaml
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
  workflow_dispatch:  # For testing without creating real issues/PRs
    inputs:
      test_url:
        description: 'Test issue/PR URL'
        required: false
```

This enables:
- Automated execution on real events
- Manual testing without creating test issues
- Debugging production problems with specific examples

## Related Documentation

- [Manual Workflows Example](/gh-aw/examples/manual/) - Example manual workflows
- [Triggers Reference](/gh-aw/reference/triggers/) - Complete trigger syntax including workflow_dispatch
- [TrialOps Guide](/gh-aw/guides/trialops/) - Testing workflows in isolation
- [CLI Commands](/gh-aw/setup/cli/) - Complete gh aw run command reference
- [Templating](/gh-aw/reference/templating/) - Using expressions and conditionals
- [Security Best Practices](/gh-aw/guides/security/) - Securing workflow execution
- [Quick Start](/gh-aw/setup/quick-start/) - Getting started with agentic workflows
