Each time this orchestrator runs on its daily schedule (or when manually dispatched), generate a concise status report for this campaign. Summarize current metrics{{if .ReportBlockers}}, highlight blockers,{{end}} and update any tracker issues using the campaign label.
{{if .CompletionGuidance}}
If all issues with the campaign label are closed, the campaign is complete. This is a normal terminal state indicating successful completion, not a blocker or error. When the campaign is complete, mark the project as finished and take no further action. Do not report closed issues as blockers.
{{end}}
