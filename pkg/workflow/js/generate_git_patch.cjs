// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Extract branch name from safe-outputs JSONL file
 * @param {string} safeOutputsPath - Path to the safe-outputs JSONL file
 * @returns {string} Extracted branch name or empty string
 */
function extractBranchFromSafeOutputs(safeOutputsPath) {
  const fs = require("fs");

  if (!fs.existsSync(safeOutputsPath)) {
    return "";
  }

  try {
    const content = fs.readFileSync(safeOutputsPath, "utf8");
    const lines = content.split("\n");

    for (const line of lines) {
      if (!line.trim()) {
        continue;
      }

      try {
        const entry = JSON.parse(line);

        // Check for create_pull_request or push_to_pull_request_branch types
        if (entry.type === "create_pull_request" || entry.type === "push_to_pull_request_branch") {
          if (entry.branch) {
            core.info(`Found ${entry.type} line with branch: ${entry.branch}`);
            return entry.branch;
          }
        }
      } catch (parseError) {
        // Skip invalid JSON lines
        continue;
      }
    }
  } catch (readError) {
    core.warning(`Failed to read safe-outputs file: ${readError instanceof Error ? readError.message : String(readError)}`);
  }

  return "";
}

/**
 * Determine which branch to use for patch generation
 * @param {string} branchFromSafeOutputs - Branch name from safe-outputs
 * @returns {Promise<string>} Target branch name or empty string
 */
async function determineTargetBranch(branchFromSafeOutputs) {
  if (branchFromSafeOutputs) {
    core.info(`Branch name from safe-outputs: ${branchFromSafeOutputs}`);

    // Check if the branch exists locally
    try {
      await exec.exec("git", ["show-ref", "--verify", "--quiet", `refs/heads/${branchFromSafeOutputs}`]);
      core.info(`Branch ${branchFromSafeOutputs} exists locally`);
      return branchFromSafeOutputs;
    } catch (error) {
      core.info(`Branch ${branchFromSafeOutputs} does not exist locally, falling back to current HEAD`);
    }
  } else {
    core.info("No branch name found in safe-outputs, using current branch");
  }

  // Fall back to current branch
  try {
    const { stdout } = await exec.getExecOutput("git", ["rev-parse", "--abbrev-ref", "HEAD"]);
    const currentBranch = stdout.trim();

    if (currentBranch && currentBranch !== "HEAD") {
      core.info(`Using current branch: ${currentBranch}`);
      return currentBranch;
    } else {
      core.warning("Detached HEAD state, using HEAD directly");
      return "HEAD";
    }
  } catch (error) {
    core.error(`Failed to get current branch: ${error instanceof Error ? error.message : String(error)}`);
    return "";
  }
}

/**
 * Determine the base ref for patch generation
 * @param {string} targetBranch - The target branch for patch generation
 * @param {string} defaultBranch - The default branch of the repository
 * @returns {Promise<string>} Base ref for patch generation
 */
async function determineBaseRef(targetBranch, defaultBranch) {
  // For detached HEAD, use merge-base with default branch
  if (targetBranch === "HEAD") {
    core.info(`Default branch: ${defaultBranch}`);
    await exec.exec("git", ["fetch", "origin", defaultBranch]);

    const { stdout } = await exec.getExecOutput("git", ["merge-base", `origin/${defaultBranch}`, "HEAD"]);
    const baseRef = stdout.trim();
    core.info(`Using merge-base as base: ${baseRef}`);
    return baseRef;
  }

  // Check if origin/targetBranch exists
  try {
    await exec.exec("git", ["show-ref", "--verify", "--quiet", `refs/remotes/origin/${targetBranch}`]);
    core.info(`Using origin/${targetBranch} as base for patch generation`);
    return `origin/${targetBranch}`;
  } catch (error) {
    core.info(`origin/${targetBranch} does not exist, using merge-base with default branch`);
  }

  // Fall back to merge-base with default branch
  core.info(`Default branch: ${defaultBranch}`);
  await exec.exec("git", ["fetch", "origin", defaultBranch]);

  const { stdout } = await exec.getExecOutput("git", ["merge-base", `origin/${defaultBranch}`, targetBranch]);
  const baseRef = stdout.trim();
  core.info(`Using merge-base as base: ${baseRef}`);
  return baseRef;
}

