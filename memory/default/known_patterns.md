## Known Patterns (2025-12-08)

- Recurring report cadence: daily code metrics, firewall reports, prompt-analysis, documentation checks, and static analyses show a strong automated reporting pipeline with heavy GitHub Actions coverage.
- Failure hotspots: "Daily Copilot PR Merged Report" and "Weekly Issue Summary" workflows failed in the latest 20 runs; Super Linter also failed once, suggesting flaky or data-dependent steps.
- Issue labeling skew: `ai-generated` and `plan` dominate weekly labels (136 and 126 counts), indicating automation-heavy issue creation; automation/code-quality/documentation labels also common.
- Issue flow: 250 issues in last 7 days with 194 closed, 56 open—closure rate is high but daily creation spikes (35 on 2025-12-08) merit monitoring.
- Workflow resource use: last 20 runs consumed ~4.0M tokens (~$1.44) with 183 errors logged, implying some runs retry or emit noisy errors despite successful conclusions.
