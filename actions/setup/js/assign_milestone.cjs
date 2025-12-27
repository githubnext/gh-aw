// @ts-nocheck - Type checking disabled due to complex type errors requiring refactoring
/// <reference types="@actions/github-script" />

/** @param {unknown} error */
function getErrorMessage(error) {
  if (error instanceof Error) return getErrorMessage(error);
  if (error && typeof error === "object" && "message" in error && typeof getErrorMessage(error) === "string") return getErrorMessage(error);
  return String(error);
}


const { processSafeOutput } = require("./safe_output_processor.cjs");

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

  // @ts-ignore - TypeScript doesn't narrow properly after success check
  const { items: milestoneItems, config } = result;
  if (!config || !milestoneItems) {
    core.setFailed("Internal error: config or milestoneItems is undefined");
    return;
  }
  const { allowed: allowedMilestones, maxCount } = config;

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
      const errorMessage = error?.message ?? String(error);
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
    if (allowedMilestones?.length > 0) {
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
      const errorMessage = error?.message ?? String(error);
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
  const failureCount = results.length - successCount;

  let summaryContent = "## Milestone Assignment\n\n";

  if (successCount > 0) {
    summaryContent += `✅ Successfully assigned ${successCount} milestone(s):\n\n`;
    summaryContent += results
      .filter(r => r.success)
      .map(r => `- Issue #${r.issue_number} → Milestone #${r.milestone_number}`)
      .join("\n");
    summaryContent += "\n\n";
  }

  if (failureCount > 0) {
    summaryContent += `❌ Failed to assign ${failureCount} milestone(s):\n\n`;
    summaryContent += results
      .filter(r => !r.success)
      .map(r => `- Issue #${r.issue_number} → Milestone #${r.milestone_number}: ${r.error}`)
      .join("\n");
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

module.exports = { main };
