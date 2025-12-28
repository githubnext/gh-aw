// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  const fs = require("fs");

  // Get environment variables
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT || "";
  const maxReports = process.env.GH_AW_MISSING_TOOL_MAX ? parseInt(process.env.GH_AW_MISSING_TOOL_MAX) : null;

  core.info("Processing missing-tool reports...");
  if (maxReports) {
    core.info(`Maximum reports allowed: ${maxReports}`);
  }

  /** @type {any[]} */
  const missingTools = [];

  // Return early if no agent output
  if (!agentOutputFile.trim()) {
    core.info("No agent output to process");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  // Read agent output from file
  let agentOutput;
  try {
    agentOutput = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.info(`Agent output file not found or unreadable: ${getErrorMessage(error)}`);
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  if (agentOutput.trim() === "") {
    core.info("No agent output to process");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  core.info(`Agent output length: ${agentOutput.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(agentOutput);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${getErrorMessage(error)}`);
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
    if (entry.type === "missing_tool") {
      // Validate required fields
      if (!entry.tool) {
        core.warning(`missing-tool entry missing 'tool' field: ${JSON.stringify(entry)}`);
        continue;
      }
      if (!entry.reason) {
        core.warning(`missing-tool entry missing 'reason' field: ${JSON.stringify(entry)}`);
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
        core.info(`Reached maximum number of missing tool reports (${maxReports})`);
        break;
      }
    }
  }

  core.info(`Total missing tools reported: ${missingTools.length}`);

  // Output results
  core.setOutput("tools_reported", JSON.stringify(missingTools));
  core.setOutput("total_count", missingTools.length.toString());

  // Log details for debugging and create step summary
  if (missingTools.length > 0) {
    core.info("Missing tools summary:");

    // Create structured summary for GitHub Actions step summary
    core.summary.addHeading("Missing Tools Report", 3).addRaw(`Found **${missingTools.length}** missing tool${missingTools.length > 1 ? "s" : ""} in this workflow execution.\n\n`);

    missingTools.forEach((tool, index) => {
      core.info(`${index + 1}. Tool: ${tool.tool}`);
      core.info(`   Reason: ${tool.reason}`);
      if (tool.alternatives) {
        core.info(`   Alternatives: ${tool.alternatives}`);
      }
      core.info(`   Reported at: ${tool.timestamp}`);
      core.info("");

      // Add to summary with structured formatting
      core.summary.addRaw(`#### ${index + 1}. \`${tool.tool}\`\n\n`).addRaw(`**Reason:** ${tool.reason}\n\n`);

      if (tool.alternatives) {
        core.summary.addRaw(`**Alternatives:** ${tool.alternatives}\n\n`);
      }

      core.summary.addRaw(`**Reported at:** ${tool.timestamp}\n\n---\n\n`);
    });

    core.summary.write();
  } else {
    core.info("No missing tools reported in this workflow execution.");
    core.summary.addHeading("Missing Tools Report", 3).addRaw("✅ No missing tools reported in this workflow execution.").write();
  }
}

module.exports = { main };
