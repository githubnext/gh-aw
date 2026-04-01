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

  // Cached results from paginated title searches
  /** @type {Map<string, Object>} */
  const milestoneByTitle = new Map();
  /** @type {Array<Object>} All milestones fetched so far (for error messages) */
  let allFetchedMilestones = [];
  let milestonesExhausted = false;

  /**
   * Find a milestone by title using lazy paginated search with early exit.
   * Results are cached so repeated lookups don't re-paginate.
   * @param {string} title
   * @returns {Promise<Object|null>}
   */
  async function findMilestoneByTitle(title) {
    if (milestoneByTitle.has(title)) {
      return milestoneByTitle.get(title);
    }
    if (milestonesExhausted) {
      return null;
    }
    let found = false;
    await githubClient.paginate(githubClient.rest.issues.listMilestones, { owner: context.repo.owner, repo: context.repo.repo, state: "all", per_page: 100 }, (response, done) => {
      for (const m of response.data) {
        if (!milestoneByTitle.has(m.title)) {
          milestoneByTitle.set(m.title, m);
          allFetchedMilestones.push(m);
        }
        if (m.title === title) {
          found = true;
          done();
          return;
        }
      }
    });
    if (!found) {
      milestonesExhausted = true;
    }
    core.info(`Searched ${allFetchedMilestones.length} milestones (exhausted=${milestonesExhausted})`);
    return milestoneByTitle.get(title) || null;
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
          milestoneByTitle.set(created.data.title, created.data);
          allFetchedMilestones.push(created.data);
          core.info(`Auto-created milestone "${milestoneTitle}" as #${milestoneNumber}`);
        } else {
          const available = formatAvailableMilestones(allFetchedMilestones);
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
        const { data: milestone } = await githubClient.rest.issues.getMilestone({
          owner: context.repo.owner,
          repo: context.repo.repo,
          milestone_number: milestoneNumber,
        });

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
        core.error(`Failed to validate milestone #${milestoneNumber}: ${errorMessage}`);
        return {
          success: false,
          error: `Milestone #${milestoneNumber} not found or failed to validate: ${errorMessage}`,
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
