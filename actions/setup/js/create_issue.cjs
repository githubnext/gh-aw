// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { generateTemporaryId, isTemporaryId, normalizeTemporaryId, replaceTemporaryIdReferences, serializeTemporaryIdMap } = require("./temporary_id.cjs");
const { parseAllowedRepos, getDefaultTargetRepo, validateRepo, parseRepoSlug } = require("./repo_helpers.cjs");
const { addExpirationComment } = require("./expiration_helpers.cjs");
const { removeDuplicateTitleFromDescription } = require("./remove_duplicate_title.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Main function - supports both factory pattern and standalone mode
 * @param {Object} [config] - Handler configuration (factory mode) or undefined (standalone mode)
 * @returns {Promise<Function|void>} Message processor function (factory mode) or Promise<void> (standalone mode)
 */
async function main(config) {
  // Factory pattern mode: return a message processor function
  if (config !== undefined) {
    core.info(`Initializing create_issue handler in factory mode`);
    
    // Create a processor that handles single items
    return async function processMessage(outputItem, resolvedTemporaryIds) {
      core.info(`Processing create_issue item: ${outputItem.title}`);
      
      // Call the standalone function with a single-item array
      // We'll need to refactor the standalone logic into a helper function
      // For now, return a placeholder that demonstrates the pattern
      const temporaryId = outputItem.temporary_id || generateTemporaryId();
      
      // This is a simplified version - full implementation would call processCreateIssueInternal
      core.warning("create_issue factory mode not fully implemented yet - use standalone mode");
      
      return {
        temporaryId: temporaryId,
        repo: `${context.repo.owner}/${context.repo.repo}`,
        number: 1, // Placeholder
      };
    };
  }

  // Standalone mode: original behavior
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("issue_number", "");
  core.setOutput("issue_url", "");
  core.setOutput("temporary_id_map", "{}");
  core.setOutput("issues_to_assign_copilot", "");

  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const createIssueItems = result.items.filter(item => item.type === "create_issue");
  if (createIssueItems.length === 0) {
    core.info("No create-issue items found in agent output");
    return;
  }
  core.info(`Found ${createIssueItems.length} create-issue item(s)`);

  // Parse allowed repos and default target
  const allowedRepos = parseAllowedRepos();
  const defaultTargetRepo = getDefaultTargetRepo();
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  if (isStaged) {
    await generateStagedPreview({
      title: "Create Issues",
      description: "The following issues would be created if staged mode was disabled:",
      items: createIssueItems,
      renderItem: (item, index) => {
        let content = `#### Issue ${index + 1}\n`;
        content += `**Title:** ${item.title || "No title provided"}\n\n`;
        if (item.temporary_id) {
          content += `**Temporary ID:** ${item.temporary_id}\n\n`;
        }
        if (item.repo) {
          content += `**Repository:** ${item.repo}\n\n`;
        }
        if (item.body) {
          content += `**Body:**\n${item.body}\n\n`;
        }
        if (item.labels && item.labels.length > 0) {
          content += `**Labels:** ${item.labels.join(", ")}\n\n`;
        }
        if (item.parent) {
          content += `**Parent:** ${item.parent}\n\n`;
        }
        return content;
      },
    });
    return;
  }
  const parentIssueNumber = context.payload?.issue?.number;

  // Map to track temporary_id -> {repo, number} relationships
  /** @type {Map<string, {repo: string, number: number}>} */
  const temporaryIdMap = new Map();

  // Extract triggering context for footer generation
  const triggeringIssueNumber = context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber = context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const labelsEnv = process.env.GH_AW_ISSUE_LABELS;
  let envLabels = labelsEnv
    ? labelsEnv
        .split(",")
        .map(label => label.trim())
        .filter(label => label)
    : [];
  const createdIssues = [];
  for (let i = 0; i < createIssueItems.length; i++) {
    const createIssueItem = createIssueItems[i];

    // Determine target repository for this issue
    const itemRepo = createIssueItem.repo ? String(createIssueItem.repo).trim() : defaultTargetRepo;

    // Validate the repository is allowed
    const repoValidation = validateRepo(itemRepo, defaultTargetRepo, allowedRepos);
    if (!repoValidation.valid) {
      core.warning(`Skipping issue: ${repoValidation.error}`);
      continue;
    }

    // Parse the repository slug
    const repoParts = parseRepoSlug(itemRepo);
    if (!repoParts) {
      core.warning(`Skipping issue: Invalid repository format '${itemRepo}'. Expected 'owner/repo'.`);
      continue;
    }

    // Get or generate the temporary ID for this issue
    const temporaryId = createIssueItem.temporary_id || generateTemporaryId();
    core.info(`Processing create-issue item ${i + 1}/${createIssueItems.length}: title=${createIssueItem.title}, bodyLength=${createIssueItem.body.length}, temporaryId=${temporaryId}, repo=${itemRepo}`);

    // Debug logging for parent field
    core.info(`Debug: createIssueItem.parent = ${JSON.stringify(createIssueItem.parent)}`);
    core.info(`Debug: parentIssueNumber from context = ${JSON.stringify(parentIssueNumber)}`);

    // Resolve parent: check if it's a temporary ID reference
    let effectiveParentIssueNumber;
    let effectiveParentRepo = itemRepo; // Default to same repo
    if (createIssueItem.parent !== undefined) {
      if (isTemporaryId(createIssueItem.parent)) {
        // It's a temporary ID, look it up in the map
        const resolvedParent = temporaryIdMap.get(normalizeTemporaryId(createIssueItem.parent));
        if (resolvedParent !== undefined) {
          effectiveParentIssueNumber = resolvedParent.number;
          effectiveParentRepo = resolvedParent.repo;
          core.info(`Resolved parent temporary ID '${createIssueItem.parent}' to ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
        } else {
          core.warning(`Parent temporary ID '${createIssueItem.parent}' not found in map. Ensure parent issue is created before sub-issues.`);
          effectiveParentIssueNumber = undefined;
        }
      } else {
        // It's a real issue number (may have # prefix)
        const parentStr = String(createIssueItem.parent).trim();
        const parentWithoutHash = parentStr.startsWith("#") ? parentStr.substring(1) : parentStr;
        effectiveParentIssueNumber = parseInt(parentWithoutHash, 10);
        if (isNaN(effectiveParentIssueNumber)) {
          core.warning(`Invalid parent value: ${createIssueItem.parent}`);
          effectiveParentIssueNumber = undefined;
        }
      }
    } else {
      // Only use context parent if we're in the same repo as context
      const contextRepo = `${context.repo.owner}/${context.repo.repo}`;
      if (itemRepo === contextRepo) {
        effectiveParentIssueNumber = parentIssueNumber;
      }
    }
    core.info(`Debug: effectiveParentIssueNumber = ${JSON.stringify(effectiveParentIssueNumber)}, effectiveParentRepo = ${effectiveParentRepo}`);

    if (effectiveParentIssueNumber && createIssueItem.parent !== undefined) {
      core.info(`Using explicit parent issue number from item: ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
    }
    let labels = [...envLabels];
    if (createIssueItem.labels && Array.isArray(createIssueItem.labels)) {
      labels = [...labels, ...createIssueItem.labels];
    }
    labels = labels
      .filter(label => !!label)
      .map(label => String(label).trim())
      .filter(label => label)
      .map(label => sanitizeLabelContent(label))
      .filter(label => label)
      .map(label => (label.length > 64 ? label.substring(0, 64) : label))
      .filter((label, index, arr) => arr.indexOf(label) === index);
    let title = createIssueItem.title ? createIssueItem.title.trim() : "";

    // Replace temporary ID references in the body using already-created issues
    let processedBody = replaceTemporaryIdReferences(createIssueItem.body, temporaryIdMap, itemRepo);

    // Remove duplicate title from description if it starts with a header matching the title
    processedBody = removeDuplicateTitleFromDescription(title, processedBody);

    let bodyLines = processedBody.split("\n");

    if (!title) {
      title = createIssueItem.body || "Agent Output";
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
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Add tracker-id comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    // Add expiration comment if expires is set
    addExpirationComment(bodyLines, "GH_AW_ISSUE_EXPIRES", "Issue");

    bodyLines.push(``, ``, generateFooter(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber).trimEnd(), "");
    const body = bodyLines.join("\n").trim();
    core.info(`Creating issue in ${itemRepo} with title: ${title}`);
    core.info(`Labels: ${labels}`);
    core.info(`Body length: ${body.length}`);
    try {
      const { data: issue } = await github.rest.issues.create({
        owner: repoParts.owner,
        repo: repoParts.repo,
        title: title,
        body: body,
        labels: labels,
      });
      core.info(`Created issue ${itemRepo}#${issue.number}: ${issue.html_url}`);
      createdIssues.push({ ...issue, _repo: itemRepo });

      // Store the mapping of temporary_id -> {repo, number}
      temporaryIdMap.set(normalizeTemporaryId(temporaryId), { repo: itemRepo, number: issue.number });
      core.info(`Stored temporary ID mapping: ${temporaryId} -> ${itemRepo}#${issue.number}`);

      // Debug logging for sub-issue linking
      core.info(`Debug: About to check if sub-issue linking is needed. effectiveParentIssueNumber = ${effectiveParentIssueNumber}`);

      // Sub-issue linking only works within the same repository
      if (effectiveParentIssueNumber && effectiveParentRepo === itemRepo) {
        core.info(`Attempting to link issue #${issue.number} as sub-issue of #${effectiveParentIssueNumber}`);
        try {
          // First, get the node IDs for both parent and child issues
          core.info(`Fetching node ID for parent issue #${effectiveParentIssueNumber}...`);
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
          core.info(`Parent issue node ID: ${parentNodeId}`);

          // Get child issue node ID
          core.info(`Fetching node ID for child issue #${issue.number}...`);
          const childResult = await github.graphql(getIssueNodeIdQuery, {
            owner: repoParts.owner,
            repo: repoParts.repo,
            issueNumber: issue.number,
          });
          const childNodeId = childResult.repository.issue.id;
          core.info(`Child issue node ID: ${childNodeId}`);

          // Link the child issue as a sub-issue of the parent
          core.info(`Executing addSubIssue mutation...`);
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
          core.info(`Warning: Could not link sub-issue to parent: ${getErrorMessage(error)}`);
          core.info(`Error details: ${error instanceof Error ? error.stack : String(error)}`);
          // Fallback: add a comment if sub-issue linking fails
          try {
            core.info(`Attempting fallback: adding comment to parent issue #${effectiveParentIssueNumber}...`);
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
      } else if (effectiveParentIssueNumber && effectiveParentRepo !== itemRepo) {
        core.info(`Skipping sub-issue linking: parent is in different repository (${effectiveParentRepo})`);
      } else {
        core.info(`Debug: No parent issue number set, skipping sub-issue linking`);
      }
      if (i === createIssueItems.length - 1) {
        core.setOutput("issue_number", issue.number);
        core.setOutput("issue_url", issue.html_url);
      }
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      if (errorMessage.includes("Issues has been disabled in this repository")) {
        core.info(`⚠ Cannot create issue "${title}" in ${itemRepo}: Issues are disabled for this repository`);
        core.info("Consider enabling issues in repository settings if you want to create issues automatically");
        continue;
      }
      core.error(`✗ Failed to create issue "${title}" in ${itemRepo}: ${errorMessage}`);
      throw error;
    }
  }
  if (createdIssues.length > 0) {
    let summaryContent = "\n\n## GitHub Issues\n";
    for (const issue of createdIssues) {
      const repoLabel = issue._repo !== defaultTargetRepo ? ` (${issue._repo})` : "";
      summaryContent += `- Issue #${issue.number}${repoLabel}: [${issue.title}](${issue.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  // Output the temporary ID map as JSON for use by downstream jobs
  const tempIdMapOutput = serializeTemporaryIdMap(temporaryIdMap);
  core.setOutput("temporary_id_map", tempIdMapOutput);
  core.info(`Temporary ID map: ${tempIdMapOutput}`);

  // Output issues that need copilot assignment for assign_to_agent job
  // This is used when create-issue has assignees: [copilot]
  const assignCopilot = process.env.GH_AW_ASSIGN_COPILOT === "true";
  if (assignCopilot && createdIssues.length > 0) {
    // Format: repo:number for each issue (for cross-repo support)
    const issuesToAssign = createdIssues.map(issue => `${issue._repo}:${issue.number}`).join(",");
    core.setOutput("issues_to_assign_copilot", issuesToAssign);
    core.info(`Issues to assign copilot: ${issuesToAssign}`);
  }

  core.info(`Successfully created ${createdIssues.length} issue(s)`);
}

module.exports = { main };
