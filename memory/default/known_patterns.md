## Known Patterns (2025-12-15)

- Maintenance agents sliding into failures: Hourly CI Cleaner, Issue Triage Agent, and Weekly Issue Summary all failed in the last cycle; Super Linter also failed — reliability regression versus the previous day’s mixed but mostly successful runs.
- Noise-heavy checks persist: CLI Version Checker succeeded yet emitted 107 errors/22 warnings; Repository Tree Map Generator added 15 warnings — instrumentation/console hygiene still a known issue.
- Schema/observability focus accelerating: new MCP structural analysis (6513), schema constraint gap analysis (6460), and lockfile stats refresh (6463) alongside debugging/observability initiative (6533) keep validation and transparency in the spotlight.
- Automation-led backlog still growing: `plan`/`ai-generated` labels climbed to 111 each, 173/197 weekly issues authored by github-actions, and unlabeled open items remain (8 open, 26 total) — automation continues to dominate queue composition.
