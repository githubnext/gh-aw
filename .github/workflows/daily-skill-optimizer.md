---
emoji: "⚡"
name: Daily Skill Optimizer Improvements
description: Runs fastxyz/skill-optimizer daily across all agentic workflows, packages results, and creates one issue with 3 improvements
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

      - name: Validate SkillOpt config and workflow files
        shell: bash
        run: |
          if [ ! -f .skill-optimizer/skill-optimizer.json ]; then
            echo "::error file=.skill-optimizer/skill-optimizer.json::.skill-optimizer/skill-optimizer.json is required by skill-optimizer."
            exit 1
          fi
          if ! find .github/workflows -maxdepth 1 -type f -name "*.md" | grep -q .; then
            echo "::error file=.github/workflows::No workflow .md files found under .github/workflows."
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

          RESULT_DIR="/tmp/gh-aw/agent/skill-optimizer-results"
          TOOL_DIR="$RESULT_DIR/skill-optimizer-src"
          RUNS_DIR="$RESULT_DIR/runs"
          CONFIG_DIR="$RESULT_DIR/configs"
          mkdir -p "$RESULT_DIR" "$RUNS_DIR" "$CONFIG_DIR"

          git clone --depth 1 https://github.com/fastxyz/skill-optimizer "$TOOL_DIR" >"$RESULT_DIR/clone.log" 2>&1

          pushd "$TOOL_DIR" >/dev/null
          npm ci >"$RESULT_DIR/npm-ci.log" 2>&1
          npm run build >"$RESULT_DIR/npm-build.log" 2>&1
          popd >/dev/null

          BASE_CONFIG="$GITHUB_WORKSPACE/.skill-optimizer/skill-optimizer.json"
          if ! jq -e '(.target | type == "object") and (.optimize | type == "object")' "$BASE_CONFIG" >/dev/null; then
            echo "::error file=.skill-optimizer/skill-optimizer.json::Expected .target and .optimize objects in base SkillOpt config."
            exit 1
          fi

          WORKFLOWS_FILE="$RESULT_DIR/workflows.txt"
          find "$GITHUB_WORKSPACE/.github/workflows" -maxdepth 1 -type f -name "*.md" | sort >"$WORKFLOWS_FILE"

          TOTAL_WORKFLOWS=$(wc -l <"$WORKFLOWS_FILE" | tr -d ' ')
          if [ "$TOTAL_WORKFLOWS" -eq 0 ]; then
            echo "::error::No workflow markdown files found to optimize."
            exit 1
          fi

          RUN_MODE="dry-run"
          if [ -n "${OPENROUTER_API_KEY:-}" ]; then
            RUN_MODE="optimize"
          fi

          SUCCESS_COUNT=0
          FAILURE_COUNT=0
          RUN_STATUS=0
          echo "[]" >"$RESULT_DIR/results.json"

          while IFS= read -r workflow_file; do
            rel_workflow="${workflow_file#"$GITHUB_WORKSPACE"/}"
            rel_workflow="${rel_workflow#/}"
            workflow_name="$(basename "$rel_workflow" .md)"
            workflow_slug="$(echo "$workflow_name" | tr -cd '[:alnum:]_-')"
            if [ -z "$workflow_slug" ] || ! [[ "$workflow_slug" =~ ^[[:alnum:]] ]]; then
              workflow_slug="workflow-$(printf '%s' "$workflow_name" | sha256sum | cut -c1-12)"
            fi
            workflow_dir="$RUNS_DIR/$workflow_slug"
            workflow_log="$workflow_dir/run.log"
            workflow_config="$CONFIG_DIR/$workflow_slug.json"
            mkdir -p "$workflow_dir"

            jq --arg skill "$rel_workflow" --arg path "$rel_workflow" \
              '.target.skill = $skill | .optimize.allowedPaths = [$path]' \
              "$BASE_CONFIG" >"$workflow_config"

            set +e
            if [ "$RUN_MODE" = "optimize" ]; then
              node "$TOOL_DIR/dist/cli.js" optimize --config "$workflow_config" >"$workflow_log" 2>&1
            else
              node "$TOOL_DIR/dist/cli.js" run --dry-run --config "$workflow_config" >"$workflow_log" 2>&1
            fi
            status=$?
            set -e

            if [ "$status" -eq 0 ]; then
              SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            else
              FAILURE_COUNT=$((FAILURE_COUNT + 1))
              RUN_STATUS=1
            fi

            jq \
              --arg workflow "$rel_workflow" \
              --arg mode "$RUN_MODE" \
              --argjson status "$status" \
              --arg log_file "${workflow_dir#"$RESULT_DIR"/}/run.log" \
              '. += [{workflow: $workflow, mode: $mode, status: $status, log_file: $log_file}]' \
              "$RESULT_DIR/results.json" >"$RESULT_DIR/results.tmp.json"
            mv "$RESULT_DIR/results.tmp.json" "$RESULT_DIR/results.json"
          done <"$WORKFLOWS_FILE"

          jq -n \
            --arg repository "${GITHUB_REPOSITORY}" \
            --arg run_mode "$RUN_MODE" \
            --argjson run_status "$RUN_STATUS" \
            --argjson total_workflows "$TOTAL_WORKFLOWS" \
            --argjson success_count "$SUCCESS_COUNT" \
            --argjson failure_count "$FAILURE_COUNT" \
            --arg run_url "${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}" \
            '{
              repository: $repository,
              run_mode: $run_mode,
              run_status: $run_status,
              total_workflows: $total_workflows,
              success_count: $success_count,
              failure_count: $failure_count,
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
          path: /tmp/gh-aw/agent/skill-optimizer-results
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
      path: /tmp/gh-aw/agent/skill-optimizer-results

tools:
  cli-proxy: true
  bash:
    - "*"
  edit:

imports:
  - shared/otlp.md
---

# Daily Skill Optimizer Improvements

You are a workflow quality analyst for `${{ github.repository }}`.

## Inputs

- Downloaded artifact directory: `/tmp/gh-aw/agent/skill-optimizer-results`
- Required file: `/tmp/gh-aw/agent/skill-optimizer-results/summary.json`
- Required file: `/tmp/gh-aw/agent/skill-optimizer-results/results.json`
- Required file: `/tmp/gh-aw/agent/skill-optimizer-results/workflows.txt`
- Optional logs:
  - `clone.log`
  - `npm-ci.log`
  - `npm-build.log`
  - `runs/<workflow-id>/run.log` (one per workflow)
  - `configs/<workflow-id>.json` (generated config per workflow)

The separate `skill_optimizer` job already ran `fastxyz/skill-optimizer` across all workflow markdown files in `.github/workflows` and packaged these results.

## Task

1. Read `summary.json`, `results.json`, and relevant logs from the downloaded artifact.
2. Identify exactly **3** actionable improvements for this repository's agentic workflow quality, prioritizing items that impact multiple workflows.
3. Create exactly **one** GitHub issue using `create_issue`.

## Issue Requirements

- Title format: `Daily Skill Optimizer Improvements - YYYY-MM-DD`
- Include:
  - Run mode (`dry-run` or `optimize`) and status from `summary.json`
  - Workflow coverage (`total_workflows`, `success_count`, `failure_count`) from `summary.json`
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
- Run mode: dry-run / optimize
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
