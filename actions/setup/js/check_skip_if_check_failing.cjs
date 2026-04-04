// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage, isRateLimitError } = require("./error_helpers.cjs");
const { ERR_API } = require("./error_codes.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");

/**
 * Determines the ref to check for CI status.
 * Uses GH_AW_SKIP_BRANCH if set as an explicit override, otherwise delegates
 * to the shared getBaseBranch() helper which handles PR base branch, issue_comment
 * on PR, and repository default branch resolution.
 *
 * @returns {Promise<string>} The ref to use for the check run query
 */
async function resolveRef() {
  const explicitBranch = process.env.GH_AW_SKIP_BRANCH;
  if (explicitBranch) {
    return explicitBranch;
  }
  return getBaseBranch();
}

/**
 * Parses a JSON list from an environment variable.
 *
 * @param {string | undefined} envValue
 * @returns {string[] | null}
 */
function parseListEnv(envValue) {
  if (!envValue) {
    return null;
  }
  try {
    const parsed = JSON.parse(envValue);
    if (!Array.isArray(parsed)) {
      return null;
    }
    // Trim, filter out empty strings, and remove duplicates
    const filtered = [
      ...new Set(
        parsed
          .filter(item => typeof item === "string")
          .map(item => item.trim())
          .filter(item => item !== "")
      ),
    ];
    return filtered.length > 0 ? filtered : null;
  } catch {
    return null;
  }
}

/**
 * Returns true for check runs that represent deployment environment gates rather
 * than CI checks. These should be ignored by default so that a pending deployment
 * approval does not falsely block the agentic workflow.
 *
 * Deployment gate checks are identified by the GitHub App that created them:
 *   - "github-deployments" – the built-in GitHub Deployments service
 *
 * @param {object} run - A check run object from the GitHub API
 * @returns {boolean}
 */
function isDeploymentCheck(run) {
  const slug = run.app?.slug;
  return slug === "github-deployments";
}

/**
 * Fetches the check run IDs for all jobs in the current workflow run.
 * These IDs are used to filter out the current workflow's own checks
 * when evaluating the skip-if-check-failing condition, so that a workflow
 * does not block itself due to its own in-progress jobs.
 *
 * @param {string} owner
 * @param {string} repo
 * @param {string | undefined} runId - The current workflow run ID (GITHUB_RUN_ID)
 * @returns {Promise<Set<number>>} Set of check run IDs belonging to the current run
 */
async function getCurrentRunCheckRunIds(owner, repo, runId) {
  if (!runId) return new Set();
  const numericRunId = parseInt(runId, 10);
  if (isNaN(numericRunId)) return new Set();
  try {
    const jobs = await github.paginate(github.rest.actions.listJobsForWorkflowRun, {
      owner,
      repo,
      run_id: numericRunId,
      per_page: 100,
    });
    const ids = new Set();
    for (const job of jobs) {
      if (typeof job.id === "number") {
        ids.add(job.id);
      }
    }
    return ids;
  } catch (error) {
    core.warning(`Could not fetch jobs for current workflow run (run_id=${numericRunId}): ${getErrorMessage(error)}. Current workflow's checks will not be filtered.`);
    return new Set();
  }
}

