// @ts-check
/// <reference types="@actions/github-script" />

const { SafeOutputHandlerManager } = require("./safe_output_handler_manager.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { serializeTemporaryIdMap } = require("./temporary_id.cjs");
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
  
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";
  
  // Load agent output
  const result = loadAgentOutput();
  if (!result.success) {
    core.info("No agent output to process");
    return;
  }
  
  core.info(`Loaded ${result.items.length} item(s) from agent output`);
  
  // Create handler manager
  const manager = new SafeOutputHandlerManager();
  
  // Register all handlers
  core.info("Registering safe output handlers...");
  manager.registerHandler("create_issue", handleCreateIssue);
  // TODO: Register other handlers as they are converted
  // manager.registerHandler("create_pull_request", handleCreatePullRequest);
  // manager.registerHandler("create_discussion", handleCreateDiscussion);
  // manager.registerHandler("add_comment", handleAddComment);
  // etc.
  
  const registeredTypes = manager.getRegisteredTypes();
  core.info(`Registered handlers for: ${registeredTypes.join(", ")}`);
  
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
        if (item.labels && item.labels.length > 0) {
          summaryContent += `**Labels:** ${item.labels.join(", ")}\n\n`;
        }
        
        summaryContent += "---\n\n";
      }
    }
    
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Safe outputs preview written to step summary");
    return;
  }
  
  // Create handler context
  const handlerContext = {
    core,
    github,
    context,
    exec,
    temporaryIdMap: new Map(),
  };
  
  // Process all items
  core.info("Processing all safe output items...");
  const processResult = await manager.processAll(result.items, handlerContext);
  
  // Report results
  if (processResult.success) {
    core.info(`✓ Successfully processed all items`);
  } else {
    core.error(`✗ Processing completed with ${processResult.errors.length} error(s)`);
    for (const error of processResult.errors) {
      core.error(`  - ${error}`);
    }
  }
  
  // Output temporary ID map
  const tempIdMapOutput = serializeTemporaryIdMap(processResult.temporaryIdMap);
  core.setOutput("temporary_id_map", tempIdMapOutput);
  core.info(`Temporary ID map: ${tempIdMapOutput}`);
  
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
          // Reconstruct URL (this is a simplification)
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
      .map(value => `${value.repo}:${value.number}`)
      .join(",");
    core.setOutput("issues_to_assign_copilot", issuesToAssign);
    core.info(`Issues to assign copilot: ${issuesToAssign}`);
  }
  
  // Add summary
  if (processResult.temporaryIdMap.size > 0) {
    let summaryContent = "\n\n## Safe Outputs\n";
    
    // Group by type
    const issuesCreated = [];
    for (const [tempId, value] of processResult.temporaryIdMap.entries()) {
      issuesCreated.push({ tempId, ...value });
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
  await main();
})();
