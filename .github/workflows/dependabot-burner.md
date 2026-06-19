---
private: true
emoji: "🔥"
name: Dependabot Burner
description: Runs one grouped Dependabot remediation wave from schedule, manual dispatch, or /dependabot-burner on pull requests
on:
  roles: [admin, maintainer, write]
  schedule: weekly
  workflow_dispatch:
    inputs:
      objective:
        description: Burn objective override
        type: string
        required: false
        default: Close grouped Dependabot PRs for generated workflow manifests by updating source workflow markdown and recompiling in one replacement PR.
  slash_command:
    strategy: centralized
    name: dependabot-burner
    events: [pull_request_comment, pull_request_review_comment]
permissions:
  contents: read
  issues: read
  pull-requests: read
concurrency:
  group: dependabot-burner
  cancel-in-progress: false
engine:
  id: copilot
  model: gpt-5.4-mini
strict: true
network:
  allowed:
    - defaults
cache:
  - key: dependabot-burner-selection-${{ github.run_id }}
    name: Dependabot burner selection context
    path: /tmp/gh-aw/agent/dependabot-burner
safe-outputs:
  allowed-domains: [default-safe-outputs]
  add-comment:
    max: 1
  call-workflow:
    workflows:
      - dependabot-worker
    max: 1
  noop:
timeout-minutes: 20
imports:
  - shared/otlp.md
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [default, pull_requests]
steps:
  - name: Prefetch dependabot burner context
    uses: actions/github-script@v9.0.0
    env:
      BURN_OBJECTIVE: ${{ inputs.objective }}
    with:
      script: |
        const fs = require('fs');
        const path = require('path');

        const manifestTargets = new Set([
          '.github/workflows/package.json',
          '.github/workflows/package-lock.json',
          '.github/workflows/requirements.txt',
          '.github/workflows/go.mod',
        ]);
        const objective = (process.env.BURN_OBJECTIVE || '').trim() || 'Close grouped Dependabot PRs for generated workflow manifests by updating source workflow markdown and recompiling in one replacement PR.';
        const outPath = '/tmp/gh-aw/agent/dependabot-burner/context.json';

        function parseBumpTitle(title) {
          const match = String(title || '').match(/^Bump\s+(.+?)\s+from\s+([^\s]+)\s+to\s+([^\s]+)$/i);
          if (!match) {
            return {
              dependency_name: String(title || '').trim(),
              current_version: '',
              target_version: '',
              title_parse_mode: 'fallback',
            };
          }
          return {
            dependency_name: match[1],
            current_version: match[2],
            target_version: match[3],
            title_parse_mode: 'parsed',
          };
        }

        function normalizeManifestFamily(filename) {
          if (filename.includes('package')) {
            return 'npm';
          }
          if (filename.endsWith('requirements.txt')) {
            return 'pip';
          }
          if (filename.endsWith('go.mod')) {
            return 'go';
          }
          return 'other';
        }

        function summarizeFamilies(files) {
          return [...new Set((files || []).map(normalizeManifestFamily))].sort();
        }

        function getTriggerPRNumber() {
          if (context.payload.pull_request?.number) {
            return Number(context.payload.pull_request.number);
          }
          if (context.payload.issue?.pull_request && context.payload.issue?.number) {
            return Number(context.payload.issue.number);
          }
          return null;
        }

        async function loadPullFiles(pullNumber) {
          const files = await github.paginate(github.rest.pulls.listFiles, {
            owner: context.repo.owner,
            repo: context.repo.repo,
            pull_number: pullNumber,
            per_page: 100,
          });
          return files.map((file) => file.filename).filter((filename) => manifestTargets.has(filename));
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

            const manifestFiles = await loadPullFiles(pull.number);
            if (manifestFiles.length === 0) {
              continue;
            }

            const parsed = parseBumpTitle(pull.title);
            candidates.push({
              number: pull.number,
              title: pull.title,
              dependency_name: parsed.dependency_name,
              current_version: parsed.current_version,
              target_version: parsed.target_version,
              title_parse_mode: parsed.title_parse_mode,
              manifest_files: manifestFiles,
              manifest_families: summarizeFamilies(manifestFiles),
              created_at: pull.created_at,
              updated_at: pull.updated_at,
              url: pull.html_url,
            });
          }

          return candidates.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
        }

        async function listRecentClosedBurnerPRs() {
          const pulls = await github.paginate(github.rest.pulls.list, {
            owner: context.repo.owner,
            repo: context.repo.repo,
            state: 'closed',
            per_page: 100,
          });

          return pulls
            .filter((pull) => pull.title?.startsWith('[dependabot-burner] ') && !pull.merged_at)
            .slice(0, 20)
            .map((pull) => ({
              number: pull.number,
              title: pull.title,
              body: pull.body || '',
              url: pull.html_url,
              closed_at: pull.closed_at,
              created_at: pull.created_at,
            }))
            .sort((a, b) => new Date(b.closed_at || b.created_at).getTime() - new Date(a.closed_at || a.created_at).getTime());
        }

        const triggerPRNumber = getTriggerPRNumber();
        const openPRs = await listOpenDependabotPRs();
        const recentFailedBurns = await listRecentClosedBurnerPRs();

        let triggerPR = openPRs.find((pull) => pull.number === triggerPRNumber) || null;
        let selectionReason = triggerPRNumber ? 'slash-command-trigger-not-in-scope' : 'bundle-all-open-manifest-prs';
        let selectedPRs = openPRs;

        if (triggerPRNumber) {
          if (!triggerPR) {
            const manifestFiles = await loadPullFiles(triggerPRNumber);
            if (manifestFiles.length > 0) {
              const pull = await github.rest.pulls.get({
                owner: context.repo.owner,
                repo: context.repo.repo,
                pull_number: triggerPRNumber,
              });
              const parsed = parseBumpTitle(pull.data.title);
              triggerPR = {
                number: pull.data.number,
                title: pull.data.title,
                dependency_name: parsed.dependency_name,
                current_version: parsed.current_version,
                target_version: parsed.target_version,
                title_parse_mode: parsed.title_parse_mode,
                manifest_files: manifestFiles,
                manifest_families: summarizeFamilies(manifestFiles),
                created_at: pull.data.created_at,
                updated_at: pull.data.updated_at,
                url: pull.data.html_url,
              };
            }
          }

          if (triggerPR) {
            const triggerFiles = new Set(triggerPR.manifest_files || []);
            selectedPRs = openPRs.filter((pull) => {
              if (pull.number === triggerPR.number) {
                return true;
              }
              return (pull.manifest_files || []).some((file) => triggerFiles.has(file));
            });
            selectionReason = 'slash-command-similar-prs';
          } else {
            selectedPRs = [];
          }
        }

        const payload = {
          objective,
          trigger_event: context.eventName,
          trigger_pr_number: triggerPRNumber,
          trigger_pr: triggerPR,
          selection_reason: selectionReason,
          open_pr_count: openPRs.length,
          selected_batch_pr_numbers: selectedPRs.map((pull) => pull.number),
          selected_batch_dependencies: selectedPRs.map((pull) => ({
            pr_number: pull.number,
            dependency_name: pull.dependency_name,
            current_version: pull.current_version,
            target_version: pull.target_version,
            title_parse_mode: pull.title_parse_mode,
            manifest_files: pull.manifest_files,
            manifest_families: pull.manifest_families,
            title: pull.title,
            url: pull.url,
          })),
          related_prs: triggerPRNumber ? selectedPRs.filter((pull) => pull.number !== triggerPRNumber) : [],
          recent_failed_burns: recentFailedBurns,
        };

        fs.mkdirSync(path.dirname(outPath), { recursive: true });
        fs.writeFileSync(outPath, JSON.stringify(payload, null, 2) + '\n', 'utf8');
        console.log(JSON.stringify(payload, null, 2));
