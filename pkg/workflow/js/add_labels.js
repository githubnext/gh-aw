function sanitizeLabelContent(content) {
  if (!content || typeof content !== "string") {
    return "";
  }
  let sanitized = content.trim();
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");
  sanitized = sanitized.replace(
    /(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
    (_m, p1, p2) => `${p1}\`@${p2}\``
  );
  sanitized = sanitized.replace(/[<>&'"]/g, "");
  return sanitized.trim();
}
async function main() {
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
      summaryContent += `**Labels to add:** ${labelsItem.labels.join(", ")}\n\n`;
    }
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Label addition preview written to step summary");
    return;
  }
  const allowedLabelsEnv = process.env.GITHUB_AW_LABELS_ALLOWED?.trim();
  const allowedLabels = allowedLabelsEnv
    ? allowedLabelsEnv
        .split(",")
        .map(label => label.trim())
        .filter(label => label)
    : undefined;
  if (allowedLabels) {
    core.debug(`Allowed labels: ${JSON.stringify(allowedLabels)}`);
  } else {
    core.debug("No label restrictions - any labels are allowed");
  }
  const maxCountEnv = process.env.GITHUB_AW_LABELS_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 3;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }
  core.debug(`Max count: ${maxCount}`);
  
  // Get the target configuration from environment variable
  const labelsTarget = process.env.GITHUB_AW_LABELS_TARGET || "triggering";
  core.info(`Labels target configuration: ${labelsTarget}`);
  
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";
  
  // Validate context based on target configuration
  if (labelsTarget === "triggering" && !isIssueContext && !isPRContext) {
    core.info('Target is "triggering" but not running in issue or pull request context, skipping label addition');
    return;
  }
  
  // Determine the issue/PR number based on target configuration
  let issueNumber;
  let contextType;
  
  if (labelsTarget === "*") {
    // For target "*", we need an explicit issue number from the labels item
    if (labelsItem.issue_number) {
      issueNumber = parseInt(labelsItem.issue_number, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.setFailed(`Invalid issue number specified: ${labelsItem.issue_number}`);
        return;
      }
      contextType = "issue/PR";
    } else {
      core.setFailed('Target is "*" but no issue_number specified in labels item');
      return;
    }
  } else if (labelsTarget && labelsTarget !== "triggering") {
    // Explicit issue number specified in target
    issueNumber = parseInt(labelsTarget, 10);
    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.setFailed(`Invalid issue number in target configuration: ${labelsTarget}`);
      return;
    }
    contextType = "issue/PR";
  } else {
    // Default behavior: use triggering issue/PR
    if (isIssueContext) {
      if (context.payload.issue) {
        issueNumber = context.payload.issue.number;
        contextType = "issue";
      } else {
        core.setFailed("Issue context detected but no issue found in payload");
        return;
      }
    } else if (isPRContext) {
      if (context.payload.pull_request) {
        issueNumber = context.payload.pull_request.number;
        contextType = "pull request";
      } else {
        core.setFailed("Pull request context detected but no pull request found in payload");
        return;
      }
    }
  }
  
  if (!issueNumber) {
    core.setFailed("Could not determine issue or pull request number");
    return;
  }
  const requestedLabels = labelsItem.labels || [];
  core.debug(`Requested labels: ${JSON.stringify(requestedLabels)}`);
  for (const label of requestedLabels) {
    if (label && typeof label === "string" && label.startsWith("-")) {
      core.setFailed(`Label removal is not permitted. Found line starting with '-': ${label}`);
      return;
    }
  }
  let validLabels;
  if (allowedLabels) {
    validLabels = requestedLabels.filter(label => allowedLabels.includes(label));
  } else {
    validLabels = requestedLabels;
  }
  let uniqueLabels = validLabels
    .filter(label => label != null && label !== false && label !== 0)
    .map(label => String(label).trim())
    .filter(label => label)
    .map(label => sanitizeLabelContent(label))
    .filter(label => label)
    .map(label => (label.length > 64 ? label.substring(0, 64) : label))
    .filter((label, index, arr) => arr.indexOf(label) === index);
  if (uniqueLabels.length > maxCount) {
    core.debug(`too many labels, keep ${maxCount}`);
    uniqueLabels = uniqueLabels.slice(0, maxCount);
  }
  if (uniqueLabels.length === 0) {
    core.info("No labels to add");
    core.setOutput("labels_added", "");
    await core.summary
      .addRaw(
        `
## Label Addition

No labels were added (no valid labels found in agent output).
`
      )
      .write();
    return;
  }
  core.info(`Adding ${uniqueLabels.length} labels to ${contextType} #${issueNumber}: ${JSON.stringify(uniqueLabels)}`);
  try {
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
      labels: uniqueLabels,
    });
    core.info(`Successfully added ${uniqueLabels.length} labels to ${contextType} #${issueNumber}`);
    core.setOutput("labels_added", uniqueLabels.join("\n"));
    const labelsListMarkdown = uniqueLabels.map(label => `- \`${label}\``).join("\n");
    await core.summary
      .addRaw(
        `
## Label Addition

Successfully added ${uniqueLabels.length} label(s) to ${contextType} #${issueNumber}:

${labelsListMarkdown}
`
      )
      .write();
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to add labels: ${errorMessage}`);
    core.setFailed(`Failed to add labels: ${errorMessage}`);
  }
}
await main();
