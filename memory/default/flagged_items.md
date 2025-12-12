## Flagged Items for Monitoring (2025-12-12)

- Copilot PR merged stream still fragile: discussion [6040](https://github.com/githubnext/gh-aw/discussions/6040) remains the last erroring state; 6144 just published and today’s run is in-flight — stability unresolved.
- Schedule agents noisy/expensive: Hourly CI Cleaner success consumed 4.9M tokens with 17 errors/22 warnings ([§20169264541](https://github.com/githubnext/gh-aw/actions/runs/20169264541)); Issue Triage Agent recorded a fresh failure ([§20169226915](https://github.com/githubnext/gh-aw/actions/runs/20169226915)); Issue Monster recovered in the latest pass.
- CLI Version Checker still high-noise: recent run ([§20170786087](https://github.com/githubnext/gh-aw/actions/runs/20170786087)) succeeded but emitted 124 errors/30 warnings and drove most spend.
- Backlog composition unchanged: `plan`/`ai-generated` labels dominate (97 each) and automation authorship continues to skew metrics; unlabeled/stale open issues need a refreshed tally.
- New cost-optimization initiative (discussion 6264) suggests imminent policy/tuning changes — monitor for config shifts impacting agent runs.
