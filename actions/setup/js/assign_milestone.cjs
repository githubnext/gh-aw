// @ts-check
/// <reference types="@actions/github-script" />

const { processSafeOutput } = require("./safe_output_processor.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  // Use shared processor for common steps
  const result = await processSafeOutput(
    {
      itemType: "assign_milestone",
      configKey: "assign_milestone",
      displayName: "Milestone",
      itemTypeName: "milestone assignment",
      supportsPR: true,
      supportsIssue: true,
      findMultiple: true, // This processor finds multiple items
      envVars: {
        allowed: "GH_AW_MILESTONE_ALLOWED",
        maxCount: "GH_AW_MILESTONE_MAX_COUNT",
        target: "GH_AW_MILESTONE_TARGET",
      },
    },
    {
      title: "Assign Milestone",
      description: "The following milestone assignments would be made if staged mode was disabled:",
      renderItem: item => {
        let content = `**Issue:** #${item.issue_number}\n`;
        content += `**Milestone Number:** ${item.milestone_number}\n\n`;
        return content;
      },
    }
  );

  if (!result.success) {
    return;
  }

  const { items: milestoneItems, config } = result;
  if (!config || !milestoneItems) {
    core.setFailed("Internal error: config or milestoneItems is undefined");
    return;
  }
  const { allowed: allowedMilestones, maxCount = 1 } = config;

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
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to fetch milestones: ${errorMessage}`);
      core.setFailed(`Failed to fetch milestones for validation: ${errorMessage}`);
      return;
    }
  }

  // Process each milestone assignment
  const results = [];
  for (const item of itemsToProcess) {
    const issueNumber = Number(item.issue_number);
    const milestoneNumber = Number(item.milestone_number);

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
      const errorMessage = getErrorMessage(error);
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
  const successResults = results.filter(r => r.success);
  const failureResults = results.filter(r => !r.success);

  const summaryParts = ["## Milestone Assignment\n"];

  if (successResults.length > 0) {
    summaryParts.push(`✅ Successfully assigned ${successResults.length} milestone(s):\n`);
    summaryParts.push(successResults.map(r => `- Issue #${r.issue_number} → Milestone #${r.milestone_number}`).join("\n"));
    summaryParts.push("\n");
  }

  if (failureResults.length > 0) {
    summaryParts.push(`❌ Failed to assign ${failureResults.length} milestone(s):\n`);
    summaryParts.push(failureResults.map(r => `- Issue #${r.issue_number} → Milestone #${r.milestone_number}: ${r.error}`).join("\n"));
  }

  const summaryContent = summaryParts.join("\n");

  await core.summary.addRaw(summaryContent).write();

  const assignedMilestones = successResults.map(r => `${r.issue_number}:${r.milestone_number}`).join("\n");
  core.setOutput("assigned_milestones", assignedMilestones);

  if (failureResults.length > 0) {
    core.setFailed(`Failed to assign ${failureResults.length} milestone(s)`);
  }
}

module.exports = { main };
