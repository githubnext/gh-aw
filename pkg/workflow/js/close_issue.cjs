// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all close-issue items
  const closeItems = result.items.filter(/** @param {any} item */ item => item.type === "close_issue");
  if (closeItems.length === 0) {
    core.info("No close-issue items found in agent output");
    return;
  }

  core.info(`Found ${closeItems.length} close-issue item(s)`);

  // If in staged mode, emit step summary instead of closing issues
  if (isStaged) {
    await generateStagedPreview({
      title: "Close Issues",
      description: "The following issues would be closed if staged mode was disabled:",
      items: closeItems,
      renderItem: (item, index) => {
        let content = `### Issue Close ${index + 1}\n`;
        content += `**Issue:** #${item.issue_number}\n\n`;
        
        if (item.comment) {
          content += `**Comment:**\n${item.comment}\n\n`;
        }
        
        if (item.state_reason) {
          content += `**Reason:** ${item.state_reason}\n\n`;
        }
        
        return content;
      },
    });
    return;
  }

  // Get filter configuration from environment variables
  const allowedLabels = process.env.GH_AW_CLOSE_ISSUE_LABELS 
    ? process.env.GH_AW_CLOSE_ISSUE_LABELS.split(",").map(l => l.trim()).filter(l => l.length > 0)
    : [];
  const titlePrefix = process.env.GH_AW_CLOSE_ISSUE_TITLE_PREFIX || "";

  core.info(`Filter configuration - Labels: ${allowedLabels.length > 0 ? allowedLabels.join(", ") : "none"}, Title prefix: ${titlePrefix || "none"}`);

  const closedIssues = [];

  // Process each close item
  for (let i = 0; i < closeItems.length; i++) {
    const closeItem = closeItems[i];
    core.info(`Processing close-issue item ${i + 1}/${closeItems.length}`);

    // Parse issue number
    const issueNumber = parseInt(closeItem.issue_number, 10);
    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.info(`Invalid issue number: ${closeItem.issue_number}`);
      continue;
    }

    core.info(`Processing close request for issue #${issueNumber}`);

    try {
      // First, fetch the issue to check if it matches filters
      const { data: issue } = await github.rest.issues.get({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issueNumber,
      });

      // Check if issue is already closed
      if (issue.state === "closed") {
        core.info(`Issue #${issueNumber} is already closed, skipping`);
        continue;
      }

      // Apply label filter if configured
      if (allowedLabels.length > 0) {
        const issueLabels = issue.labels.map(label => typeof label === "string" ? label : label.name);
        const hasMatchingLabel = allowedLabels.some(allowedLabel => issueLabels.includes(allowedLabel));
        
        if (!hasMatchingLabel) {
          core.info(`Issue #${issueNumber} does not have any of the required labels [${allowedLabels.join(", ")}], skipping`);
          continue;
        }
      }

      // Apply title prefix filter if configured
      if (titlePrefix && !issue.title.startsWith(titlePrefix)) {
        core.info(`Issue #${issueNumber} title does not start with "${titlePrefix}", skipping`);
        continue;
      }

      // Add comment if provided
      if (closeItem.comment && closeItem.comment.trim().length > 0) {
        core.info(`Adding comment to issue #${issueNumber}`);
        await github.rest.issues.createComment({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: issueNumber,
          body: closeItem.comment.trim(),
        });
        core.info(`Comment added to issue #${issueNumber}`);
      }

      // Close the issue
      const updateData = {
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issueNumber,
        state: "closed",
      };

      // Add state_reason if provided
      if (closeItem.state_reason) {
        updateData.state_reason = closeItem.state_reason;
      }

      const { data: closedIssue } = await github.rest.issues.update(updateData);

      core.info(`✓ Closed issue #${closedIssue.number}: ${closedIssue.html_url}`);
      closedIssues.push(closedIssue);

      // Set output for the last closed issue
      if (i === closeItems.length - 1) {
        core.setOutput("issue_number", closedIssue.number);
        core.setOutput("issue_url", closedIssue.html_url);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`✗ Failed to close issue #${issueNumber}: ${errorMessage}`);
      // Continue processing other issues instead of failing the entire job
    }
  }

  // Write summary for all closed issues
  if (closedIssues.length > 0) {
    let summaryContent = "\n\n## Closed Issues\n";
    for (const issue of closedIssues) {
      summaryContent += `- Issue #${issue.number}: [${issue.title}](${issue.html_url})`;
      if (issue.state_reason) {
        summaryContent += ` (${issue.state_reason})`;
      }
      summaryContent += "\n";
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully closed ${closedIssues.length} issue(s)`);
  return closedIssues;
}

await main();
