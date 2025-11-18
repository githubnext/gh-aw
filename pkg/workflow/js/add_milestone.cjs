// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const milestoneItem = result.items.find(item => item.type === "add_milestone");
  if (!milestoneItem) {
    core.warning("No add-milestone item found in agent output");
    return;
  }

  core.info(`Found add-milestone item with milestone: ${JSON.stringify(milestoneItem.milestone)}`);

  // Check if we're in staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Add Milestone",
      description: "The following milestone assignment would be performed if staged mode was disabled:",
      items: [milestoneItem],
      renderItem: item => {
        let content = "";
        if (item.item_number) {
          content += `**Target Issue:** #${item.item_number}\n\n`;
        } else {
          content += `**Target:** Current issue\n\n`;
        }
        content += `**Milestone:** ${item.milestone}\n\n`;
        return content;
      },
    });
    return;
  }

  // Parse allowed milestones from environment
  const allowedMilestonesEnv = process.env.GH_AW_MILESTONES_ALLOWED?.trim();
  if (!allowedMilestonesEnv) {
    core.setFailed("No allowed milestones configured. Please configure safe-outputs.add-milestone.allowed in your workflow.");
    return;
  }

  const allowedMilestones = allowedMilestonesEnv
    .split(",")
    .map(m => m.trim())
    .filter(m => m);

  if (allowedMilestones.length === 0) {
    core.setFailed("Allowed milestones list is empty");
    return;
  }

  core.info(`Allowed milestones: ${JSON.stringify(allowedMilestones)}`);

  // Parse target configuration
  const milestoneTarget = process.env.GH_AW_MILESTONE_TARGET || "triggering";
  core.info(`Milestone target configuration: ${milestoneTarget}`);

  // Determine if we're in issue context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";

  if (milestoneTarget === "triggering" && !isIssueContext) {
    core.info('Target is "triggering" but not running in issue context, skipping milestone addition');
    return;
  }

  // Determine the issue number
  let issueNumber;
  if (milestoneTarget === "*") {
    if (milestoneItem.item_number) {
      issueNumber = typeof milestoneItem.item_number === "number" ? milestoneItem.item_number : parseInt(String(milestoneItem.item_number), 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.setFailed(`Invalid item_number specified: ${milestoneItem.item_number}`);
        return;
      }
    } else {
      core.setFailed('Target is "*" but no item_number specified in milestone item');
      return;
    }
  } else if (milestoneTarget && milestoneTarget !== "triggering") {
    issueNumber = parseInt(milestoneTarget, 10);
    if (isNaN(issueNumber) || issueNumber <= 0) {
      core.setFailed(`Invalid issue number in target configuration: ${milestoneTarget}`);
      return;
    }
  } else {
    // Use triggering issue
    if (isIssueContext) {
      if (context.payload.issue) {
        issueNumber = context.payload.issue.number;
      } else {
        core.setFailed("Issue context detected but no issue found in payload");
        return;
      }
    } else {
      core.setFailed("Could not determine issue number");
      return;
    }
  }

  if (!issueNumber) {
    core.setFailed("Could not determine issue number");
    return;
  }

  core.info(`Target issue number: ${issueNumber}`);

  // Validate milestone is in allowed list
  const requestedMilestone = milestoneItem.milestone;
  let milestoneIdentifier = String(requestedMilestone);

  // Check if milestone is in allowed list (either as name or number)
  const isAllowed = allowedMilestones.some(allowed => {
    if (typeof requestedMilestone === "number") {
      // Check if allowed is a number or matches the number as string
      return allowed === String(requestedMilestone) || parseInt(allowed, 10) === requestedMilestone;
    }
    // For string milestones, do case-insensitive comparison
    return allowed.toLowerCase() === String(requestedMilestone).toLowerCase();
  });

  if (!isAllowed) {
    core.setFailed(`Milestone '${requestedMilestone}' is not in the allowed list: ${JSON.stringify(allowedMilestones)}`);
    return;
  }

  core.info(`Milestone '${requestedMilestone}' is allowed`);

  // Resolve milestone to milestone number if it's a title
  let milestoneNumber;
  if (typeof requestedMilestone === "number") {
    milestoneNumber = requestedMilestone;
  } else {
    // Fetch milestones from repository to resolve title to number
    try {
      core.info(`Fetching milestones to resolve title: ${requestedMilestone}`);
      const { data: milestones } = await github.rest.issues.listMilestones({
        owner: context.repo.owner,
        repo: context.repo.repo,
        state: "open",
        per_page: 100,
      });

      // Try to find milestone by title (case-insensitive)
      const milestone = milestones.find(m => m.title.toLowerCase() === requestedMilestone.toLowerCase());

      if (!milestone) {
        // Also check closed milestones
        const { data: closedMilestones } = await github.rest.issues.listMilestones({
          owner: context.repo.owner,
          repo: context.repo.repo,
          state: "closed",
          per_page: 100,
        });

        const closedMilestone = closedMilestones.find(m => m.title.toLowerCase() === requestedMilestone.toLowerCase());

        if (!closedMilestone) {
          core.setFailed(`Milestone '${requestedMilestone}' not found in repository. Available milestones: ${milestones.map(m => m.title).join(", ")}`);
          return;
        }

        milestoneNumber = closedMilestone.number;
        core.info(`Resolved closed milestone '${requestedMilestone}' to number: ${milestoneNumber}`);
      } else {
        milestoneNumber = milestone.number;
        core.info(`Resolved milestone '${requestedMilestone}' to number: ${milestoneNumber}`);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to fetch milestones: ${errorMessage}`);
      core.setFailed(`Failed to resolve milestone '${requestedMilestone}': ${errorMessage}`);
      return;
    }
  }

  // Add issue to milestone
  try {
    core.info(`Adding issue #${issueNumber} to milestone #${milestoneNumber}`);
    await github.rest.issues.update({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
      milestone: milestoneNumber,
    });

    core.info(`Successfully added issue #${issueNumber} to milestone`);
    core.setOutput("milestone_added", String(milestoneNumber));
    core.setOutput("issue_number", String(issueNumber));

    await core.summary
      .addRaw(
        `
## Milestone Assignment

Successfully added issue #${issueNumber} to milestone: **${milestoneIdentifier}**
`
      )
      .write();
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to add milestone: ${errorMessage}`);
    core.setFailed(`Failed to add milestone: ${errorMessage}`);
  }
}

await main();
