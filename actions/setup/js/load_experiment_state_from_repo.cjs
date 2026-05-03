// @ts-check
/// <reference types="@actions/github-script" />

/**
 * load_experiment_state_from_repo
 *
 * Fetches the experiment state file from a git branch using the GitHub API and writes
 * it to the local experiments directory so that pick_experiment.cjs can read it.
 *
 * Falls back gracefully to an empty state when the branch or file does not yet exist
 * (first run), or when any other error occurs while fetching the file.
 *
 * Environment variables (set by the compiled workflow step):
 *   GH_AW_EXPERIMENT_STATE_FILE - Absolute path to the local state file to write
 *                                  e.g. /tmp/gh-aw/experiments/state.json
 *   GH_AW_EXPERIMENT_STATE_DIR  - Directory that holds the state file (created if missing)
 *                                  e.g. /tmp/gh-aw/experiments
 *   GH_AW_EXPERIMENT_BRANCH     - Git branch name to fetch state from
 *                                  e.g. experiments/myworkflow
 */

const fs = require("fs");
const path = require("path");

/**
 * Fetch experiment state from the git branch via the GitHub API.
 * Returns the raw file content as a string, or null when the branch/file is absent.
 *
 * @param {any} octokit - Authenticated Octokit instance
 * @param {string} owner - Repository owner
 * @param {string} repo  - Repository name
 * @param {string} branch - Branch name (e.g. "experiments/myworkflow")
 * @param {string} filePath - File path within the branch (e.g. "state.json")
 * @returns {Promise<string|null>}
 */
async function fetchFileFromBranch(octokit, owner, repo, branch, filePath) {
  try {
    const response = await octokit.rest.repos.getContent({
      owner,
      repo,
      path: filePath,
      ref: branch,
    });
    const data = response.data;
    if (Array.isArray(data) || data.type !== "file" || !data.content) {
      return null;
    }
    // GitHub API returns base64-encoded content.
    return Buffer.from(data.content, "base64").toString("utf8");
  } catch (/** @type {any} */ err) {
    // 404 means the branch or file does not exist yet – that is normal on first run.
    if (err.status === 404) {
      return null;
    }
    throw err;
  }
}

/**
 * Main entry point called by the actions/github-script step.
 */
async function main() {
  const stateFile = process.env.GH_AW_EXPERIMENT_STATE_FILE || "/tmp/gh-aw/experiments/state.json";
  const stateDir = process.env.GH_AW_EXPERIMENT_STATE_DIR || "/tmp/gh-aw/experiments";
  const branch = process.env.GH_AW_EXPERIMENT_BRANCH || "";

  if (!branch) {
    core.warning("GH_AW_EXPERIMENT_BRANCH is not set – starting with empty experiment state");
    fs.mkdirSync(stateDir, { recursive: true });
    return;
  }

  const [owner, repo] = (process.env.GITHUB_REPOSITORY || "/").split("/");
  if (!owner || !repo) {
    core.warning("GITHUB_REPOSITORY is not set – starting with empty experiment state");
    fs.mkdirSync(stateDir, { recursive: true });
    return;
  }

  // Use the authenticated `github` client provided by actions/github-script (via setupGlobals).
  // This avoids requiring GITHUB_TOKEN to be explicitly set in the step env.
  const octokit = github;
  const stateFileName = path.basename(stateFile);

  core.info(`Loading experiment state from branch "${branch}" (file: ${stateFileName})`);

  let content = null;
  try {
    content = await fetchFileFromBranch(octokit, owner, repo, branch, stateFileName);
  } catch (/** @type {any} */ err) {
    core.warning(`Failed to fetch experiment state from branch "${branch}": ${err.message} – starting fresh`);
  }

  // Ensure the directory exists regardless of whether we fetched the file.
  fs.mkdirSync(stateDir, { recursive: true });

  if (content === null) {
    core.info(`No experiment state found in branch "${branch}" – starting with empty state`);
    return;
  }

  // Validate that the content is parseable JSON before writing.
  try {
    const parsed = JSON.parse(content);
    if (!parsed || typeof parsed.counts !== "object") {
      core.warning(`Experiment state in branch "${branch}" is invalid JSON – starting fresh`);
      return;
    }
  } catch {
    core.warning(`Experiment state in branch "${branch}" could not be parsed – starting fresh`);
    return;
  }

  fs.writeFileSync(stateFile, content, "utf8");
  core.info(`Experiment state written to ${stateFile}`);
}

module.exports = { main, fetchFileFromBranch };
