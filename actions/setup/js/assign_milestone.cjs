// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

async function main(config = {}) {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("assigned_milestones", "");
  core.setOutput("milestone_assigned", "");
  core.setOutput("assignment_date", "");

  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const milestoneItems = result.items.filter(item => item.type === "assign_milestone");
  if (milestoneItems.length === 0) {
    core.info("No assign-milestone items found in agent output");
    return;
  }
  core.info(`Found ${milestoneItems.length} assign-milestone item(s)`);

  // Get configuration from config parameter or environment variable
  let effectiveConfig = config;
  if (Object.keys(config).length === 0 && process.env.GH_AW_ASSIGN_MILESTONE_CONFIG) {
    try {
      effectiveConfig = JSON.parse(process.env.GH_AW_ASSIGN_MILESTONE_CONFIG);
      core.info(`Loaded config from GH_AW_ASSIGN_MILESTONE_CONFIG: ${JSON.stringify(effectiveConfig)}`);
    } catch (error) {
      core.warning(`Failed to parse GH_AW_ASSIGN_MILESTONE_CONFIG: ${getErrorMessage(error)}`);
      effectiveConfig = {};
    }
  }
  
  const allowedMilestones = effectiveConfig.allowed || [];
  const maxCount = effectiveConfig.max || 1;

  if (isStaged) {
    await generateStagedPreview({
      title: "Assign Milestone",
      description: "The following milestone assignments would be made if staged mode was disabled:",
      items: milestoneItems.slice(0, maxCount),
      renderItem: (item, index) => {
        let content = `#### Assignment ${index + 1}\n`;
        content += `**Issue:** #${item.issue_number}\n`;
        content += `**Milestone Number:** ${item.milestone_number}\n\n`;
        return content;
      },
    });
    return;
  }

  // Limit items to max count
  const itemsToProcess = milestoneItems.slice(0, maxCount);
  if (milestoneItems.length > maxCount) {
    core.warning(`Found ${milestoneItems.length} milestone assignments, but max is ${maxCount}. Processing first ${maxCount}.`);
  }

  // Fetch all milestones to validate against allowed list
  let allMilestones = [];
  if (allowedMilestones && allowedMilestones.length > 0) {
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
  const assignmentDate = new Date().toISOString();
  
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

  // Set outputs
  const assignedMilestones = successResults.map(r => `${r.issue_number}:${r.milestone_number}`).join("\n");
  core.setOutput("assigned_milestones", assignedMilestones);
  
  // Set milestone_assigned output (true/false string)
  core.setOutput("milestone_assigned", successResults.length > 0 ? "true" : "false");
  
  // Set assignment_date output with ISO 8601 timestamp
  if (successResults.length > 0) {
    core.setOutput("assignment_date", assignmentDate);
    core.info(`Milestone assignment completed at: ${assignmentDate}`);
  }

  if (failureResults.length > 0) {
    core.setFailed(`Failed to assign ${failureResults.length} milestone(s)`);
  }
}

module.exports = { main };
