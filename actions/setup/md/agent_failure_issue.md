## Problem

The agentic workflow **{workflow_name}** has failed. This typically indicates a configuration or runtime error that requires user intervention.

## Failed Run

- **Workflow:** [{workflow_name}]({workflow_source_url})
- **Failed Run:** {run_url}
- **Source:** {workflow_source}
{github_context_section}
## How to investigate

Use the **debug-agentic-workflow** agent to investigate this failure.

In GitHub Copilot Chat, type `/agent` and select **debug-agentic-workflow**.

When prompted, provide the workflow run URL: {run_url}

The debug agent will help you:
- Analyze the failure logs
- Identify the root cause
- Suggest fixes for configuration or runtime errors
