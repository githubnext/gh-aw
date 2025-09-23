import type { SafeOutputItems } from "./types/safe-outputs.js";

async function main() {
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

  // Find all update-release items
  const updateItems = validatedOutput.items.filter((item: any) => item.type === "update-release");
  if (updateItems.length === 0) {
    core.info("No update-release items found in agent output");
    return;
  }

  core.info(`Found ${updateItems.length} update-release item(s)`);

  // If in staged mode, emit step summary instead of updating releases
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Update Releases Preview\n\n";
    summaryContent += "The following release updates would be applied if staged mode was disabled:\n\n";

    for (let i = 0; i < updateItems.length; i++) {
      const item = updateItems[i] as any;
      summaryContent += `### Release Update ${i + 1}\n`;
      if (item.release_id) {
        summaryContent += `- **Release ID**: ${item.release_id}\n`;
      }
      if (item.body) {
        summaryContent += `- **Body Update**: ${item.body.length > 100 ? item.body.substring(0, 100) + "..." : item.body}\n`;
      }
      summaryContent += "\n";
    }

    core.summary.addRaw(summaryContent).write();
    core.info("Staged mode: Release update preview added to step summary");
    return;
  }

  // Get configuration
  const updateTarget = process.env.GITHUB_AW_UPDATE_TARGET || "triggering";
  const releaseIdFromEnv = process.env.GITHUB_AW_RELEASE_ID;

  core.info(`Update target configuration: ${updateTarget}`);
  core.info(`Release ID from environment: ${releaseIdFromEnv || "not set"}`);

  // Check if we're in a release context
  const isReleaseContext = context.eventName === "release";

  // Validate context based on target configuration
  if (updateTarget === "triggering" && !isReleaseContext && !releaseIdFromEnv) {
    core.info('Target is "triggering" but not running in release context and no explicit release ID provided, skipping release update');
    return;
  }

  const updatedReleases: Array<{ id: number; url: string; tag_name: string }> = [];

  // Process each update item
  for (let i = 0; i < updateItems.length; i++) {
    const updateItem = updateItems[i] as any;
    core.info(`Processing update-release item ${i + 1}/${updateItems.length}`);

    // Determine the release ID for this update
    let releaseId;

    if (updateTarget === "*") {
      // For target "*", we need an explicit release ID from the update item
      if (updateItem.release_id) {
        releaseId = updateItem.release_id;
      } else {
        core.info('Target is "*" but no release_id specified in update item');
        continue;
      }
    } else if (releaseIdFromEnv) {
      // Use release ID from environment (frontmatter configuration)
      releaseId = releaseIdFromEnv;
    } else if (updateTarget && updateTarget !== "triggering") {
      // Explicit release ID specified in target
      releaseId = updateTarget;
    } else {
      // Default behavior: use triggering release
      if (isReleaseContext) {
        if (context.payload.release) {
          releaseId = context.payload.release.id;
        } else {
          core.info("Release context detected but no release found in payload");
          continue;
        }
      } else {
        core.info("Could not determine release ID");
        continue;
      }
    }

    if (!releaseId) {
      core.info("Could not determine release ID");
      continue;
    }

    core.info(`Updating release ${releaseId}`);

    // Build the update data
    if (!updateItem.body) {
      core.info("No body content provided for release update, skipping");
      continue;
    }

    try {
      // Get the current release to read existing body
      const { data: currentRelease } = await github.rest.repos.getRelease({
        owner: context.repo.owner,
        repo: context.repo.repo,
        release_id: parseInt(String(releaseId), 10),
      });

      // Create updated body by appending to existing description in a section
      const sectionMarker = "\n\n## AI Agent Update\n\n";
      let updatedBody = currentRelease.body || "";

      // Check if the AI Agent Update section already exists
      const sectionIndex = updatedBody.indexOf(sectionMarker);
      if (sectionIndex !== -1) {
        // Section exists, append to it
        updatedBody = updatedBody + "\n\n" + updateItem.body;
      } else {
        // Section doesn't exist, create it
        updatedBody = updatedBody + sectionMarker + updateItem.body;
      }

      // Update the release
      const { data: updatedRelease } = await github.rest.repos.updateRelease({
        owner: context.repo.owner,
        repo: context.repo.repo,
        release_id: parseInt(String(releaseId), 10),
        body: updatedBody,
      });

      updatedReleases.push({
        id: updatedRelease.id,
        url: updatedRelease.html_url,
        tag_name: updatedRelease.tag_name,
      });

      core.info(`Successfully updated release ${updatedRelease.tag_name} (${updatedRelease.id}): ${updatedRelease.html_url}`);
    } catch (error) {
      core.error(`Error updating release ${releaseId}: ${error instanceof Error ? error.message : String(error)}`);
      continue;
    }
  }

  // Set outputs for the first updated release (for backwards compatibility)
  if (updatedReleases.length > 0) {
    const firstRelease = updatedReleases[0];
    core.setOutput("release_id", firstRelease.id);
    core.setOutput("release_url", firstRelease.url);
  }

  core.info(`Successfully updated ${updatedReleases.length} release(s)`);
}

// Execute main function
main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
