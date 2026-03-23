// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API } = require("./error_codes.cjs");

/**
 * Determines the ref to check for CI status.
 * Uses GH_AW_SKIP_BRANCH if set, otherwise falls back to the PR base branch
 * (for pull_request events) or the current ref.
 *
 * @returns {string} The ref to use for the check run query
 */
function resolveRef() {
  const explicitBranch = process.env.GH_AW_SKIP_BRANCH;
  if (explicitBranch) {
    return explicitBranch;
  }

  // For pull_request events, default to the base (target) branch
  const payload = context.payload;
  if (payload && payload.pull_request && payload.pull_request.base && payload.pull_request.base.ref) {
    return payload.pull_request.base.ref;
  }

  // Fall back to the triggering ref, stripping the "refs/heads/" prefix if present
  const ref = context.ref;
  if (ref && ref.startsWith("refs/heads/")) {
    return ref.slice("refs/heads/".length);
  }
  return ref;
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
    return parsed.filter(item => typeof item === "string");
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

async function main() {
  const includeEnv = process.env.GH_AW_SKIP_CHECK_INCLUDE;
  const excludeEnv = process.env.GH_AW_SKIP_CHECK_EXCLUDE;

  const includeList = parseListEnv(includeEnv);
  const excludeList = parseListEnv(excludeEnv);

  const ref = resolveRef();
  if (!ref) {
    core.setFailed("skip-if-check-failed: could not determine the ref to check.");
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

  try {
    // Fetch all check runs for the ref (paginate to handle repos with many checks)
    const checkRuns = await github.paginate(github.rest.checks.listForRef, {
      owner,
      repo,
      ref,
      per_page: 100,
    });

    core.info(`Found ${checkRuns.length} check run(s) on ref "${ref}"`);

    // Filter to the latest run per check name (GitHub may have multiple runs per name).
    // Deployment gate checks are silently skipped here so they never influence the gate.
    /** @type {Map<string, object>} */
    const latestByName = new Map();
    let deploymentCheckCount = 0;
    for (const run of checkRuns) {
      if (isDeploymentCheck(run)) {
        deploymentCheckCount++;
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

    // A check is considered "failed" if it has completed with a non-success conclusion
    const failedConclusions = new Set(["failure", "cancelled", "timed_out"]);

    const failingChecks = relevant.filter(run => run.status === "completed" && run.conclusion != null && failedConclusions.has(run.conclusion));

    if (failingChecks.length > 0) {
      const names = failingChecks.map(r => `${r.name} (${r.conclusion})`).join(", ");
      core.warning(`⚠️ Failing CI checks detected on "${ref}": ${names}. Workflow execution will be prevented by activation job.`);
      core.setOutput("skip_if_check_failed_ok", "false");
      return;
    }

    core.info(`✓ No failing checks found on "${ref}", workflow can proceed`);
    core.setOutput("skip_if_check_failed_ok", "true");
  } catch (error) {
    core.setFailed(`${ERR_API}: Failed to fetch check runs for ref "${ref}": ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
