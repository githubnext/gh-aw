## Missing Secret Configuration

The workflow requires one or more secrets that are not configured in the repository. The AI agent cannot proceed without proper authentication credentials.

### Required Secrets

{missing_secrets_list}

## Why This Matters

AI agents require valid API credentials to function. The workflow has been configured to use **{engine_name}** which requires specific secrets to be set up in your repository.

Without these secrets:
- The AI agent cannot authenticate with external services
- Workflow execution will fail immediately
- No tasks can be completed

## Action Required

Configure the required secret(s) in your repository settings:

1. **Navigate to repository settings:** [Repository Secrets](https://github.com/{repo_owner}/{repo_name}/settings/secrets/actions)
2. **Add the secret:** Click "New repository secret"
3. **Set the value:** Paste your API key or token
4. **Save:** The secret will be available for future workflow runs

### Documentation

For detailed instructions on obtaining and configuring these secrets, see:
- **Secret Documentation:** {docs_url}

## Agent Instructions

**Agent:** `agentic-workflows`
**Task:** Debug and resolve missing secret issue

**Steps:**

1. Invoke agent: `/agent agentic-workflows`
2. Command: "Debug this missing secret issue"
3. Review which secret(s) are missing
4. Follow the documentation to obtain the required credentials
5. Add the secret(s) to repository settings
6. Re-run the workflow to validate the configuration

## References

- **Workflow:** [{workflow_name}]({workflow_source_url})
- **Failed Run:** {run_url}
- **Engine:** {engine_name}
- **Source:** {workflow_source}
