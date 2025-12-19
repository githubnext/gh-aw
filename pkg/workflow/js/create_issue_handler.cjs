// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { generateTemporaryId, isTemporaryId, normalizeTemporaryId, replaceTemporaryIdReferences, serializeTemporaryIdMap } = require("./temporary_id.cjs");
const { parseAllowedRepos, getDefaultTargetRepo, validateRepo, parseRepoSlug } = require("./repo_helpers.cjs");
const { addExpirationComment } = require("./expiration_helpers.cjs");
const { removeDuplicateTitleFromDescription } = require("./remove_duplicate_title.cjs");

/**
 * Create issue handler - registers with the handler manager
 * @param {any} item - The create_issue item from agent output
 * @param {import("./safe_output_handler_manager.cjs").HandlerContext} context - Handler context
 * @returns {Promise<import("./safe_output_handler_manager.cjs").HandlerResult>}
 */
async function handleCreateIssue(item, context) {
  const { core, github, exec } = context;
  const { context: ghContext } = context;

  // Map to track temporary_id -> {repo, number} relationships for this handler
  /** @type {Map<string, {repo: string, number: number}>} */
  const localTemporaryIdMap = new Map();

  try {
    // Parse allowed repos and default target
    const allowedRepos = parseAllowedRepos();
    const defaultTargetRepo = getDefaultTargetRepo();

    // Determine target repository for this issue
    const itemRepo = item.repo ? String(item.repo).trim() : defaultTargetRepo;

    // Validate the repository is allowed
    const repoValidation = validateRepo(itemRepo, defaultTargetRepo, allowedRepos);
    if (!repoValidation.valid) {
      core.warning(`Skipping issue: ${repoValidation.error}`);
      return { success: true }; // Not a failure, just skipped
    }

    // Parse the repository slug
    const repoParts = parseRepoSlug(itemRepo);
    if (!repoParts) {
      core.warning(`Skipping issue: Invalid repository format '${itemRepo}'. Expected 'owner/repo'.`);
      return { success: true }; // Not a failure, just skipped
    }

    // Get or generate the temporary ID for this issue
    const temporaryId = item.temporary_id || generateTemporaryId();
    core.info(`Processing create-issue item: title=${item.title}, bodyLength=${item.body.length}, temporaryId=${temporaryId}, repo=${itemRepo}`);

    // Resolve parent: check if it's a temporary ID reference
    let effectiveParentIssueNumber;
    let effectiveParentRepo = itemRepo; // Default to same repo

    if (item.parent !== undefined) {
      if (isTemporaryId(item.parent)) {
        // It's a temporary ID, look it up in the map
        const resolvedParent = context.temporaryIdMap.get(normalizeTemporaryId(item.parent));
        if (resolvedParent !== undefined) {
          effectiveParentIssueNumber = resolvedParent.number;
          effectiveParentRepo = resolvedParent.repo;
          core.info(`Resolved parent temporary ID '${item.parent}' to ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
        } else {
          core.warning(`Parent temporary ID '${item.parent}' not found in map. Ensure parent issue is created before sub-issues.`);
          effectiveParentIssueNumber = undefined;
        }
      } else {
        // It's a real issue number
        effectiveParentIssueNumber = parseInt(String(item.parent), 10);
        if (isNaN(effectiveParentIssueNumber)) {
          core.warning(`Invalid parent value: ${item.parent}`);
          effectiveParentIssueNumber = undefined;
        }
      }
    } else {
      // Only use context parent if we're in the same repo as context
      const contextRepo = `${ghContext.repo.owner}/${ghContext.repo.repo}`;
      if (itemRepo === contextRepo) {
        effectiveParentIssueNumber = ghContext.payload?.issue?.number;
      }
    }

    // Get labels
    const labelsEnv = process.env.GH_AW_ISSUE_LABELS;
    let envLabels = labelsEnv
      ? labelsEnv
          .split(",")
          .map(label => label.trim())
          .filter(label => label)
      : [];

    let labels = [...envLabels];
    if (item.labels && Array.isArray(item.labels)) {
      labels = [...labels, ...item.labels];
    }
    labels = labels
      .filter(label => !!label)
      .map(label => String(label).trim())
      .filter(label => label)
      .map(label => sanitizeLabelContent(label))
      .filter(label => label)
      .map(label => (label.length > 64 ? label.substring(0, 64) : label))
      .filter((label, index, arr) => arr.indexOf(label) === index);

    let title = item.title ? item.title.trim() : "";

    // Replace temporary ID references in the body using already-created issues
    let processedBody = replaceTemporaryIdReferences(item.body, context.temporaryIdMap, itemRepo);

    // Remove duplicate title from description if it starts with a header matching the title
    processedBody = removeDuplicateTitleFromDescription(title, processedBody);

    let bodyLines = processedBody.split("\n");

    if (!title) {
      title = item.body || "Agent Output";
    }
    const titlePrefix = process.env.GH_AW_ISSUE_TITLE_PREFIX;
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }

    if (effectiveParentIssueNumber) {
      core.info("Detected issue context, parent issue " + effectiveParentRepo + "#" + effectiveParentIssueNumber);
      // Use full repo reference if cross-repo, short reference if same repo
      if (effectiveParentRepo === itemRepo) {
        bodyLines.push(`Related to #${effectiveParentIssueNumber}`);
      } else {
        bodyLines.push(`Related to ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
      }
    }

    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const runId = ghContext.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = ghContext.payload.repository ? `${ghContext.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${ghContext.repo.owner}/${ghContext.repo.repo}/actions/runs/${runId}`;

    // Add tracker-id comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    // Add expiration comment if expires is set
    addExpirationComment(bodyLines, "GH_AW_ISSUE_EXPIRES", "Issue");

    // Extract triggering context for footer generation
    const triggeringIssueNumber = ghContext.payload?.issue?.number && !ghContext.payload?.issue?.pull_request ? ghContext.payload.issue.number : undefined;
    const triggeringPRNumber = ghContext.payload?.pull_request?.number || (ghContext.payload?.issue?.pull_request ? ghContext.payload.issue.number : undefined);
    const triggeringDiscussionNumber = ghContext.payload?.discussion?.number;

    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";

    bodyLines.push(``, ``, generateFooter(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber).trimEnd(), "");
    const body = bodyLines.join("\n").trim();

    core.info(`Creating issue in ${itemRepo} with title: ${title}`);
    core.info(`Labels: ${labels}`);
    core.info(`Body length: ${body.length}`);

    const { data: issue } = await github.rest.issues.create({
      owner: repoParts.owner,
      repo: repoParts.repo,
      title: title,
      body: body,
      labels: labels,
    });

    core.info(`Created issue ${itemRepo}#${issue.number}: ${issue.html_url}`);

    // Store the mapping of temporary_id -> {repo, number}
    localTemporaryIdMap.set(normalizeTemporaryId(temporaryId), { repo: itemRepo, number: issue.number });
    core.info(`Stored temporary ID mapping: ${temporaryId} -> ${itemRepo}#${issue.number}`);

    // Sub-issue linking only works within the same repository
    if (effectiveParentIssueNumber && effectiveParentRepo === itemRepo) {
      core.info(`Attempting to link issue #${issue.number} as sub-issue of #${effectiveParentIssueNumber}`);
      try {
        const getIssueNodeIdQuery = `
          query($owner: String!, $repo: String!, $issueNumber: Int!) {
            repository(owner: $owner, name: $repo) {
              issue(number: $issueNumber) {
                id
              }
            }
          }
        `;

        // Get parent issue node ID
        const parentResult = await github.graphql(getIssueNodeIdQuery, {
          owner: repoParts.owner,
          repo: repoParts.repo,
          issueNumber: effectiveParentIssueNumber,
        });
        const parentNodeId = parentResult.repository.issue.id;

        // Get child issue node ID
        const childResult = await github.graphql(getIssueNodeIdQuery, {
          owner: repoParts.owner,
          repo: repoParts.repo,
          issueNumber: issue.number,
        });
        const childNodeId = childResult.repository.issue.id;

        // Link the child issue as a sub-issue of the parent
        const addSubIssueMutation = `
          mutation($issueId: ID!, $subIssueId: ID!) {
            addSubIssue(input: {
              issueId: $issueId,
              subIssueId: $subIssueId
            }) {
              subIssue {
                id
                number
              }
            }
          }
        `;

        await github.graphql(addSubIssueMutation, {
          issueId: parentNodeId,
          subIssueId: childNodeId,
        });

        core.info("✓ Successfully linked issue #" + issue.number + " as sub-issue of #" + effectiveParentIssueNumber);
      } catch (error) {
        core.info(`Warning: Could not link sub-issue to parent: ${error instanceof Error ? error.message : String(error)}`);
        // Fallback: add a comment if sub-issue linking fails
        try {
          await github.rest.issues.createComment({
            owner: repoParts.owner,
            repo: repoParts.repo,
            issue_number: effectiveParentIssueNumber,
            body: `Created related issue: #${issue.number}`,
          });
          core.info("✓ Added comment to parent issue #" + effectiveParentIssueNumber + " (sub-issue linking not available)");
        } catch (commentError) {
          core.info(`Warning: Could not add comment to parent issue: ${commentError instanceof Error ? commentError.message : String(commentError)}`);
        }
      }
    }

    return {
      success: true,
      temporaryIds: localTemporaryIdMap,
      data: {
        issue_number: issue.number,
        issue_url: issue.html_url,
        repo: itemRepo,
      },
    };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    if (errorMessage.includes("Issues has been disabled in this repository")) {
      core.info(`⚠ Cannot create issue "${item.title}": Issues are disabled for this repository`);
      return { success: true }; // Not a failure, just skipped
    }
    core.error(`✗ Failed to create issue "${item.title}": ${errorMessage}`);
    return { success: false, error: errorMessage };
  }
}

module.exports = { handleCreateIssue };
