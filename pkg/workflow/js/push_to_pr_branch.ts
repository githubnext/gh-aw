async function pushToPrBranchMain(): Promise<void> {
  const fs = require("fs");

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

  // If patch is empty or contains only whitespace, handle gracefully
  if (patchContent.trim() === "") {
    const message = "Patch file is empty - no changes to push";

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

  core.info(`Patch file size: ${patchContent.length} characters`);
  core.debug(`Patch content preview: ${patchContent.substring(0, 200)}...`);

  // Determine the target branch
  let targetBranch: string;
  
  if (target === "triggering") {
    // Use the branch from the triggering PR or current branch
    if (context.eventName === "pull_request" || context.eventName === "pull_request_review") {
      targetBranch = context.payload.pull_request?.head?.ref || "main";
    } else {
      targetBranch = context.ref.replace("refs/heads/", "");
    }
  } else {
    // Explicit branch name specified
    targetBranch = target;
  }

  core.info(`Target branch: ${targetBranch}`);

  // Get current branch and ensure we're on the target branch
  try {
    // Fetch the latest changes to ensure we have the most recent state
    await exec.exec("git", ["fetch", "origin", targetBranch]);

    // Check if the target branch exists locally
    let branchExists = false;
    try {
      await exec.exec("git", ["rev-parse", "--verify", targetBranch]);
      branchExists = true;
    } catch {
      branchExists = false;
    }

    if (branchExists) {
      // Branch exists locally, check it out and pull latest
      await exec.exec("git", ["checkout", targetBranch]);
      await exec.exec("git", ["pull", "origin", targetBranch]);
    } else {
      // Branch doesn't exist locally, create and track it
      await exec.exec("git", ["checkout", "-b", targetBranch, `origin/${targetBranch}`]);
    }

    core.info(`Successfully checked out branch: ${targetBranch}`);
  } catch (error) {
    core.setFailed(
      `Failed to checkout target branch '${targetBranch}': ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  // Apply the patch
  try {
    core.info("Applying patch to working directory...");
    await exec.exec("git", ["apply", "/tmp/aw.patch"]);
    core.info("Patch applied successfully");
  } catch (error) {
    core.setFailed(
      `Failed to apply patch: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  // Stage all changes
  try {
    await exec.exec("git", ["add", "."]);
    core.info("Changes staged successfully");
  } catch (error) {
    core.setFailed(
      `Failed to stage changes: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  // Check if there are actually changes to commit
  let hasChanges = false;
  try {
    const { exitCode } = await exec.getExecOutput("git", ["diff", "--cached", "--quiet"]);
    hasChanges = exitCode !== 0;
  } catch {
    // If git diff fails, assume there are changes
    hasChanges = true;
  }

  if (!hasChanges) {
    const message = "No changes to commit after applying patch";

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

  // Commit changes
  const commitMessage = process.env.GITHUB_AW_COMMIT_MESSAGE || "Apply changes from agentic workflow";
  const runId = context.runId;
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  const fullCommitMessage = `${commitMessage}\n\nGenerated by: ${runUrl}`;

  try {
    await exec.exec("git", ["commit", "-m", fullCommitMessage]);
    core.info(`Changes committed with message: "${commitMessage}"`);
  } catch (error) {
    core.setFailed(
      `Failed to commit changes: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  // Push changes
  try {
    await exec.exec("git", ["push", "origin", targetBranch]);
    core.info(`Changes pushed to branch: ${targetBranch}`);

    // Set outputs
    core.setOutput("branch_name", targetBranch);
    core.setOutput("commit_sha", await getLatestCommitSha());
    core.setOutput("pushed", "true");

    // Write summary
    let summaryContent = "\n\n## Git Push Summary\n";
    summaryContent += `- **Branch**: \`${targetBranch}\`\n`;
    summaryContent += `- **Commit**: [View Changes](${context.payload.repository?.html_url}/commit/${await getLatestCommitSha()})\n`;
    summaryContent += `- **Status**: ✅ Successfully pushed\n`;
    await core.summary.addRaw(summaryContent).write();
  } catch (error) {
    core.setFailed(
      `Failed to push changes: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }
}

async function getLatestCommitSha(): Promise<string> {
  try {
    const { stdout } = await exec.getExecOutput("git", ["rev-parse", "HEAD"]);
    return stdout.trim();
  } catch {
    return "unknown";
  }
}

(async () => {
  await pushToPrBranchMain();
})();