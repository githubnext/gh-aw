### Workflow Failure

**Workflow:** [{workflow_name}]({workflow_source_url})  
**Branch:** {branch}  
**Run URL:** {run_url}{pull_request_info}

{secret_verification_context}{assignment_errors_context}{create_discussion_errors_context}{code_push_failure_context}{repo_memory_validation_context}{missing_data_context}{missing_safe_outputs_context}

### Action Required

**Option 1: Assign this issue to agent using agentic-workflows**

Assign this issue to the `agentic-workflows` agent to automatically debug and fix the workflow failure.

**Option 2: Manually invoke the agent**

Debug this workflow failure using the `agentic-workflows` agent:

```
/agent agentic-workflows debug the agentic workflow {workflow_id} failure in {run_url}
```