async function main() {
  const includeEnv = process.env.GH_AW_SKIP_CHECK_INCLUDE;
  const excludeEnv = process.env.GH_AW_SKIP_CHECK_EXCLUDE;
  const allowPending = process.env.GH_AW_SKIP_CHECK_ALLOW_PENDING === "true";

  const includeList = parseListEnv(includeEnv);
  const excludeList = parseListEnv(excludeEnv);

  const ref = await resolveRef();
  if (!ref) {
    core.setFailed("skip-if-check-failing: could not determine the ref to check.");
    return;
  }

  const { owner, repo } = context.repo;
  core.info(`Checking CI checks on ref: ${ref} (${owner}/${repo})`);

  if (includeList && includeList.length > 0) {
    core.info(`Including only checks: ${includeList.join(", ")}`);
  }
  if (excludeList && excludeList.length > 0) {
    core.info(`Excluding checks: ${excludeList.join(", ")}`);
  }
  if (allowPending) {
    core.info("Pending/in-progress checks will be ignored (allow-pending: true)");
  }

  try {
    // Fetch all check runs for the ref (paginate to handle repos with many checks)
    const checkRuns = await github.paginate(github.rest.checks.listForRef, {
      owner,
      repo,
      ref,
      per_page: 100,
    });

    core.info(`Found ${checkRuns.length} check run(s) on ref "${ref}"`);

    // Fetch check run IDs for the current workflow run so we can exclude them.
    // This prevents a workflow from blocking itself due to its own in-progress jobs
    // appearing as failing checks on the ref.
    const currentRunCheckRunIds = await getCurrentRunCheckRunIds(owner, repo, process.env.GITHUB_RUN_ID);

    // Filter to the latest run per check name (GitHub may have multiple runs per name).
    // Deployment gate checks and the current run's own checks are silently skipped here
    // so they never influence the gate.
    /** @type {Map<string, object>} */
    const latestByName = new Map();
    let deploymentCheckCount = 0;
    let currentRunFilterCount = 0;
    for (const run of checkRuns) {
      if (isDeploymentCheck(run)) {
        deploymentCheckCount++;
        continue;
      }
      if (currentRunCheckRunIds.has(run.id)) {
        currentRunFilterCount++;
        continue;
      }
      const name = run.name;
      const existing = latestByName.get(name);
      if (!existing || new Date(run.started_at ?? 0) > new Date(existing.started_at ?? 0)) {
        latestByName.set(name, run);
      }
    }

    if (deploymentCheckCount > 0) {
      core.info(`Skipping ${deploymentCheckCount} deployment gate check(s) (app: github-deployments)`);
    }
    if (currentRunFilterCount > 0) {
      core.info(`Skipping ${currentRunFilterCount} check run(s) from the current workflow run`);
    }

    // Apply user-defined include/exclude filtering
    const relevant = [];
    for (const [name, run] of latestByName) {
      if (includeList && includeList.length > 0 && !includeList.includes(name)) {
        continue;
      }
      if (excludeList && excludeList.length > 0 && excludeList.includes(name)) {
        continue;
      }
      relevant.push(run);
    }

    core.info(`Evaluating ${relevant.length} check run(s) after filtering`);

    // A check is "failing" if it either:
    //   1. Completed with a non-success conclusion (failure, cancelled, timed_out), OR
    //   2. Is still pending/in-progress — unless allow-pending is set
    const failedConclusions = new Set(["failure", "cancelled", "timed_out"]);

    const failingChecks = relevant.filter(run => {
      if (run.status === "completed") {
        return run.conclusion != null && failedConclusions.has(run.conclusion);
      }
      // Pending/queued/in_progress: treat as failing unless allow-pending is true
      return !allowPending;
    });

    if (failingChecks.length > 0) {
      const names = failingChecks.map(r => (r.status === "completed" ? `${r.name} (${r.conclusion})` : `${r.name} (${r.status})`)).join(", ");
      core.warning(`⚠️ Failing CI checks detected on "${ref}": ${names}. Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_if_check_failing_ok", "false");
      return;
    }

    core.info(`✓ No failing checks found on "${ref}", workflow can proceed`);
    core.setOutput("skip_if_check_failing_ok", "true");
  } catch (error) {
    const errorMsg = getErrorMessage(error);
    // Gracefully handle API rate limit errors (fail-open) to avoid blocking the workflow
    // due to transient GitHub API availability issues. When multiple workflows run
    // simultaneously, they can exhaust the installation API rate limit, causing this
    // check to fail. Failing open matches the behavior of other pre-activation checks.
    if (isRateLimitError(error)) {
      core.warning(`⚠️ API rate limit exceeded while checking CI status for ref "${ref}": ${errorMsg}`);
      core.warning(`Allowing workflow to proceed (fail-open on rate limit)`);
      core.setOutput("skip_if_check_failing_ok", "true");
    } else {
      core.setFailed(`${ERR_API}: Failed to fetch check runs for ref "${ref}": ${errorMsg}`);
    }
  }
}

module.exports = { main };
