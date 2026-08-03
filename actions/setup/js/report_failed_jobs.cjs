// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { fetchAndLogRateLimit } = require("./github_rate_limit_logger.cjs");
const { renderTemplateFromFile, getPromptPath } = require("./messages_core.cjs");
const { generateFooterWithExpiration, createExpirationLine } = require("./ephemerals.cjs");
const { generateXMLMarker } = require("./messages.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");

const GITHUB_API_VERSION = "2022-11-28";
const FAILED_JOBS_ISSUE_EXPIRES_HOURS = 24 * 7; // 1 week

/**
 * Known builtin job names whose failures are already reported
 * by the handle_agent_failure step (the agent job) or are framework-internal
 * (conclusion = current job, pre_activation/activation = reported via agent failure issue flags).
 */
const BUILTIN_REPORTED_JOB_NAMES = new Set(["agent", "conclusion", "activation", "pre_activation", "pre-activation"]);

/**
 * Check whether an error is a GitHub permission error for issues write.
 * @param {unknown} error
 * @returns {boolean}
 */
function isIssueWritePermissionError(error) {
  if (!(error instanceof Error)) return false;
  const msg = error.message.toLowerCase();
  return msg.includes("resource not accessible") || msg.includes("must have push access") || msg.includes("403");
}

/**
 * Check whether an error is a GitHub permission error for actions read.
 * @param {unknown} error
 * @returns {boolean}
 */
function isActionsReadPermissionError(error) {
  if (!(error instanceof Error)) return false;
  const msg = error.message.toLowerCase();
  return msg.includes("resource not accessible") || msg.includes("403");
}

/**
 * Format the list of failed jobs as a markdown bulleted list.
 * @param {Array<{name: string, html_url: string | null}>} jobs
 * @returns {string}
 */
function formatFailedJobsList(jobs) {
  return jobs
    .map(job => {
      if (job.html_url) {
        return `- [\`${job.name}\`](${job.html_url})`;
      }
      return `- \`${job.name}\``;
    })
    .join("\n");
}

/**
 * Query all jobs for the current workflow run and return failed non-builtin jobs.
 * @returns {Promise<Array<{name: string, html_url: string | null}>>}
 */
async function getFailedNonBuiltinJobs() {
  const { owner, repo } = context.repo;
  const runId = context.runId;

  core.info(`Querying jobs for workflow run ${runId}`);

  /** @type {Array<{name: string, html_url: string | null}>} */
  const failedJobs = [];

  let page = 1;
  const perPage = 100;

  while (true) {
    const response = await github.rest.actions.listJobsForWorkflowRun({
      owner,
      repo,
      run_id: runId,
      per_page: perPage,
      page,
      filter: "latest",
    });

    const jobs = response.data.jobs;
    core.info(`Page ${page}: retrieved ${jobs.length} jobs`);

    for (const job of jobs) {
      if (job.conclusion !== "failure") {
        continue;
      }
      if (BUILTIN_REPORTED_JOB_NAMES.has(job.name)) {
        core.info(`Skipping builtin job: ${job.name} (already reported by agent failure handler)`);
        continue;
      }
      core.info(`Found failed non-builtin job: ${job.name}`);
      failedJobs.push({ name: job.name, html_url: job.html_url });
    }

    if (jobs.length < perPage) {
      break;
    }
    page++;
  }

  return failedJobs;
}

/**
 * Report failed non-builtin jobs by creating a GitHub issue.
 * Checks API rate limits before and after querying the jobs list.
 * Respects GH_AW_REPORT_FAILED_JOBS env var (default: true).
 */
async function main() {
  try {
    // Check if reporting is enabled
    const reportFailedJobs = parseBoolTemplatable(process.env.GH_AW_REPORT_FAILED_JOBS, true);
    if (!reportFailedJobs) {
      core.info("Failed jobs reporting is disabled (report-failed-jobs: false), skipping");
      return;
    }

    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runUrl = process.env.GH_AW_RUN_URL || "";
    const { owner, repo } = context.repo;

    // Check rate limit before querying jobs
    core.info("Checking GitHub API rate limit before querying jobs");
    await fetchAndLogRateLimit(github, "report_failed_jobs_before");

    /** @type {Array<{name: string, html_url: string | null}>} */
    let failedJobs;
    try {
      failedJobs = await getFailedNonBuiltinJobs();
    } catch (error) {
      if (isActionsReadPermissionError(error)) {
        core.info(`Skipping failed jobs reporting: token lacks actions:read permission (${getErrorMessage(error)})`);
        return;
      }
      core.warning(`Failed to query jobs for run: ${getErrorMessage(error)}`);
      return;
    } finally {
      // Check rate limit after querying jobs
      core.info("Checking GitHub API rate limit after querying jobs");
      await fetchAndLogRateLimit(github, "report_failed_jobs_after");
    }

    if (failedJobs.length === 0) {
      core.info("No failed non-builtin jobs found, skipping issue creation");
      return;
    }

    core.info(`Found ${failedJobs.length} failed non-builtin job(s): ${failedJobs.map(j => j.name).join(", ")}`);

    // Render the issue body from template
    const failedJobsList = formatFailedJobsList(failedJobs);
    const templatePath = getPromptPath("failed_jobs_issue.md");
    const issueBodyContent = renderTemplateFromFile(templatePath, {
      workflow_name: workflowName,
      workflow_source_url: workflowSourceURL,
      run_url: runUrl,
      failed_jobs_list: failedJobsList,
    });

    const xmlMarker = generateXMLMarker(workflowName, runUrl);
    const issueBody = generateFooterWithExpiration({
      footerText: `${issueBodyContent}\n\n${xmlMarker}`,
      expiresHours: FAILED_JOBS_ISSUE_EXPIRES_HOURS,
    });

    const issueTitle = `[aw] Failed jobs: ${workflowName}`;

    core.info(`Creating failed jobs issue: "${issueTitle}"`);

    try {
      const newIssue = await github.rest.issues.create({
        owner,
        repo,
        title: issueTitle,
        body: issueBody,
        labels: ["agentic-workflows"],
        headers: { "X-GitHub-Api-Version": GITHUB_API_VERSION },
      });

      core.info(`Created failed jobs issue #${newIssue.data.number}: ${newIssue.data.html_url}`);
    } catch (error) {
      if (isIssueWritePermissionError(error)) {
        core.info(`Skipping failed jobs issue creation: token lacks issues:write permission (${getErrorMessage(error)})`);
      } else {
        core.warning(`Failed to create failed jobs issue: ${getErrorMessage(error)}`);
      }
    }
  } catch (error) {
    core.warning(`report_failed_jobs: unexpected error: ${getErrorMessage(error)}`);
  }
}

module.exports = { main, getFailedNonBuiltinJobs, formatFailedJobsList, BUILTIN_REPORTED_JOB_NAMES };
