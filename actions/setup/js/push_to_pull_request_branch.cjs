// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { updateActivationCommentWithCommit } = require("./update_activation_comment.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Environment validation - fail early if required variables are missing
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT || "";
  if (agentOutputFile.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.setFailed(`Error reading agent output file: ${getErrorMessage(error)}`);
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  const target = process.env.GH_AW_PUSH_TARGET || "triggering";
  const ifNoChanges = process.env.GH_AW_PUSH_IF_NO_CHANGES || "warn";

  // Check if patch file exists and has valid content
  if (!fs.existsSync("/tmp/gh-aw/aw.patch")) {
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

  const patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");

  // Check for actual error conditions (but allow empty patches as valid noop)
  if (patchContent.includes("Failed to generate patch")) {
    const message = "Patch file contains error message - cannot push without changes";

    // Log diagnostic information to help with troubleshooting
    core.error("Patch file generation failed - this is an error condition that requires investigation");
    core.error(`Patch file location: /tmp/gh-aw/aw.patch`);
    core.error(`Patch file size: ${Buffer.byteLength(patchContent, "utf8")} bytes`);

    // Show first 500 characters of patch content for diagnostics
    const previewLength = Math.min(500, patchContent.length);
    core.error(`Patch file preview (first ${previewLength} characters):`);
    core.error(patchContent.substring(0, previewLength));

    // This is always a failure regardless of if-no-changes configuration
    // because the patch file contains an error message from the patch generation process
    core.setFailed(message);
    return;
  }

  // Validate patch size (unless empty)
  const isEmpty = !patchContent || !patchContent.trim();
  if (!isEmpty) {
    // Get maximum patch size from environment (default: 1MB = 1024 KB)
    const maxSizeKb = parseInt(process.env.GH_AW_MAX_PATCH_SIZE || "1024", 10);
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
    core.setFailed(`Error parsing agent output JSON: ${getErrorMessage(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find the push-to-pull-request-branch item
  const pushItem = validatedOutput.items.find(/** @param {any} item */ item => item.type === "push_to_pull_request_branch");
  if (!pushItem) {
    core.info("No push-to-pull-request-branch item found in agent output");
    return;
  }

  core.info("Found push-to-pull-request-branch item");

  // If in staged mode, emit step summary instead of pushing changes
  if (isStaged) {
    await generateStagedPreview({
      title: "Push to PR Branch",
      description: "The following changes would be pushed if staged mode was disabled:",
      items: [{ target, commit_message: pushItem.commit_message }],
      renderItem: item => {
        let content = "";
        content += `**Target:** ${item.target}\n\n`;

        if (item.commit_message) {
          content += `**Commit Message:** ${item.commit_message}\n\n`;
        }

        if (fs.existsSync("/tmp/gh-aw/aw.patch")) {
          const patchStats = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
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

  // Validate pull number is defined before fetching
  if (!pullNumber) {
    core.setFailed("Pull request number is required but not found");
    return;
  }

  // Fetch the specific PR to get its head branch, title, and labels
  try {
    const { data: pullRequest } = await github.rest.pulls.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
      pull_number: pullNumber,
    });
    branchName = pullRequest.head.ref;
    prTitle = pullRequest.title || "";
    prLabels = pullRequest.labels.map(label => label.name);
  } catch (error) {
    core.info(`Warning: Could not fetch PR ${pullNumber} details: ${getErrorMessage(error)}`);
    // Exit with failure if we cannot determine the branch name
    core.setFailed(`Failed to determine branch name for PR ${pullNumber}`);
    return;
  }

  core.info(`Target branch: ${branchName}`);
  core.info(`PR title: ${prTitle}`);
  core.info(`PR labels: ${prLabels.join(", ")}`);

  // Validate title prefix if specified
  const titlePrefix = process.env.GH_AW_PR_TITLE_PREFIX;
  if (titlePrefix && !prTitle.startsWith(titlePrefix)) {
    core.setFailed(`Pull request title "${prTitle}" does not start with required prefix "${titlePrefix}"`);
    return;
  }

  // Validate labels if specified
  const requiredLabelsStr = process.env.GH_AW_PR_LABELS;
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

  // Fetch the specific target branch from origin (since we use shallow checkout)
  try {
    core.info(`Fetching branch: ${branchName}`);
    await exec.exec(`git fetch origin ${branchName}:refs/remotes/origin/${branchName}`);
  } catch (fetchError) {
    core.setFailed(`Failed to fetch branch ${branchName}: ${fetchError instanceof Error ? fetchError.message : String(fetchError)}`);
    return;
  }

  // Check if branch exists on origin
  try {
    await exec.exec(`git rev-parse --verify origin/${branchName}`);
  } catch (verifyError) {
    core.setFailed(`Branch ${branchName} does not exist on origin, can't push to it: ${verifyError instanceof Error ? verifyError.message : String(verifyError)}`);
    return;
  }

  // Checkout the branch from origin
  try {
    await exec.exec(`git checkout -B ${branchName} origin/${branchName}`);
    core.info(`Checked out existing branch from origin: ${branchName}`);
  } catch (checkoutError) {
    core.setFailed(`Failed to checkout branch ${branchName}: ${checkoutError instanceof Error ? checkoutError.message : String(checkoutError)}`);
    return;
  }

  // Apply the patch using git CLI (skip if empty)
  if (!isEmpty) {
    core.info("Applying patch...");
    try {
      // Check if commit title suffix is configured
      const commitTitleSuffix = process.env.GH_AW_COMMIT_TITLE_SUFFIX;

      if (commitTitleSuffix) {
        core.info(`Appending commit title suffix: "${commitTitleSuffix}"`);

        // Read the patch file
        let patchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");

        // Modify Subject lines in the patch to append the suffix
        // Patch format has "Subject: [PATCH] <original title>" or "Subject: <original title>"
        // Append the suffix at the end of the title to avoid git am stripping brackets
        patchContent = patchContent.replace(/^Subject: (?:\[PATCH\] )?(.*)$/gm, (match, title) => `Subject: [PATCH] ${title}${commitTitleSuffix}`);

        // Write the modified patch back
        fs.writeFileSync("/tmp/gh-aw/aw.patch", patchContent, "utf8");
        core.info(`Patch modified with commit title suffix: "${commitTitleSuffix}"`);
      }

      // Log first 100 lines of patch for debugging
      const finalPatchContent = fs.readFileSync("/tmp/gh-aw/aw.patch", "utf8");
      const patchLines = finalPatchContent.split("\n");
      const previewLineCount = Math.min(100, patchLines.length);
      core.info(`Patch preview (first ${previewLineCount} of ${patchLines.length} lines):`);
      for (let i = 0; i < previewLineCount; i++) {
        core.info(patchLines[i]);
      }

      // Patches are created with git format-patch, so use git am to apply them
      await exec.exec("git am /tmp/gh-aw/aw.patch");
      core.info("Patch applied successfully");

      // Push the applied commits to the branch
      await exec.exec(`git push origin ${branchName}`);
      core.info(`Changes committed and pushed to branch: ${branchName}`);
    } catch (error) {
      core.error(`Failed to apply patch: ${getErrorMessage(error)}`);

      // Investigate why the patch failed by logging git status and the failed patch
      try {
        core.info("Investigating patch failure...");

        // Log git status to see the current state
        const statusResult = await exec.getExecOutput("git", ["status"]);
        core.info("Git status output:");
        core.info(statusResult.stdout);

        // Log recent commits for context
        const logResult = await exec.getExecOutput("git", ["log", "--oneline", "-5"]);
        core.info("Recent commits (last 5):");
        core.info(logResult.stdout);

        // Log uncommitted changes
        const diffResult = await exec.getExecOutput("git", ["diff", "HEAD"]);
        core.info("Uncommitted changes:");
        core.info(diffResult.stdout && diffResult.stdout.trim() ? diffResult.stdout : "(no uncommitted changes)");

        // Log the failed patch diff
        const patchDiffResult = await exec.getExecOutput("git", ["am", "--show-current-patch=diff"]);
        core.info("Failed patch diff:");
        core.info(patchDiffResult.stdout);

        // Log the full failed patch for complete context
        const patchFullResult = await exec.getExecOutput("git", ["am", "--show-current-patch"]);
        core.info("Failed patch (full):");
        core.info(patchFullResult.stdout);
      } catch (investigateError) {
        core.warning(`Failed to investigate patch failure: ${investigateError instanceof Error ? investigateError.message : String(investigateError)}`);
      }

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
  const commitShaRes = await exec.getExecOutput("git", ["rev-parse", "HEAD"]);
  if (commitShaRes.exitCode !== 0) throw new Error("Failed to get commit SHA");
  const commitSha = commitShaRes.stdout.trim();

  // Get repository base URL and construct URLs
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const repoUrl = context.payload.repository ? context.payload.repository.html_url : `${githubServer}/${context.repo.owner}/${context.repo.repo}`;
  const pushUrl = `${repoUrl}/tree/${branchName}`;
  const commitUrl = `${repoUrl}/commit/${commitSha}`;

  // Set outputs
  core.setOutput("branch_name", branchName);
  core.setOutput("commit_sha", commitSha);
  core.setOutput("push_url", pushUrl);
  core.setOutput("commit_url", commitUrl);

  // Update the activation comment with commit link (if a comment was created and changes were pushed)
  if (hasChanges) {
    await updateActivationCommentWithCommit(github, context, core, commitSha, commitUrl);
  }

  // Write summary to GitHub Actions summary
  const summaryTitle = hasChanges ? "Push to Branch" : "Push to Branch (No Changes)";
  const summaryContent = hasChanges
    ? `
## ${summaryTitle}
- **Branch**: \`${branchName}\`
- **Commit**: [${commitSha.substring(0, 7)}](${commitUrl})
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

module.exports = { main };
