// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (!agentOutputFile) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = require("fs").readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.setFailed(`Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

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

  // Find all close-issue items
  const closeItems = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "close_issue");
  if (closeItems.length === 0) {
    core.info("No close-issue items found in agent output");
    return;
  }

  core.info(`Found ${closeItems.length} close-issue item(s)`);

  // If in staged mode, emit step summary instead of closing issues
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Close Issues Preview\n\n";
    summaryContent += "The following issues would be closed if staged mode was disabled:\n\n";

    for (let i = 0; i < closeItems.length; i++) {
      const item = closeItems[i];
      summaryContent += `### Close Issue ${i + 1}\n`;
      if (item.issue_number) {
        summaryContent += `**Target Issue:** #${item.issue_number}\n\n`;
      } else {
        summaryContent += `**Target:** Current issue\n\n`;
      }

      if (item.outcome) {
        summaryContent += `**Outcome:** ${item.outcome}\n\n`;
      }
      if (item.reason) {
        summaryContent += `**Reason:** ${item.reason}\n\n`;
      }
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Close issue preview written to step summary");
    return;
  }

  // Get the configuration from environment variables
  const closeTarget = process.env.GH_AW_CLOSE_TARGET || "triggering";
  const requiredLabels = process.env.GH_AW_REQUIRED_LABELS ? process.env.GH_AW_REQUIRED_LABELS.split(",").map(l => l.trim()) : [];
  const allowedOutcomes = process.env.GH_AW_ALLOWED_OUTCOMES
    ? process.env.GH_AW_ALLOWED_OUTCOMES.split(",").map(o => o.trim())
    : ["completed", "not_planned"];

  core.info(`Close target configuration: ${closeTarget}`);
  core.info(`Required labels: ${requiredLabels.length > 0 ? requiredLabels.join(", ") : "none"}`);
  core.info(`Allowed outcomes: ${allowedOutcomes.join(", ")}`);

  // Check if we're in an issue context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";

  // Validate context based on target configuration
  if (closeTarget === "triggering" && !isIssueContext) {
    core.info('Target is "triggering" but not running in issue context, skipping issue close');
    return;
  }

  const closedIssues = [];

  // Process each close item
  for (let i = 0; i < closeItems.length; i++) {
    const closeItem = closeItems[i];
    core.info(`Processing close-issue item ${i + 1}/${closeItems.length}`);

    // Determine the issue number for this close operation
    let issueNumber;

    if (closeTarget === "*") {
      // For target "*", we need an explicit issue number from the close item
      if (closeItem.issue_number) {
        issueNumber = parseInt(closeItem.issue_number, 10);
        if (isNaN(issueNumber) || issueNumber <= 0) {
          core.info(`Invalid issue number specified: ${closeItem.issue_number}`);
          continue;
        }
      } else {
        core.info('Target is "*" but no issue_number specified in close item');
        continue;
      }
    } else if (closeTarget && closeTarget !== "triggering") {
      // Explicit issue number specified in target
      issueNumber = parseInt(closeTarget, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.info(`Invalid issue number in target configuration: ${closeTarget}`);
        continue;
      }
    } else {
      // Default behavior: use triggering issue
      if (isIssueContext) {
        if (context.payload.issue) {
          issueNumber = context.payload.issue.number;
        } else {
          core.info("Issue context detected but no issue found in payload");
          continue;
        }
      } else {
        core.info("Could not determine issue number");
        continue;
      }
    }

    if (!issueNumber) {
      core.info("Could not determine issue number");
      continue;
    }

    core.info(`Attempting to close issue #${issueNumber}`);

    try {
      // Fetch the issue to check its labels
      const { data: issue } = await github.rest.issues.get({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issueNumber,
      });

      // Check if the issue has all required labels
      if (requiredLabels.length > 0) {
        const issueLabels = issue.labels.map(/** @param {any} label */ label => (typeof label === "string" ? label : label.name));
        const missingLabels = requiredLabels.filter(required => !issueLabels.includes(required));

        if (missingLabels.length > 0) {
          core.info(`Issue #${issueNumber} does not have required labels: ${missingLabels.join(", ")}. Skipping.`);
          continue;
        }
        core.info(`Issue #${issueNumber} has all required labels`);
      }

      // Validate the outcome if specified
      let stateReason = closeItem.outcome;
      if (stateReason) {
        // Validate that the outcome is allowed
        if (!allowedOutcomes.includes(stateReason)) {
          core.info(`Outcome '${stateReason}' is not in allowed outcomes: ${allowedOutcomes.join(", ")}. Skipping.`);
          continue;
        }
        core.info(`Using outcome: ${stateReason}`);
      } else {
        // Default to "completed" if not specified
        stateReason = "completed";
        core.info(`No outcome specified, using default: ${stateReason}`);
      }

      // Close the issue
      const updateData = {
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issueNumber,
        state: "closed",
        state_reason: stateReason,
      };

      const { data: closedIssue } = await github.rest.issues.update(updateData);

      core.info(`✓ Closed issue #${closedIssue.number}: ${closedIssue.html_url}`);
      closedIssues.push(closedIssue);

      // Set output for the last closed issue (for backward compatibility)
      if (i === closeItems.length - 1) {
        core.setOutput("issue_number", closedIssue.number);
        core.setOutput("issue_url", closedIssue.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to close issue #${issueNumber}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all closed issues
  if (closedIssues.length > 0) {
    let summaryContent = "\n\n## Closed Issues\n";
    for (const issue of closedIssues) {
      summaryContent += `- Issue #${issue.number}: [${issue.title}](${issue.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully closed ${closedIssues.length} issue(s)`);
  return closedIssues;
}
await main();
