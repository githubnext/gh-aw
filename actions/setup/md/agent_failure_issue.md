## Problem

The agentic workflow **{workflow_name}** has failed. This typically indicates a configuration or runtime error that requires user intervention.

## Failed Run

- **Workflow:** [{workflow_name}]({workflow_source_url})
- **Failed Run:** {run_url}{pull_request_info}

## How to investigate

Use the **agentic-workflows** custom agent to investigate this failure.

In GitHub Copilot Chat, type `/agent` and select **agentic-workflows**.

When prompted, tell it to **debug** the problem and provide the workflow run URL: {run_url}

The agent will help you:
- Analyze the failure logs
- Identify the root cause
- Suggest fixes for configuration or runtime errors
