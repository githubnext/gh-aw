// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { loadTemporaryIdMapFromResolved, resolveRepoIssueTarget } = require("./temporary_id.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "assign_milestone";

/**
 * Formats milestones as a human-readable list of titles (e.g., '"v1.0", "v2.0"').
 * @param {Array<{title: string}>|null} milestones
 * @returns {string}
 */
function formatAvailableMilestones(milestones) {
  if (!milestones || milestones.length === 0) return "none";
  return milestones.map(m => `"${m.title}"`).join(", ");
}

/**
 * Formats milestones as a human-readable list with numbers (e.g., '"v1.0" (#5), "v2.0" (#6)').
 * @param {Array<{title: string, number: number}>|null} milestones
 * @returns {string}
 */
function formatAvailableMilestonesWithNumbers(milestones) {
  if (!milestones || milestones.length === 0) return "none";
  return milestones.map(m => `"${m.title}" (#${m.number})`).join(", ");
}

/**
 * Main handler factory for assign_milestone
 * Returns a message handler function that processes individual assign_milestone messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedMilestones = config.allowed || [];
  const maxCount = config.max || 10;
  const autoCreate = config.auto_create === true;
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

  core.info(`Assign milestone configuration: max=${maxCount}, auto_create=${autoCreate}`);
  if (allowedMilestones.length > 0) {
    core.info(`Allowed milestones: ${allowedMilestones.join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Per-handler milestone cache shared between title and number lookups.
  // Populated incrementally as pages are fetched; avoids re-requesting the same pages.
  const milestoneCache = {
    /** @type {Map<string, Object>} */
    byTitle: new Map(),
    /** @type {Map<number, Object>} */
    byNumber: new Map(),
    /** All milestone objects fetched so far (in page order). */
    allFetched: /** @type {Array} */ [],
    /** True once pagination has been fully exhausted (no more pages). */
    exhausted: false,
  };

  /**
   * Populate milestoneCache with one page of results and signal early exit when target is found.
   * @param {Array} pageData - Response data from one page of listMilestones.
   * @param {function():void} done - Octokit paginate early-exit callback.
   * @param {function(Object):boolean} predicate - Returns true when the desired milestone is found.
   */
  function processMilestonePage(pageData, done, predicate) {
    for (const m of pageData) {
      if (!milestoneCache.byTitle.has(m.title)) {
        milestoneCache.byTitle.set(m.title, m);
        milestoneCache.byNumber.set(m.number, m);
        milestoneCache.allFetched.push(m);
      }
    }
    if (pageData.some(predicate)) {
      done();
    }
  }

  /**
   * Fetch milestones from the API, paginating with early exit via `predicate`.
   * Shared by findMilestoneByTitle and findMilestoneByNumber.
   * @param {function(Object):boolean} predicate
   * @returns {Promise<void>}
   */
  async function paginateMilestonesUntil(predicate) {
    if (milestoneCache.exhausted) {
      return;
    }
    let earlyExit = false;
    await githubClient.paginate(
      githubClient.rest.issues.listMilestones,
      {
        owner: context.repo.owner,
        repo: context.repo.repo,
        state: "all",
        per_page: 100,
      },
      (response, done) => {
        processMilestonePage(
          response.data,
          () => {
            earlyExit = true;
            done();
          },
          predicate
        );
      }
    );
    if (!earlyExit) {
      milestoneCache.exhausted = true;
    }
    core.info(`Fetched ${milestoneCache.allFetched.length} milestones so far (exhausted=${milestoneCache.exhausted})`);
  }

  /**
   * Find a milestone by title, stopping pagination as soon as it is found.
   * @param {string} title
   * @returns {Promise<Object|null>}
   */
  async function findMilestoneByTitle(title) {
    if (milestoneCache.byTitle.has(title)) {
      return milestoneCache.byTitle.get(title);
    }
    await paginateMilestonesUntil(m => m.title === title);
    return milestoneCache.byTitle.get(title) || null;
  }

  /**
   * Find a milestone by number, stopping pagination as soon as it is found.
   * @param {number} number
   * @returns {Promise<Object|null>}
   */
  async function findMilestoneByNumber(number) {
    if (milestoneCache.byNumber.has(number)) {
      return milestoneCache.byNumber.get(number);
    }
    await paginateMilestonesUntil(m => m.number === number);
    return milestoneCache.byNumber.get(number) || null;
  }

  /**
   * Message handler function that processes a single assign_milestone message
   * @param {Object} message - The assign_milestone message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleAssignMilestone(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping assign_milestone: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const item = message;

    // Convert resolvedTemporaryIds to a normalized Map for resolveRepoIssueTarget
    const temporaryIdMap = loadTemporaryIdMapFromResolved(resolvedTemporaryIds);

    // Resolve issue_number, which may be a temporary ID (e.g. "aw_abc123") or a plain number
    const resolvedIssueTarget = resolveRepoIssueTarget(item.issue_number, temporaryIdMap, context.repo.owner, context.repo.repo);

    // If the issue_number is a temporary ID that hasn't been resolved yet, defer processing
    if (resolvedIssueTarget.wasTemporaryId && !resolvedIssueTarget.resolved) {
      core.info(`Deferring assign_milestone: unresolved temporary ID (${item.issue_number})`);
      return {
        success: false,
        deferred: true,
        error: resolvedIssueTarget.errorMessage || `Unresolved temporary ID: ${item.issue_number}`,
      };
    }

    if (resolvedIssueTarget.errorMessage || !resolvedIssueTarget.resolved) {
      core.error(`Invalid issue_number: ${item.issue_number}`);
      return {
        success: false,
        error: `Invalid issue_number: ${item.issue_number}`,
      };
    }

    const issueNumber = resolvedIssueTarget.resolved.number;
    if (resolvedIssueTarget.wasTemporaryId) {
      core.info(`Resolved temporary ID '${item.issue_number}' to issue #${issueNumber}`);
    }

    let milestoneNumber = Number(item.milestone_number);
    const milestoneTitle = item.milestone_title || null;
    const hasMilestoneNumber = !isNaN(milestoneNumber) && milestoneNumber > 0;

    // Validate that at least one of milestone_number or milestone_title is provided
    if (!hasMilestoneNumber && !milestoneTitle) {
      const msg = "Either milestone_number or milestone_title must be provided";
      core.error(msg);
      return {
        success: false,
        error: msg,
      };
    }

    // Resolve milestone by title if milestone_number is not valid
    if (!hasMilestoneNumber && milestoneTitle !== null) {
      try {
        const match = await findMilestoneByTitle(milestoneTitle);
        if (match) {
          milestoneNumber = match.number;
          core.info(`Resolved milestone title "${milestoneTitle}" to #${milestoneNumber}`);
        } else if (autoCreate) {
          // Create the milestone automatically
          const created = await githubClient.rest.issues.createMilestone({
            owner: context.repo.owner,
            repo: context.repo.repo,
            title: milestoneTitle,
          });
          milestoneNumber = created.data.number;
          milestoneCache.byTitle.set(created.data.title, created.data);
          milestoneCache.byNumber.set(created.data.number, created.data);
          milestoneCache.allFetched.push(created.data);
          core.info(`Auto-created milestone "${milestoneTitle}" as #${milestoneNumber}`);
        } else {
          const available = formatAvailableMilestones(milestoneCache.allFetched);
          core.warning(`Milestone "${milestoneTitle}" not found in repository. Available: ${available}. Set auto_create: true to create it automatically.`);
          return {
            success: false,
            error: `Milestone "${milestoneTitle}" not found in repository. Set auto_create: true to create it automatically.`,
          };
        }
      } catch (error) {
        const errorMessage = getErrorMessage(error);
        core.error(`Failed to resolve milestone "${milestoneTitle}": ${errorMessage}`);
        return {
          success: false,
          error: `Failed to resolve milestone "${milestoneTitle}": ${errorMessage}`,
        };
      }
    }

    // Validate against allowed list if configured
    if (allowedMilestones.length > 0) {
      try {
        const milestone = await findMilestoneByNumber(milestoneNumber);

        if (!milestone) {
          const available = formatAvailableMilestonesWithNumbers(milestoneCache.allFetched);
          core.warning(`Milestone #${milestoneNumber} not found in repository. Available milestones: ${available}`);
          return {
            success: false,
            error: `Milestone #${milestoneNumber} not found in repository`,
          };
        }

        const isAllowed = allowedMilestones.includes(milestone.title) || allowedMilestones.includes(String(milestoneNumber));

        if (!isAllowed) {
          core.warning(`Milestone "${milestone.title}" (#${milestoneNumber}) is not in the allowed list`);
          return {
            success: false,
            error: `Milestone "${milestone.title}" (#${milestoneNumber}) is not in the allowed list`,
          };
        }
      } catch (error) {
        const errorMessage = getErrorMessage(error);
        core.error(`Failed to validate milestone: ${errorMessage}`);
        return {
          success: false,
          error: `Failed to fetch milestones for validation: ${errorMessage}`,
        };
      }
    }

    // Assign the milestone to the issue
    try {
      // If in staged mode, preview without executing
      if (isStaged) {
        logStagedPreviewInfo(`Would assign milestone #${milestoneNumber} to issue #${issueNumber}`);
        return {
          success: true,
          staged: true,
          previewInfo: {
            issue_number: issueNumber,
            milestone_number: milestoneNumber,
          },
        };
      }

      await githubClient.rest.issues.update({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issueNumber,
        milestone: milestoneNumber,
      });

      core.info(`Successfully assigned milestone #${milestoneNumber} to issue #${issueNumber}`);
      return {
        success: true,
        issue_number: issueNumber,
        milestone_number: milestoneNumber,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to assign milestone #${milestoneNumber} to issue #${issueNumber}: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main };
