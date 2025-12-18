Each time this orchestrator runs on its daily schedule (or when manually dispatched), generate a concise status report for this campaign. Summarize current metrics{{if .ReportBlockers}}, highlight blockers,{{end}} and update any tracker issues using the campaign label.

**Understanding Empty Boards**: If you find zero items with the campaign label on the project board, this is normal when a campaign is just starting or when all work has been completed. This is not an error condition - simply report the current state. The associated worker workflows will create tracker issues as they discover work to be done.

{{if .CompletionGuidance}}
If all issues with the campaign label are closed, the campaign is complete. This is a normal terminal state indicating successful completion, not a blocker or error. When the campaign is complete, mark the project as finished and take no further action. Do not report closed issues as blockers.
{{end}}
