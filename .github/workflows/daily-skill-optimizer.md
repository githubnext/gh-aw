---
name: Daily Skill Optimizer Improvements
description: Runs fastxyz/skill-optimizer daily, packages results as an artifact, and creates one issue with 3 improvements
on:
  schedule:
    - cron: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
tracker-id: daily-skill-optimizer
engine: copilot
strict: true
timeout-minutes: 45

jobs:
  skill_optimizer:
    runs-on: ubuntu-latest
    needs: [activation]
    permissions:
      contents: read
    outputs:
      run_mode: ${{ steps.run_skill_optimizer.outputs.run_mode }}
      run_status: ${{ steps.run_skill_optimizer.outputs.run_status }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v6.0.2
        with:
          persist-credentials: false

      - name: Setup Node.js
        uses: actions/setup-node@v6.4.0
        with:
          node-version: "24"

      - name: Validate SKILL.md exists
        shell: bash
        run: |
          if [ ! -f SKILL.md ]; then
            echo "::error file=SKILL.md::SKILL.md is required by skill-optimizer. See .skill-optimizer/skill-optimizer.json for setup instructions."
            exit 1
          fi

      - name: Stash any uncommitted changes
        shell: bash
        run: |
          git stash --include-untracked || true

      - name: Run skill-optimizer
        id: run_skill_optimizer
        shell: bash
        env:
          OPENROUTER_API_KEY: ${{ secrets.OPENROUTER_API_KEY }}
        run: |
          set -euo pipefail

          RESULT_DIR="/tmp/gh-aw/skill-optimizer-results"
          TOOL_DIR="$RESULT_DIR/skill-optimizer-src"
          mkdir -p "$RESULT_DIR"

          git clone --depth 1 https://github.com/fastxyz/skill-optimizer "$TOOL_DIR" >"$RESULT_DIR/clone.log" 2>&1

          pushd "$TOOL_DIR" >/dev/null
          npm ci >"$RESULT_DIR/npm-ci.log" 2>&1
          npm run build >"$RESULT_DIR/npm-build.log" 2>&1
          popd >/dev/null

          cat >"$RESULT_DIR/answers.json" <<EOF
          {
            "surface": "prompt",
            "repoPath": "${GITHUB_WORKSPACE}",
            "models": [
              "openrouter/anthropic/claude-sonnet-4.6"
            ],
            "maxTasks": 20,
            "maxIterations": 3
          }
          EOF

          set +e
          node "$TOOL_DIR/dist/cli.js" init --answers "$RESULT_DIR/answers.json" >"$RESULT_DIR/init.log" 2>&1
          INIT_STATUS=$?
          set -e

          RUN_MODE="dry-run"
          RUN_STATUS="$INIT_STATUS"

          if [ "$INIT_STATUS" -eq 0 ]; then
            set +e
            node "$TOOL_DIR/dist/cli.js" run --config "$GITHUB_WORKSPACE/.skill-optimizer/skill-optimizer.json" --dry-run >"$RESULT_DIR/run.log" 2>&1
            RUN_STATUS=$?
            set -e

            if [ -n "${OPENROUTER_API_KEY:-}" ]; then
              RUN_MODE="benchmark"
              set +e
              node "$TOOL_DIR/dist/cli.js" run --config "$GITHUB_WORKSPACE/.skill-optimizer/skill-optimizer.json" >"$RESULT_DIR/benchmark.log" 2>&1
              RUN_STATUS=$?
              set -e
            fi
          fi

          jq -n \
            --arg repository "${GITHUB_REPOSITORY}" \
            --arg run_mode "$RUN_MODE" \
            --argjson init_status "$INIT_STATUS" \
            --argjson run_status "$RUN_STATUS" \
            --arg run_url "${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}" \
            '{
              repository: $repository,
              run_mode: $run_mode,
              init_status: $init_status,
              run_status: $run_status,
              run_url: $run_url
            }' >"$RESULT_DIR/summary.json"

          echo "run_mode=$RUN_MODE" >> "$GITHUB_OUTPUT"
          echo "run_status=$RUN_STATUS" >> "$GITHUB_OUTPUT"

      - name: Restore stashed changes
        if: always()
        shell: bash
        run: |
          git stash pop || true

      - name: Upload skill-optimizer artifact
        if: always()
        uses: actions/upload-artifact@v7.0.1
        with:
          name: skill-optimizer-results
          path: /tmp/gh-aw/skill-optimizer-results
          if-no-files-found: error
          retention-days: 7

safe-outputs:
  create-issue:
    title-prefix: "[skill-optimizer] "
    labels: [automation, documentation, prompt-quality]
    max: 1
    expires: 7d

steps:
  - name: Download skill-optimizer artifact
    uses: actions/download-artifact@v8.0.1
    with:
      name: skill-optimizer-results
      path: /tmp/gh-aw/skill-optimizer-results

tools:
  mount-as-clis: true
  bash:
    - "*"
  edit:
features:
  mcp-cli: true
---

# Daily Skill Optimizer Improvements

You are a workflow quality analyst for `${{ github.repository }}`.

## Inputs

- Downloaded artifact directory: `/tmp/gh-aw/skill-optimizer-results`
- Required file: `/tmp/gh-aw/skill-optimizer-results/summary.json`
- Optional logs:
  - `clone.log`
  - `npm-ci.log`
  - `npm-build.log`
  - `init.log`
  - `run.log`
  - `benchmark.log`

The separate `skill_optimizer` job already ran `fastxyz/skill-optimizer` and packaged these results.

## Task

1. Read `summary.json` and relevant logs from the downloaded artifact.
2. Identify exactly **3** actionable improvements for this repository's workflow/skill guidance quality.
3. Create exactly **one** GitHub issue using `create_issue`.

## Issue Requirements

- Title format: `Daily Skill Optimizer Improvements - YYYY-MM-DD`
- Include:
  - Run mode (`dry-run` or `benchmark`) and status from `summary.json`
  - A short evidence section with concrete references to artifact files
  - A numbered list with exactly **3** improvements
  - Expected impact for each improvement
- Keep recommendations specific to this repository and immediately actionable.

## Issue Format Guidelines

Use h3 (`###`) or lower for all headers in your report. Never use h1 (`#`) or h2 (`##`) — these are reserved for the issue title.

Wrap long sections in `<details><summary><b>Section Name</b></summary>` tags to improve readability. Example:

```markdown
<details>
<summary><b>Full Analysis Details</b></summary>

[Long detailed content here...]

</details>
```

Structure the issue body as follows:

```markdown
### Summary
- Run mode: dry-run / benchmark
- Status: ✅/⚠️/❌

### Key Findings
[Always visible — the 3 improvements with expected impact]

<details>
<summary><b>Evidence from Artifact</b></summary>

[Concrete references to artifact files and log excerpts]

</details>

### Recommendations
[Numbered list of 3 actionable improvements]
```

Do not call `noop` for this workflow; always create exactly one issue with exactly 3 improvements.
