// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { parseAllowedItems, resolveTarget } = require("./safe_output_helpers.cjs");
const { getSafeOutputConfig, validateMaxCount } = require("./safe_output_validator.cjs");

// GitHub Copilot reviewer bot username
const COPILOT_REVIEWER_BOT = "copilot-pull-request-reviewer[bot]";

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const reviewerItem = result.items.find(item => item.type === "add_reviewer");
  if (!reviewerItem) {
    core.warning("No add-reviewer item found in agent output");
    return;
  }
  core.info(`Found add-reviewer item with ${reviewerItem.reviewers.length} reviewers`);

  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Add Reviewers",
      description: "The following reviewers would be added if staged mode was disabled:",
      items: [reviewerItem],
      renderItem: item => {
        let content = "";
        if (item.pull_request_number) {
          content += `**Target Pull Request:** #${item.pull_request_number}\n\n`;
        } else {
          content += `**Target:** Current pull request\n\n`;
        }
        if (item.reviewers && item.reviewers.length > 0) {
          content += `**Reviewers to add:** ${item.reviewers.join(", ")}\n\n`;
        }
        return content;
      },
    });
    return;
  }

  // Get configuration from config.json
  const config = getSafeOutputConfig("add_reviewer");

  // Parse allowed reviewers (from env or config)
  const allowedReviewers = parseAllowedItems(process.env.GH_AW_REVIEWERS_ALLOWED) || config.reviewers;
  if (allowedReviewers) {
    core.info(`Allowed reviewers: ${JSON.stringify(allowedReviewers)}`);
  } else {
    core.info("No reviewer restrictions - any reviewers are allowed");
  }

  // Parse max count (env takes priority, then config)
  const maxCountResult = validateMaxCount(process.env.GH_AW_REVIEWERS_MAX_COUNT, config.max);
  if (!maxCountResult.valid) {
    core.setFailed(maxCountResult.error);
    return;
  }
  const maxCount = maxCountResult.value;
  core.info(`Max count: ${maxCount}`);

  // Resolve target
  const reviewersTarget = process.env.GH_AW_REVIEWERS_TARGET || "triggering";
  core.info(`Reviewers target configuration: ${reviewersTarget}`);

  const targetResult = resolveTarget({
    targetConfig: reviewersTarget,
    item: reviewerItem,
    context,
    itemType: "reviewer addition",
    supportsPR: false,
  });

  if (!targetResult.success) {
    if (targetResult.shouldFail) {
      core.setFailed(targetResult.error);
    } else {
      core.info(targetResult.error);
    }
    return;
  }

  const prNumber = targetResult.number;

  const requestedReviewers = reviewerItem.reviewers || [];
  core.info(`Requested reviewers: ${JSON.stringify(requestedReviewers)}`);

  // Filter by allowed reviewers if configured
  let validReviewers;
  if (allowedReviewers) {
    validReviewers = requestedReviewers.filter(reviewer => allowedReviewers.includes(reviewer));
  } else {
    validReviewers = requestedReviewers;
  }

  // Sanitize and deduplicate reviewers
  let uniqueReviewers = validReviewers
    .filter(reviewer => reviewer != null && reviewer !== false && reviewer !== 0)
    .map(reviewer => String(reviewer).trim())
    .filter(reviewer => reviewer)
    .filter((reviewer, index, arr) => arr.indexOf(reviewer) === index);

  // Apply max count limit
  if (uniqueReviewers.length > maxCount) {
    core.info(`Too many reviewers, keeping ${maxCount}`);
    uniqueReviewers = uniqueReviewers.slice(0, maxCount);
  }

  if (uniqueReviewers.length === 0) {
    core.info("No reviewers to add");
    core.setOutput("reviewers_added", "");
    await core.summary
      .addRaw(
        `
## Reviewer Addition

No reviewers were added (no valid reviewers found in agent output).
`
      )
      .write();
    return;
  }

  core.info(`Adding ${uniqueReviewers.length} reviewers to PR #${prNumber}: ${JSON.stringify(uniqueReviewers)}`);

  try {
    // Special handling for "copilot" reviewer - separate it from other reviewers in a single pass
    const hasCopilot = uniqueReviewers.includes("copilot");
    const otherReviewers = hasCopilot ? uniqueReviewers.filter(r => r !== "copilot") : uniqueReviewers;

    // Add non-copilot reviewers first
    if (otherReviewers.length > 0) {
      await github.rest.pulls.requestReviewers({
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: prNumber,
        reviewers: otherReviewers,
      });
      core.info(`Successfully added ${otherReviewers.length} reviewer(s) to PR #${prNumber}`);
    }

    // Add copilot reviewer separately if requested
    if (hasCopilot) {
      try {
        await github.rest.pulls.requestReviewers({
          owner: context.repo.owner,
          repo: context.repo.repo,
          pull_number: prNumber,
          reviewers: [COPILOT_REVIEWER_BOT],
        });
        core.info(`Successfully added copilot as reviewer to PR #${prNumber}`);
      } catch (copilotError) {
        core.warning(`Failed to add copilot as reviewer: ${copilotError instanceof Error ? copilotError.message : String(copilotError)}`);
        // Don't fail the whole step if copilot reviewer fails
      }
    }

    core.setOutput("reviewers_added", uniqueReviewers.join("\n"));

    const reviewersListMarkdown = uniqueReviewers.map(reviewer => `- \`${reviewer}\``).join("\n");
    await core.summary
      .addRaw(
        `
## Reviewer Addition

Successfully added ${uniqueReviewers.length} reviewer(s) to PR #${prNumber}:

${reviewersListMarkdown}
`
      )
      .write();
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to add reviewers: ${errorMessage}`);
    core.setFailed(`Failed to add reviewers: ${errorMessage}`);
  }
}

await main();
