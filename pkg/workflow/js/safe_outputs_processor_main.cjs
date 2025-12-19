// @ts-check
/// <reference types="@actions/github-script" />

const { SafeOutputHandlerManager } = require("./safe_output_handler_manager.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { serializeTemporaryIdMap, loadTemporaryIdMap } = require("./temporary_id.cjs");
const { handleCreateIssue } = require("./create_issue_handler.cjs");

/**
 * Main processor for all safe output messages.
 * This replaces the individual main() functions in each handler script.
 *
 * The processor:
 * 1. Loads agent output
 * 2. Registers all safe output handlers
 * 3. Processes all messages in sequence
 * 4. Collects temporary IDs and outputs
 */
async function main() {
  // Initialize outputs to empty strings
  core.setOutput("issue_number", "");
  core.setOutput("issue_url", "");
  core.setOutput("temporary_id_map", "{}");
  core.setOutput("pull_request_number", "");
  core.setOutput("pull_request_url", "");
  core.setOutput("discussion_number", "");
  core.setOutput("discussion_url", "");
  core.setOutput("issues_to_assign_copilot", "");
  core.setOutput("comment_id", "");
  core.setOutput("comment_url", "");

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Load agent output
  const result = loadAgentOutput();
  if (!result.success) {
    core.info("No agent output to process");
    return;
  }

  core.info(`Loaded ${result.items.length} item(s) from agent output`);

  // Load any existing temporary ID map (from previous steps or jobs)
  const existingTempIdMap = loadTemporaryIdMap();

  // Create handler manager
  const manager = new SafeOutputHandlerManager();

  // Register all handlers
  core.info("Registering safe output handlers...");
  manager.registerHandler("create_issue", handleCreateIssue);

  // Note: Other handlers will be registered here as they are converted.
  // For now, unregistered types will be skipped with an info message.
  // TODO: Register remaining handlers:
  // - create_pull_request
  // - create_discussion
  // - add_comment
  // - close_issue, close_discussion, close_pull_request
  // - update_issue, update_pull_request
  // - add_labels, add_reviewer
  // - assign_milestone, assign_to_agent, assign_to_user
  // - create_pr_review_comment
  // - create_code_scanning_alert
  // - push_to_pull_request_branch
  // - upload_assets
  // - update_release
  // - link_sub_issue
  // - hide_comment
  // - update_project

  const registeredTypes = manager.getRegisteredTypes();
  core.info(`Registered handlers for: ${registeredTypes.join(", ")}`);

  // Count items by type
  const typeCount = new Map();
  for (const item of result.items) {
    typeCount.set(item.type, (typeCount.get(item.type) || 0) + 1);
  }
  core.info(
    `Items by type: ${Array.from(typeCount.entries())
      .map(([type, count]) => `${type}(${count})`)
      .join(", ")}`
  );

  // If in staged mode, show preview for all items
  if (isStaged) {
    core.info("🎭 Staged mode enabled - generating preview");

    // Group items by type for preview
    const itemsByType = new Map();
    for (const item of result.items) {
      if (!itemsByType.has(item.type)) {
        itemsByType.set(item.type, []);
      }
      itemsByType.get(item.type).push(item);
    }

    let summaryContent = "## 🎭 Staged Mode: Safe Outputs Preview\n\n";
    summaryContent += "The following safe outputs would be processed if staged mode was disabled:\n\n";

    for (const [type, items] of itemsByType) {
      summaryContent += `### ${type} (${items.length} item(s))\n\n`;
      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        summaryContent += `#### Item ${i + 1}\n`;

        // Show relevant fields based on type
        if (item.title) {
          summaryContent += `**Title:** ${item.title}\n\n`;
        }
        if (item.temporary_id) {
          summaryContent += `**Temporary ID:** ${item.temporary_id}\n\n`;
        }
        if (item.repo) {
          summaryContent += `**Repository:** ${item.repo}\n\n`;
        }
        if (item.body) {
          const bodyPreview = item.body.length > 500 ? item.body.substring(0, 500) + "..." : item.body;
          summaryContent += `**Body:**\n${bodyPreview}\n\n`;
        }
        if (item.labels && Array.isArray(item.labels) && item.labels.length > 0) {
          summaryContent += `**Labels:** ${item.labels.join(", ")}\n\n`;
        }

        summaryContent += "---\n\n";
      }
    }

    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Safe outputs preview written to step summary");
    return;
  }

  // Create handler context with existing temporary IDs
  const handlerContext = {
    core,
    github,
    context,
    exec,
    temporaryIdMap: existingTempIdMap,
  };

  if (existingTempIdMap.size > 0) {
    core.info(`Loaded ${existingTempIdMap.size} existing temporary ID mapping(s)`);
  }

  // Process all items
  core.info("Processing all safe output items...");
  const processResult = await manager.processAll(result.items, handlerContext);

  // Report results
  if (processResult.success) {
    core.info(`✓ Successfully processed all items`);
  } else {
    core.warning(`⚠ Processing completed with ${processResult.errors.length} error(s)`);
    for (const error of processResult.errors) {
      core.error(`  - ${error}`);
    }
    // Don't fail the workflow for handler errors - they may be non-critical
  }

  // Output temporary ID map
  const tempIdMapOutput = serializeTemporaryIdMap(processResult.temporaryIdMap);
  core.setOutput("temporary_id_map", tempIdMapOutput);
  if (processResult.temporaryIdMap.size > 0) {
    core.info(`Temporary ID map: ${tempIdMapOutput}`);
  }

  // Set outputs from last item of each type
  // This maintains compatibility with existing workflows that expect these outputs
  const issueItems = result.items.filter(item => item.type === "create_issue");
  if (issueItems.length > 0) {
    // Find the last issue that was successfully created
    for (let i = issueItems.length - 1; i >= 0; i--) {
      const tempId = issueItems[i].temporary_id;
      if (tempId) {
        const normalized = require("./temporary_id.cjs").normalizeTemporaryId(tempId);
        const resolved = processResult.temporaryIdMap.get(normalized);
        if (resolved) {
          core.setOutput("issue_number", resolved.number);
          // Reconstruct URL
          const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
          const issueUrl = `${githubServer}/${resolved.repo}/issues/${resolved.number}`;
          core.setOutput("issue_url", issueUrl);
          break;
        }
      }
    }
  }

  // Check for copilot assignment
  const assignCopilot = process.env.GH_AW_ASSIGN_COPILOT === "true";
  if (assignCopilot && processResult.temporaryIdMap.size > 0) {
    const issuesToAssign = Array.from(processResult.temporaryIdMap.values())
      .filter(value => value.repo && value.number) // Filter valid entries
      .map(value => `${value.repo}:${value.number}`)
      .join(",");
    if (issuesToAssign) {
      core.setOutput("issues_to_assign_copilot", issuesToAssign);
      core.info(`Issues to assign copilot: ${issuesToAssign}`);
    }
  }

  // Add summary
  if (processResult.temporaryIdMap.size > 0) {
    let summaryContent = "\n\n## Safe Outputs\n";

    // Group by type
    const issuesCreated = [];
    for (const [tempId, value] of processResult.temporaryIdMap.entries()) {
      if (value.repo && value.number) {
        issuesCreated.push({ tempId, ...value });
      }
    }

    if (issuesCreated.length > 0) {
      summaryContent += "\n### Created Issues\n";
      for (const item of issuesCreated) {
        const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
        const url = `${githubServer}/${item.repo}/issues/${item.number}`;
        summaryContent += `- [${item.repo}#${item.number}](${url})\n`;
      }
    }

    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Completed processing ${result.items.length} safe output item(s)`);
}

// Execute main function
(async () => {
  try {
    await main();
  } catch (error) {
    core.setFailed(`Safe outputs processor failed: ${error instanceof Error ? error.message : String(error)}`);
  }
})();
