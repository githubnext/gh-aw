async function addLabelsMain() {
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
  let validatedOutput;
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
  const labelsItem = validatedOutput.items.find(item => item.type === "add-labels");
  if (!labelsItem) {
    core.warning("No add-labels item found in agent output");
    return;
  }
  core.debug(`Found add-labels item with ${labelsItem.labels.length} labels`);
  if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Add Labels Preview\n\n";
    summaryContent += "The following labels would be added if staged mode was disabled:\n\n";
    if (labelsItem.issue_number) {
      summaryContent += `**Target Issue:** #${labelsItem.issue_number}\n\n`;
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
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Label addition preview written to step summary");
    return;
  }
  const labelTarget = process.env.GITHUB_AW_LABEL_TARGET || "triggering";
  core.info(`Label target configuration: ${labelTarget}`);
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";
  if (labelTarget === "triggering" && !isIssueContext && !isPRContext) {
    core.info('Target is "triggering" but not running in issue or pull request context, skipping label addition');
    return;
  }
  let issueNumber;
  if (labelTarget === "*") {
    if (labelsItem.issue_number) {
      issueNumber = parseInt(labelsItem.issue_number, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.info(`Invalid issue number specified: ${labelsItem.issue_number}`);
        return;
      }
    } else {
      core.info('Target is "*" but no issue_number specified in labels item');
      return;
    }
  } else if (labelTarget && labelTarget !== "triggering") {
    issueNumber = parseInt(labelTarget, 10);
    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.info(`Invalid issue number in target configuration: ${labelTarget}`);
      return;
    }
  } else {
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
  const labels = labelsItem.labels || [];
  if (labels.length === 0) {
    core.info("No labels to add");
    return;
  }
  core.info(`Adding ${labels.length} label(s) to issue #${issueNumber}: ${labels.join(", ")}`);
  try {
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
      labels: labels,
    });
    core.info(`Successfully added labels to issue #${issueNumber}`);
    core.setOutput("issue_number", issueNumber.toString());
    core.setOutput("labels_added", labels.join(","));
    core.setOutput("label_count", labels.length.toString());
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
