// Embedded files for bundling
const FILES = {
    "load_agent_output.cjs": "// @ts-check\n/// \u003creference types=\"@actions/github-script\" /\u003e\n\nconst fs = require(\"fs\");\n\n/**\n * Maximum content length to log for debugging purposes\n * @type {number}\n */\nconst MAX_LOG_CONTENT_LENGTH = 10000;\n\n/**\n * Truncate content for logging if it exceeds the maximum length\n * @param {string} content - Content to potentially truncate\n * @returns {string} Truncated content with indicator if truncated\n */\nfunction truncateForLogging(content) {\n  if (content.length \u003c= MAX_LOG_CONTENT_LENGTH) {\n    return content;\n  }\n  return content.substring(0, MAX_LOG_CONTENT_LENGTH) + `\\n... (truncated, total length: ${content.length})`;\n}\n\n/**\n * Load and parse agent output from the GH_AW_AGENT_OUTPUT file\n *\n * This utility handles the common pattern of:\n * 1. Reading the GH_AW_AGENT_OUTPUT environment variable\n * 2. Loading the file content\n * 3. Validating the JSON structure\n * 4. Returning parsed items array\n *\n * @returns {{\n *   success: true,\n *   items: any[]\n * } | {\n *   success: false,\n *   items?: undefined,\n *   error?: string\n * }} Result object with success flag and items array (if successful) or error message\n */\nfunction loadAgentOutput() {\n  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;\n\n  // No agent output file specified\n  if (!agentOutputFile) {\n    core.info(\"No GH_AW_AGENT_OUTPUT environment variable found\");\n    return { success: false };\n  }\n\n  // Read agent output from file\n  let outputContent;\n  try {\n    outputContent = fs.readFileSync(agentOutputFile, \"utf8\");\n  } catch (error) {\n    const errorMessage = `Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`;\n    core.error(errorMessage);\n    return { success: false, error: errorMessage };\n  }\n\n  // Check for empty content\n  if (outputContent.trim() === \"\") {\n    core.info(\"Agent output content is empty\");\n    return { success: false };\n  }\n\n  core.info(`Agent output content length: ${outputContent.length}`);\n\n  // Parse the validated output JSON\n  let validatedOutput;\n  try {\n    validatedOutput = JSON.parse(outputContent);\n  } catch (error) {\n    const errorMessage = `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`;\n    core.error(errorMessage);\n    core.info(`Failed to parse content:\\n${truncateForLogging(outputContent)}`);\n    return { success: false, error: errorMessage };\n  }\n\n  // Validate items array exists\n  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {\n    core.info(\"No valid items found in agent output\");\n    core.info(`Parsed content: ${truncateForLogging(JSON.stringify(validatedOutput))}`);\n    return { success: false };\n  }\n\n  return { success: true, items: validatedOutput.items };\n}\n\nmodule.exports = { loadAgentOutput, truncateForLogging, MAX_LOG_CONTENT_LENGTH };\n"
  };

// Helper to load embedded files
function requireFile(filename) {
  const content = FILES[filename];
  if (!content) {
    throw new Error(`File not found: ${filename}`);
  }
  const exports = {};
  const module = { exports };
  const func = new Function('exports', 'module', 'require', content);
  func(exports, module, requireFile);
  return module.exports;
}

// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = requireFile('load_agent_output.cjs');

/**
 * Main function to handle noop safe output
 * No-op is a fallback output type that logs messages for transparency
 * without taking any GitHub API actions
 */
async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all noop items
  const noopItems = result.items.filter(/** @param {any} item */ item => item.type === "noop");
  if (noopItems.length === 0) {
    core.info("No noop items found in agent output");
    return;
  }

  core.info(`Found ${noopItems.length} noop item(s)`);

  // If in staged mode, emit step summary instead of logging
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: No-Op Messages Preview\n\n";
    summaryContent += "The following messages would be logged if staged mode was disabled:\n\n";

    for (let i = 0; i < noopItems.length; i++) {
      const item = noopItems[i];
      summaryContent += `### Message ${i + 1}\n`;
      summaryContent += `${item.message}\n\n`;
      summaryContent += "---\n\n";
    }

    await core.summary.addRaw(summaryContent).write();
    core.info("📝 No-op message preview written to step summary");
    return;
  }

  // Process each noop item - just log the messages for transparency
  let summaryContent = "\n\n## No-Op Messages\n\n";
  summaryContent += "The following messages were logged for transparency:\n\n";

  for (let i = 0; i < noopItems.length; i++) {
    const item = noopItems[i];
    core.info(`No-op message ${i + 1}: ${item.message}`);
    summaryContent += `- ${item.message}\n`;
  }

  // Write summary for all noop messages
  await core.summary.addRaw(summaryContent).write();

  // Export the first noop message for use in add-comment default reporting
  if (noopItems.length > 0) {
    core.setOutput("noop_message", noopItems[0].message);
    core.exportVariable("GH_AW_NOOP_MESSAGE", noopItems[0].message);
  }

  core.info(`Successfully processed ${noopItems.length} noop message(s)`);
}

await main();
