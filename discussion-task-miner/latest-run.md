# Task Mining Run - 2026-07-31

## Summary
- Discussions scanned: 12 new (since discussion #49141)
- Tasks identified: 3
- Issues created: 3
- Duplicates/saturated topics avoided: 2 (Sentrux god-files, ESLint Monster remediation)

## Created Issues
- Fix gh-aw-detection frontmatter/lock.yml drift across 6 workflows (PR Sous Chef, Matt Pocock Skills Reviewer, Design Decision Gate, Impeccable Skills Reviewer, PR Code Quality Reviewer, PR Description Updater) — from discussion #49221
- Widen engine-detection heuristics: 44% of compiled workflows classified as unclassified/other — from discussion #49205
- Investigate and fix daily-code-metrics reporting-cadence gap (23-day scheduling failure) — from discussion #49188

## Top Patterns Observed
- Config drift between markdown frontmatter and compiled `.lock.yml` (detection settings)
- Metrics/classification heuristics falling short as workflow diversity grows
- Scheduling reliability gaps in daily reporting workflows
