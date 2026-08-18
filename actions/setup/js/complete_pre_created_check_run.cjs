// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const checkRunId = Number.parseInt(process.env.GH_AW_PRE_CREATED_CHECK_RUN_ID || "", 10);
  if (!Number.isInteger(checkRunId)) {
    core.info("No pre-created pull request check run to complete");
    return;
  }

  const needs = JSON.parse(process.env.GH_AW_NEEDS || "{}");
  const results = Object.values(needs)
    .map(value => value?.result)
    .filter(Boolean);
  const conclusion = results.includes("failure") ? "failure" : results.includes("cancelled") ? "cancelled" : "success";
  const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

  await github.rest.checks.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    check_run_id: checkRunId,
    status: "completed",
    conclusion,
    completed_at: new Date().toISOString(),
    output: {
      title: `${context.workflow} ${conclusion}`,
      summary: `The [workflow run](${runUrl}) completed with conclusion: ${conclusion}.`,
    },
  });
}

module.exports = { main };
