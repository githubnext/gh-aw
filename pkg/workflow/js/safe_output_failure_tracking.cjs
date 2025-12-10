// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Main function to track and report safe output job failures
 * This creates or updates an issue when safe output jobs fail
 */
async function main() {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Unknown Workflow";
  const runId = process.env.GH_AW_RUN_ID || "unknown";
  const runUrl = process.env.GH_AW_RUN_URL || "";
  const repository = process.env.GH_AW_REPOSITORY || "";

  // Parse repository to get owner and repo
  const [owner, repo] = repository.split("/");
  if (!owner || !repo) {
    core.info("Invalid repository format, skipping failure tracking");
    return;
  }

  // Collect all failed safe output jobs
  const failedJobs = [];
  for (const [key, value] of Object.entries(process.env)) {
    if (key.startsWith("GH_AW_JOB_") && key.endsWith("_RESULT")) {
      if (value === "failure") {
        // Extract job name from environment variable name
        const jobName = key.replace("GH_AW_JOB_", "").replace("_RESULT", "").toLowerCase().replace(/_/g, "_");

        const errorKey = `GH_AW_JOB_${jobName.toUpperCase().replace(/-/g, "_")}_ERROR`;
        const errorMessage = process.env[errorKey] || "No error message available";

        failedJobs.push({
          name: jobName,
          error: errorMessage,
        });
      }
    }
  }

  // If no jobs failed, nothing to report
  if (failedJobs.length === 0) {
    core.info("No safe output jobs failed, skipping failure tracking");
    return;
  }

  core.info(`Found ${failedJobs.length} failed safe output job(s)`);
  for (const job of failedJobs) {
    core.info(`  - ${job.name}`);
  }

  // Issue configuration
  const titlePrefix = "[ca] ";
  const label = "agentics";
  const issueTitle = `${titlePrefix}Safe Output Failure: ${workflowName}`;

  // Check if an issue already exists with this title
  let existingIssue = null;
  try {
    const issues = await github.rest.issues.listForRepo({
      owner,
      repo,
      state: "open",
      labels: label,
      per_page: 100,
    });

    existingIssue = issues.data.find(issue => issue.title === issueTitle);
  } catch (error) {
    core.warning(`Failed to search for existing issues: ${error instanceof Error ? error.message : String(error)}`);
  }

  // Build the issue body or comment
  const failureDetails = failedJobs.map(job => `- **${job.name}**: Run [#${runId}](${runUrl})`).join("\n");

  const timestamp = new Date().toISOString();
  const commentBody = `## Safe Output Failure Report

**Workflow:** ${workflowName}
**Run ID:** [${runId}](${runUrl})
**Timestamp:** ${timestamp}

### Failed Jobs:
${failureDetails}

---
*This is an automated report. Do not include error messages as they may be tainted.*`;

  if (existingIssue) {
    // Add a comment to the existing issue
    try {
      await github.rest.issues.createComment({
        owner,
        repo,
        issue_number: existingIssue.number,
        body: commentBody,
      });
      core.info(`Added comment to existing issue #${existingIssue.number}: ${existingIssue.html_url}`);
      await core.summary
        .addRaw(`\n\n## ⚠️ Safe Output Failure Tracked\n\nAdded comment to issue [#${existingIssue.number}](${existingIssue.html_url})\n`)
        .write();
    } catch (error) {
      core.error(`Failed to add comment to issue: ${error instanceof Error ? error.message : String(error)}`);
    }
  } else {
    // Create a new issue
    const issueBody = `## Safe Output Failure Tracking

This issue tracks failures in safe output jobs for the workflow: **${workflowName}**

### Latest Failure:
${commentBody}`;

    try {
      const issue = await github.rest.issues.create({
        owner,
        repo,
        title: issueTitle,
        body: issueBody,
        labels: [label],
      });
      core.info(`Created new issue #${issue.data.number}: ${issue.data.html_url}`);
      await core.summary
        .addRaw(`\n\n## ⚠️ Safe Output Failure Tracked\n\nCreated issue [#${issue.data.number}](${issue.data.html_url})\n`)
        .write();
    } catch (error) {
      core.error(`Failed to create issue: ${error instanceof Error ? error.message : String(error)}`);
    }
  }
}

await main();
