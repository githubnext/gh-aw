// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
const os = require("os");
const nodePath = require("path");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { getGitAuthEnv } = require("./git_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { pushSignedCommits } = require("./push_signed_commits.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "edit_wiki";

/**
 * Main handler factory for edit_wiki
 * Returns a message handler function that processes individual edit_wiki messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const ifNoChanges = config.if_no_changes || "warn";
  const commitTitleSuffix = config.commit_title_suffix || "";
  const maxSizeKb = config.max_patch_size ? parseInt(String(config.max_patch_size), 10) : 1024;
  const maxCount = config.max || 0; // 0 means no limit

  // Cross-repo support: resolve target repository from config
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);

  // Build git auth env once for all network operations in this handler.
  const gitAuthEnv = getGitAuthEnv(config["github-token"]);

  // Create authenticated GitHub client (same pattern as push_to_pull_request_branch).
  // Used by pushSignedCommits for the signed-commit GraphQL path and fallback git push.
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

  core.info(`If no changes: ${ifNoChanges}`);
  if (commitTitleSuffix) {
    core.info(`Commit title suffix: ${commitTitleSuffix}`);
  }
  core.info(`Max patch size: ${maxSizeKb} KB`);
  core.info(`Max count: ${maxCount || "unlimited"}`);
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function - processes individual edit_wiki messages
   * @param {any} message - The edit_wiki message to process
   * @param {import('./types/handler-factory').ResolvedTemporaryIds} resolvedTemporaryIds - Map of temporary IDs to resolved IDs
   * @returns {Promise<import('./types/handler-factory').HandlerResult>}
   */
  return async function handleEditWiki(message, resolvedTemporaryIds) {
    // Check max count
    if (maxCount > 0 && processedCount >= maxCount) {
      core.info(`Skipping message - max count (${maxCount}) reached`);
      return { success: false, error: `Max count (${maxCount}) reached`, skipped: true };
    }

    processedCount++;

    // Determine the patch file path from the message
    const patchFilePath = message.patch_path;
    core.info(`Patch file path: ${patchFilePath || "(not set)"}`);

    // Check if patch file exists and has valid content
    if (!patchFilePath || !fs.existsSync(patchFilePath)) {
      const msg = "No patch file found - cannot push wiki changes without a patch";

      switch (ifNoChanges) {
        case "error":
          return { success: false, error: msg };
        case "ignore":
          return { success: false, error: msg, skipped: true };
        case "warn":
        default:
          core.info(msg);
          return { success: false, error: msg, skipped: true };
      }
    }

    const patchContent = fs.readFileSync(patchFilePath, "utf8");

    // Check for actual error conditions
    if (patchContent.includes("Failed to generate patch")) {
      const msg = "Patch file contains error message - cannot push wiki changes";
      core.error("Patch file generation failed");
      core.error(`Patch file location: ${patchFilePath}`);
      return { success: false, error: msg };
    }

    const isEmpty = !patchContent || !patchContent.trim();

    // Validate patch size
    if (!isEmpty) {
      const patchSizeBytes = Buffer.byteLength(patchContent, "utf8");
      const patchSizeKb = Math.ceil(patchSizeBytes / 1024);

      const diffSizeBytesRaw = message.diff_size;
      const haveDiffSize = typeof diffSizeBytesRaw === "number" && diffSizeBytesRaw >= 0;

      let sizeForCheckBytes;
      let sizeLabel;
      if (haveDiffSize) {
        sizeForCheckBytes = diffSizeBytesRaw;
        sizeLabel = "Incremental diff size";
      } else {
        sizeForCheckBytes = patchSizeBytes;
        sizeLabel = "Patch size";
      }
      const sizeForCheckKb = Math.ceil(sizeForCheckBytes / 1024);

      core.info(`Patch file size: ${patchSizeKb} KB`);
      core.info(`${sizeLabel}: ${sizeForCheckKb} KB (maximum allowed: ${maxSizeKb} KB)`);

      if (sizeForCheckKb > maxSizeKb) {
        const msg = `${sizeLabel} (${sizeForCheckKb} KB) exceeds maximum allowed size (${maxSizeKb} KB)`;
        return { success: false, error: msg };
      }

      core.info("Patch size validation passed");
    }

    if (isEmpty) {
      const msg = "Patch file is empty - no changes to apply to wiki";

      switch (ifNoChanges) {
        case "error":
          return { success: false, error: "No wiki changes to push - failing as configured by if-no-changes: error" };
        case "ignore":
          return { success: false, error: msg, skipped: true };
        case "warn":
        default:
          core.info(msg);
          return { success: false, error: msg, skipped: true };
      }
    }

    // If in staged mode, emit staged preview
    if (isStaged) {
      await generateStagedPreview({
        title: "Edit Wiki",
        description: "The following wiki changes would be pushed if staged mode was disabled:",
        items: [{ commit_message: message.message || message.commit_message || "(no message)" }],
        renderItem: item => {
          let content = `**Wiki:** ${defaultTargetRepo || process.env.GITHUB_REPOSITORY || "(current repo)"}\n\n`;
          if (item.commit_message) {
            content += `**Commit Message:** ${item.commit_message}\n\n`;
          }
          if (patchFilePath && fs.existsSync(patchFilePath)) {
            const patchStats = fs.readFileSync(patchFilePath, "utf8");
            if (patchStats.trim()) {
              content += `**Changes:** Patch file exists with ${patchStats.split("\n").length} lines\n\n`;
              content += `<details><summary>Show patch preview</summary>\n\n\`\`\`diff\n${patchStats.slice(0, 2000)}${patchStats.length > 2000 ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>\n\n`;
            } else {
              content += `**Changes:** No changes (empty patch)\n\n`;
            }
          }
          return content;
        },
      });
      return { success: true, staged: true };
    }

    // Resolve and validate target repository
    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "edit wiki");
    if (!repoResult.success) {
      return { success: false, error: repoResult.error };
    }
    const itemRepo = repoResult.repo;
    const repoParts = repoResult.repoParts;

    core.info(`Target repository: ${itemRepo}`);

    // Build the wiki git URL
    const serverUrl = (process.env.GITHUB_SERVER_URL || "https://github.com").replace(/\/$/, "");
    const wikiGitUrl = `${serverUrl}/${repoParts.owner}/${repoParts.repo}.wiki.git`;
    core.info(`Wiki git URL: ${wikiGitUrl.replace(/:\/\/[^@]+@/, "://***@")}`);

    // Clone wiki to a temp directory
    const wikiCloneDir = fs.mkdtempSync(nodePath.join(os.tmpdir(), "gh-aw-wiki-"));
    core.info(`Cloning wiki to: ${wikiCloneDir}`);

    try {
      // Clone the wiki repo using auth env vars for authentication
      const cloneResult = await exec.getExecOutput("git", ["clone", wikiGitUrl, wikiCloneDir], {
        env: { ...process.env, ...gitAuthEnv },
        ignoreReturnCode: true,
      });

      if (cloneResult.exitCode !== 0) {
        const stderr = (cloneResult.stderr || "").trim();
        // Distinguish empty wiki (no commits yet) from a real error
        if (stderr.includes("You appear to have cloned an empty repository") || stderr.includes("warning: You appear to have cloned an empty repository")) {
          core.info("Wiki repository is empty - will initialize with first commit");
          // Initialize a new git repo in the clone dir
          await exec.exec("git", ["init", wikiCloneDir]);
          await exec.exec("git", ["remote", "add", "origin", wikiGitUrl], { cwd: wikiCloneDir, env: { ...process.env, ...gitAuthEnv } });
        } else {
          return {
            success: false,
            error: `Failed to clone wiki repository: ${stderr || `git clone exited with code ${cloneResult.exitCode}`}`,
          };
        }
      }

      core.info("Wiki cloned successfully");

      // Configure git identity in the wiki clone
      await exec.exec("git", ["config", "user.email", "github-actions[bot]@users.noreply.github.com"], { cwd: wikiCloneDir });
      await exec.exec("git", ["config", "user.name", "github-actions[bot]"], { cwd: wikiCloneDir });
      await exec.exec("git", ["config", "am.keepcr", "true"], { cwd: wikiCloneDir });

      // Determine the current branch name (wiki default is typically 'master')
      let wikiBranch = "master";
      try {
        const branchResult = await exec.getExecOutput("git", ["rev-parse", "--abbrev-ref", "HEAD"], {
          cwd: wikiCloneDir,
          ignoreReturnCode: true,
        });
        if (branchResult.exitCode === 0 && branchResult.stdout.trim() && branchResult.stdout.trim() !== "HEAD") {
          wikiBranch = branchResult.stdout.trim();
        }
      } catch {
        // Use default 'master'
      }
      core.info(`Wiki branch: ${wikiBranch}`);

      // Apply patch to wiki clone
      let patchFileToApply = patchFilePath;

      if (commitTitleSuffix) {
        core.info(`Appending commit title suffix: "${commitTitleSuffix}"`);
        let patchFileContent = fs.readFileSync(patchFilePath, "utf8");
        patchFileContent = patchFileContent.replace(/^Subject: (\[PATCH[^\]]*\] )?(.*)$/gm, (match, patchPrefix, title) => `Subject: ${patchPrefix || "[PATCH] "}${title}${commitTitleSuffix}`);
        patchFileToApply = nodePath.join(wikiCloneDir, "..", `aw-wiki-modified-${Date.now()}.patch`);
        fs.writeFileSync(patchFileToApply, patchFileContent, "utf8");
        core.info(`Patch modified with commit title suffix`);
      }

      // Log first 100 lines of patch for debugging
      const finalPatchContent = fs.readFileSync(patchFileToApply, "utf8");
      const patchLines = finalPatchContent.split("\n");
      const previewLineCount = Math.min(100, patchLines.length);
      core.info(`Patch preview (first ${previewLineCount} of ${patchLines.length} lines):`);
      for (let i = 0; i < previewLineCount; i++) {
        core.info(patchLines[i]);
      }

      // Apply the patch with git am --3way
      // --3way handles cases where the patch base may differ from the wiki state
      const amResult = await exec.getExecOutput("git", ["am", "--3way", patchFileToApply], {
        cwd: wikiCloneDir,
        ignoreReturnCode: true,
      });

      if (amResult.exitCode !== 0) {
        const amError = (amResult.stderr || amResult.stdout || "").trim();
        core.error(`Failed to apply patch to wiki: ${amError}`);

        // Log debug info
        try {
          const statusResult = await exec.getExecOutput("git", ["status"], { cwd: wikiCloneDir });
          core.info(`Git status:\n${statusResult.stdout}`);
        } catch {
          // Non-fatal
        }

        // Abort git am in case of partial apply
        try {
          await exec.exec("git", ["am", "--abort"], { cwd: wikiCloneDir });
        } catch {
          // Ignore abort errors
        }

        return { success: false, error: `Failed to apply patch to wiki: ${amError || "git am failed"}` };
      }

      core.info("Patch applied to wiki successfully");

      // Configure an authenticated remote URL before pushing so that git push
      // works reliably from a fresh clone. This mirrors what actions/checkout does
      // internally (https://x-access-token:TOKEN@...) and ensures the push succeeds
      // even when GIT_CONFIG_* environment variables are not propagated correctly to
      // child processes in the github-script execution environment.
      const authToken = config["github-token"] || process.env.GITHUB_TOKEN;
      if (authToken) {
        core.setSecret(authToken);
        const serverUrlHost = serverUrl.replace(/^https?:\/\//, "").replace(/\/$/, "");
        const authWikiUrl = `https://x-access-token:${authToken}@${serverUrlHost}/${repoParts.owner}/${repoParts.repo}.wiki.git`;
        // Use silent: true to suppress any command logging that could expose the token
        // (core.setSecret also masks the value, but defense-in-depth)
        const setUrlResult = await exec.getExecOutput("git", ["remote", "set-url", "origin", authWikiUrl], {
          cwd: wikiCloneDir,
          silent: true,
          ignoreReturnCode: true,
        });
        if (setUrlResult.exitCode !== 0) {
          core.warning(`Failed to set authenticated remote URL: ${(setUrlResult.stderr || "").trim()}`);
        }
      }

      // Push commits using pushSignedCommits (the same helper used by push_to_pull_request_branch).
      // Wiki repos are not accessible via the GitHub GraphQL createCommitOnBranch mutation,
      // so the GraphQL path will fail gracefully and the fallback git push will be used.
      core.info(`Pushing wiki changes to: ${wikiBranch}`);
      let commitSha = "";
      try {
        const pushedSha = await pushSignedCommits({
          githubClient,
          owner: repoParts.owner,
          repo: `${repoParts.repo}.wiki`,
          branch: wikiBranch,
          baseRef: `origin/${wikiBranch}`,
          cwd: wikiCloneDir,
          gitAuthEnv,
        });
        if (pushedSha) {
          commitSha = pushedSha;
        }
        core.info("Wiki changes pushed successfully");
      } catch (pushError) {
        const pushErrorMessage = getErrorMessage(pushError);
        core.error(`Failed to push wiki changes: ${pushErrorMessage}`);
        return { success: false, error: `Failed to push wiki changes: ${pushErrorMessage}` };
      }

      // Get the commit SHA for the activation comment update if not set by pushSignedCommits
      if (!commitSha) {
        try {
          const shaResult = await exec.getExecOutput("git", ["rev-parse", "HEAD"], { cwd: wikiCloneDir });
          commitSha = shaResult.stdout.trim();
        } catch {
          // Non-fatal
        }
      }

      const wikiUrl = `${serverUrl}/${repoParts.owner}/${repoParts.repo}/wiki`;
      core.info(`Wiki URL: ${wikiUrl}`);

      return {
        success: true,
        url: wikiUrl,
        sha: commitSha,
      };
    } finally {
      // Clean up wiki clone directory
      try {
        fs.rmSync(wikiCloneDir, { recursive: true, force: true });
        core.info("Cleaned up wiki clone directory");
      } catch (cleanupError) {
        core.warning(`Failed to clean up wiki clone directory: ${getErrorMessage(cleanupError)}`);
      }
    }
  };
}

module.exports = { main };
