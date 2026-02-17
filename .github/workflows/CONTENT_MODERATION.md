# Content Moderation Workflow

A classic GitHub Actions workflow that implements automated content moderation by removing issues, PRs, discussions, and comments created by users on a configurable blocklist.

## Features

- ✅ **Blocklist-based filtering** - Configure usernames to automatically moderate
- ✅ **Automatic triggers** - Monitors issues, PRs, discussions, and comments
- ✅ **Manual moderation** - Use workflow_dispatch with GitHub URL for on-demand moderation
- ✅ **Staging mode** - Dry run capability for testing without actual moderation
- ✅ **Extensive logging** - Detailed logging using `core.info`, `core.warning`, `core.error`
- ✅ **Step summaries** - Generates comprehensive reports of all moderation actions
- ✅ **Flexible blocklist** - Support for additional runtime usernames

## Quick Start

### 1. Configure the Blocklist

Edit the workflow file `.github/workflows/content-moderation.yml` and add usernames to the `DEFAULT_BLOCKLIST` array (around line 75):

```javascript
const DEFAULT_BLOCKLIST = [
  'spammer123',
  'bot-account',
  'suspicious-user',
];
```

### 2. Automatic Moderation

Once configured, the workflow automatically runs when:
- Issues are opened, edited, or reopened
- Pull requests are opened, edited, reopened, or synchronized
- Issue comments are created or edited
- Discussions are created or edited
- Discussion comments are created or edited

### 3. Manual Moderation

For on-demand moderation, use the workflow_dispatch trigger:

1. Go to **Actions** → **Content Moderation** → **Run workflow**
2. Fill in the inputs:
   - **target_url**: GitHub URL of the content to moderate (required)
   - **staging**: Check to run in dry-run mode (optional, default: false)
   - **blocklist**: Additional usernames to block, comma-separated (optional)

#### Example URLs:
- Issue: `https://github.com/owner/repo/issues/123`
- Pull Request: `https://github.com/owner/repo/pull/456`
- Discussion: `https://github.com/owner/repo/discussions/789`
- Issue Comment: `https://github.com/owner/repo/issues/123#issuecomment-123456`
- Discussion Comment: `https://github.com/owner/repo/discussions/789#discussioncomment-123456`

## Actions Performed

When content from a blocklisted user is detected:

| Content Type | Action | Details |
|--------------|--------|---------|
| **Issue** | Closed | Issue is closed with `state_reason: not_planned` and a comment explaining the closure |
| **Pull Request** | Closed | PR is closed and a comment is added explaining the closure |
| **Issue Comment** | Hidden | Comment content is replaced with `[Content removed - user on moderation blocklist]` |
| **Discussion** | Closed | Discussion is closed using GraphQL API with reason `RESOLVED` |
| **Discussion Comment** | Deleted | Comment is permanently deleted using GraphQL API |

## Staging Mode

Use staging mode to test the workflow without actually performing any moderation actions:

1. Run the workflow with `staging: true`
2. Check the logs to see what actions would have been taken
3. Review the step summary for a comprehensive report

In staging mode:
- All blocklist checks are performed
- All logging occurs as normal
- No actual content is modified or removed
- Actions are logged with "(Staged)" suffix in the summary

## Logging and Monitoring

The workflow provides extensive logging:

### Console Logs
- **Headers**: Major workflow sections (e.g., "Content Moderation Workflow")
- **Sections**: Subsections (e.g., "Blocklist Check", "Taking Moderation Action")
- **Actions**: Details of each action taken with all relevant metadata
- **Success**: Confirmation messages for completed actions (✅)
- **Warnings**: Non-critical issues like empty blocklist (⚠️)
- **Errors**: Failure messages with error details (❌)

### Step Summary
After each run, a comprehensive summary is generated showing:
- Run timestamp and trigger event
- Blocklist configuration
- Blocked users detected
- All actions taken with timestamps and details
- Any errors encountered

## Permissions Required

The workflow requires the following permissions:

```yaml
permissions:
  issues: write
  pull-requests: write
  discussions: write
```

These permissions are automatically configured in the workflow file.

## Examples

### Example 1: Automatic Moderation
When a blocklisted user creates an issue:
1. Workflow is triggered by `issues.opened` event
2. User is checked against blocklist
3. If match found, issue is closed with explanation comment
4. Summary is generated showing the action taken

### Example 2: Manual Moderation with Staging
To test moderation on a specific comment:
1. Copy the comment URL (e.g., `https://github.com/owner/repo/issues/123#issuecomment-456789`)
2. Run workflow with:
   - `target_url`: The comment URL
   - `staging`: `true`
3. Review logs to see what would happen
4. Review step summary for details
5. Run again with `staging: false` to execute

### Example 3: Additional Blocklist Users
To temporarily block additional users without editing the workflow:
1. Run workflow_dispatch
2. Set `blocklist`: `temp-spammer,another-user`
3. These users are added to the blocklist for this run only

## Troubleshooting

### Blocklist is empty warning
If you see "Blocklist is empty - no moderation will occur":
- Edit the workflow file and add usernames to `DEFAULT_BLOCKLIST`
- Or use workflow_dispatch with the `blocklist` input

### Invalid GitHub URL format
If you see "Invalid GitHub URL format" in workflow_dispatch:
- Ensure the URL matches the expected format
- Include the full GitHub URL (not a shortened link)
- Check that the URL is for issues, PRs, or discussions

### Action failed to execute
Check the error logs for details:
- Verify the workflow has the required permissions
- Ensure the target content exists and is accessible
- For discussions, ensure GraphQL API access is available

## Advanced Configuration

### Custom Actions
To customize the actions taken for different content types, edit the `main()` function in the workflow file and modify the switch statement around line 580.

### Custom Comments
To change the comment added when closing issues/PRs, edit the `closeIssue()` and `closePullRequest()` functions around lines 210 and 250.

### Logging Verbosity
The workflow uses extensive logging by default. To reduce verbosity:
- Comment out `core.info()` calls in the logging utilities section
- Keep `core.warning()` and `core.error()` for critical information

## Security Considerations

- **Manual verification**: Review the blocklist regularly to ensure it's accurate
- **False positives**: Use staging mode first to avoid accidentally blocking legitimate users
- **Permissions**: The workflow has write access to issues, PRs, and discussions
- **Audit trail**: All actions are logged in the step summary for accountability

## License

This workflow is part of the gh-aw repository and follows the same license.
