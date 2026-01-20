# Security Alert Burndown Campaign - Setup Instructions

This document provides step-by-step instructions for completing the setup of the Security Alert Burndown campaign.

## Overview

The Security Alert Burndown campaign has been created to automatically fix code security alerts in the `githubnext/gh-aw` repository. The campaign uses two worker workflows:

1. **code-scanning-fixer** - Runs every 30 minutes, creates full PRs for high-severity alerts
2. **security-fix-pr** - Runs every 4 hours, uses GitHub's native autofix API

## ✅ Completed Steps

- [x] Campaign spec file created at `.github/workflows/security-alert-burndown.campaign.md`
- [x] Campaign validates successfully
- [x] Worker workflows exist and are compiled
- [x] Campaign structure follows best practices
- [x] Risk level set to medium (automated changes with review)
- [x] Governance limits configured

## ⚠️ Required Manual Steps

### 1. Update GitHub Project URL

**Action Required**: Find the "Security Alert Burndown" GitHub Project and update the URL in the campaign spec.

**Steps**:
1. Navigate to https://github.com/orgs/githubnext/projects
2. Search for "Security Alert Burndown" project
3. Copy the project URL (format: `https://github.com/orgs/githubnext/projects/[NUMBER]`)
4. Edit `.github/workflows/security-alert-burndown.campaign.md`
5. Replace line 8: `project-url: https://github.com/orgs/githubnext/projects/1234`
   With: `project-url: https://github.com/orgs/githubnext/projects/[ACTUAL_NUMBER]`
6. Commit the change

**Verification**:
```bash
gh aw campaign validate  # Should show no errors
gh aw campaign status    # Should display the campaign
```

### 2. Configure Required Project Fields

**Action Required**: Add custom fields to the GitHub Project board.

According to the issue description, the project needs these custom fields configured:

#### Field Configuration Table

| Field Name | Type | Options/Format |
|------------|------|----------------|
| `status` | Single-select | `Todo`, `In Progress`, `Review required`, `Blocked`, `Done` |
| `campaign_id` | Text | Free text |
| `worker_workflow` | Text | Free text |
| `repository` | Text | Free text |
| `priority` | Single-select | `High`, `Medium`, `Low` |
| `size` | Single-select | `Small`, `Medium`, `Large` |
| `start_date` | Date | Date picker |
| `end_date` | Date | Date picker |

**Steps**:
1. Go to the GitHub Project: `https://github.com/orgs/githubnext/projects/[NUMBER]`
2. Click on the "⚙️ Settings" icon (top right)
3. Navigate to "Custom fields"
4. For each field above:
   - Click "+ New field"
   - Enter the field name exactly as shown
   - Select the field type
   - For single-select fields, add all options listed
   - Save the field

**Alternative (Using GitHub CLI)**:
```bash
# Install gh project extension if not already installed
gh extension install github/gh-projects

# Add custom fields (requires appropriate permissions)
gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "status" --data-type "SINGLE_SELECT" \
  --single-select-options "Todo,In Progress,Review required,Blocked,Done"

gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "campaign_id" --data-type "TEXT"

gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "worker_workflow" --data-type "TEXT"

gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "repository" --data-type "TEXT"

gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "priority" --data-type "SINGLE_SELECT" \
  --single-select-options "High,Medium,Low"

gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "size" --data-type "SINGLE_SELECT" \
  --single-select-options "Small,Medium,Large"

gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "start_date" --data-type "DATE"

gh project field-create [PROJECT_NUMBER] --owner githubnext \
  --name "end_date" --data-type "DATE"
```

### 3. Verify Campaign Setup

After completing the manual steps above:

```bash
# Validate the campaign
gh aw campaign validate

# Check campaign status
gh aw campaign status

# View campaign details
gh aw campaign security-alert-burndown
```

## Campaign Operation

Once the project URL is updated and fields are configured, the campaign will:

1. **Discover** security issues via the tracker label `campaign:security-alert-burndown`
2. **Coordinate** by adding discovered items to the GitHub Project
3. **Track Progress** using the configured KPIs:
   - Primary: File Write Alerts Fixed (target: 100 alerts)
   - Supporting: Alert Backlog Size (target: 0 alerts)
4. **Dispatch** worker workflows according to their schedules:
   - `code-scanning-fixer`: Every 30 minutes
   - `security-fix-pr`: Every 4 hours
5. **Report** campaign status and progress

## Risk Management

**Risk Level**: Medium

- All fixes are submitted as pull requests (no direct commits)
- Human review required before merging
- Cache memory prevents duplicate fixes
- Focus on high-severity issues
- Up to 3 alerts clustered per run (governance limit)

## Troubleshooting

### Campaign not showing in status
- Ensure `.campaign.md` file is in `.github/workflows/`
- Run `gh aw campaign validate` to check for errors
- Verify YAML frontmatter is valid

### Worker workflows not dispatching
- Check that workflows exist: `code-scanning-fixer.md` and `security-fix-pr.md`
- Ensure workflows are compiled (`.lock.yml` files exist)
- Verify workflow permissions allow `workflow_dispatch`

### Project updates not working
- Verify project URL is correct
- Ensure project fields are configured exactly as specified
- Check that `GH_AW_PROJECT_GITHUB_TOKEN` secret has appropriate permissions

## References

- **Issue**: #10820
- **Campaign Spec**: `.github/workflows/security-alert-burndown.campaign.md`
- **Worker Workflow 1**: `.github/workflows/code-scanning-fixer.md`
- **Worker Workflow 2**: `.github/workflows/security-fix-pr.md`
- **Documentation**: [Campaign Specs Guide](https://github.com/githubnext/gh-aw/blob/main/docs/src/content/docs/guides/campaigns/specs.md)

## Support

For questions or issues with campaign setup:
- Review the [gh-aw campaigns documentation](https://github.com/githubnext/gh-aw/tree/main/docs/src/content/docs/guides/campaigns)
- Check existing campaigns for examples
- Contact @mnkiefer (campaign owner)
