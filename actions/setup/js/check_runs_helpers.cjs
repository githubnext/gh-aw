// @ts-check

/**
 * Returns true for check runs that represent deployment environment gates rather
 * than CI checks.
 * @param {any} run
 * @returns {boolean}
 */
function isDeploymentCheck(run) {
  return run?.app?.slug === "github-deployments";
}

/**
 * Select latest check run per name and apply standard filtering.
 * @param {any[]} checkRuns
 * @param {{
 *   includeList?: string[]|null,
 *   excludeList?: string[]|null,
 *   excludedCheckRunIds?: Set<number>,
 * }} [options]
 * @returns {{relevant: any[], deploymentCheckCount: number, currentRunFilterCount: number}}
 */
function selectLatestRelevantChecks(checkRuns, options = {}) {
  const includeList = options.includeList || null;
  const excludeList = options.excludeList || null;
  const excludedCheckRunIds = options.excludedCheckRunIds || new Set();

  /** @type {Map<string, any>} */
  const latestByName = new Map();
  let deploymentCheckCount = 0;
  let currentRunFilterCount = 0;

  for (const run of checkRuns) {
    if (isDeploymentCheck(run)) {
      deploymentCheckCount++;
      continue;
    }
    if (excludedCheckRunIds.has(run.id)) {
      currentRunFilterCount++;
      continue;
    }
    const existing = latestByName.get(run.name);
    if (!existing || new Date(run.started_at ?? 0) > new Date(existing.started_at ?? 0)) {
      latestByName.set(run.name, run);
    }
  }

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

  return { relevant, deploymentCheckCount, currentRunFilterCount };
}

/**
 * Computes failing checks with shared semantics.
 * @param {any[]} checkRuns
 * @param {{allowPending?: boolean}} [options]
 * @returns {any[]}
 */
function getFailingChecks(checkRuns, options = {}) {
  const allowPending = options.allowPending === true;
  const failedConclusions = new Set(["failure", "cancelled", "timed_out"]);
  return checkRuns.filter(run => {
    if (run.status === "completed") {
      return run.conclusion != null && failedConclusions.has(run.conclusion);
    }
    return !allowPending;
  });
}

module.exports = {
  isDeploymentCheck,
  selectLatestRelevantChecks,
  getFailingChecks,
};
