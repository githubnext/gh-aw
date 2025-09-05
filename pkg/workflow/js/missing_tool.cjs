async function main() {
  const fs = require("fs");

  // Get environment variables
  const agentOutput = process.env.GITHUB_AW_AGENT_OUTPUT || "";
  const maxReports = process.env.GITHUB_AW_MISSING_TOOL_MAX
    ? parseInt(process.env.GITHUB_AW_MISSING_TOOL_MAX)
    : null;

  core.info("Processing missing-tool reports...");
  core.info(`Agent output length: ${agentOutput.length}`);
  if (maxReports) {
    core.info(`Maximum reports allowed: ${maxReports}`);
  }

  const missingTools = [];

  // Return early if no agent output
  if (!agentOutput.trim()) {
    core.info("No agent output to process");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(agentOutput);
  } catch (error) {
    core.error(
      `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  core.info(`Parsed agent output with ${validatedOutput.items.length} entries`);

  // Process all parsed entries
  for (const entry of validatedOutput.items) {
    if (entry.type === "missing-tool") {
      // Validate required fields
      if (!entry.tool) {
        core.warning(
          `missing-tool entry missing 'tool' field: ${JSON.stringify(entry)}`
        );
        continue;
      }
      if (!entry.reason) {
        core.warning(
          `missing-tool entry missing 'reason' field: ${JSON.stringify(entry)}`
        );
        continue;
      }

      const missingTool = {
        tool: entry.tool,
        reason: entry.reason,
        alternatives: entry.alternatives || null,
        timestamp: new Date().toISOString(),
      };

      missingTools.push(missingTool);
      core.info(`Recorded missing tool: ${missingTool.tool}`);

      // Check max limit
      if (maxReports && missingTools.length >= maxReports) {
        core.info(
          `Reached maximum number of missing tool reports (${maxReports})`
        );
        break;
      }
    }
  }

  core.info(`Total missing tools reported: ${missingTools.length}`);

  // Output results
  core.setOutput("tools_reported", JSON.stringify(missingTools));
  core.setOutput("total_count", missingTools.length.toString());

  // Log details for debugging
  if (missingTools.length > 0) {
    core.info("Missing tools summary:");
    missingTools.forEach((tool, index) => {
      core.info(`${index + 1}. Tool: ${tool.tool}`);
      core.info(`   Reason: ${tool.reason}`);
      if (tool.alternatives) {
        core.info(`   Alternatives: ${tool.alternatives}`);
      }
      core.info(`   Reported at: ${tool.timestamp}`);
      core.info("");
    });
  } else {
    core.info("No missing tools reported in this workflow execution.");
  }
}

main().catch(error => {
  core.error(`Error processing missing-tool reports: ${error}`);
  process.exit(1);
});
