// @ts-check
/// <reference types="@actions/github-script" />

const { processSafeOutput, processItems } = require("./safe_output_processor.cjs");

// GitHub Copilot reviewer bot username
const COPILOT_REVIEWER_BOT = "copilot-pull-request-reviewer[bot]";

async function main() {
  // Use shared processor for common steps
  const result = await processSafeOutput(
    {
      itemType: "add_reviewer",
      configKey: "add_reviewer",
      displayName: "Reviewers",
      itemTypeName: "reviewer addition",
      supportsPR: false, // PR-only: supportsPR=false means ONLY PR context (not issues)
      supportsIssue: false,
      envVars: {
        allowed: "GH_AW_REVIEWERS_ALLOWED",
        maxCount: "GH_AW_REVIEWERS_MAX_COUNT",
        target: "GH_AW_REVIEWERS_TARGET",
      },
    },
    {
      title: "Add Reviewers",
      description: "The following reviewers would be added if staged mode was disabled:",
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
    }
  );

  if (!result.success) {
    return;
  }

  // @ts-ignore - TypeScript doesn't narrow properly after success check
  const { item: reviewerItem, config, targetResult } = result;
  if (!config || !targetResult || targetResult.number === undefined) {
    core.setFailed("Internal error: config, targetResult, or targetResult.number is undefined");
    return;
  }
  const { allowed: allowedReviewers, maxCount } = config;
  const prNumber = targetResult.number;

  const requestedReviewers = reviewerItem.reviewers || [];
  core.info(`Requested reviewers: ${JSON.stringify(requestedReviewers)}`);

  // Use shared helper to filter, sanitize, dedupe, and limit
  const uniqueReviewers = processItems(requestedReviewers, allowedReviewers, maxCount);

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
