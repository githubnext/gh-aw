---
private: true
emoji: "🧽"
name: Daily Copilot Branch Cleaner
description: Deletes stale branches whose names start with copilot/ when their latest commit is older than 24 hours
on:
  schedule:
    - cron: "17 3 * * *"
  workflow_dispatch:
permissions:
  contents: read
jobs:
  cleanup_copilot_branches:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - name: Delete stale copilot branches
        uses: actions/github-script@3a2844b7e9c422d3c10d287c895573f7108da1b3 # v9.0.0
        with:
          script: |
            const cutoffMs = 24 * 60 * 60 * 1000;
            const now = Date.now();
            const owner = context.repo.owner;
            const repo = context.repo.repo;
            const refs = await github.paginate(github.rest.git.listMatchingRefs, {
              owner,
              repo,
              ref: 'heads/copilot/',
            });

            const results = [];
            const failures = [];

            if (refs.length === 0) {
              core.notice('No branches matching copilot/ were found.');
              await core.summary
                .addHeading('Daily Copilot Branch Cleaner', 2)
                .addRaw('No branches matching `copilot/` were found.')
                .write();
              return;
            }

            for (const ref of refs) {
              const branch = ref.ref.replace('refs/heads/', '');
              const sha = ref.object?.sha;

              if (!sha) {
                results.push({ branch, action: 'skipped', reason: 'missing head sha' });
                continue;
              }

              const { data: commit } = await github.rest.repos.getCommit({
                owner,
                repo,
                ref: sha,
              });

              const timestamps = [commit.commit?.author?.date, commit.commit?.committer?.date]
                .filter(Boolean)
                .map((value) => Date.parse(value))
                .filter(Number.isFinite);

              if (timestamps.length === 0) {
                results.push({ branch, action: 'skipped', reason: 'missing commit timestamp' });
                continue;
              }

              const lastActivityMs = Math.max(...timestamps);
              const ageHours = (now - lastActivityMs) / (60 * 60 * 1000);

              if (ageHours < 24) {
                results.push({
                  branch,
                  action: 'kept',
                  reason: `active ${ageHours.toFixed(1)}h ago`,
                });
                continue;
              }

              try {
                await github.rest.git.deleteRef({
                  owner,
                  repo,
                  ref: `heads/${branch}`,
                });
                results.push({
                  branch,
                  action: 'deleted',
                  reason: `inactive for ${ageHours.toFixed(1)}h`,
                });
              } catch (error) {
                const message = error instanceof Error ? error.message : String(error);
                failures.push(`${branch}: ${message}`);
                results.push({
                  branch,
                  action: 'failed',
                  reason: message,
                });
              }
            }

            const deleted = results.filter((result) => result.action === 'deleted').length;
            const kept = results.filter((result) => result.action === 'kept').length;
            const skipped = results.filter((result) => result.action === 'skipped').length;
            const failed = results.filter((result) => result.action === 'failed').length;

            await core.summary
              .addHeading('Daily Copilot Branch Cleaner', 2)
              .addRaw(`Deleted ${deleted} branch(es); kept ${kept}; skipped ${skipped}; failed ${failed}.`)
              .addTable([
                [
                  { data: 'Branch', header: true },
                  { data: 'Action', header: true },
                  { data: 'Reason', header: true },
                ],
                ...results.map((result) => [result.branch, result.action, result.reason]),
              ])
              .write();

            if (failures.length > 0) {
              core.setFailed(`Failed to process ${failures.length} branch(es): ${failures.join('; ')}`);
            }
---

# Daily Copilot Branch Cleaner

Runs once per day and deletes branches whose names start with `copilot/` when the head commit is older than 24 hours.
