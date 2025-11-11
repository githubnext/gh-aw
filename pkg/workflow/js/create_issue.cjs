// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { generateFingerprintComment } = require("./generate_fingerprint_comment.cjs");
const { getFingerprint } = require("./get_fingerprint.cjs");

async function main() {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("issue_number", "");
  core.setOutput("issue_url", "");

  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const createIssueItems = result.items.filter(item => item.type === "create_issue");
  if (createIssueItems.length === 0) {
    core.info("No create-issue items found in agent output");
    return;
  }
  core.info(`Found ${createIssueItems.length} create-issue item(s)`);
  if (isStaged) {
    await generateStagedPreview({
      title: "Create Issues",
      description: "The following issues would be created if staged mode was disabled:",
      items: createIssueItems,
      renderItem: (item, index) => {
        let content = `### Issue ${index + 1}\n`;
        content += `**Title:** ${item.title || "No title provided"}\n\n`;
        if (item.body) {
          content += `**Body:**\n${item.body}\n\n`;
        }
        if (item.labels && item.labels.length > 0) {
          content += `**Labels:** ${item.labels.join(", ")}\n\n`;
        }
        return content;
      },
    });
    return;
  }
  const parentIssueNumber = context.payload?.issue?.number;

  // Extract triggering context for footer generation
  const triggeringIssueNumber =
    context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber =
    context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const labelsEnv = process.env.GH_AW_ISSUE_LABELS;
  let envLabels = labelsEnv
    ? labelsEnv
        .split(",")
        .map(label => label.trim())
        .filter(label => label)
    : [];
  const createdIssues = [];
  for (let i = 0; i < createIssueItems.length; i++) {
    const createIssueItem = createIssueItems[i];
    core.info(
      `Processing create-issue item ${i + 1}/${createIssueItems.length}: title=${createIssueItem.title}, bodyLength=${createIssueItem.body.length}`
    );

    // Debug logging for parent field
    core.info(`Debug: createIssueItem.parent = ${JSON.stringify(createIssueItem.parent)}`);
    core.info(`Debug: parentIssueNumber from context = ${JSON.stringify(parentIssueNumber)}`);

    // Use the parent field from the item if provided, otherwise fall back to context
    const effectiveParentIssueNumber = createIssueItem.parent !== undefined ? createIssueItem.parent : parentIssueNumber;
    core.info(`Debug: effectiveParentIssueNumber = ${JSON.stringify(effectiveParentIssueNumber)}`);

    if (effectiveParentIssueNumber && createIssueItem.parent !== undefined) {
      core.info(`Using explicit parent issue number from item: #${effectiveParentIssueNumber}`);
    }
    let labels = [...envLabels];
    if (createIssueItem.labels && Array.isArray(createIssueItem.labels)) {
      labels = [...labels, ...createIssueItem.labels];
    }
    labels = labels
      .filter(label => !!label)
      .map(label => String(label).trim())
      .filter(label => label)
      .map(label => sanitizeLabelContent(label))
      .filter(label => label)
      .map(label => (label.length > 64 ? label.substring(0, 64) : label))
      .filter((label, index, arr) => arr.indexOf(label) === index);
    let title = createIssueItem.title ? createIssueItem.title.trim() : "";
    let bodyLines = createIssueItem.body.split("\n");
    if (!title) {
      title = createIssueItem.body || "Agent Output";
    }
    const titlePrefix = process.env.GH_AW_ISSUE_TITLE_PREFIX;
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }
    if (effectiveParentIssueNumber) {
      core.info("Detected issue context, parent issue #" + effectiveParentIssueNumber);
      bodyLines.push(`Related to #${effectiveParentIssueNumber}`);
    }
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const fingerprint = getFingerprint();
    const runId = context.runId;
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

    // Add fingerprint comment if present
    const fingerprintComment = generateFingerprintComment(fingerprint);
    if (fingerprintComment) {
      bodyLines.push(fingerprintComment);
    }

    bodyLines.push(
      ``,
      ``,
      generateFooter(
        workflowName,
        runUrl,
        workflowSource,
        workflowSourceURL,
        triggeringIssueNumber,
        triggeringPRNumber,
        triggeringDiscussionNumber
      ).trimEnd(),
      ""
    );
    const body = bodyLines.join("\n").trim();
    core.info(`Creating issue with title: ${title}`);
    core.info(`Labels: ${labels}`);
    core.info(`Body length: ${body.length}`);
    try {
      const { data: issue } = await github.rest.issues.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title: title,
        body: body,
        labels: labels,
      });
      core.info("Created issue #" + issue.number + ": " + issue.html_url);
      createdIssues.push(issue);

      // Debug logging for sub-issue linking
      core.info(`Debug: About to check if sub-issue linking is needed. effectiveParentIssueNumber = ${effectiveParentIssueNumber}`);

      if (effectiveParentIssueNumber) {
        core.info(`Attempting to link issue #${issue.number} as sub-issue of #${effectiveParentIssueNumber}`);
        try {
          // First, get the node IDs for both parent and child issues
          core.info(`Fetching node ID for parent issue #${effectiveParentIssueNumber}...`);
          const getIssueNodeIdQuery = `
            query($owner: String!, $repo: String!, $issueNumber: Int!) {
              repository(owner: $owner, name: $repo) {
                issue(number: $issueNumber) {
                  id
                }
              }
            }
          `;

          // Get parent issue node ID
          const parentResult = await github.graphql(getIssueNodeIdQuery, {
            owner: context.repo.owner,
            repo: context.repo.repo,
            issueNumber: effectiveParentIssueNumber,
          });
          const parentNodeId = parentResult.repository.issue.id;
          core.info(`Parent issue node ID: ${parentNodeId}`);

          // Get child issue node ID
          core.info(`Fetching node ID for child issue #${issue.number}...`);
          const childResult = await github.graphql(getIssueNodeIdQuery, {
            owner: context.repo.owner,
            repo: context.repo.repo,
            issueNumber: issue.number,
          });
          const childNodeId = childResult.repository.issue.id;
          core.info(`Child issue node ID: ${childNodeId}`);

          // Link the child issue as a sub-issue of the parent
          core.info(`Executing addSubIssue mutation...`);
          const addSubIssueMutation = `
            mutation($issueId: ID!, $subIssueId: ID!) {
              addSubIssue(input: {
                issueId: $issueId,
                subIssueId: $subIssueId
              }) {
                subIssue {
                  id
                  number
                }
              }
            }
          `;

          await github.graphql(addSubIssueMutation, {
            issueId: parentNodeId,
            subIssueId: childNodeId,
          });

          core.info("✓ Successfully linked issue #" + issue.number + " as sub-issue of #" + effectiveParentIssueNumber);
        } catch (error) {
          core.info(`Warning: Could not link sub-issue to parent: ${error instanceof Error ? error.message : String(error)}`);
          core.info(`Error details: ${error instanceof Error ? error.stack : String(error)}`);
          // Fallback: add a comment if sub-issue linking fails
          try {
            core.info(`Attempting fallback: adding comment to parent issue #${effectiveParentIssueNumber}...`);
            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: effectiveParentIssueNumber,
              body: `Created related issue: #${issue.number}`,
            });
            core.info("✓ Added comment to parent issue #" + effectiveParentIssueNumber + " (sub-issue linking not available)");
          } catch (commentError) {
            core.info(
              `Warning: Could not add comment to parent issue: ${commentError instanceof Error ? commentError.message : String(commentError)}`
            );
          }
        }
      } else {
        core.info(`Debug: No parent issue number set, skipping sub-issue linking`);
      }
      if (i === createIssueItems.length - 1) {
        core.setOutput("issue_number", issue.number);
        core.setOutput("issue_url", issue.html_url);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      if (errorMessage.includes("Issues has been disabled in this repository")) {
        core.info(`⚠ Cannot create issue "${title}": Issues are disabled for this repository`);
        core.info("Consider enabling issues in repository settings if you want to create issues automatically");
        continue;
      }
      core.error(`✗ Failed to create issue "${title}": ${errorMessage}`);
      throw error;
    }
  }
  if (createdIssues.length > 0) {
    let summaryContent = "\n\n## GitHub Issues\n";
    for (const issue of createdIssues) {
      summaryContent += `- Issue #${issue.number}: [${issue.title}](${issue.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }
  core.info(`Successfully created ${createdIssues.length} issue(s)`);
}
(async () => {
  await main();
})();
