const COPILOT_REVIEWER_BOT = "copilot-pull-request-reviewer[bot]";
async function main() {
  const prNumberStr = process.env.PR_NUMBER;
  if (!prNumberStr || "" === prNumberStr.trim()) return void core.setFailed("PR_NUMBER environment variable is required but not set");
  const prNumber = parseInt(prNumberStr.trim(), 10);
  if (isNaN(prNumber) || prNumber <= 0) core.setFailed(`Invalid PR_NUMBER: ${prNumberStr}. Must be a positive integer.`);
  else {
    core.info(`Adding Copilot as reviewer to PR #${prNumber}`);
    try {
      (await github.rest.pulls.requestReviewers({ owner: context.repo.owner, repo: context.repo.repo, pull_number: prNumber, reviewers: [COPILOT_REVIEWER_BOT] }),
        core.info(`Successfully added Copilot as reviewer to PR #${prNumber}`),
        await core.summary.addRaw(`\n## Copilot Reviewer Added\n\nSuccessfully added Copilot as a reviewer to PR #${prNumber}.\n`).write());
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      (core.error(`Failed to add Copilot as reviewer: ${errorMessage}`), core.setFailed(`Failed to add Copilot as reviewer to PR #${prNumber}: ${errorMessage}`));
    }
  }
}
main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
