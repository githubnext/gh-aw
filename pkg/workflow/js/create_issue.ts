import type { SafeOutputItems } from "./types/safe-outputs";

/**
 * Sanitizes label content for safe output
 * @param content - The label content to sanitize
 * @returns The sanitized label content
 */
function sanitizeLabelContent(content: string): string {
  if (!content || typeof content !== "string") {
    return "";
  }

  let sanitized = content.trim();

  // Remove control characters (except newlines and tabs, but labels shouldn't have these anyway)
  sanitized = sanitized.replace(/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/g, "");

  // Remove ANSI escape sequences
  sanitized = sanitized.replace(/\x1b\[[0-9;]*[mGKH]/g, "");

  // Neutralize @mentions in labels to prevent unintended notifications
  sanitized = sanitized.replace(
    /(^|[^\w`])@([A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)/g,
    (_m, p1, p2) => `${p1}\`@${p2}\``
  );

  // For labels, convert any remaining problematic characters
  sanitized = sanitized.replace(/[<>&'"]/g, "");

  return sanitized.trim();
}

interface CreatedIssue {
  number: number;
  title: string;
  html_url: string;
}

async function main(): Promise<void> {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput: SafeOutputItems;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all create-issue items
  const createIssueItems = validatedOutput.items.filter(item => item.type === "create-issue");
  if (createIssueItems.length === 0) {
    core.info("No create-issue items found in agent output");
    return;
  }

  core.info(`Found ${createIssueItems.length} create-issue item(s)`);

  // If in staged mode, emit step summary instead of creating issues
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Create Issues Preview\n\n";
    summaryContent += "The following issues would be created if staged mode was disabled:\n\n";

    for (let i = 0; i < createIssueItems.length; i++) {
      const item = createIssueItems[i];
      summaryContent += `### Issue ${i + 1}\n`;
      summaryContent += `**Title:** ${item.title || "No title provided"}\n\n`;
      if (item.body) {
        summaryContent += `**Body:**\n${item.body}\n\n`;
      }
      if (item.labels && item.labels.length > 0) {
        summaryContent += `**Labels:** ${item.labels.join(", ")}\n\n`;
      }
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Issue creation preview written to step summary");
    return;
  }

  // Check if we're in an issue context (triggered by an issue event)
  const parentIssueNumber = context.payload?.issue?.number;

  // Parse labels from environment variable (comma-separated string)
  const labelsEnv = process.env.GITHUB_AW_ISSUE_LABELS;
  let envLabels: string[] = labelsEnv
    ? labelsEnv
        .split(",")
        .map((label: string) => label.trim())
        .filter((label: string) => label)
    : [];

  const createdIssues: CreatedIssue[] = [];

  // Process each create-issue item
  for (let i = 0; i < createIssueItems.length; i++) {
    const createIssueItem = createIssueItems[i];
    core.info(
      `Processing create-issue item ${i + 1}/${createIssueItems.length}: title=${createIssueItem.title}, bodyLength=${createIssueItem.body.length}`
    );

    // Merge environment labels with item-specific labels
    let labels = [...envLabels];
    if (createIssueItem.labels && Array.isArray(createIssueItem.labels)) {
      labels = [...labels, ...createIssueItem.labels];
    }

    // Clean up labels: remove duplicates, empty labels, limit length, and sanitize
    labels = (labels as unknown[])
      .filter(label => label != null && label !== false && label !== 0) // Remove null, undefined, false, 0
      .map(label => String(label).trim()) // Ensure string and trim
      .filter(label => label) // Remove empty strings after trimming
      .map(label => sanitizeLabelContent(label)) // Sanitize content
      .filter(label => label) // Remove empty labels after sanitization
      .map(label => (label.length > 64 ? label.substring(0, 64) : label)) // Limit to 64 characters
      .filter((label, index, arr) => arr.indexOf(label) === index); // Remove duplicates

    // Extract title and body from the JSON item
    let title = createIssueItem.title ? createIssueItem.title.trim() : "";
    let bodyLines = createIssueItem.body.split("\n");

    // If no title was found, use the body content as title (or a default)
    if (!title) {
      title = createIssueItem.body || "Agent Output";
    }

    // Apply title prefix if provided via environment variable
    const titlePrefix = process.env.GITHUB_AW_ISSUE_TITLE_PREFIX;
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }

    if (parentIssueNumber) {
      core.info("Detected issue context, parent issue #" + parentIssueNumber);

      // Add reference to parent issue in the child issue body
      bodyLines.push(`Related to #${parentIssueNumber}`);
    }

    // Add AI disclaimer with run id, run htmlurl
    // Add AI disclaimer with workflow run information
    const runId = context.runId;
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `https://github.com/actions/runs/${runId}`;
    bodyLines.push(``, ``, `> Generated by Agentic Workflow [Run](${runUrl})`, "");

    // Prepare the body content
    const body = bodyLines.join("\n").trim();

    core.info(`Creating issue with title: ${title}`);
    core.info(`Labels: ${labels}`);
    core.info(`Body length: ${body.length}`);

    try {
      // Create the issue using GitHub API
      const { data: issue } = await github.rest.issues.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title: title,
        body: body,
        labels: labels,
      });

      core.info("Created issue #" + issue.number + ": " + issue.html_url);
      createdIssues.push(issue);

      // If we have a parent issue, add a comment to it referencing the new child issue
      if (parentIssueNumber) {
        try {
          await github.rest.issues.createComment({
            owner: context.repo.owner,
            repo: context.repo.repo,
            issue_number: parentIssueNumber,
            body: `Created related issue: #${issue.number}`,
          });
          core.info("Added comment to parent issue #" + parentIssueNumber);
        } catch (error) {
          core.info(`Warning: Could not add comment to parent issue: ${error instanceof Error ? error.message : String(error)}`);
        }
      }

      // Set output for the last created issue (for backward compatibility)
      if (i === createIssueItems.length - 1) {
        core.setOutput("issue_number", issue.number);
        core.setOutput("issue_url", issue.html_url);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);

      // Special handling for disabled issues repository
      if (errorMessage.includes("Issues has been disabled in this repository")) {
        core.info(`⚠ Cannot create issue "${title}": Issues are disabled for this repository`);
        core.info("Consider enabling issues in repository settings if you want to create issues automatically");
        continue; // Skip this issue but continue processing others
      }

      core.error(`✗ Failed to create issue "${title}": ${errorMessage}`);
      throw error;
    }
  }

  // Write summary for all created issues
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
