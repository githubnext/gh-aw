Each time this orchestrator runs on its daily schedule (or when manually dispatched), generate a concise status report for this campaign. Summarize current metrics{{if .ReportBlockers}}, highlight blockers,{{end}} and update any tracker issues using the campaign label.

**Tracking Worker Output**: The orchestrator tracks work created by worker workflows by querying for issues with the campaign label (e.g., `campaign:go-file-size-reduction`). Worker workflows create issues but do not manage the project board - that is the orchestrator's responsibility. The orchestrator should:

1. Query GitHub for all issues (open and closed) with the campaign label to understand current progress
2. Ensure all campaign-labeled issues are added to the project board (if not already present)
3. Update project board item statuses and fields as needed to reflect current state
4. Report on overall campaign progress based on issue counts and status

**Understanding Empty Boards**: If you find zero items with the campaign label on the project board, this is normal when a campaign is just starting or when all work has been completed. This is not an error condition - simply report the current state. The associated worker workflows will create tracker issues as they discover work to be done, and the orchestrator will add them to the board on subsequent runs.

{{if .CompletionGuidance}}
If all issues with the campaign label are closed, the campaign is complete. This is a normal terminal state indicating successful completion, not a blocker or error. When the campaign is complete, mark the project as finished and take no further action. Do not report closed issues as blockers.
{{end}}
