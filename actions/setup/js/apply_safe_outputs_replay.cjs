// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Apply Safe Outputs Replay Driver
 *
 * Downloads the agent output artifact from a previous workflow run and replays
 * the safe outputs, applying them to the repository.
 *
 * Called from the `apply_safe_outputs` job in the agentic-maintenance workflow.
 *
 * Required environment variables:
 *   GH_AW_RUN_URL   - Run URL or run ID to replay safe outputs from.
 *                     Accepts a full URL (https://github.com/{owner}/{repo}/actions/runs/{runId})
 *                     or a plain run ID (digits only).
 *   GH_TOKEN        - GitHub token for artifact download via `gh run download`.
 *
 * Optional environment variables:
 *   GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG - If set, overrides the auto-generated handler config.
 */

const fs = require("fs");
const path = require("path");

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_CONFIG, ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");
const { AGENT_OUTPUT_FILENAME, TMP_GH_AW_PATH } = require("./constants.cjs");

/**
 * Parse a run ID from a run URL or plain run ID string.
 *
 * Accepts:
 *   - A plain run ID: "23560193313"
 *   - A full run URL: "https://github.com/{owner}/{repo}/actions/runs/{runId}"
 *   - A run URL with job: "https://github.com/{owner}/{repo}/actions/runs/{runId}/job/{jobId}"
 *
 * @param {string} runUrl - The run URL or run ID to parse
 * @returns {{ runId: string, owner: string|null, repo: string|null }} Parsed components
 */
function parseRunUrl(runUrl) {
  if (!runUrl || typeof runUrl !== "string") {
    throw new Error(`${ERR_VALIDATION}: run_url is required`);
  }

  const trimmed = runUrl.trim();

  // Check if it's a plain run ID (digits only)
  if (/^\d+$/.test(trimmed)) {
    return { runId: trimmed, owner: null, repo: null };
  }

  // Parse a full GitHub Actions URL
  // Pattern: https://github.com/{owner}/{repo}/actions/runs/{runId}[/job/{jobId}]
  const match = trimmed.match(/github\.com\/([^/]+)\/([^/]+)\/actions\/runs\/(\d+)/);
  if (match) {
    return { runId: match[3], owner: match[1], repo: match[2] };
  }

  throw new Error(`${ERR_VALIDATION}: Cannot parse run ID from: ${trimmed}. Expected a plain run ID (digits only) or a GitHub Actions run URL (https://github.com/{owner}/{repo}/actions/runs/{runId}).`);
}

/**
 * Download the agent artifact from a workflow run using `gh run download`.
 *
 * @param {string} runId - The workflow run ID
 * @param {string} destDir - Destination directory for the downloaded artifact
 * @param {string|null} repoSlug - Optional repository slug (owner/repo)
 * @returns {Promise<string>} Path to the downloaded agent_output.json file
 */
async function downloadAgentArtifact(runId, destDir, repoSlug) {
  core.info(`Downloading agent artifact from run ${runId}...`);

  fs.mkdirSync(destDir, { recursive: true });

  const args = ["run", "download", runId, "--name", "agent", "--dir", destDir];
  if (repoSlug) {
    args.push("--repo", repoSlug);
  }

  const exitCode = await exec.exec("gh", args);
  if (exitCode !== 0) {
    throw new Error(`${ERR_SYSTEM}: Failed to download agent artifact from run ${runId}`);
  }

  const outputFile = path.join(destDir, AGENT_OUTPUT_FILENAME);
  if (!fs.existsSync(outputFile)) {
    throw new Error(`${ERR_SYSTEM}: Agent output file not found at ${outputFile} after download. Ensure run ${runId} has an "agent" artifact containing ${AGENT_OUTPUT_FILENAME}.`);
  }

  core.info(`✓ Agent artifact downloaded to ${outputFile}`);
  return outputFile;
}

/**
 * Build a handler config from the items present in the agent output file.
 * Each item type found in the output is enabled (with an empty config object).
 *
 * @param {string} agentOutputFile - Path to the agent_output.json file
 * @returns {Object} Handler config keyed by normalized type name
 */
function buildHandlerConfigFromOutput(agentOutputFile) {
  const content = fs.readFileSync(agentOutputFile, "utf8");
  const validatedOutput = JSON.parse(content);

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No items found in agent output; handler config will be empty");
    return {};
  }

  // Normalize type: convert dashes to underscores (mirrors safe_outputs_append.cjs)
  const config = Object.fromEntries(validatedOutput.items.filter(item => item.type && typeof item.type === "string").map(item => [item.type.replace(/-/g, "_"), {}]));

  core.info(`Handler config built from ${validatedOutput.items.length} item(s): ${Object.keys(config).join(", ")}`);
  return config;
}

/**
 * Download the agent artifact from a previous run and apply the safe outputs.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const runUrl = process.env.GH_AW_RUN_URL;
  if (!runUrl) {
    core.setFailed(`${ERR_CONFIG}: GH_AW_RUN_URL environment variable is required but not set`);
    return;
  }

  core.info(`Applying safe outputs from run: ${runUrl}`);

  // Parse run ID and optional owner/repo from the URL
  let runId, owner, repo;
  try {
    ({ runId, owner, repo } = parseRunUrl(runUrl));
  } catch (error) {
    core.setFailed(getErrorMessage(error));
    return;
  }

  core.info(`Parsed run ID: ${runId}`);

  // repoFromUrl is non-null only when the URL explicitly specified an owner/repo.
  // displayRepoSlug falls back to the current workflow context for logging.
  const repoFromUrl = owner && repo ? `${owner}/${repo}` : null;
  const displayRepoSlug = repoFromUrl ?? `${context.repo.owner}/${context.repo.repo}`;
  core.info(`Target repository: ${displayRepoSlug}`);

  // Download the agent artifact into /tmp/gh-aw/
  const destDir = TMP_GH_AW_PATH;
  let agentOutputFile;
  try {
    agentOutputFile = await downloadAgentArtifact(runId, destDir, repoFromUrl);
  } catch (error) {
    core.setFailed(getErrorMessage(error));
    return;
  }

  // Set GH_AW_AGENT_OUTPUT so the handler manager can find the output file
  process.env.GH_AW_AGENT_OUTPUT = agentOutputFile;
  core.info(`Set GH_AW_AGENT_OUTPUT=${agentOutputFile}`);

  // Auto-build GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG from the output if not already set
  if (!process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG) {
    try {
      const handlerConfig = buildHandlerConfigFromOutput(agentOutputFile);
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify(handlerConfig);
      core.info("Auto-configured GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG from agent output");
    } catch (error) {
      core.setFailed(`Failed to build handler config: ${getErrorMessage(error)}`);
      return;
    }
  }

  // Apply safe outputs via the handler manager
  core.info("Applying safe outputs...");
  const { main: runHandlerManager } = require("./safe_output_handler_manager.cjs");
  await runHandlerManager();
}

module.exports = { main, parseRunUrl, buildHandlerConfigFromOutput };
