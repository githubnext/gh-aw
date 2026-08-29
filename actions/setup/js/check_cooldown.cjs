// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { fetchAndLogRateLimit } = require("./github_rate_limit_logger.cjs");

function resolveWorkflowId() {
  const workflowRef = process.env.GITHUB_WORKFLOW_REF ?? "";
  const workflowRefMatch = workflowRef.match(/\.github\/workflows\/([^@]+)/);
  return workflowRefMatch?.[1] ?? context.workflow;
}

async function main() {
  const {
    repo: { owner, repo },
    runId,
  } = context;
  const cooldownSeconds = Number(process.env.GH_AW_COOLDOWN_SECONDS);
  if (!Number.isFinite(cooldownSeconds) || !Number.isSafeInteger(cooldownSeconds) || cooldownSeconds < 300) {
    throw new Error("Workflow cooldown must be an integer of at least 300 seconds");
  }

  const workflowId = resolveWorkflowId();
  const threshold = Date.now() - cooldownSeconds * 1000;

  try {
    await fetchAndLogRateLimit(github, "check_cooldown_start");
    core.info(`Checking ${cooldownSeconds}-second cooldown for workflow '${workflowId}'`);

    let page = 1;
    const perPage = 100;

    while (true) {
      const response = await github.rest.actions.listWorkflowRuns({
        owner,
        repo,
        workflow_id: workflowId,
        status: "completed",
        per_page: perPage,
        page,
      });
      const runs = response.data.workflow_runs;

      for (const run of runs) {
        if (run.id === runId) {
          continue;
        }

        const completedAt = new Date(run.updated_at ?? "");
        if (Number.isNaN(completedAt.getTime())) {
          core.warning(`Skipping run ${run.id} with an invalid completion time`);
          continue;
        }
        const jobs = await github.paginate(github.rest.actions.listJobsForWorkflowRun, {
          owner,
          repo,
          run_id: run.id,
          filter: "latest",
          per_page: 100,
        });
        const agentExecuted = jobs.some(job => job.name === "agent" && job.conclusion !== "skipped" && job.started_at);
        if (!agentExecuted) {
          continue;
        }

        if (completedAt.getTime() <= threshold) {
          core.info(`Cooldown passed since agent run ${run.id} completed`);
          core.setOutput("cooldown_ok", "true");
          return;
        }

        const remainingSeconds = Math.max(0, Math.ceil((completedAt.getTime() + cooldownSeconds * 1000 - Date.now()) / 1000));
        core.warning(`Skipping agent execution because run ${run.id} completed within the cooldown period (${remainingSeconds} seconds remaining)`);
        core.setOutput("cooldown_ok", "false");
        return;
      }

      if (runs.length < perPage) {
        break;
      }
      page++;
    }

    core.info("Cooldown passed; no recent completed run executed the agent job");
    core.setOutput("cooldown_ok", "true");
  } catch (error) {
    core.warning(`Cooldown check failed: ${getErrorMessage(error)}`);
    core.warning("Allowing agent execution because workflow run history could not be checked");
    core.setOutput("cooldown_ok", "true");
  }
}

module.exports = { main };
