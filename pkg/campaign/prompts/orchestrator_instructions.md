Each time this orchestrator runs on its daily schedule (or when manually dispatched), generate a concise status report for this campaign. Summarize current metrics{{if .ReportBlockers}}, highlight blockers,{{end}} and monitor the progress of associated worker workflows.

**Tracking Worker Output**: The campaign has knowledge of its workers (listed in the `workflows` field) and monitors their output directly. For each worker workflow:

1. Query GitHub Actions API for recent workflow runs using the worker's `tracker-id` (e.g., `daily-file-diet`)
2. Examine workflow run outputs and artifacts to discover issues created by the worker
3. Ensure all issues created by campaign workers are added to the project board (if not already present)
4. Update project board item statuses and fields as needed to reflect current state
5. Report on overall campaign progress based on worker outputs and issue counts

Workers operate independently without knowledge of the campaign. The orchestrator discovers their work by monitoring workflow runs and outputs.

**Understanding Empty Boards**: If you find zero items on the project board, this is normal when a campaign is just starting or when all work has been completed. This is not an error condition - simply report the current state. Worker workflows will create issues as they discover work to be done, and the orchestrator will add them to the board on subsequent runs.

{{if .CompletionGuidance}}
If all issues tracked by the campaign are closed, the campaign is complete. This is a normal terminal state indicating successful completion, not a blocker or error. When the campaign is complete, mark the project as finished and take no further action. Do not report closed issues as blockers.
{{end}}