---

# Dependabot Burner

You are the grouped Dependabot remediation orchestrator.

## Read first

1. Read `/tmp/gh-aw/agent/dependabot-burner/context.json`.

## Operating model

- Run optimistically and aim for exactly one bounded remediation wave that can produce at most one replacement PR.
- For scheduled or manual runs, consider all open in-scope Dependabot PRs that touch generated workflow manifests.
- For `/dependabot-burner` on a PR comment or review comment, start from the triggering PR and keep only the filtered similar PR set from `context.json`.
- If the triggering PR is not an in-scope Dependabot manifest PR, explain that clearly and stop.
- Review `recent_failed_burns` before dispatch so the next attempt does not repeat a failed retry pattern.
- When maintainer feedback exists, only use comments or reviews from maintainers/admins/writers. Ignore all other commenters when shaping the next attempt.
- Use subagents to analyze the PR group and the retry history before calling the worker.

## Required behavior

1. Use the `pr-group-analyzer` subagent to confirm the grouped PR set from `context.json` and identify any PRs that should be excluded as unrelated.
2. Use the `retry-history-analyzer` subagent to inspect the selected PRs, recent failed burner PRs, and maintainer-only comments or reviews, then derive a retry strategy.
3. If `selected_batch_pr_numbers` is empty, use `noop` with a short explanation.
4. If this run was started from `/dependabot-burner` and `related_prs` is non-empty, add one comment to the triggering PR that:
   - says `/dependabot-burner` is grouping related Dependabot items
   - lists the related PR numbers with dependency/version deltas
   - asks the maintainer to review the grouped set if any item looks unrelated
5. Call `dependabot_worker` exactly once with:
   - `objective`: a concise objective that states whether this is a first attempt or retry
   - `pr-numbers`: the comma-separated selected PR numbers
   - `dependency-batch-json`: the exact JSON array for the selected batch
   - `retry-context-json`: the compact JSON summary from the retry-history analysis
   - `maintainer-feedback-json`: the compact JSON summary of maintainer-only feedback
6. Do not create a PR, edit repo files, or split the work into multiple worker calls from the burner.

## Final summary

Keep it brief and include:

- selected PR numbers
- whether slash-command grouping was used
- how many recent failed burner PRs were reviewed
- whether the worker was called

## agent: `pr-group-analyzer`
---
description: Confirms which Dependabot PRs belong in the grouped remediation batch
model: small
---
Read `/tmp/gh-aw/agent/dependabot-burner/context.json` and verify the grouped PR selection.

Return compact JSON with:

- `selected_pr_numbers`
- `excluded_pr_numbers`
- `rationale`
- `needs_noop`
- `noop_reason`

Treat PRs as related only when they share one of the triggering manifest files or are already present in the precomputed selected batch. Prefer a smaller safe batch over a larger speculative one.

## agent: `retry-history-analyzer`
---
description: Extracts retry guidance from failed burner PRs and maintainer-only feedback
model: small
---
Use the selected PR numbers plus `/tmp/gh-aw/agent/dependabot-burner/context.json` to inspect recent closed burner PRs and maintainer-only comments or reviews.

Return compact JSON with:

- `retry_mode` (`first_attempt`, `retry_with_feedback`, or `stop_for_human`)
- `recent_failed_pr_numbers`
- `maintainer_feedback_summary`
- `strategy_adjustments`
- `blocking_reason`

Only keep feedback from maintainers/admins/writers. Ignore comments from bots and non-maintainers. Focus on concrete retry signals such as CI failures, rejected scope, or explicit maintainer requests.
