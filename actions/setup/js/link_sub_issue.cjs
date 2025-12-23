// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { loadTemporaryIdMap, resolveIssueNumber } = require("./temporary_id.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const linkItems = result.items.filter(item => item.type === "link_sub_issue");
  if (linkItems.length === 0) {
    core.info("No link_sub_issue items found in agent output");
    return;
  }

  core.info(`Found ${linkItems.length} link_sub_issue item(s)`);

  // Load the temporary ID map from create_issue job
  const temporaryIdMap = loadTemporaryIdMap();
  if (temporaryIdMap.size > 0) {
    core.info(`Loaded temporary ID map with ${temporaryIdMap.size} entries`);
  }

  // Check if we're in staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Link Sub-Issue",
      description: "The following sub-issue links would be created if staged mode was disabled:",
      items: linkItems,
      renderItem: item => {
        // Resolve temporary IDs for display
        const parentResolved = resolveIssueNumber(item.parent_issue_number, temporaryIdMap);
        const subResolved = resolveIssueNumber(item.sub_issue_number, temporaryIdMap);

        let parentDisplay = parentResolved.resolved ? `${parentResolved.resolved.repo}#${parentResolved.resolved.number}` : `${item.parent_issue_number} (unresolved)`;
        let subDisplay = subResolved.resolved ? `${subResolved.resolved.repo}#${subResolved.resolved.number}` : `${item.sub_issue_number} (unresolved)`;

        if (parentResolved.wasTemporaryId && parentResolved.resolved) {
          parentDisplay += ` (from ${item.parent_issue_number})`;
        }
        if (subResolved.wasTemporaryId && subResolved.resolved) {
          subDisplay += ` (from ${item.sub_issue_number})`;
        }

        let content = `**Parent Issue:** ${parentDisplay}\n`;
        content += `**Sub-Issue:** ${subDisplay}\n\n`;
        return content;
      },
    });
    return;
  }

  // Get filter configurations
  const parentRequiredLabelsEnv = process.env.GH_AW_LINK_SUB_ISSUE_PARENT_REQUIRED_LABELS?.trim();
  const parentRequiredLabels = parentRequiredLabelsEnv
    ? parentRequiredLabelsEnv
        .split(",")
        .map(l => l.trim())
        .filter(l => l)
    : [];

  const parentTitlePrefix = process.env.GH_AW_LINK_SUB_ISSUE_PARENT_TITLE_PREFIX?.trim() || "";

  const subRequiredLabelsEnv = process.env.GH_AW_LINK_SUB_ISSUE_SUB_REQUIRED_LABELS?.trim();
  const subRequiredLabels = subRequiredLabelsEnv
    ? subRequiredLabelsEnv
        .split(",")
        .map(l => l.trim())
        .filter(l => l)
    : [];

  const subTitlePrefix = process.env.GH_AW_LINK_SUB_ISSUE_SUB_TITLE_PREFIX?.trim() || "";

  if (parentRequiredLabels.length > 0) {
    core.info(`Parent required labels: ${JSON.stringify(parentRequiredLabels)}`);
  }
  if (parentTitlePrefix) {
    core.info(`Parent title prefix: ${parentTitlePrefix}`);
  }
  if (subRequiredLabels.length > 0) {
    core.info(`Sub-issue required labels: ${JSON.stringify(subRequiredLabels)}`);
  }
  if (subTitlePrefix) {
    core.info(`Sub-issue title prefix: ${subTitlePrefix}`);
  }

  // Get max count configuration
  const maxCountEnv = process.env.GH_AW_LINK_SUB_ISSUE_MAX_COUNT;
  const maxCount = maxCountEnv ? parseInt(maxCountEnv, 10) : 5;
  if (isNaN(maxCount) || maxCount < 1) {
    core.setFailed(`Invalid max value: ${maxCountEnv}. Must be a positive integer`);
    return;
  }
  core.info(`Max count: ${maxCount}`);

  // Limit items to max count
  const itemsToProcess = linkItems.slice(0, maxCount);
  if (linkItems.length > maxCount) {
    core.warning(`Found ${linkItems.length} link_sub_issue items, but max is ${maxCount}. Processing first ${maxCount}.`);
  }

  // Process each link request
  const results = [];
  for (const item of itemsToProcess) {
    // Resolve issue numbers, supporting temporary IDs from create_issue job
    const parentResolved = resolveIssueNumber(item.parent_issue_number, temporaryIdMap);
    const subResolved = resolveIssueNumber(item.sub_issue_number, temporaryIdMap);

    // Check for resolution errors
    if (parentResolved.errorMessage) {
      core.warning(`Failed to resolve parent issue: ${parentResolved.errorMessage}`);
      results.push({
        parent_issue_number: item.parent_issue_number,
        sub_issue_number: item.sub_issue_number,
        success: false,
        error: parentResolved.errorMessage,
      });
      continue;
    }

    if (subResolved.errorMessage) {
      core.warning(`Failed to resolve sub-issue: ${subResolved.errorMessage}`);
      results.push({
        parent_issue_number: item.parent_issue_number,
        sub_issue_number: item.sub_issue_number,
        success: false,
        error: subResolved.errorMessage,
      });
      continue;
    }

    const parentIssueNumber = parentResolved.resolved.number;
    const subIssueNumber = subResolved.resolved.number;

    if (parentResolved.wasTemporaryId) {
      core.info(`Resolved parent temporary ID '${item.parent_issue_number}' to ${parentResolved.resolved.repo}#${parentIssueNumber}`);
    }
    if (subResolved.wasTemporaryId) {
      core.info(`Resolved sub-issue temporary ID '${item.sub_issue_number}' to ${subResolved.resolved.repo}#${subIssueNumber}`);
    }

    // Fetch parent issue to validate filters
    let parentIssue;
    try {
      const parentResponse = await github.rest.issues.get({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: parentIssueNumber,
      });
      parentIssue = parentResponse.data;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.warning(`Failed to fetch parent issue #${parentIssueNumber}: ${errorMessage}`);
      results.push({
        parent_issue_number: parentIssueNumber,
        sub_issue_number: subIssueNumber,
        success: false,
        error: `Failed to fetch parent issue: ${errorMessage}`,
      });
      continue;
    }

    // Validate parent issue filters
    if (parentRequiredLabels.length > 0) {
      const parentLabels = parentIssue.labels.map(l => (typeof l === "string" ? l : l.name || ""));
      const missingLabels = parentRequiredLabels.filter(required => !parentLabels.includes(required));
      if (missingLabels.length > 0) {
        core.warning(`Parent issue #${parentIssueNumber} is missing required labels: ${missingLabels.join(", ")}. Skipping.`);
        results.push({
          parent_issue_number: parentIssueNumber,
          sub_issue_number: subIssueNumber,
          success: false,
          error: `Parent issue missing required labels: ${missingLabels.join(", ")}`,
        });
        continue;
      }
    }

    if (parentTitlePrefix && !parentIssue.title.startsWith(parentTitlePrefix)) {
      core.warning(`Parent issue #${parentIssueNumber} title does not start with "${parentTitlePrefix}". Skipping.`);
      results.push({
        parent_issue_number: parentIssueNumber,
        sub_issue_number: subIssueNumber,
        success: false,
        error: `Parent issue title does not start with "${parentTitlePrefix}"`,
      });
      continue;
    }

    // Fetch sub-issue to validate filters
    let subIssue;
    try {
      const subResponse = await github.rest.issues.get({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: subIssueNumber,
      });
      subIssue = subResponse.data;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to fetch sub-issue #${subIssueNumber}: ${errorMessage}`);
      results.push({
        parent_issue_number: parentIssueNumber,
        sub_issue_number: subIssueNumber,
        success: false,
        error: `Failed to fetch sub-issue: ${errorMessage}`,
      });
      continue;
    }

    // Check if the sub-issue already has a parent using GraphQL
    try {
      const parentCheckQuery = `
        query($owner: String!, $repo: String!, $number: Int!) {
          repository(owner: $owner, name: $repo) {
            issue(number: $number) {
              parent {
                number
                title
              }
            }
          }
        }
      `;
      const parentCheckResult = await github.graphql(parentCheckQuery, {
        owner: context.repo.owner,
        repo: context.repo.repo,
        number: subIssueNumber,
      });

      const existingParent = parentCheckResult?.repository?.issue?.parent;
      if (existingParent) {
        core.warning(`Sub-issue #${subIssueNumber} is already a sub-issue of #${existingParent.number} ("${existingParent.title}"). Skipping.`);
        results.push({
          parent_issue_number: parentIssueNumber,
          sub_issue_number: subIssueNumber,
          success: false,
          error: `Sub-issue is already a sub-issue of #${existingParent.number}`,
        });
        continue;
      }
    } catch (error) {
      // If the GraphQL query fails (e.g., parent field not available), log warning but continue
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.warning(`Could not check if sub-issue #${subIssueNumber} has a parent: ${errorMessage}. Proceeding with link attempt.`);
    }

    // Validate sub-issue filters
    if (subRequiredLabels.length > 0) {
      const subLabels = subIssue.labels.map(l => (typeof l === "string" ? l : l.name || ""));
      const missingLabels = subRequiredLabels.filter(required => !subLabels.includes(required));
      if (missingLabels.length > 0) {
        core.warning(`Sub-issue #${subIssueNumber} is missing required labels: ${missingLabels.join(", ")}. Skipping.`);
        results.push({
          parent_issue_number: parentIssueNumber,
          sub_issue_number: subIssueNumber,
          success: false,
          error: `Sub-issue missing required labels: ${missingLabels.join(", ")}`,
        });
        continue;
      }
    }

    if (subTitlePrefix && !subIssue.title.startsWith(subTitlePrefix)) {
      core.warning(`Sub-issue #${subIssueNumber} title does not start with "${subTitlePrefix}". Skipping.`);
      results.push({
        parent_issue_number: parentIssueNumber,
        sub_issue_number: subIssueNumber,
        success: false,
        error: `Sub-issue title does not start with "${subTitlePrefix}"`,
      });
      continue;
    }

    // Link the sub-issue using GraphQL mutation
    try {
      // Get the parent issue's node ID for GraphQL
      const parentNodeId = parentIssue.node_id;
      const subNodeId = subIssue.node_id;

      // Use GraphQL mutation to add sub-issue
      await github.graphql(
        `
        mutation AddSubIssue($parentId: ID!, $subIssueId: ID!) {
          addSubIssue(input: { issueId: $parentId, subIssueId: $subIssueId }) {
            issue {
              id
              number
            }
            subIssue {
              id
              number
            }
          }
        }
      `,
        {
          parentId: parentNodeId,
          subIssueId: subNodeId,
        }
      );

      core.info(`Successfully linked issue #${subIssueNumber} as sub-issue of #${parentIssueNumber}`);
      results.push({
        parent_issue_number: parentIssueNumber,
        sub_issue_number: subIssueNumber,
        success: true,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.warning(`Failed to link issue #${subIssueNumber} as sub-issue of #${parentIssueNumber}: ${errorMessage}`);
      results.push({
        parent_issue_number: parentIssueNumber,
        sub_issue_number: subIssueNumber,
        success: false,
        error: errorMessage,
      });
    }
  }

  // Generate step summary
  const successCount = results.filter(r => r.success).length;
  const failureCount = results.filter(r => !r.success).length;

  let summaryContent = "## Link Sub-Issue\n\n";

  if (successCount > 0) {
    summaryContent += `✅ Successfully linked ${successCount} sub-issue(s):\n\n`;
    for (const result of results.filter(r => r.success)) {
      summaryContent += `- Issue #${result.sub_issue_number} → Parent #${result.parent_issue_number}\n`;
    }
    summaryContent += "\n";
  }

  if (failureCount > 0) {
    summaryContent += `⚠️ Failed to link ${failureCount} sub-issue(s):\n\n`;
    for (const result of results.filter(r => !r.success)) {
      summaryContent += `- Issue #${result.sub_issue_number} → Parent #${result.parent_issue_number}: ${result.error}\n`;
    }
  }

  await core.summary.addRaw(summaryContent).write();

  // Set outputs
  const linkedIssues = results
    .filter(r => r.success)
    .map(r => `${r.parent_issue_number}:${r.sub_issue_number}`)
    .join("\n");
  core.setOutput("linked_issues", linkedIssues);

  // Warn if any linking failed (do not fail the job)
  if (failureCount > 0) {
    core.warning(`Failed to link ${failureCount} sub-issue(s). See step summary for details.`);
  }
}

module.exports = { main };
