// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const labelsItem = result.items.find(item => item.type === "remove_labels");
  if (!labelsItem) {
    core.warning("No remove-labels item found in agent output");
    return;
  }
  core.info(`Found remove-labels item with ${labelsItem.labels.length} labels`);
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Remove Labels",
      description: "The following labels would be removed if staged mode was disabled:",
      items: [labelsItem],
      renderItem: item => {
        let content = "";
        if (item.item_number) {
          content += `**Target Issue:** #${item.item_number}\n\n`;
        } else {
          content += `**Target:** Current issue/PR\n\n`;
        }
        if (item.labels && item.labels.length > 0) {
          content += `**Labels to remove:** ${item.labels.join(", ")}\n\n`;
        }
        return content;
      },
    });
    return;
  }
  const allowedLabelsEnv = process.env.GH_AW_LABELS_ALLOWED?.trim();
  const allowedLabels = allowedLabelsEnv
    ? allowedLabelsEnv
        .split(",")
        .map(label => label.trim())
        .filter(label => label)
    : undefined;
  if (allowedLabels) {
    core.info(`Allowed labels: ${JSON.stringify(allowedLabels)}`);
  } else {
    core.info("No label restrictions - any labels are allowed");
  }
  const maxCountEnv = process.env.GH_AW_LABELS_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 3;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }
  core.info(`Max count: ${maxCount}`);
  const labelsTarget = process.env.GH_AW_LABELS_TARGET || "triggering";
  core.info(`Labels target configuration: ${labelsTarget}`);
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";
  if (labelsTarget === "triggering" && !isIssueContext && !isPRContext) {
    core.info('Target is "triggering" but not running in issue or pull request context, skipping label removal');
    return;
  }
  let itemNumber;
  let contextType;
  if (labelsTarget === "*") {
    if (labelsItem.item_number) {
      itemNumber = typeof labelsItem.item_number === "number" ? labelsItem.item_number : parseInt(String(labelsItem.item_number), 10);
      if (isNaN(itemNumber) || itemNumber <= 0) {
        core.setFailed(`Invalid item_number specified: ${labelsItem.item_number}`);
        return;
      }
      contextType = "issue";
    } else {
      core.setFailed('Target is "*" but no item_number specified in labels item');
      return;
    }
  } else if (labelsTarget && labelsTarget !== "triggering") {
    itemNumber = parseInt(labelsTarget, 10);
    if (isNaN(itemNumber) || itemNumber <= 0) {
      core.setFailed(`Invalid issue number in target configuration: ${labelsTarget}`);
      return;
    }
    contextType = "issue";
  } else {
    if (isIssueContext) {
      if (context.payload.issue) {
        itemNumber = context.payload.issue.number;
        contextType = "issue";
      } else {
        core.setFailed("Issue context detected but no issue found in payload");
        return;
      }
    } else if (isPRContext) {
      if (context.payload.pull_request) {
        itemNumber = context.payload.pull_request.number;
        contextType = "pull request";
      } else {
        core.setFailed("Pull request context detected but no pull request found in payload");
        return;
      }
    }
  }
  if (!itemNumber) {
    core.setFailed("Could not determine issue or pull request number");
    return;
  }
  const requestedLabels = labelsItem.labels || [];
  core.info(`Requested labels: ${JSON.stringify(requestedLabels)}`);
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
    core.info(`too many labels, keep ${maxCount}`);
    uniqueLabels = uniqueLabels.slice(0, maxCount);
  }
  if (uniqueLabels.length === 0) {
    core.info("No labels to remove");
    core.setOutput("labels_removed", "");
    await core.summary
      .addRaw(
        `
## Label Removal

No labels were removed (no valid labels found in agent output).
`
      )
      .write();
    return;
  }
  core.info(`Removing ${uniqueLabels.length} labels from ${contextType} #${itemNumber}: ${JSON.stringify(uniqueLabels)}`);

  // Remove labels one by one
  const removedLabels = [];
  const failedLabels = [];

  for (const label of uniqueLabels) {
    try {
      await github.rest.issues.removeLabel({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: itemNumber,
        name: label,
      });
      removedLabels.push(label);
      core.info(`Successfully removed label '${label}' from ${contextType} #${itemNumber}`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      // Check if error is because label doesn't exist on the issue
      if (errorMessage.includes("Label does not exist") || errorMessage.includes("404")) {
        core.warning(`Label '${label}' not found on ${contextType} #${itemNumber}, skipping`);
      } else {
        core.error(`Failed to remove label '${label}': ${errorMessage}`);
        failedLabels.push(label);
      }
    }
  }

  if (failedLabels.length > 0) {
    const failedMessage = `Failed to remove ${failedLabels.length} label(s): ${failedLabels.join(", ")}`;
    core.setFailed(failedMessage);
  }

  core.setOutput("labels_removed", removedLabels.join("\n"));

  const labelsListMarkdown = removedLabels.map(label => `- \`${label}\``).join("\n");
  let summaryContent = `
## Label Removal

Successfully removed ${removedLabels.length} label(s) from ${contextType} #${itemNumber}:

${labelsListMarkdown}
`;

  if (failedLabels.length > 0) {
    const failedListMarkdown = failedLabels.map(label => `- \`${label}\``).join("\n");
    summaryContent += `\n### Failed to Remove\n\n${failedListMarkdown}\n`;
  }

  await core.summary.addRaw(summaryContent).write();
}
await main();
