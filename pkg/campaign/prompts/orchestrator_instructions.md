Each time this orchestrator runs on its daily schedule (or when manually dispatched), generate a concise status report for this campaign. Summarize current metrics{{if .ReportBlockers}}, highlight blockers,{{end}} and monitor the progress of associated worker workflows.

**Tracking Worker Output**: The campaign has knowledge of its workers (listed in the `workflows` field) and monitors their output via tracker-id. Each worker workflow has a unique tracker-id that gets embedded in all created assets. The orchestrator discovers issues created by workers by searching for this tracker-id:

1. **Search for issues created by workers** using the GitHub MCP server:
   - For each worker in the `workflows` list, determine its tracker-id (e.g., `daily-file-diet`)
   - Search for issues in the repository containing the tracker-id HTML comment: `<!-- tracker-id: WORKER_TRACKER_ID -->`
   - Use `github-search_issues` to find issues containing the tracker-id in their body
   - Example query: `repo:owner/repo "<!-- tracker-id: daily-file-diet -->" in:body`
   
2. **Filter discovered issues**:
   - Focus on recently created or updated issues (within the last week)
   - Check if issues are already on the project board to avoid duplicates
   - Identify new issues that need to be added to the board
   
3. **Add discovered issues to the project board**:
   - Use `update-project` safe-output to add issue URLs to the project board
   - Set appropriate status fields (e.g., "Todo", "In Progress", "Done")
   - Preserve any existing project item metadata
   
4. **Update project board status**:
   - For issues already on the board, check their current state (open/closed)
   - Update project board fields to reflect current status
   - Move items between columns as appropriate (e.g., closed issues to "Done")
   
5. **Report on campaign progress**:
   - Count total issues discovered via tracker-id
   - Track open vs closed issues
   - Calculate progress percentage
   - Highlight any issues that need attention

Workers operate independently without knowledge of the campaign. The orchestrator discovers their work by searching for issues containing the worker's tracker-id, which is automatically embedded in all created assets.

**Understanding Empty Boards**: If you find zero items on the project board, this is normal when a campaign is just starting or when all work has been completed. This is not an error condition - simply report the current state. Worker workflows will create issues as they discover work to be done, and the orchestrator will add them to the board on subsequent runs.

{{if .CompletionGuidance}}
If all issues tracked by the campaign are closed, the campaign is complete. This is a normal terminal state indicating successful completion, not a blocker or error. When the campaign is complete, mark the project as finished and take no further action. Do not report closed issues as blockers.
{{end}}
