// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");

/**
 * Determine if the conclusion job should fail based on safe output entries.
 *
 * This script checks if the agent produced any meaningful safe outputs.
 * If no safe outputs exist (other than "noop"), it indicates that either:
 * - The safe output server failed to run
 * - The prompt completely failed to generate any meaningful result
 *
 * In such cases, the agent should have called "noop" to explicitly indicate
 * no action was taken. The absence of any outputs is considered a failure.
 *
 * The check only applies when the agent job succeeded. If the agent job
 * already failed, we don't need to check safe outputs.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "";

  core.info(`Checking if conclusion job should fail`);
  core.info(`Agent conclusion: ${agentConclusion}`);

  // Only check safe outputs if the agent job succeeded
  // If agent already failed, no need to check outputs
  if (agentConclusion !== "success") {
    core.info(`Agent job did not succeed (conclusion: ${agentConclusion}), skipping safe output check`);
    return;
  }

  // Load agent output to check for safe output entries
  const agentOutputResult = loadAgentOutput();

  if (!agentOutputResult.success) {
    // No agent output file or failed to load - this is a failure
    core.error("No agent output found. The workflow should have produced safe outputs or called noop.");
    core.setFailed("Workflow run failed: No safe outputs were generated. The agent should produce outputs or explicitly call noop.");
    return;
  }

  const items = agentOutputResult.items;
  core.info(`Found ${items.length} safe output item(s)`);

  // Filter out "noop" entries to check if there are any actual outputs
  const nonNoopItems = items.filter(item => item.type !== "noop");

  if (nonNoopItems.length === 0 && items.length === 0) {
    // No items at all (including noop) - this is a failure
    core.error("No safe output entries found. The workflow should have produced outputs or called noop.");
    core.setFailed("Workflow run failed: No safe outputs were generated. The agent should produce outputs or explicitly call noop.");
    return;
  }

  core.info(`Found ${nonNoopItems.length} non-noop safe output item(s)`);
  core.info("Safe output check passed - workflow produced meaningful outputs");
}

module.exports = { main };