/**
 * Generate git patch from base to target branch
 * @param {string} baseRef - Base ref for patch generation
 * @param {string} targetBranch - Target branch for patch generation
 * @param {string} patchPath - Path where patch should be written
 * @returns {Promise<boolean>} True if patch was generated successfully
 */
async function generatePatch(baseRef, targetBranch, patchPath) {
  const fs = require("fs");

  try {
    const { stdout, exitCode } = await exec.getExecOutput("git", ["format-patch", `${baseRef}..${targetBranch}`, "--stdout"], {
      ignoreReturnCode: true,
    });

    if (exitCode === 0 && stdout) {
      fs.writeFileSync(patchPath, stdout);
      core.info(`Patch file created from: ${targetBranch} (base: ${baseRef})`);
      return true;
    } else {
      fs.writeFileSync(patchPath, "Failed to generate patch from branch");
      core.warning("Failed to generate patch - git format-patch returned no output or error");
      return false;
    }
  } catch (error) {
    const fs = require("fs");
    fs.writeFileSync(patchPath, "Failed to generate patch from branch");
    core.error(`Failed to generate patch: ${error instanceof Error ? error.message : String(error)}`);
    return false;
  }
}

/**
 * Add patch info to step summary
 * @param {string} patchPath - Path to the patch file
 */
async function addPatchToSummary(patchPath) {
  const fs = require("fs");

  if (!fs.existsSync(patchPath)) {
    return;
  }

  const stats = fs.statSync(patchPath);
  core.info(`Patch file size: ${stats.size} bytes`);

  // Read patch content
  const patchContent = fs.readFileSync(patchPath, "utf8");

  // Show the first 500 lines of the patch for review
  const lines = patchContent.split("\n");
  const previewLines = lines.slice(0, 500);
  const preview = previewLines.join("\n");

  let summary = "## Git Patch\n\n";
  summary += "```diff\n";
  summary += preview;
  if (lines.length > 500) {
    summary += "\n...";
  }
  summary += "\n```\n\n";

  await core.summary.addRaw(summary).write();
}

async function main() {
  const fs = require("fs");

  // Show current git status
  core.info("Current git status:");
  await exec.exec("git", ["status"]);

  // Get environment variables
  const safeOutputsPath = process.env.GH_AW_SAFE_OUTPUTS || "";
  const defaultBranch = process.env.GH_AW_DEFAULT_BRANCH || context.payload?.repository?.default_branch || "main";
  const patchPath = "/tmp/gh-aw/aw.patch";

  // Extract branch name from safe-outputs
  const branchFromSafeOutputs = safeOutputsPath ? extractBranchFromSafeOutputs(safeOutputsPath) : "";

  // Determine target branch
  const targetBranch = await determineTargetBranch(branchFromSafeOutputs);

  if (!targetBranch) {
    core.info("No target branch determined, no patch generation");
    return;
  }

  core.info(`Generating patch for: ${targetBranch}`);

  // Determine base ref
  const baseRef = await determineBaseRef(targetBranch, defaultBranch);

  if (!baseRef) {
    core.error("Failed to determine base ref for patch generation");
    return;
  }

  // Generate patch
  const success = await generatePatch(baseRef, targetBranch, patchPath);

  if (success) {
    // Add patch to step summary
    await addPatchToSummary(patchPath);
  }
}

// Only run main if this script is being executed directly
if (typeof module !== "undefined" && require.main === module) {
  main().catch(error => {
    core.setFailed(error instanceof Error ? error.message : String(error));
  });
}

// Export functions for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    extractBranchFromSafeOutputs,
    determineTargetBranch,
    determineBaseRef,
    generatePatch,
    addPatchToSummary,
    main,
  };
}
