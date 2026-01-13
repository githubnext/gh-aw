# Secrets Diagnostics Workflow

This workflow provides automated diagnostics for all secrets used in the `gh-aw` repository. It tests each secret's configuration and generates a comprehensive diagnostic report.

## Purpose

The Secrets Diagnostics workflow helps maintain the health of repository secrets by:

- **Validating Configuration**: Tests that secrets are properly configured and accessible
- **Checking Permissions**: Verifies that tokens have appropriate API access
- **Identifying Issues**: Reports failures and missing configurations
- **Tracking Changes**: Provides historical reports via artifacts

## Workflow Details

### Triggers

- **Manual**: Can be triggered via `workflow_dispatch` from the Actions tab
- **Scheduled**: Runs weekly on Mondays at 9:00 AM UTC

### Secrets Tested

The workflow tests the following secrets:

#### GitHub Tokens
- `GH_AW_GITHUB_TOKEN` - Tests REST API and GraphQL API access to the repository
- `GH_AW_GITHUB_MCP_SERVER_TOKEN` - Tests GitHub MCP server authentication
- `GH_AW_PROJECT_GITHUB_TOKEN` - Tests GitHub Projects API access

#### AI Engine API Keys
- `ANTHROPIC_API_KEY` - Tests Claude API authentication
- `OPENAI_API_KEY` - Tests OpenAI API authentication
- `GH_AW_COPILOT_TOKEN` - Tests GitHub Copilot CLI availability
- `BRAVE_API_KEY` - Tests Brave Search API authentication

#### Integration Tokens
- `NOTION_API_TOKEN` - Tests Notion API authentication

### Test Results

Each secret test can have one of four statuses:

- ✅ **Success**: Secret is configured and working correctly
- ❌ **Failure**: Secret is configured but failed validation (invalid key, insufficient permissions, etc.)
- ⚪ **Not Set**: Secret is not configured (expected if the service is not used)
- ⏭️ **Skipped**: Test was skipped (e.g., Copilot CLI not installed)

### Outputs

1. **Step Summary**: The diagnostic report is displayed in the GitHub Actions step summary for quick viewing
2. **Artifact**: The full `diagnostics.md` report is uploaded as an artifact with 30-day retention
3. **Console Output**: Real-time test results are logged to the console with color-coded status indicators

## Report Structure

The generated `diagnostics.md` report includes:

### Summary Section
- Total number of tests
- Count of successful, failed, not set, and skipped tests

### Detailed Results
- Per-secret status and messages
- API response codes and error details
- JSON details for debugging

### Recommendations
- List of secrets that are not configured
- List of secrets that failed validation
- Suggestions for remediation

## Implementation

### Script Location
`.github/scripts/secrets-diagnostics.cjs`

The diagnostic script is a standalone Node.js script that:
- Uses native `https` module for API calls (no external dependencies)
- Implements timeout handling (10 seconds per request)
- Provides colored console output
- Generates structured markdown reports
- Returns informational exit codes (never fails the workflow)

### Workflow File
`.github/workflows/secrets-diagnostics.yml`

The workflow:
- Runs on Ubuntu Latest
- Uses Node.js 22
- Has read-only permissions
- 10-minute timeout
- Never fails on test failures (informational only)

## Usage

### Running Manually

1. Go to the Actions tab in the GitHub repository
2. Select "Secrets Diagnostics" workflow
3. Click "Run workflow"
4. Select the branch (usually `main`)
5. Click "Run workflow" button

### Viewing Results

After the workflow completes:

1. Click on the workflow run
2. View the step summary for a quick overview
3. Download the `secrets-diagnostic-report` artifact for the full report
4. Check individual test logs in the "Run diagnostics script" step

### Interpreting Results

- **All Not Set**: Normal for a fresh setup or if secrets aren't needed
- **Some Successful**: Indicates those secrets are working correctly
- **Some Failed**: Requires investigation - check the error details in the report
- **All Successful**: Ideal state - all configured secrets are working

## Maintenance

### Adding New Secrets

To add a new secret to the diagnostic workflow:

1. Add the secret test function in `.github/scripts/secrets-diagnostics.cjs`:
   ```javascript
   async function testNewSecret(token) {
     if (!token) {
       return { status: Status.NOT_SET, message: 'Token not set' };
     }
     // Implement test logic
   }
   ```

2. Add the test call in `runDiagnostics()`:
   ```javascript
   const newToken = process.env.NEW_SECRET;
   const newResult = await testNewSecret(newToken);
   results.push({
     secret: 'NEW_SECRET',
     test: 'API Test Name',
     ...newResult
   });
   ```

3. Add the secret to the workflow environment in `.github/workflows/secrets-diagnostics.yml`:
   ```yaml
   env:
     NEW_SECRET: ${{ secrets.NEW_SECRET }}
   ```

### Modifying Test Logic

The test functions follow a consistent pattern:
1. Check if the secret is set (return `NOT_SET` if missing)
2. Attempt to authenticate with the service
3. Return `SUCCESS`, `FAILURE`, or `SKIPPED` with a descriptive message
4. Include relevant details for debugging

## Security Considerations

- The workflow has **read-only** permissions
- Secrets are never logged or exposed in output
- API calls use minimal endpoints that don't modify data
- The workflow runs in isolation and doesn't affect other workflows
- Failed secrets don't block the repository or other workflows

## Troubleshooting

### Common Issues

**Secret shows as "Not Set" but it's configured**
- The secret may not be added to the workflow environment variables
- Check `.github/workflows/secrets-diagnostics.yml` to ensure the secret is listed

**Test fails with timeout**
- API endpoint may be temporarily unavailable
- Timeout is set to 10 seconds - network issues can cause timeouts
- Re-run the workflow to verify if it's a transient issue

**API returns 403 Forbidden**
- Secret may have insufficient permissions
- For GitHub tokens, ensure the token has `repo` scope for private repositories
- Check the API provider's documentation for required permissions

**Copilot CLI test skipped**
- The Copilot CLI is not installed in the GitHub Actions runner by default
- This is expected behavior - the test is informational only

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub REST API](https://docs.github.com/en/rest)
- [GitHub GraphQL API](https://docs.github.com/en/graphql)
- [Anthropic Claude API](https://docs.anthropic.com/claude/reference)
- [OpenAI API](https://platform.openai.com/docs/api-reference)
- [Brave Search API](https://brave.com/search/api/)
- [Notion API](https://developers.notion.com/)
