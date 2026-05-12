---
name: Dependabot Campaign
description: Lean campaign that bundles open Dependabot PRs for compiler-generated workflow manifests into one remediation wave
on:
  schedule: daily
  workflow_dispatch:
    inputs:
      objective:
        description: Campaign objective override
        type: string
        required: false
        default: Close open Dependabot PRs for generated workflow manifests by updating source workflow markdown and recompiling.
permissions:
  contents: read
  issues: read
  pull-requests: read
concurrency:
  group: dependabot-campaign
  cancel-in-progress: false
tracker-id: dependabot-campaign
engine:
  id: copilot
  model: gpt-5.4-mini
strict: true
network:
  allowed:
    - defaults
    - node
    - python
    - go
imports:
  - shared/observability-otlp.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [default]
safe-outputs:
  allowed-domains: [default-safe-outputs]
  call-workflow:
    workflows:
      - dependabot-worker
    max: 1
  noop:
timeout-minutes: 15
steps:
  - name: Compute dependabot campaign scoreboard
    uses: actions/github-script@v9
    env:
      CAMPAIGN_OBJECTIVE: ${{ inputs.objective }}
    with:
      script: |
        const fs = require('fs');
        const path = require('path');

        const objective = (process.env.CAMPAIGN_OBJECTIVE || '').trim() || 'Close open Dependabot PRs for generated workflow manifests by updating source workflow markdown and recompiling.';
        const baselinePath = '/tmp/gh-aw/cache-memory/campaigns/dependabot/baseline.json';
        const scoreboardPath = '/tmp/gh-aw/agent/campaigns/dependabot-scoreboard.json';
        const manifestTargets = new Set([
          '.github/workflows/package.json',
          '.github/workflows/package-lock.json',
          '.github/workflows/requirements.txt',
          '.github/workflows/go.mod',
        ]);

        function readJson(filePath, fallback) {
          if (!fs.existsSync(filePath)) {
            return fallback;
          }
          return JSON.parse(fs.readFileSync(filePath, 'utf8'));
        }

        function writeJson(filePath, value) {
          fs.mkdirSync(path.dirname(filePath), { recursive: true });
          fs.writeFileSync(filePath, JSON.stringify(value, null, 2) + '\n', 'utf8');
        }

        function normalizeBaseline(value) {
          if (!value || typeof value !== 'object') {
            return null;
          }
          const openPRCount = Number(value.open_pr_count);
          if (!Number.isFinite(openPRCount)) {
            return null;
          }
          return { open_pr_count: openPRCount };
        }

        function parseBumpTitle(title) {
          const match = String(title || '').match(/^Bump\s+(.+?)\s+from\s+([^\s]+)\s+to\s+([^\s]+)$/i);
          if (!match) {
            return { dependency_name: '', current_version: '', target_version: '' };
          }
          return {
            dependency_name: match[1],
            current_version: match[2],
            target_version: match[3],
          };
        }

        async function listOpenDependabotPRs() {
          const pulls = await github.paginate(github.rest.pulls.list, {
            owner: context.repo.owner,
            repo: context.repo.repo,
            state: 'open',
            per_page: 100,
          });

          const candidates = [];
          for (const pull of pulls) {
            const author = pull.user?.login || '';
            if (author !== 'dependabot[bot]' && author !== 'app/dependabot') {
              continue;
            }

            const files = await github.paginate(github.rest.pulls.listFiles, {
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: pull.number,
              per_page: 100,
            });

            const touchedManifestFiles = files
              .map((file) => file.filename)
              .filter((filename) => manifestTargets.has(filename));

            if (touchedManifestFiles.length === 0) {
              continue;
            }

            const parsed = parseBumpTitle(pull.title);
            candidates.push({
              number: pull.number,
              title: pull.title,
              dependency_name: parsed.dependency_name,
              current_version: parsed.current_version,
              target_version: parsed.target_version,
              manifest_files: touchedManifestFiles,
              created_at: pull.created_at,
              updated_at: pull.updated_at,
              url: pull.html_url,
            });
          }

          return candidates.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
        }

        const openPRs = await listOpenDependabotPRs();

        let baseline = {
          open_pr_count: openPRs.length,
        };
        if (fs.existsSync(baselinePath)) {
          const parsedBaseline = normalizeBaseline(readJson(baselinePath, null));
          if (parsedBaseline) {
            baseline = parsedBaseline;
          } else {
            writeJson(baselinePath, baseline);
          }
        } else {
          writeJson(baselinePath, baseline);
        }

        const baselineCount = Math.max(Number(baseline.open_pr_count ?? openPRs.length), 1);
        const score = Math.round(((baselineCount - openPRs.length) * 1000) / baselineCount) / 10;
        const scoreboard = {
          campaign_id: 'dependabot',
          objective,
          metric: 'open_dependabot_manifest_prs_remaining',
          baseline_open_pr_count: baseline.open_pr_count ?? openPRs.length,
          current_open_pr_count: openPRs.length,
          goal_met: openPRs.length === 0,
          score,
          selected_batch_pr_numbers: openPRs.map((pull) => pull.number),
          selected_batch_dependencies: openPRs.map((pull) => ({
            pr_number: pull.number,
            dependency_name: pull.dependency_name,
            current_version: pull.current_version,
            target_version: pull.target_version,
            manifest_files: pull.manifest_files,
            title: pull.title,
          })),
          selection_reason: openPRs.length > 0 ? 'bundle-all-open-manifest-prs' : 'goal-met',
          open_prs: openPRs.slice(0, 20),
        };

        writeJson(scoreboardPath, scoreboard);
        console.log(JSON.stringify(scoreboard, null, 2));
---

# Dependabot Campaign

You are the Dependabot campaign orchestrator. Your job is to bundle the current in-scope Dependabot backlog into one safe remediation wave.

## Read first

1. Read `/tmp/gh-aw/agent/campaigns/dependabot-scoreboard.json`.

## Operating model

- This campaign has one objective: close open Dependabot PRs that touch generated workflow manifests by updating source workflow markdown and recompiling.
- For this repo, the preferred remediation is to bundle all currently open in-scope Dependabot PRs into one source-of-truth update pass.
- Reuse `dependabot-worker` to execute one bounded remediation wave across the current backlog snapshot.
- Treat `/tmp/gh-aw/agent/campaigns/dependabot-scoreboard.json` as the current deterministic campaign score.

## Behavior

For this campaign:

0. If `goal_met` is true in the scoreboard, summarize that the campaign goal is already met and stop.
1. Read `selected_batch_pr_numbers`, `selected_batch_dependencies`, `selection_reason`, and `open_prs` from the scoreboard.
2. If `selected_batch_pr_numbers` is empty, summarize that no in-scope open Dependabot PRs remain and stop.
3. Call the `dependabot_worker` MCP tool with:
  - `objective`: the objective from the scoreboard
  - `pr-numbers`: the comma-separated contents of `selected_batch_pr_numbers`
  - `dependency-batch-json`: the JSON stringified contents of `selected_batch_dependencies`

## Constraints

- Do not open a PR from the orchestrator. The worker owns code changes and PR creation.
- Do not edit generated manifests in the orchestrator.
- Always mention the deterministic scoreboard value, the number of open PRs in scope, and the selection reason in your final summary.

{{#runtime-import shared/noop-reminder.md}}
