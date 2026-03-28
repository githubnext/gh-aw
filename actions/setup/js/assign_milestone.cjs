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

  // Cache milestones to avoid fetching multiple times
  let allMilestones = null;

  /**
   * Fetch all milestones from the repository and cache the result.
   * @returns {Promise<boolean>} True on success, false on failure (result already set).
   */
  async function fetchMilestonesIfNeeded() {
    if (allMilestones !== null) {
      return true;
    }
    try {
      const milestonesResponse = await githubClient.rest.issues.listMilestones({
        owner: context.repo.owner,
        repo: context.repo.repo,
        state: "all",
        per_page: 100,
      });
      allMilestones = milestonesResponse.data;
      core.info(`Fetched ${allMilestones.length} milestones from repository`);
      return true;
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to fetch milestones: ${errorMessage}`);
      return false;
    }
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

    // Fetch milestones when we have an allowed list or need to resolve a title
    const needsMilestoneFetch = allowedMilestones.length > 0 || (!hasMilestoneNumber && milestoneTitle !== null);
    if (needsMilestoneFetch) {
      const fetched = await fetchMilestonesIfNeeded();
      if (!fetched) {
        return {
          success: false,
          error: `Failed to fetch milestones for validation`,
        };
      }
    }

    // Resolve milestone by title if milestone_number is not valid
    if (!hasMilestoneNumber && milestoneTitle !== null) {
      const match = allMilestones ? allMilestones.find(m => m.title === milestoneTitle) : null;
      if (match) {
        milestoneNumber = match.number;
        core.info(`Resolved milestone title "${milestoneTitle}" to #${milestoneNumber}`);
      } else if (autoCreate) {
        // Create the milestone automatically
        try {
          const created = await githubClient.rest.issues.createMilestone({
            owner: context.repo.owner,
            repo: context.repo.repo,
            title: milestoneTitle,
          });
          milestoneNumber = created.data.number;
          if (allMilestones) {
            allMilestones.push(created.data);
          }
          core.info(`Auto-created milestone "${milestoneTitle}" as #${milestoneNumber}`);
        } catch (error) {
          const errorMessage = getErrorMessage(error);
          core.error(`Failed to create milestone "${milestoneTitle}": ${errorMessage}`);
          return {
            success: false,
            error: `Failed to create milestone "${milestoneTitle}": ${errorMessage}`,
          };
        }
      } else {
        const available = formatAvailableMilestones(allMilestones);
        core.warning(`Milestone "${milestoneTitle}" not found in repository. Available: ${available}. Set auto_create: true to create it automatically.`);
        return {
          success: false,
          error: `Milestone "${milestoneTitle}" not found in repository. Set auto_create: true to create it automatically.`,
        };
      }
    }

    // Validate against allowed list if configured
    if (allowedMilestones.length > 0 && allMilestones) {
      const milestone = allMilestones.find(m => m.number === milestoneNumber);

      if (!milestone) {
        const available = formatAvailableMilestonesWithNumbers(allMilestones);
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
