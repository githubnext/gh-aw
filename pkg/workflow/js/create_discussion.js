async function main() {
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }
  core.debug(`Agent output content length: ${outputContent.length}`);
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }
  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.warning("No valid items found in agent output");
    return;
  }
  const createDiscussionItems = validatedOutput.items.filter(item => item.type === "create-discussion");
  if (createDiscussionItems.length === 0) {
    core.warning("No create-discussion items found in agent output");
    return;
  }
  core.debug(`Found ${createDiscussionItems.length} create-discussion item(s)`);
  if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Create Discussions Preview\n\n";
    summaryContent += "The following discussions would be created if staged mode was disabled:\n\n";
    for (let i = 0; i < createDiscussionItems.length; i++) {
      const item = createDiscussionItems[i];
      summaryContent += `### Discussion ${i + 1}\n`;
      summaryContent += `**Title:** ${item.title || "No title provided"}\n\n`;
      if (item.body) {
        summaryContent += `**Body:**\n${item.body}\n\n`;
      }
      if (item.category_id) {
        summaryContent += `**Category ID:** ${item.category_id}\n\n`;
      }
      summaryContent += "---\n\n";
    }
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Discussion creation preview written to step summary");
    return;
  }
  let discussionCategories = [];
  let repositoryId = undefined;
  try {
    const repositoryQuery = `
      query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          id
          discussionCategories(first: 20) {
            nodes {
              id
              name
              slug
              description
            }
          }
        }
      }
    `;
    const queryResult = await github.graphql(repositoryQuery, {
      owner: context.repo.owner,
      repo: context.repo.repo,
    });
    if (!queryResult || !queryResult.repository) throw new Error("Failed to fetch repository information via GraphQL");
    repositoryId = queryResult.repository.id;
    discussionCategories = queryResult.repository.discussionCategories.nodes || [];
    core.info(`Available categories: ${JSON.stringify(discussionCategories.map(cat => ({ name: cat.name, id: cat.id })))}`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    if (
      errorMessage.includes("Not Found") ||
      errorMessage.includes("not found") ||
      errorMessage.includes("Could not resolve to a Repository")
    ) {
      core.info("⚠ Cannot create discussions: Discussions are not enabled for this repository");
      core.info("Consider enabling discussions in repository settings if you want to create discussions automatically");
      return;
    }
    core.error(`Failed to get discussion categories: ${errorMessage}`);
    throw error;
  }
  let categoryId = process.env.GITHUB_AW_DISCUSSION_CATEGORY_ID;
  if (!categoryId && discussionCategories.length > 0) {
    categoryId = discussionCategories[0].id;
    core.info(`No category-id specified, using default category: ${discussionCategories[0].name} (${categoryId})`);
  }
  if (!categoryId) {
    core.error("No discussion category available and none specified in configuration");
    throw new Error("Discussion category is required but not available");
  }
  if (!repositoryId) {
    core.error("Repository ID is required for creating discussions");
    throw new Error("Repository ID is required but not available");
  }
  const createdDiscussions = [];
  for (let i = 0; i < createDiscussionItems.length; i++) {
    const createDiscussionItem = createDiscussionItems[i];
    core.info(
      `Processing create-discussion item ${i + 1}/${createDiscussionItems.length}: title=${createDiscussionItem.title}, bodyLength=${createDiscussionItem.body.length}`
    );
    let title = createDiscussionItem.title ? createDiscussionItem.title.trim() : "";
    let bodyLines = createDiscussionItem.body.split("\n");
    if (!title) {
      title = createDiscussionItem.body || "Agent Output";
    }
    const titlePrefix = process.env.GITHUB_AW_DISCUSSION_TITLE_PREFIX;
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }
    const runId = context.runId;
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `https://github.com/actions/runs/${runId}`;
    bodyLines.push(``, ``, `> Generated by Agentic Workflow [Run](${runUrl})`, "");
    const body = bodyLines.join("\n").trim();
    core.info(`Creating discussion with title: ${title}`);
    core.info(`Category ID: ${categoryId}`);
    core.info(`Body length: ${body.length}`);
    try {
      const createDiscussionMutation = `
        mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!) {
          createDiscussion(input: {
            repositoryId: $repositoryId,
            categoryId: $categoryId,
            title: $title,
            body: $body
          }) {
            discussion {
              id
              number
              title
              url
            }
          }
        }
      `;
      const mutationResult = await github.graphql(createDiscussionMutation, {
        repositoryId: repositoryId,
        categoryId: categoryId,
        title: title,
        body: body,
      });
      const discussion = mutationResult.createDiscussion.discussion;
      if (!discussion) {
        core.error("Failed to create discussion: No discussion data returned");
        continue;
      }
      core.info("Created discussion #" + discussion.number + ": " + discussion.url);
      createdDiscussions.push(discussion);
      if (i === createDiscussionItems.length - 1) {
        core.setOutput("discussion_number", discussion.number);
        core.setOutput("discussion_url", discussion.url);
      }
    } catch (error) {
      core.error(`✗ Failed to create discussion "${title}": ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }
  if (createdDiscussions.length > 0) {
    let summaryContent = "\n\n## GitHub Discussions\n";
    for (const discussion of createdDiscussions) {
      summaryContent += `- Discussion #${discussion.number}: [${discussion.title}](${discussion.url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }
  core.info(`Successfully created ${createdDiscussions.length} discussion(s)`);
}
await main();
