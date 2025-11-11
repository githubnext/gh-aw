// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find commit_status items
  const commitStatusItems = result.items.filter(/** @param {any} item */ item => item.type === "commit_status");

  if (commitStatusItems.length === 0) {
    core.info("No commit_status items found in agent output");
    return;
  }

  core.info(`Found ${commitStatusItems.length} commit_status item(s)`);

  // If in staged mode, emit step summary instead of performing actions
  if (isStaged) {
    await generateStagedPreview({
      title: "Commit Status Update",
      description: "The following commit status would be updated if staged mode was disabled:",
      items: commitStatusItems,
      renderItem: item => {
        let content = "";
        content += `**State:** ${item.state}\n`;
        content += `**Description:** ${item.description}\n`;
        if (item.context) {
          content += `**Context:** ${item.context}\n`;
        }
        content += "\n";
        return content;
      },
    });
    return;
  }

  // Get commit SHA from environment
  const commitSha = process.env.GH_AW_COMMIT_SHA;
  if (!commitSha) {
    core.info("No commit SHA available - skipping commit status update");
    return;
  }

  // Get context from environment or use default
  const defaultContext = process.env.GH_AW_COMMIT_STATUS_CONTEXT || "agentic-workflow";

  // Get the run URL for target_url
  const runUrl = `${context.payload.repository.html_url}/actions/runs/${context.runId}`;

  // Process each commit status item
  for (const item of commitStatusItems) {
    try {
      const statusContext = item.context || defaultContext;

      core.info(`Updating commit status for ${commitSha}`);
      core.info(`State: ${item.state}`);
      core.info(`Description: ${item.description}`);
      core.info(`Context: ${statusContext}`);

      // Update commit status using GitHub REST API
      await github.rest.repos.createCommitStatus({
        owner: context.repo.owner,
        repo: context.repo.repo,
        sha: commitSha,
        state: item.state,
        description: item.description,
        context: statusContext,
        target_url: runUrl,
      });

      core.info(`Successfully updated commit status to '${item.state}'`);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to update commit status: ${errorMessage}`);
      core.setFailed(`Failed to update commit status: ${errorMessage}`);
      return;
    }
  }

  // Write summary
  await core.summary
    .addRaw(
      `
## Commit Status Updated

Successfully updated commit status for commit \`${commitSha.substring(0, 7)}\`:

${commitStatusItems.map((item, i) => `**Status ${i + 1}:** ${item.state} - ${item.description}`).join("\n")}
`
    )
    .write();
}

await main();
