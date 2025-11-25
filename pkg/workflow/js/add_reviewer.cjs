// @ts-check
/// <reference types="@actions/github-script" />

const { executeListAction, validateListItems, renderMarkdownList } = require("./safe_output_list_action.cjs");
const { parseAllowedItems } = require("./safe_output_helpers.cjs");
const { getSafeOutputConfig, validateMaxCount } = require("./safe_output_validator.cjs");

// GitHub Copilot reviewer bot username
const COPILOT_REVIEWER_BOT = "copilot-pull-request-reviewer[bot]";

await executeListAction({
  itemType: "add_reviewer",
  singularNoun: "reviewer",
  pluralNoun: "reviewers",
  itemsField: "reviewers",
  configKey: "add_reviewer",
  configAllowedField: "reviewers",
  envAllowedVar: "GH_AW_REVIEWERS_ALLOWED",
  envMaxCountVar: "GH_AW_REVIEWERS_MAX_COUNT",
  envTargetVar: "GH_AW_REVIEWERS_TARGET",
  targetNumberField: "pull_request_number",
  supportsPR: false,
  stagedPreviewTitle: "Add Reviewers",
  stagedPreviewDescription: "The following reviewers would be added if staged mode was disabled:",
  renderStagedItem: item => {
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
  validateItems: requestedReviewers => {
    // Get configuration for custom validation
    const config = getSafeOutputConfig("add_reviewer");
    const allowedReviewers = parseAllowedItems(process.env.GH_AW_REVIEWERS_ALLOWED) || config.reviewers;
    const maxCountResult = validateMaxCount(process.env.GH_AW_REVIEWERS_MAX_COUNT, config.max);
    const maxCount = maxCountResult.valid ? maxCountResult.value : 1;
    return validateListItems(requestedReviewers, allowedReviewers, maxCount);
  },
  applyAction: async (reviewers, contextType, prNumber) => {
    // Special handling for "copilot" reviewer - separate it from other reviewers in a single pass
    const hasCopilot = reviewers.includes("copilot");
    const otherReviewers = hasCopilot ? reviewers.filter(r => r !== "copilot") : reviewers;

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
  },
  outputField: "reviewers_added",
  summaryTitle: "Reviewer Addition",
  renderSuccessSummary: (reviewers, contextType, prNumber) => {
    const reviewersListMarkdown = renderMarkdownList(reviewers);
    return `
## Reviewer Addition

Successfully added ${reviewers.length} reviewer(s) to PR #${prNumber}:

${reviewersListMarkdown}
`;
  },
});
