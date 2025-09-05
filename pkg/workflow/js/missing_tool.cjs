async function main() {
  const fs = require("fs");
  const core = require("@actions/core");

  // Get environment variables
  const agentOutput = process.env.GITHUB_AW_AGENT_OUTPUT || "";
  const maxReports = process.env.GITHUB_AW_MISSING_TOOL_MAX
    ? parseInt(process.env.GITHUB_AW_MISSING_TOOL_MAX)
    : null;

  console.log("Processing missing-tool reports...");
  console.log("Agent output length:", agentOutput.length);
  if (maxReports) {
    console.log("Maximum reports allowed:", maxReports);
  }

  const missingTools = [];

  if (agentOutput.trim()) {
    let parsedData;

    try {
      // First try to parse as JSON array
      parsedData = JSON.parse(agentOutput);

      // If it's not an array, wrap it in an array
      if (!Array.isArray(parsedData)) {
        parsedData = [parsedData];
      }

      console.log(
        "Parsed agent output as JSON array with",
        parsedData.length,
        "entries"
      );
    } catch (arrayError) {
      console.log("Agent output is not a JSON array, trying JSONL format...");

      // Fall back to JSONL parsing (newline-delimited JSON)
      const lines = agentOutput.split("\n").filter(line => line.trim());
      parsedData = [];

      for (const line of lines) {
        try {
          const entry = JSON.parse(line);
          parsedData.push(entry);
        } catch (lineError) {
          console.log("Warning: Failed to parse line as JSON:", line);
          console.log("Parse error:", lineError.message);
        }
      }

      console.log(
        "Parsed agent output as JSONL with",
        parsedData.length,
        "entries"
      );
    }

    // Process all parsed entries
    for (const entry of parsedData) {
      if (entry.type === "missing-tool") {
        // Validate required fields
        if (!entry.tool) {
          console.log(
            "Warning: missing-tool entry missing 'tool' field:",
            JSON.stringify(entry)
          );
          continue;
        }
        if (!entry.reason) {
          console.log(
            "Warning: missing-tool entry missing 'reason' field:",
            JSON.stringify(entry)
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
        console.log("Recorded missing tool:", missingTool.tool);

        // Check max limit
        if (maxReports && missingTools.length >= maxReports) {
          console.log(
            `Reached maximum number of missing tool reports (${maxReports})`
          );
          break;
        }
      }
    }
  }

  console.log("Total missing tools reported:", missingTools.length);

  // Output results
  core.setOutput("tools_reported", JSON.stringify(missingTools));
  core.setOutput("total_count", missingTools.length.toString());

  // Log details for debugging
  if (missingTools.length > 0) {
    console.log("Missing tools summary:");
    missingTools.forEach((tool, index) => {
      console.log(`${index + 1}. Tool: ${tool.tool}`);
      console.log(`   Reason: ${tool.reason}`);
      if (tool.alternatives) {
        console.log(`   Alternatives: ${tool.alternatives}`);
      }
      console.log(`   Reported at: ${tool.timestamp}`);
      console.log("");
    });
  } else {
    console.log("No missing tools reported in this workflow execution.");
  }
}

main().catch(error => {
  console.error("Error processing missing-tool reports:", error);
  process.exit(1);
});
