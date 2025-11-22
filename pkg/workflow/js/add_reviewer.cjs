// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

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

  // Parse allowed reviewers from environment (if configured)
  const allowedReviewersEnv = process.env.GH_AW_REVIEWERS_ALLOWED?.trim();
  const allowedReviewers = allowedReviewersEnv
    ? allowedReviewersEnv
        .split(",")
        .map(reviewer => reviewer.trim())
        .filter(reviewer => reviewer)
    : undefined;

  if (allowedReviewers) {
    core.info(`Allowed reviewers: ${JSON.stringify(allowedReviewers)}`);
  } else {
    core.info("No reviewer restrictions - any reviewers are allowed");
  }

  // Parse max count from environment
  const maxCountEnv = process.env.GH_AW_REVIEWERS_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 3;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }
  core.info(`Max count: ${maxCount}`);

  // Parse target configuration
  const reviewersTarget = process.env.GH_AW_REVIEWERS_TARGET || "triggering";
  core.info(`Reviewers target configuration: ${reviewersTarget}`);

  // Check if we're in a PR context
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";

  if (reviewersTarget === "triggering" && !isPRContext) {
    core.info('Target is "triggering" but not running in pull request context, skipping reviewer addition');
    return;
  }

  // Determine the pull request number
  let prNumber;
  if (reviewersTarget === "*") {
    if (reviewerItem.pull_request_number) {
      prNumber = typeof reviewerItem.pull_request_number === "number" 
        ? reviewerItem.pull_request_number 
        : parseInt(String(reviewerItem.pull_request_number), 10);
      if (isNaN(prNumber) || prNumber <= 0) {
        core.setFailed(`Invalid pull_request_number specified: ${reviewerItem.pull_request_number}`);
        return;
      }
    } else {
      core.setFailed('Target is "*" but no pull_request_number specified in reviewer item');
      return;
    }
  } else if (reviewersTarget && reviewersTarget !== "triggering") {
    prNumber = parseInt(reviewersTarget, 10);
    if (isNaN(prNumber) || prNumber <= 0) {
      core.setFailed(`Invalid pull request number in target configuration: ${reviewersTarget}`);
      return;
    }
  } else {
    // Use triggering PR
    if (isPRContext) {
      if (context.payload.pull_request) {
        prNumber = context.payload.pull_request.number;
      } else {
        core.setFailed("Pull request context detected but no pull request found in payload");
        return;
      }
    }
  }

  if (!prNumber) {
    core.setFailed("Could not determine pull request number");
    return;
  }

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
    const otherReviewers = hasCopilot 
      ? uniqueReviewers.filter(r => r !== "copilot")
      : uniqueReviewers;

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
