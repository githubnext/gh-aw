/** @type {typeof import("fs")} */
const fs = require("fs");

async function main() {
  // Environment validation - fail early if required variables are missing
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT || "";
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  const target = process.env.GITHUB_AW_PUSH_TARGET || "triggering";
  const ifNoChanges = process.env.GITHUB_AW_PUSH_IF_NO_CHANGES || "warn";

  // Check if patch file exists and has valid content
  if (!fs.existsSync("/tmp/aw.patch")) {
    const message = "No patch file found - cannot push without changes";

    switch (ifNoChanges) {
      case "error":
        core.setFailed(message);
        return;
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.info(message);
        return;
    }
  }

  const patchContent = fs.readFileSync("/tmp/aw.patch", "utf8");

  // Check for actual error conditions (but allow empty patches as valid noop)
  if (patchContent.includes("Failed to generate patch")) {
    const message = "Patch file contains error message - cannot push without changes";

    switch (ifNoChanges) {
      case "error":
        core.setFailed(message);
        return;
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.info(message);
        return;
    }
  }

  // Validate patch size (unless empty)
  const isEmpty = !patchContent || !patchContent.trim();
  if (!isEmpty) {
    // Get maximum patch size from environment (default: 1MB = 1024 KB)
    const maxSizeKb = parseInt(process.env.GITHUB_AW_MAX_PATCH_SIZE || "1024", 10);
    const patchSizeBytes = Buffer.byteLength(patchContent, "utf8");
    const patchSizeKb = Math.ceil(patchSizeBytes / 1024);

    core.info(`Patch size: ${patchSizeKb} KB (maximum allowed: ${maxSizeKb} KB)`);

    if (patchSizeKb > maxSizeKb) {
      const message = `Patch size (${patchSizeKb} KB) exceeds maximum allowed size (${maxSizeKb} KB)`;
      core.setFailed(message);
      return;
    }

    core.info("Patch size validation passed");
  }
  if (isEmpty) {
    const message = "Patch file is empty - no changes to apply (noop operation)";

    switch (ifNoChanges) {
      case "error":
        core.setFailed("No changes to push - failing as configured by if-no-changes: error");
        return;
      case "ignore":
        // Silent success - no console output
        break;
      case "warn":
      default:
        core.info(message);
        break;
    }
  }

  core.info(`Agent output content length: ${outputContent.length}`);
  if (!isEmpty) {
    core.info("Patch content validation passed");
  }
  core.info(`Target configuration: ${target}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find the push-to-pull-request-branch item
  const pushItem = validatedOutput.items.find(/** @param {any} item */ item => item.type === "push-to-pull-request-branch");
  if (!pushItem) {
    core.info("No push-to-pull-request-branch item found in agent output");
    return;
  }

  core.info("Found push-to-pull-request-branch item");

  // If in staged mode, emit step summary instead of pushing changes
  if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Push to PR Branch Preview\n\n";
    summaryContent += "The following changes would be pushed if staged mode was disabled:\n\n";

    summaryContent += `**Target:** ${target}\n\n`;

    if (pushItem.commit_message) {
      summaryContent += `**Commit Message:** ${pushItem.commit_message}\n\n`;
    }

    if (fs.existsSync("/tmp/aw.patch")) {
      const patchStats = fs.readFileSync("/tmp/aw.patch", "utf8");
      if (patchStats.trim()) {
        summaryContent += `**Changes:** Patch file exists with ${patchStats.split("\n").length} lines\n\n`;
        summaryContent += `<details><summary>Show patch preview</summary>\n\n\`\`\`diff\n${patchStats.slice(0, 2000)}${patchStats.length > 2000 ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>\n\n`;
      } else {
        summaryContent += `**Changes:** No changes (empty patch)\n\n`;
      }
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Push to PR branch preview written to step summary");
    return;
  }

  // Validate target configuration for pull request context
  if (target !== "*" && target !== "triggering") {
    // If target is a specific number, validate it's a valid pull request number
    const pullNumber = parseInt(target, 10);
    if (isNaN(pullNumber)) {
      core.setFailed('Invalid target configuration: must be "triggering", "*", or a valid pull request number');
      return;
    }
  }

  // Compute the target branch name based on target configuration
  let pullNumber;
  if (target === "triggering") {
    // Use the number of the triggering pull request
    pullNumber = context.payload?.pull_request?.number || context.payload?.issue?.number;

    // Check if we're in a pull request context when required
    if (!pullNumber) {
      core.setFailed('push-to-pull-request-branch with target "triggering" requires pull request context');
      return;
    }
  } else if (target === "*") {
    if (pushItem.pull_number) {
      pullNumber = parseInt(pushItem.pull_number, 10);
    }
  } else {
    // Target is a specific pull request number
    pullNumber = parseInt(target, 10);
  }
  let branchName;
  let prTitle = "";
  let prLabels = [];

  // Fetch the specific PR to get its head branch, title, and labels
  try {
    let prInfo = "";
    const prInfoRes = await exec.exec(
      `gh`,
      [
        `pr`,
        `view`,
        `${pullNumber}`,
        `--json`,
        `headRefName,title,labels`,
        `--jq`,
        `{headRefName, title, labels: (.labels // [] | map(.name))}`,
      ],
      {
        listeners: { stdout: data => (prInfo += data.toString()) },
      }
    );
    if (!prInfoRes) {
      const prData = JSON.parse(prInfo.trim());
      branchName = prData.headRefName;
      prTitle = prData.title || "";
      prLabels = prData.labels || [];
    } else {
      throw new Error("No PR data found");
    }
  } catch (error) {
    core.info(`Warning: Could not fetch PR ${pullNumber} details: ${error instanceof Error ? error.message : String(error)}`);
    // Exit with failure if we cannot determine the branch name
    core.setFailed(`Failed to determine branch name for PR ${pullNumber}`);
    return;
  }

  core.info(`Target branch: ${branchName}`);
  core.info(`PR title: ${prTitle}`);
  core.info(`PR labels: ${prLabels.join(", ")}`);

  // Validate title prefix if specified
  const titlePrefix = process.env.GITHUB_AW_PR_TITLE_PREFIX;
  if (titlePrefix && !prTitle.startsWith(titlePrefix)) {
    core.setFailed(`Pull request title "${prTitle}" does not start with required prefix "${titlePrefix}"`);
    return;
  }

  // Validate labels if specified
  const requiredLabelsStr = process.env.GITHUB_AW_PR_LABELS;
  if (requiredLabelsStr) {
    const requiredLabels = requiredLabelsStr.split(",").map(label => label.trim());
    const missingLabels = requiredLabels.filter(label => !prLabels.includes(label));
    if (missingLabels.length > 0) {
      core.setFailed(`Pull request is missing required labels: ${missingLabels.join(", ")}. Current labels: ${prLabels.join(", ")}`);
      return;
    }
  }

  if (titlePrefix) {
    core.info(`✓ Title prefix validation passed: "${titlePrefix}"`);
  }
  if (requiredLabelsStr) {
    core.info(`✓ Labels validation passed: ${requiredLabelsStr}`);
  }

  // Check if patch has actual changes (not just empty)
  const hasChanges = !isEmpty;

  // Switch to or create the target branch
  core.info(`Switching to branch: ${branchName}`);
  try {
    // Try to checkout existing branch first
    await exec.exec("git fetch origin");

    // Check if branch exists on origin
    try {
      await exec.exec(`git rev-parse --verify origin/${branchName}`);
      await exec.exec(`git checkout -B ${branchName} origin/${branchName}`);
      core.info(`Checked out existing branch from origin: ${branchName}`);
    } catch (originError) {
      // Give an error if branch doesn't exist on origin
      core.setFailed(
        `Branch ${branchName} does not exist on origin, can't push to it: ${originError instanceof Error ? originError.message : String(originError)}`
      );
      return;
    }
  } catch (error) {
    core.setFailed(`Failed to switch to branch ${branchName}: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  // Apply the patch using git CLI (skip if empty)
  if (!isEmpty) {
    core.info("Applying patch...");
    try {
      // Patches are created with git format-patch, so use git am to apply them
      await exec.exec("git am /tmp/aw.patch");
      core.info("Patch applied successfully");

      // Push the applied commits to the branch
      await exec.exec(`git push origin ${branchName}`);
      core.info(`Changes committed and pushed to branch: ${branchName}`);
    } catch (error) {
      core.error(`Failed to apply patch: ${error instanceof Error ? error.message : String(error)}`);
      core.setFailed("Failed to apply patch");
      return;
    }
  } else {
    core.info("Skipping patch application (empty patch)");

    // Handle if-no-changes configuration for empty patches
    const message = "No changes to apply - noop operation completed successfully";

    switch (ifNoChanges) {
      case "error":
        core.setFailed("No changes to apply - failing as configured by if-no-changes: error");
        return;
      case "ignore":
        // Silent success - no console output
        break;
      case "warn":
      default:
        core.info(message);
        break;
    }
  }

  // Get commit SHA and push URL
  let commitSha = "";
  const commitShaRes = await exec.exec("git", ["rev-parse", "HEAD"], {
    listeners: { stdout: data => (commitSha += data.toString()) },
  });
  if (commitShaRes) throw new Error("Failed to get commit SHA");
  commitSha = commitSha.trim();

  // Get commit SHA and push URL
  const pushUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/tree/${branchName}`
    : `https://github.com/${context.repo.owner}/${context.repo.repo}/tree/${branchName}`;

  // Set outputs
  core.setOutput("branch_name", branchName);
  core.setOutput("commit_sha", commitSha);
  core.setOutput("push_url", pushUrl);

  // Write summary to GitHub Actions summary
  const summaryTitle = hasChanges ? "Push to Branch" : "Push to Branch (No Changes)";
  const summaryContent = hasChanges
    ? `
## ${summaryTitle}
- **Branch**: \`${branchName}\`
- **Commit**: [${commitSha.substring(0, 7)}](${pushUrl})
- **URL**: [${pushUrl}](${pushUrl})
`
    : `
## ${summaryTitle}
- **Branch**: \`${branchName}\`
- **Status**: No changes to apply (noop operation)
- **URL**: [${pushUrl}](${pushUrl})
`;

  await core.summary.addRaw(summaryContent).write();
}

await main();
