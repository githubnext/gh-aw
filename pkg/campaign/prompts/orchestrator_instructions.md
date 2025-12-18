Each time this orchestrator runs on its daily schedule (or when manually dispatched), generate a concise status report for this campaign. Summarize current metrics{{if .ReportBlockers}}, highlight blockers,{{end}} and monitor the progress of associated worker workflows.

**Tracking Worker Output**: The campaign has knowledge of its workers (listed in the `workflows` field) and monitors their output directly. For each worker workflow:

1. **Query GitHub Actions API for recent workflow runs** using the GitHub MCP server:
   - Use `list_workflow_runs` with the worker's workflow file (e.g., `daily-file-diet.lock.yml`)
   - Filter for recent completed runs (last 24-48 hours) to find worker activity
   - Use `get_workflow_run` to get detailed information about specific runs
   
2. **Inspect workflow runs and retrieve logs** (using GitHub CLI or API):
   - Use `gh run list` to find recent workflow runs for workers
   - Use `gh run view <run-id>` to get detailed information about specific runs
   - Use `gh run view <run-id> --log` to retrieve logs from worker executions
   - Parse logs to understand what actions the worker took and what outputs were generated
   - Look for safe-output action results in the logs (e.g., issue URLs, PR URLs)
   
3. **Extract issue URLs from workflow run artifacts**:
   - Use `list_workflow_run_artifacts` to list artifacts from each workflow run
   - Download the `agent-output` artifact which contains workflow outputs
   - Parse the artifact to extract issue URLs created by safe-output actions
   - Look for outputs like: `GH_AW_OUTPUT_CREATE_ISSUE_ISSUE_URL` in the run logs
   
4. **Alternative: Use GitHub Issues search** as a fallback:
   - Search for recently created issues with labels matching the worker's output patterns
   - Filter by creation date (within the worker's run time window)
   - Cross-reference with workflow run timestamps to confirm correlation
   
5. **Add discovered issues to the project board**:
   - Use `update-project` safe-output to add issue URLs to the project board
   - Set appropriate status fields (e.g., "Todo", "In Progress", "Done")
   - Preserve any existing project item metadata
   
6. **Generate comprehensive status reports**:
   - Count total issues discovered from worker runs
   - Track open vs closed issues
   - Calculate progress percentage
   - Highlight any issues that need attention
   - Summarize recent worker activity (successful runs, failures, completion times)
   - Report on trends over time

**Important**: Workers operate independently without knowledge of the campaign. The orchestrator discovers their work by monitoring workflow runs, inspecting jobs and logs, parsing artifacts, and extracting issue URLs from workflow outputs. **DO NOT modify worker workflow files or their job configurations** - the orchestrator's role is strictly observational and coordinating, not to interfere with worker execution.

**Understanding Empty Boards**: If you find zero items on the project board, this is normal when a campaign is just starting or when all work has been completed. This is not an error condition - simply report the current state. Worker workflows will create issues as they discover work to be done, and the orchestrator will add them to the board on subsequent runs.

{{if .CompletionGuidance}}
If all issues tracked by the campaign are closed, the campaign is complete. This is a normal terminal state indicating successful completion, not a blocker or error. When the campaign is complete, mark the project as finished and take no further action. Do not report closed issues as blockers.
{{end}}
