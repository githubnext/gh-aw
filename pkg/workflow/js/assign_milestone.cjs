// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const milestoneItems = result.items.filter(item => item.type === "assign_milestone");
  if (milestoneItems.length === 0) {
    core.info("No assign_milestone items found in agent output");
    return;
  }

  core.info(`Found ${milestoneItems.length} assign_milestone item(s)`);

  // Check if we're in staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Assign Milestone",
      description: "The following milestone assignments would be made if staged mode was disabled:",
      items: milestoneItems,
      renderItem: item => {
        let content = `**Issue:** #${item.issue_number}\n`;
        content += `**Milestone Number:** ${item.milestone_number}\n\n`;
        return content;
      },
    });
    return;
  }

  // Get allowed milestones configuration
  const allowedMilestonesEnv = process.env.GH_AW_MILESTONE_ALLOWED?.trim();
  const allowedMilestones = allowedMilestonesEnv
    ? allowedMilestonesEnv
        .split(",")
        .map(m => m.trim())
        .filter(m => m)
    : undefined;

  if (allowedMilestones) {
    core.info(`Allowed milestones: ${JSON.stringify(allowedMilestones)}`);
  } else {
    core.info("No milestone restrictions - any milestones are allowed");
  }

  // Get max count configuration
  const maxCountEnv = process.env.GH_AW_MILESTONE_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 1;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }
  core.info(`Max count: ${maxCount}`);

  // Limit items to max count
  const itemsToProcess = milestoneItems.slice(0, maxCount);
  if (milestoneItems.length > maxCount) {
    core.warning(`Found ${milestoneItems.length} milestone assignments, but max is ${maxCount}. Processing first ${maxCount}.`);
  }

  // Fetch all milestones to validate against allowed list
  let allMilestones = [];
  if (allowedMilestones) {
    try {
      const milestonesResponse = await github.rest.issues.listMilestones({
        owner: context.repo.owner,
        repo: context.repo.repo,
        state: "all",
        per_page: 100,
      });
      allMilestones = milestonesResponse.data;
      core.info(`Fetched ${allMilestones.length} milestones from repository`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to fetch milestones: ${errorMessage}`);
      core.setFailed(`Failed to fetch milestones for validation: ${errorMessage}`);
      return;
    }
  }

  // Process each milestone assignment
  const results = [];
  for (const item of itemsToProcess) {
    const issueNumber = typeof item.issue_number === "number" ? item.issue_number : parseInt(String(item.issue_number), 10);
    const milestoneNumber = typeof item.milestone_number === "number" ? item.milestone_number : parseInt(String(item.milestone_number), 10);

    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.error(`Invalid issue_number: ${item.issue_number}`);
      continue;
    }

    if (isNaN(milestoneNumber) || milestoneNumber <= 0) {
      core.error(`Invalid milestone_number: ${item.milestone_number}`);
      continue;
    }

    // Validate against allowed list if configured
    if (allowedMilestones && allowedMilestones.length > 0) {
      const milestone = allMilestones.find(m => m.number === milestoneNumber);

      if (!milestone) {
        core.warning(`Milestone #${milestoneNumber} not found in repository. Skipping.`);
        continue;
      }

      // Check if milestone title or number (as string) is in allowed list
      const isAllowed = allowedMilestones.includes(milestone.title) || allowedMilestones.includes(String(milestoneNumber));

      if (!isAllowed) {
        core.warning(`Milestone "${milestone.title}" (#${milestoneNumber}) is not in the allowed list. Skipping.`);
        continue;
      }
    }

    // Assign the milestone to the issue
    try {
      await github.rest.issues.update({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issueNumber,
        milestone: milestoneNumber,
      });

      core.info(`Successfully assigned milestone #${milestoneNumber} to issue #${issueNumber}`);
      results.push({
        issue_number: issueNumber,
        milestone_number: milestoneNumber,
        success: true,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to assign milestone #${milestoneNumber} to issue #${issueNumber}: ${errorMessage}`);
      results.push({
        issue_number: issueNumber,
        milestone_number: milestoneNumber,
        success: false,
        error: errorMessage,
      });
    }
  }

  // Generate step summary
  const successCount = results.filter(r => r.success).length;
  const failureCount = results.filter(r => !r.success).length;

  let summaryContent = "## Milestone Assignment\n\n";

  if (successCount > 0) {
    summaryContent += `✅ Successfully assigned ${successCount} milestone(s):\n\n`;
    for (const result of results.filter(r => r.success)) {
      summaryContent += `- Issue #${result.issue_number} → Milestone #${result.milestone_number}\n`;
    }
    summaryContent += "\n";
  }

  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign ${failureCount} milestone(s):\n\n`;
    for (const result of results.filter(r => !r.success)) {
      summaryContent += `- Issue #${result.issue_number} → Milestone #${result.milestone_number}: ${result.error}\n`;
    }
  }

  await core.summary.addRaw(summaryContent).write();

  // Set outputs
  const assignedMilestones = results
    .filter(r => r.success)
    .map(r => `${r.issue_number}:${r.milestone_number}`)
    .join("\n");
  core.setOutput("assigned_milestones", assignedMilestones);

  // Fail if any assignments failed
  if (failureCount > 0) {
    core.setFailed(`Failed to assign ${failureCount} milestone(s)`);
  }
}

await main();
