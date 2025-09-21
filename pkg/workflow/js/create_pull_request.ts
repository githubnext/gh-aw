import type { SafeOutputItems } from "./types/safe-outputs";

interface CreatedPullRequest {
  number: number;
  title: string;
  html_url: string;
  head: {
    ref: string;
    sha: string;
  };
}

async function createPullRequestMain(): Promise<CreatedPullRequest | null> {
  const fs = require("fs");
  const crypto = require("crypto");

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
    const message = "No patch file found - cannot create pull request without changes";

    // If in staged mode, still show preview
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
      summaryContent += `**Status:** ⚠️ No patch file found\n\n`;
      summaryContent += `**Message:** ${message}\n\n`;

      // Write to step summary
      await core.summary.addRaw(summaryContent).write();
      core.info("📝 Pull request creation preview written to step summary (no patch file)");
      return null;
    }

    switch (ifNoChanges) {
      case "error":
        throw new Error(message);
      case "ignore":
        // Silent success - no console output
        return null;
      case "warn":
      default:
        core.info(message);
        return null;
    }
  }

  const patchContent = fs.readFileSync("/tmp/aw.patch", "utf8");

  // Check for actual error conditions (but allow empty patches as valid noop)
  if (patchContent.includes("Failed to generate patch")) {
    const message = "Patch file contains error message - cannot create pull request without changes";

    // If in staged mode, still show preview with error
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
      summaryContent += `**Status:** ❌ Patch generation failed\n\n`;
      summaryContent += `**Message:** ${message}\n\n`;

      // Write to step summary
      await core.summary.addRaw(summaryContent).write();
      core.info("📝 Pull request creation preview written to step summary (patch failed)");
      return null;
    }

    switch (ifNoChanges) {
      case "error":
        throw new Error(message);
      case "ignore":
        // Silent success - no console output
        return null;
      case "warn":
      default:
        core.info(message);
        return null;
    }
  }

  // If patch is empty or contains only whitespace, handle gracefully
  if (patchContent.trim() === "") {
    const message = "Patch file is empty - no changes to create pull request for";

    // If in staged mode, still show preview
    if (isStaged) {
      let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
      summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
      summaryContent += `**Status:** ⚠️ No changes detected\n\n`;
      summaryContent += `**Message:** ${message}\n\n`;

      // Write to step summary
      await core.summary.addRaw(summaryContent).write();
      core.info("📝 Pull request creation preview written to step summary (no changes)");
      return null;
    }

    switch (ifNoChanges) {
      case "error":
        throw new Error(message);
      case "ignore":
        // Silent success - no console output
        return null;
      case "warn":
      default:
        core.info(message);
        return null;
    }
  }

  core.info(`Patch file size: ${patchContent.length} characters`);
  core.debug(`Patch content preview: ${patchContent.substring(0, 200)}...`);

  // Generate a unique branch name using cryptographic random
  const randomSuffix = crypto.randomBytes(8).toString("hex");
  const branchName = `agentic-workflow-${workflowId}-${randomSuffix}`;

  // Extract PR details from environment or generate defaults
  const titlePrefix = process.env.GITHUB_AW_PR_TITLE_PREFIX || "[AI]";
  const labels = process.env.GITHUB_AW_PR_LABELS
    ? process.env.GITHUB_AW_PR_LABELS.split(",").map(label => label.trim()).filter(label => label)
    : [];
  const isDraft = process.env.GITHUB_AW_PR_DRAFT === "true";

  // Generate PR title and body
  const runId = context.runId;
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  const defaultTitle = `${titlePrefix} Changes from workflow ${workflowId}`;
  const prTitle = process.env.GITHUB_AW_PR_TITLE || defaultTitle;

  let prBody = `This pull request contains changes generated by an agentic workflow.\n\n`;
  prBody += `**Workflow:** ${workflowId}\n`;
  prBody += `**Generated by:** [Workflow Run #${runId}](${runUrl})\n`;
  prBody += `**Base branch:** ${baseBranch}\n\n`;

  if (outputContent.trim()) {
    // Add a summary of the agent output (truncated for readability)
    const outputSummary = outputContent.length > 500 
      ? outputContent.substring(0, 500) + "...\n\n[Content truncated]"
      : outputContent;
    prBody += `## Agent Output\n\n\`\`\`\n${outputSummary}\n\`\`\`\n\n`;
  }

  prBody += `> This PR was created automatically by the GitHub Agentic Workflows system.`;

  // If in staged mode, emit step summary instead of creating PR
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Create Pull Request Preview\n\n";
    summaryContent += "The following pull request would be created if staged mode was disabled:\n\n";
    summaryContent += `**Title:** ${prTitle}\n\n`;
    summaryContent += `**Base Branch:** ${baseBranch}\n`;
    summaryContent += `**Head Branch:** ${branchName}\n`;
    summaryContent += `**Labels:** ${labels.length > 0 ? labels.join(", ") : "None"}\n`;
    summaryContent += `**Draft:** ${isDraft ? "Yes" : "No"}\n\n`;
    summaryContent += `**Body:**\n${prBody}\n\n`;
    summaryContent += `**Patch Size:** ${patchContent.length} characters\n\n`;

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Pull request creation preview written to step summary");
    return null;
  }

  try {
    // Create and checkout the new branch
    await exec.exec("git", ["checkout", "-b", branchName]);
    core.info(`Created and checked out branch: ${branchName}`);

    // Apply the patch
    await exec.exec("git", ["apply", "/tmp/aw.patch"]);
    core.info("Applied patch to working directory");

    // Stage all changes
    await exec.exec("git", ["add", "."]);

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
          throw new Error(message);
        case "ignore":
          return null;
        case "warn":
        default:
          core.info(message);
          return null;
      }
    }

    // Commit the changes
    const commitMessage = `${titlePrefix} Apply changes from workflow ${workflowId}\n\nGenerated by: ${runUrl}`;
    await exec.exec("git", ["commit", "-m", commitMessage]);
    core.info("Committed changes to branch");

    // Push the branch
    await exec.exec("git", ["push", "origin", branchName]);
    core.info(`Pushed branch ${branchName} to origin`);

    // Create the pull request
    const { data: pullRequest } = await github.rest.pulls.create({
      owner: context.repo.owner,
      repo: context.repo.repo,
      title: prTitle,
      body: prBody,
      head: branchName,
      base: baseBranch,
      draft: isDraft,
    });

    core.info(`Created pull request #${pullRequest.number}: ${pullRequest.html_url}`);

    // Add labels if specified
    if (labels.length > 0) {
      try {
        await github.rest.issues.addLabels({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: pullRequest.number,
          labels: labels,
        });
        core.info(`Added labels to PR: ${labels.join(", ")}`);
      } catch (labelError) {
        core.warning(`Failed to add labels: ${labelError instanceof Error ? labelError.message : String(labelError)}`);
      }
    }

    // Set outputs
    core.setOutput("pull_request_number", pullRequest.number);
    core.setOutput("pull_request_url", pullRequest.html_url);
    core.setOutput("branch_name", branchName);
    core.setOutput("created", "true");

    // Write summary
    let summaryContent = "\n\n## GitHub Pull Request\n";
    summaryContent += `- **PR #${pullRequest.number}**: [${pullRequest.title}](${pullRequest.html_url})\n`;
    summaryContent += `- **Branch**: \`${branchName}\`\n`;
    summaryContent += `- **Status**: ${isDraft ? "Draft" : "Ready for review"}\n`;
    if (labels.length > 0) {
      summaryContent += `- **Labels**: ${labels.join(", ")}\n`;
    }
    await core.summary.addRaw(summaryContent).write();

    return {
      number: pullRequest.number,
      title: pullRequest.title,
      html_url: pullRequest.html_url,
      head: {
        ref: branchName,
        sha: pullRequest.head.sha,
      },
    };

  } catch (error) {
    core.setFailed(`Failed to create pull request: ${error instanceof Error ? error.message : String(error)}`);
    return null;
  }
}

(async () => {
  await createPullRequestMain();
})();