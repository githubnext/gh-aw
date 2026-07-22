// @ts-check
/// <reference types="@actions/github-script" />

/**
 * push_experiment_state
 *
 * Commits state files to a git
 * branch using the GitHub GraphQL `createCommitOnBranch` mutation so commits
 * are cryptographically signed (verified) by GitHub. Falls back to a plain
 * `git push` via pushSignedCommits when the GraphQL path is unavailable.
 *
 * Generic environment variables:
 *   GH_AW_STATE_DIR             - Directory containing files to commit
 *   GH_AW_STATE_BRANCH          - Target git branch
 *   GH_AW_STATE_FILES           - Comma-separated filenames to copy from GH_AW_STATE_DIR
 *   GH_AW_STATE_LABEL           - Human-readable label used in logs/messages
 *
 * Backward-compatible experiment aliases:
 *   GH_AW_EXPERIMENT_STATE_DIR  - Directory containing state.json / assignments.json
 *   GH_AW_EXPERIMENT_BRANCH     - Target git branch for experiment state
 *   GH_TOKEN / GITHUB_TOKEN     - GitHub token for API access and git operations
 *   GITHUB_RUN_ID               - Run ID used in commit messages
 *   GITHUB_SERVER_URL           - GitHub server URL (defaults to https://github.com)
 *   GITHUB_REPOSITORY           - "owner/repo" of the current repository
 */

const fs = require("fs");
const path = require("path");

const { getErrorMessage } = require("./error_helpers.cjs");
const { execGitSync, getGitAuthEnv } = require("./git_helpers.cjs");
const { pushSignedCommits } = require("./push_signed_commits.cjs");

/**
 * Checkout or create an orphan git branch for experiment state.
 * Returns the remote HEAD SHA (empty string for a new branch).
 *
 * @param {string} branchName - Target branch name (e.g. "experiments/myworkflow")
 * @param {string} repoUrl    - Authenticated HTTPS URL of the target repo
 * @param {string} workspaceDir - Local git workspace directory
 * @returns {string} baseRef (empty string when branch is brand new)
 */
function checkoutOrCreateBranch(branchName, repoUrl, workspaceDir) {
  try {
    execGitSync(["fetch", repoUrl, `${branchName}:${branchName}`], { stdio: "pipe", cwd: workspaceDir, suppressLogs: true });
    execGitSync(["checkout", branchName], { stdio: "inherit", cwd: workspaceDir });
    const baseRef = execGitSync(["rev-parse", "HEAD"], { cwd: workspaceDir }).trim();
    core.info(`Checked out existing branch ${branchName}, baseRef=${baseRef}`);
    return baseRef;
  } catch (fetchErr) {
    const msg = getErrorMessage(fetchErr);
    const isMissing = /couldn't find remote ref/i.test(msg) || /remote branch .* not found/i.test(msg);
    if (!isMissing) throw fetchErr;

    // Branch does not exist yet – create an orphan branch.
    core.info(`Branch ${branchName} does not exist, creating orphan branch...`);
    execGitSync(["checkout", "--orphan", branchName], { stdio: "inherit", cwd: workspaceDir });
    execGitSync(["read-tree", "--empty"], { stdio: "pipe", cwd: workspaceDir });
    // Remove any pre-existing working-tree files (from sparse checkout).
    for (const entry of fs.readdirSync(workspaceDir)) {
      if (entry !== ".git") {
        fs.rmSync(path.join(workspaceDir, entry), { recursive: true, force: true });
      }
    }
    return "";
  }
}

/**
 * Main entry point called by the actions/github-script step.
 */
