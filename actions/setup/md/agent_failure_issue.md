### Workflow Failure

**Workflow:** [{workflow_name}]({workflow_source_url})  
**Branch:** {branch}  
**Run:** {run_url}{pull_request_info}

{secret_verification_context}{inference_access_error_context}{app_token_minting_failed_context}{lockdown_check_failed_context}{assignment_errors_context}{assign_copilot_failure_context}{create_discussion_errors_context}{code_push_failure_context}{repo_memory_validation_context}{push_repo_memory_failure_context}{missing_data_context}{missing_safe_outputs_context}{engine_failure_context}{timeout_context}{fork_context}

### Action Required

**Assign this issue to Copilot** using the `agentic-workflows` sub-agent to automatically debug and fix the workflow failure.

<details>
<summary><b>Debug with any coding agent</b></summary>

Use this prompt with any coding agent (GitHub Copilot, Claude, Gemini, etc.):

````
Debug the agentic workflow failure using https://raw.githubusercontent.com/github/gh-aw/main/debug.md

The failed workflow run is at {run_url}
````

</details>

<details>
<summary><b>Manually invoke the agent</b></summary>

Debug this workflow failure using your favorite Agent CLI and the `agentic-workflows` prompt.

- Start your agent
- Load the `agentic-workflows` prompt from `.github/agents/agentic-workflows.agent.md` or <https://github.com/github/gh-aw/blob/main/.github/agents/agentic-workflows.agent.md>
- Type `debug the agentic workflow {workflow_id} failure in {run_url}`

</details>

> [!TIP]
> <details>
> <summary><b>Stop reporting this workflow as a failure</b></summary>
>
> To stop a workflow from creating failure issues, set `report-failure-as-issue: false` in its frontmatter:
> ```yaml
> safe-outputs:
>   report-failure-as-issue: false
> ```
>
> </details>
