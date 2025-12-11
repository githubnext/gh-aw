## Known Patterns (2025-12-11)

- Workflow hygiene push: new “Repository Quality Improvement - Workflow Lifecycle Management & Staleness Detection” (6133) signals attention on pruning/refreshing recurring agents.
- Copilot PR merged report remains shaky: 6040 is still the latest published state (error) and today’s run is in-progress, continuing the fragile streak from prior days.
- Automation remains dominant but softening: github-actions authored 151/178 issues; top labels `plan`/`ai-generated` drop to 84 each (from 101/98) while open issues tick down to 52.
- Schedule jobs show fresh flakiness: Hourly CI Cleaner, Issue Monster, and Issue Triage Agent all failed on Dec 11; multiple Tidy runs cancelled; Dev/Dev Hawk reruns cluster within minutes.
- High-noise success: CLI Version Checker succeeded but emitted 132 errors/36 warnings in one run, keeping log noise elevated despite green status.