async function main() {
  const stateDir = process.env.GH_AW_STATE_DIR || process.env.GH_AW_EXPERIMENT_STATE_DIR || "/tmp/gh-aw/experiments";
  const branchName = process.env.GH_AW_STATE_BRANCH || process.env.GH_AW_EXPERIMENT_BRANCH || "";
  const stateLabel = process.env.GH_AW_STATE_LABEL || "experiment state";
  const filesEnv = process.env.GH_AW_STATE_FILES || "state.json,assignments.json";
  const candidateFiles = filesEnv
    .split(",")
    .map(name => name.trim())
    .filter(Boolean);
  const ghToken = process.env.GH_TOKEN || process.env.GITHUB_TOKEN || "";
  const githubRunId = process.env.GITHUB_RUN_ID || "unknown";
  const githubServerUrl = (process.env.GITHUB_SERVER_URL || "https://github.com").replace(/\/$/, "");
  const serverHost = githubServerUrl.replace(/^https?:\/\//, "");

  if (!branchName) {
    core.setFailed("GH_AW_STATE_BRANCH (or GH_AW_EXPERIMENT_BRANCH) is not set");
    return;
  }
  if (!ghToken) {
    core.setFailed("GH_TOKEN or GITHUB_TOKEN is not set");
    return;
  }

  const targetRepo = `${context.repo.owner}/${context.repo.repo}`;
  const allowedRepos = new Set(
    (process.env.GH_AW_ALLOWED_TARGET_REPOS || targetRepo)
      .split(",")
      .map(repo => repo.trim())
      .filter(Boolean)
  );
  if (!allowedRepos.has(targetRepo)) {
    core.setFailed(`Target repository "${targetRepo}" is not in GH_AW_ALLOWED_TARGET_REPOS. ` + `Current allowlist: ${Array.from(allowedRepos).join(", ")}`);
    return;
  }
  const [owner, repo] = targetRepo.split("/");

  core.info(`Pushing ${stateLabel} to branch "${branchName}" in ${targetRepo}`);

  // Collect the JSON files that exist in the state directory.
  const filesToPush = candidateFiles.filter(name => {
    const full = path.join(stateDir, name);
    return fs.existsSync(full) && fs.statSync(full).isFile();
  });

  if (filesToPush.length === 0) {
    core.info(`No ${stateLabel} files found – nothing to push`);
    return;
  }

  core.info(`Files to push: ${filesToPush.join(", ")}`);

  const workspaceDir = process.env.GITHUB_WORKSPACE || process.cwd();
  const repoUrl = `https://x-access-token:${ghToken}@${serverHost}/${targetRepo}.git`;

  // Checkout the target branch (or create it as an orphan on first run).
  let baseRef;
  try {
    baseRef = checkoutOrCreateBranch(branchName, repoUrl, workspaceDir);
  } catch (err) {
    core.setFailed(`Failed to checkout branch "${branchName}": ${getErrorMessage(err)}`);
    return;
  }

  // Copy state files into the workspace root.
  for (const name of filesToPush) {
    const src = path.join(stateDir, name);
    const dest = path.join(workspaceDir, name);
    try {
      fs.copyFileSync(src, dest);
      core.info(`Copied ${name}`);
    } catch (err) {
      core.setFailed(`Failed to copy ${name}: ${getErrorMessage(err)}`);
      return;
    }
  }

  // Stage all changes.
  try {
    execGitSync(["add", "--sparse", "."], { stdio: "inherit", cwd: workspaceDir });
  } catch (err) {
    core.setFailed(`Failed to stage changes: ${getErrorMessage(err)}`);
    return;
  }

  // Check whether there are any staged changes to commit.
  const status = execGitSync(["status", "--porcelain"], { cwd: workspaceDir }).trim();
  if (!status) {
    core.info(`No changes to ${stateLabel} – skipping push`);
    return;
  }

  // Commit.
  try {
    execGitSync(["commit", "-m", `Update ${stateLabel} from workflow run ${githubRunId}`], { stdio: "inherit", cwd: workspaceDir });
  } catch (err) {
    core.setFailed(`Failed to commit ${stateLabel}: ${getErrorMessage(err)}`);
    return;
  }

  // Point origin at the target repo so pushSignedCommits can resolve the remote branch HEAD.
  execGitSync(["remote", "set-url", "origin", `https://${serverHost}/${targetRepo}.git`], { stdio: "pipe", cwd: workspaceDir });

  // Push using GraphQL createCommitOnBranch (signed commits) with a plain-git fallback.
  const MAX_RETRIES = 3;
  const BASE_DELAY_MS = 1000;
  let currentBaseRef = baseRef;

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    core.info(`Pushing to ${branchName} (attempt ${attempt + 1}/${MAX_RETRIES + 1})...`);
    try {
      await pushSignedCommits({
        githubClient: github,
        owner,
        repo,
        branch: branchName,
        baseRef: currentBaseRef,
        cwd: workspaceDir,
        gitAuthEnv: getGitAuthEnv(ghToken),
      });
      core.info(`Successfully pushed ${stateLabel} to ${branchName}`);
      return;
    } catch (err) {
      const errMsg = getErrorMessage(err);
      if (attempt < MAX_RETRIES) {
        const delay = BASE_DELAY_MS * Math.pow(2, attempt);
        core.warning(`Push failed (attempt ${attempt + 1}/${MAX_RETRIES + 1}), retrying in ${delay}ms: ${errMsg}`);
        await new Promise(resolve => setTimeout(resolve, delay));

        // Refresh baseRef and fetch the updated remote history so that
        // pushSignedCommits can resolve the new baseRef in git rev-list.
        try {
          const { stdout: lsOut } = await exec.getExecOutput("git", ["ls-remote", "origin", `refs/heads/${branchName}`], { cwd: workspaceDir });
          const remoteHead = lsOut.trim().split(/\s+/)[0] || "";
          if (remoteHead && remoteHead !== currentBaseRef) {
            currentBaseRef = remoteHead;
            core.info(`Refreshed baseRef for retry: ${currentBaseRef}`);
            // Fetch the updated branch history into the local repo so pushSignedCommits
            // can resolve currentBaseRef in `git rev-list baseRef..HEAD`.
            try {
              execGitSync(["fetch", "origin", `refs/heads/${branchName}`], { stdio: "pipe", cwd: workspaceDir, suppressLogs: true });
            } catch (fetchErr) {
              core.info(`Fetch of branch "${branchName}" on retry failed (non-fatal): ${getErrorMessage(fetchErr)}`);
            }
          }
        } catch {
          // ls-remote failed; keep existing baseRef
        }
      } else {
        core.setFailed(`Failed to push ${stateLabel} after ${MAX_RETRIES + 1} attempts: ${errMsg}`);
        return;
      }
    }
  }
}

module.exports = { main, checkoutOrCreateBranch };
