# Campaign Issue to PR Automation

This GitHub Action workflow automates the process of converting campaign issue forms into campaign specification pull requests.

## Trigger

The workflow is triggered when:
- A new issue is opened
- The issue has the `campaign` label (automatically applied by the "Start a Campaign" issue form)

## What it does

1. **Creates campaign spec using CLI**: 
   - Passes the issue body to `gh aw campaign create-from-issue` command
   - The CLI parses the issue form and generates `.github/workflows/<id>.campaign.md`
   - All parsing, validation, and file generation is handled by the CLI

2. **Compiles the spec**: 
   - Runs `gh aw compile --validate --verbose` to:
     - Validate the campaign spec
     - Generate the orchestrator workflow (`.github/workflows/<id>.campaign.g.lock.yml`)

3. **Creates a pull request**:
   - Creates a new branch `campaign/<id>`
   - Commits both the campaign spec and generated orchestrator
   - Opens a PR with details about the campaign
   - Links the PR back to the originating issue
   - Adds a comment to the issue with the PR link

## CLI Command

The workflow uses the `gh aw campaign create-from-issue` command, which:
- Reads the issue body from stdin
- Extracts all campaign form fields (name, ID, project URL, etc.)
- Validates required fields and formats
- Generates the campaign spec with proper YAML frontmatter and markdown body
- Returns the path to the created file

You can use this command manually:
```bash
gh aw campaign create-from-issue < issue-body.txt
```

## Usage

1. Go to Issues → New Issue
2. Select "🚀 Start a Campaign"
3. Fill out the form with campaign details
4. Submit the issue
5. Wait for the automated workflow to run (~2-3 minutes)
6. Review the generated PR
7. Merge the PR to activate the campaign

## Example

See issue #6635 for an example of a campaign issue that would trigger this automation.

## Troubleshooting

If the workflow fails, check the Actions tab for error messages. Common issues:
- Invalid campaign identifier format
- Missing required fields (Campaign ID or Project URL)
- Compilation errors in the generated spec

## Related Files

- Issue template: `.github/ISSUE_TEMPLATE/start-campaign.yml`
- Workflow: `.github/workflows/campaign-issue-to-pr.yml`
- CLI command: `pkg/campaign/issue.go` and `pkg/campaign/command.go`
- Campaign package: `pkg/campaign/`
