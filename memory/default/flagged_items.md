## Flagged Items for Monitoring (2025-12-11)

- Copilot PR merged pipeline still unresolved: latest published run is erroring (discussion [6040](https://github.com/githubnext/gh-aw/discussions/6040)) and today’s run is currently in-progress — stability remains suspect.
- Fresh schedule failures today: Hourly CI Cleaner ([20137659816](https://github.com/githubnext/gh-aw/actions/runs/20137659816)), Issue Monster ([20135923391](https://github.com/githubnext/gh-aw/actions/runs/20135923391)), and Issue Triage Agent ([20135811205](https://github.com/githubnext/gh-aw/actions/runs/20135811205)) all failed; Tidy saw cancellations.
- High-noise success: CLI Version Checker ([20137529278](https://github.com/githubnext/gh-aw/actions/runs/20137529278)) succeeded but produced 132 errors/36 warnings — may mask real regressions.
- Automation skew persists: `plan`/`ai-generated` still lead labels (84 each) and 13 open issues lack labels; 38 open issues are older than 3 days and may need triage/closure.
- Issue volume taper (Dec 8 spike to 36 down to 6 today) could conceal stalled work; watch that open count (52) keeps falling.
