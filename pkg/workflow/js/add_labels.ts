import type { SafeOutputItems, AddLabelsItem } from "./types/safe-outputs";

async function addLabelsMain(): Promise<void> {
  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.debug(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput: SafeOutputItems;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.warning("No valid items found in agent output");
    return;
  }

  // Find the add-labels item
  const labelsItem = validatedOutput.items.find(item => item.type === "add-labels") as AddLabelsItem | undefined;
  if (!labelsItem) {
    core.warning("No add-labels item found in agent output");
    return;
  }

  core.debug(`Found add-labels item with ${labelsItem.labels.length} labels`);

  // If in staged mode, emit step summary instead of adding labels
  if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Add Labels Preview\n\n";
    summaryContent += "The following labels would be added if staged mode was disabled:\n\n";

    if ((labelsItem as any).issue_number) {
      summaryContent += `**Target Issue:** #${(labelsItem as any).issue_number}\n\n`;
    } else {
      summaryContent += `**Target:** Current issue/PR\n\n`;
    }

    if (labelsItem.labels && labelsItem.labels.length > 0) {
      summaryContent += "**Labels to add:**\n";
      for (const label of labelsItem.labels) {
        summaryContent += `- \`${label}\`\n`;
      }
    } else {
      summaryContent += "**No labels specified**\n";
    }

    summaryContent += "\n---\n\n";

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Label addition preview written to step summary");
    return;
  }

  // Get the label configuration from environment variable
  const labelTarget = process.env.GITHUB_AW_LABEL_TARGET || "triggering";
  core.info(`Label target configuration: ${labelTarget}`);

  // Check if we're in an issue or pull request context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";

  // Validate context based on target configuration
  if (labelTarget === "triggering" && !isIssueContext && !isPRContext) {
    core.info('Target is "triggering" but not running in issue or pull request context, skipping label addition');
    return;
  }

  // Determine the issue/PR number for this label operation
  let issueNumber: number | undefined;

  if (labelTarget === "*") {
    // For target "*", we need an explicit issue number from the labels item
    if ((labelsItem as any).issue_number) {
      issueNumber = parseInt((labelsItem as any).issue_number, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.info(`Invalid issue number specified: ${(labelsItem as any).issue_number}`);
        return;
      }
    } else {
      core.info('Target is "*" but no issue_number specified in labels item');
      return;
    }
  } else if (labelTarget && labelTarget !== "triggering") {
    // Explicit issue number specified in target
    issueNumber = parseInt(labelTarget, 10);
    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.info(`Invalid issue number in target configuration: ${labelTarget}`);
      return;
    }
  } else {
    // Default behavior: use triggering issue/PR
    if (isIssueContext) {
      if (context.payload.issue) {
        issueNumber = context.payload.issue.number;
      } else {
        core.info("Issue context detected but no issue found in payload");
        return;
      }
    } else if (isPRContext) {
      if (context.payload.pull_request) {
        issueNumber = context.payload.pull_request.number;
      } else {
        core.info("Pull request context detected but no pull request found in payload");
        return;
      }
    }
  }

  if (!issueNumber) {
    core.info("Could not determine issue or pull request number");
    return;
  }

  // Extract labels from the JSON item
  const labels = labelsItem.labels || [];
  if (labels.length === 0) {
    core.info("No labels to add");
    return;
  }

  core.info(`Adding ${labels.length} label(s) to issue #${issueNumber}: ${labels.join(", ")}`);

  try {
    // Add the labels using GitHub API
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
      labels: labels,
    });

    core.info(`Successfully added labels to issue #${issueNumber}`);

    // Set outputs
    core.setOutput("issue_number", issueNumber.toString());
    core.setOutput("labels_added", labels.join(","));
    core.setOutput("label_count", labels.length.toString());

    // Write summary
    let summaryContent = "\n\n## GitHub Labels\n";
    summaryContent += `Successfully added **${labels.length}** label(s) to issue #${issueNumber}:\n`;
    for (const label of labels) {
      summaryContent += `- \`${label}\`\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  } catch (error) {
    core.error(`✗ Failed to add labels: ${error instanceof Error ? error.message : String(error)}`);
    throw error;
  }
}

(async () => {
  await addLabelsMain();
})();