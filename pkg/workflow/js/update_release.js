async function main() {
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";
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
  let validatedOutput;
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
  const updateItems = validatedOutput.items.filter(item => item.type === "update-release");
  if (updateItems.length === 0) {
    core.info("No update-release items found in agent output");
    return;
  }
  core.info(`Found ${updateItems.length} update-release item(s)`);
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Update Releases Preview\n\n";
    summaryContent += "The following release updates would be applied if staged mode was disabled:\n\n";
    for (let i = 0; i < updateItems.length; i++) {
      const item = updateItems[i];
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
  const updateTarget = process.env.GITHUB_AW_UPDATE_TARGET || "triggering";
  const releaseIdFromEnv = process.env.GITHUB_AW_RELEASE_ID;
  core.info(`Update target configuration: ${updateTarget}`);
  core.info(`Release ID from environment: ${releaseIdFromEnv || "not set"}`);
  const isReleaseContext = context.eventName === "release";
  if (updateTarget === "triggering" && !isReleaseContext && !releaseIdFromEnv) {
    core.info('Target is "triggering" but not running in release context and no explicit release ID provided, skipping release update');
    return;
  }
  const updatedReleases = [];
  for (let i = 0; i < updateItems.length; i++) {
    const updateItem = updateItems[i];
    core.info(`Processing update-release item ${i + 1}/${updateItems.length}`);
    let releaseId;
    if (updateTarget === "*") {
      if (updateItem.release_id) {
        releaseId = updateItem.release_id;
      } else {
        core.info('Target is "*" but no release_id specified in update item');
        continue;
      }
    } else if (releaseIdFromEnv) {
      releaseId = releaseIdFromEnv;
    } else if (updateTarget && updateTarget !== "triggering") {
      releaseId = updateTarget;
    } else {
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
    if (!updateItem.body) {
      core.info("No body content provided for release update, skipping");
      continue;
    }
    try {
      const { data: currentRelease } = await github.rest.repos.getRelease({
        owner: context.repo.owner,
        repo: context.repo.repo,
        release_id: parseInt(String(releaseId), 10),
      });
      const sectionMarker = "\n\n## AI Agent Update\n\n";
      let updatedBody = currentRelease.body || "";
      const sectionIndex = updatedBody.indexOf(sectionMarker);
      if (sectionIndex !== -1) {
        updatedBody = updatedBody + "\n\n" + updateItem.body;
      } else {
        updatedBody = updatedBody + sectionMarker + updateItem.body;
      }
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
  if (updatedReleases.length > 0) {
    const firstRelease = updatedReleases[0];
    core.setOutput("release_id", firstRelease.id);
    core.setOutput("release_url", firstRelease.url);
  }
  core.info(`Successfully updated ${updatedReleases.length} release(s)`);
}
main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
