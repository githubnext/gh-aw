// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { updateBody } = require("./update_pr_description_helpers.cjs");

/**
 * Process a single update-release message
 * @param {Object} message - The update-release message
 * @param {boolean} isStaged - Whether in staged mode
 * @param {string} workflowName - Workflow name for attribution
 * @returns {Promise<Object>} Result with release info
 */
async function processReleaseUpdate(message, isStaged, workflowName) {
  // In staged mode, skip actual processing (preview is handled elsewhere)
  if (isStaged) {
    core.info(`Staged mode: Would update release with tag ${message.tag || "(inferred)"}`);
    return { skipped: true, reason: "staged_mode" };
  }

  core.info(`Processing update-release message`);

  try {
    // Infer tag from event context if not provided
    let releaseTag = message.tag;
    if (!releaseTag) {
      // Try to get tag from release event context
      if (context.eventName === "release" && context.payload.release && context.payload.release.tag_name) {
        releaseTag = context.payload.release.tag_name;
        core.info(`Inferred release tag from event context: ${releaseTag}`);
      } else if (context.eventName === "workflow_dispatch" && context.payload.inputs) {
        // Try to extract from release_url input
        const releaseUrl = context.payload.inputs.release_url;
        if (releaseUrl) {
          const urlMatch = releaseUrl.match(/github\.com\/[^\/]+\/[^\/]+\/releases\/tag\/([^\/\?#]+)/);
          if (urlMatch && urlMatch[1]) {
            releaseTag = decodeURIComponent(urlMatch[1]);
            core.info(`Inferred release tag from release_url input: ${releaseTag}`);
          }
        }
        // Try to fetch from release_id input
        if (!releaseTag && context.payload.inputs.release_id) {
          const releaseId = context.payload.inputs.release_id;
          core.info(`Fetching release with ID: ${releaseId}`);
          const { data: release } = await github.rest.repos.getRelease({
            owner: context.repo.owner,
            repo: context.repo.repo,
            release_id: parseInt(releaseId, 10),
          });
          releaseTag = release.tag_name;
          core.info(`Inferred release tag from release_id input: ${releaseTag}`);
        }
      }

      if (!releaseTag) {
        throw new Error("Release tag is required but not provided and cannot be inferred from event context");
      }
    }

    // Get the release by tag
    core.info(`Fetching release with tag: ${releaseTag}`);
    const { data: release } = await github.rest.repos.getReleaseByTag({
      owner: context.repo.owner,
      repo: context.repo.repo,
      tag: releaseTag,
    });

    core.info(`Found release: ${release.name || release.tag_name} (ID: ${release.id})`);

    // Get workflow run URL for AI attribution
    const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

    // Log operation type
    const operation = message.operation || "append";
    if (operation === "append") {
      core.info("Operation: append (add to end with separator)");
    } else if (operation === "prepend") {
      core.info("Operation: prepend (add to start with separator)");
    } else if (operation === "replace") {
      core.info("Operation: replace (replace entire body)");
    }

    // Use shared helper to update body based on operation
    const newBody = updateBody({
      currentBody: release.body || "",
      newContent: message.body,
      operation,
      workflowName,
      runUrl,
      runId: context.runId,
    });

    // Update the release
    const { data: updatedRelease } = await github.rest.repos.updateRelease({
      owner: context.repo.owner,
      repo: context.repo.repo,
      release_id: release.id,
      body: newBody,
    });

    core.info(`Successfully updated release: ${updatedRelease.html_url}`);

    // Return result with release info
    return {
      tag: releaseTag,
      url: updatedRelease.html_url,
      id: updatedRelease.id,
      releaseId: updatedRelease.id,
    };
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    const tagInfo = message.tag || "inferred from context";

    // Check for specific error cases
    if (errorMessage.includes("Not Found")) {
      core.error(`Release with tag '${tagInfo}' not found. Please ensure the tag exists.`);
      core.error("Error details: " + errorMessage);
      throw new Error(`Release with tag '${tagInfo}' not found. Please ensure the tag exists.`);
    }

    core.error(`Failed to update release with tag ${tagInfo}: ${errorMessage}`);
    throw new Error(`Failed to update release with tag ${tagInfo}: ${errorMessage}`);
  }
}

/**
 * Main handler function - supports both direct invocation and factory pattern
 * @param {Object} config - Handler configuration
 * @returns {Promise<Function|void>} Handler function for factory pattern, or void for direct execution
 */
async function main(config = {}) {
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";

  // Load agent output to determine execution mode
  const result = loadAgentOutput();
  
  // If no agent output, return factory handler for use by handler_manager
  if (!result.success) {
    return async function handleUpdateRelease(message, resolvedTemporaryIds = {}) {
      return await processReleaseUpdate(message, isStaged, workflowName);
    };
  }

  // Direct execution mode (for backwards compatibility with tests)
  const releaseItems = result.items.filter(item => item.type === "update_release");
  
  if (releaseItems.length === 0) {
    core.info("No update-release items found in agent output");
    return;
  }

  core.info(`Found ${releaseItems.length} update-release item(s) to process`);

  // Handle staged mode preview
  if (isStaged) {
    const previewItems = releaseItems.map((item, index) => {
      const tagDisplay = item.tag || "(inferred from context)";
      const operation = item.operation || "append";
      return `#### Release ${index + 1}: ${tagDisplay}\n\n` + `**Operation:** ${operation}\n\n` + `**Content:**\n\n${item.body || "(no content)"}`;
    });

    const previewContent = "## Update Release Preview\n\n" + "The following releases would be updated if staged mode was disabled:\n\n" + previewItems.join("\n\n---\n\n");

    await core.summary.addRaw(previewContent).write();
    core.info("Staged mode: Generated preview");
    return;
  }

  // Process each release update
  const results = [];
  for (let i = 0; i < releaseItems.length; i++) {
    const item = releaseItems[i];
    core.info(`Processing release ${i + 1}/${releaseItems.length}`);
    
    try {
      const result = await processReleaseUpdate(item, isStaged, workflowName);
      results.push(result);
      
      // Set outputs for the first (or only) release
      if (i === 0) {
        core.setOutput("release_id", result.releaseId);
        core.setOutput("release_url", result.url);
        core.setOutput("release_tag", result.tag);
      }
    } catch (error) {
      core.setFailed(getErrorMessage(error));
      return;
    }
  }

  // Generate summary
  if (results.length > 0) {
    const summaryContent =
      "\n\n## Release Updates\n" +
      results
        .map(r => `- Updated release **${r.tag}**: [View Release](${r.url})`)
        .join("\n");
    
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully updated ${results.length} release(s)`);
}

module.exports = { main };
