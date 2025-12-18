{{if .ProjectURL}}
## Project Board Updates

The campaign uses a GitHub Project board for tracking: {{.ProjectURL}}

**Orchestrator's Role**: You can update existing project items to reflect current campaign status, but **do NOT add new items to the board** - that is the responsibility of worker workflows. When worker workflows create issues, they will add them to the project board.

If you need to update project metadata or status fields for campaign coordination, use the `update-project` safe output with this exact URL: {{.ProjectURL}}

**Important**: If the project board is empty or has no items yet, this is normal - worker workflows will populate it as they create work items.
{{end}}
