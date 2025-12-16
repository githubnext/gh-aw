# Campaign Issue to PR Automation

This GitHub Action workflow automates the process of converting campaign issue forms into campaign specification pull requests.

## Trigger

The workflow is triggered when:
- A new issue is opened
- The issue has the `campaign` label (automatically applied by the "Start a Campaign" issue form)

## What it does

1. **Parses the issue form**: Extracts all campaign details from the issue body including:
   - Campaign Name
   - Campaign Identifier
   - Campaign Version
   - Project Board URL
   - Campaign Type / Playbook
   - Scope
   - Constraints
   - Prior Learnings
   - Campaign Description
   - Additional Context

2. **Validates the campaign**: 
   - Checks that required fields (ID and Project URL) are present
   - Validates ID format (lowercase letters, digits, and hyphens only)
   - Uses GitHub GraphQL API to verify the project board exists and is accessible

3. **Generates the campaign spec**: 
   - Creates `.github/workflows/<id>.campaign.md` with:
     - YAML frontmatter containing all campaign metadata
     - Markdown body with campaign details
     - State set to `active`
     - Default configurations for memory paths, metrics, and safe outputs

4. **Compiles the spec**: 
   - Runs `gh aw compile --validate --verbose` to:
     - Validate the campaign spec
     - Generate the orchestrator workflow (`.github/workflows/<id>.campaign.g.lock.yml`)

5. **Creates a pull request**:
   - Creates a new branch `campaign/<id>`
   - Commits both the campaign spec and generated orchestrator
   - Opens a PR with details about the campaign
   - Links the PR back to the originating issue
   - Adds a comment to the issue with the PR link

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
- Project board URL not accessible
- Compilation errors in the generated spec

## Related Files

- Issue template: `.github/ISSUE_TEMPLATE/start-campaign.yml`
- Workflow: `.github/workflows/campaign-issue-to-pr.yml`
- Campaign package: `pkg/campaign/`
