/** @type {typeof import("fs")} */
const fs = require("fs");
/** @type {typeof import("crypto")} */
const crypto = require("crypto");
const { execSync } = require("child_process");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  // Environment validation - fail early if required variables are missing
  const workflowId = process.env.GITHUB_AW_WORKFLOW_ID;
  if (!workflowId) {
    throw new Error("GITHUB_AW_WORKFLOW_ID environment variable is required");
  }

  const baseBranch = process.env.GITHUB_AW_BASE_BRANCH;
  if (!baseBranch) {
    throw new Error("GITHUB_AW_BASE_BRANCH environment variable is required");
  }

  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT || "";
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
  }

  const ifNoChanges = process.env.GITHUB_AW_PR_IF_NO_CHANGES || "warn";

  // Check if patch file exists and has valid content
  if (!fs.existsSync("/tmp/aw.patch")) {
    const message =
      "No patch file found - cannot create pull request without changes";

    // If in staged mode, still show preview
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent +=
        "The following pull request would be created if staged mode was disabled:\n\n";
      summaryContent += `**Status:** ⚠️ No patch file found\n\n`;
      summaryContent += `**Message:** ${message}\n\n`;

      // Write to step summary
      await core.summary.addRaw(summaryContent).write();
      core.info(
        "📝 Pull request creation preview written to step summary (no patch file)"
      );
      return;
    }

    switch (ifNoChanges) {
      case "error":
        throw new Error(message);
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.warning(message);
        return;
    }
  }

  const patchContent = fs.readFileSync("/tmp/aw.patch", "utf8");

  // Check for actual error conditions (but allow empty patches as valid noop)
  if (patchContent.includes("Failed to generate patch")) {
    const message =
      "Patch file contains error message - cannot create pull request without changes";

    // If in staged mode, still show preview
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent +=
        "The following pull request would be created if staged mode was disabled:\n\n";
      summaryContent += `**Status:** ⚠️ Patch file contains error\n\n`;
      summaryContent += `**Message:** ${message}\n\n`;

      // Write to step summary
      await core.summary.addRaw(summaryContent).write();
      core.info(
        "📝 Pull request creation preview written to step summary (patch error)"
      );
      return;
    }

    switch (ifNoChanges) {
      case "error":
        throw new Error(message);
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.warning(message);
        return;
    }
  }

  // Empty patch is valid - behavior depends on if-no-changes configuration
  const isEmpty = !patchContent || !patchContent.trim();
  if (isEmpty && !isStaged) {
    const message =
      "Patch file is empty - no changes to apply (noop operation)";

    switch (ifNoChanges) {
      case "error":
        throw new Error(
          "No changes to push - failing as configured by if-no-changes: error"
        );
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.warning(message);
        return;
    }
  }

  core.debug(`Agent output content length: ${outputContent.length}`);
  if (!isEmpty) {
    core.info("Patch content validation passed");
  } else {
    core.info("Patch file is empty - processing noop operation");
  }

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.error(
      `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.warning("No valid items found in agent output");
    return;
  }

  // Find the create-pull-request item
  const pullRequestItem = validatedOutput.items.find(
    /** @param {any} item */ item => item.type === "create-pull-request"
  );
  if (!pullRequestItem) {
    core.warning("No create-pull-request item found in agent output");
    return;
  }

  core.debug(`Found create-pull-request item: title="${pullRequestItem.title}", bodyLength=${pullRequestItem.body.length}`);

  // If in staged mode, emit step summary instead of creating PR
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
    summaryContent +=
      "The following pull request would be created if staged mode was disabled:\n\n";

    summaryContent += `**Title:** ${pullRequestItem.title || "No title provided"}\n\n`;
    summaryContent += `**Branch:** ${pullRequestItem.branch || "auto-generated"}\n\n`;
    summaryContent += `**Base:** ${baseBranch}\n\n`;

    if (pullRequestItem.body) {
      summaryContent += `**Body:**\n${pullRequestItem.body}\n\n`;
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
    core.info("📝 Pull request creation preview written to step summary");
    return;
  }

  // Extract title, body, and branch from the JSON item
  let title = pullRequestItem.title.trim();
  let bodyLines = pullRequestItem.body.split("\n");
  let branchName = pullRequestItem.branch
    ? pullRequestItem.branch.trim()
    : null;

  // If no title was found, use a default
  if (!title) {
    title = "Agent Output";
  }

  // Apply title prefix if provided via environment variable
  const titlePrefix = process.env.GITHUB_AW_PR_TITLE_PREFIX;
  if (titlePrefix && !title.startsWith(titlePrefix)) {
    title = titlePrefix + title;
  }

  // Add AI disclaimer with run id, run htmlurl
  const runId = context.runId;
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `https://github.com/actions/runs/${runId}`;
  bodyLines.push(
    ``,
    ``,
    `> Generated by Agentic Workflow [Run](${runUrl})`,
    ""
  );

  // Prepare the body content
  const body = bodyLines.join("\n").trim();

  // Parse labels from environment variable (comma-separated string)
  const labelsEnv = process.env.GITHUB_AW_PR_LABELS;
  const labels = labelsEnv
    ? labelsEnv
        .split(",")
        .map(/** @param {string} label */ label => label.trim())
        .filter(/** @param {string} label */ label => label)
    : [];

  // Parse draft setting from environment variable (defaults to true)
  const draftEnv = process.env.GITHUB_AW_PR_DRAFT;
  const draft = draftEnv ? draftEnv.toLowerCase() === "true" : true;

  core.info(`Creating pull request with title: ${title}`);
  core.debug(`Labels: ${JSON.stringify(labels)}`);
  core.debug(`Draft: ${draft}`);
  core.debug(`Body length: ${body.length}`);

  const randomHex = crypto.randomBytes(8).toString("hex");
  // Use branch name from JSONL if provided, otherwise generate unique branch name
  if (!branchName) {
    core.debug(
      "No branch name provided in JSONL, generating unique branch name"
    );
    // Generate unique branch name using cryptographic random hex
    branchName = `${workflowId}-${randomHex}`;
  } else {
    branchName = `${branchName}-${randomHex}`;
    core.debug(`Using branch name from JSONL with added salt: ${branchName}`);
  }

  core.info(`Generated branch name: ${branchName}`);
  core.debug(`Base branch: ${baseBranch}`);

  // Create a new branch using git CLI, ensuring it's based on the correct base branch

  // First, fetch latest changes and checkout the base branch
  core.debug(
    `Fetching latest changes and checking out base branch: ${baseBranch}`
  );
  execSync("git fetch origin", { stdio: "inherit" });
  execSync(`git checkout ${baseBranch}`, { stdio: "inherit" });

  // Handle branch creation/checkout
  core.debug(
    `Branch should not exist locally, creating new branch from base: ${branchName}`
  );
  execSync(`git checkout -b ${branchName}`, { stdio: "inherit" });
  core.info(`Created new branch from base: ${branchName}`);

  // Apply the patch using git CLI (skip if empty)
  if (!isEmpty) {
    core.info("Applying patch...");
    // Patches are created with git format-patch, so use git am to apply them
    execSync("git am /tmp/aw.patch", { stdio: "inherit" });
    core.info("Patch applied successfully");

    // Push the applied commits to the branch
    execSync(`git push origin ${branchName}`, { stdio: "inherit" });
    core.info("Changes pushed to branch");
  } else {
    core.info("Skipping patch application (empty patch)");

    // For empty patches, handle if-no-changes configuration
    const message =
      "No changes to apply - noop operation completed successfully";

    switch (ifNoChanges) {
      case "error":
        throw new Error(
          "No changes to apply - failing as configured by if-no-changes: error"
        );
      case "ignore":
        // Silent success - no console output
        return;
      case "warn":
      default:
        core.warning(message);
        return;
    }
  }

  // Create the pull request
  const { data: pullRequest } = await github.rest.pulls.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    title: title,
    body: body,
    head: branchName,
    base: baseBranch,
    draft: draft,
  });

  core.info(
    `Created pull request #${pullRequest.number}: ${pullRequest.html_url}`
  );

  // Add labels if specified
  if (labels.length > 0) {
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: pullRequest.number,
      labels: labels,
    });
    console.log("Added labels to pull request:", labels);
  }

  // Set output for other jobs to use
  core.setOutput("pull_request_number", pullRequest.number);
  core.setOutput("pull_request_url", pullRequest.html_url);
  core.setOutput("branch_name", branchName);

  // Write summary to GitHub Actions summary
  await core.summary
    .addRaw(
      `

## Pull Request
- **Pull Request**: [#${pullRequest.number}](${pullRequest.html_url})
- **Branch**: \`${branchName}\`
- **Base Branch**: \`${baseBranch}\`
`
    )
    .write();
}
await main();
