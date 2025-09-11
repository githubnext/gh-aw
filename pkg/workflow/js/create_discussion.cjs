async function main() {
  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    console.log("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }
  if (outputContent.trim() === "") {
    console.log("Agent output content is empty");
    return;
  }

  console.log("Agent output content length:", outputContent.length);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    console.log(
      "Error parsing agent output JSON:",
      error instanceof Error ? error.message : String(error)
    );
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    console.log("No valid items found in agent output");
    return;
  }

  // Find all create-discussion items
  const createDiscussionItems = validatedOutput.items.filter(
    /** @param {any} item */ item => item.type === "create-discussion"
  );
  if (createDiscussionItems.length === 0) {
    console.log("No create-discussion items found in agent output");
    return;
  }

  console.log(
    `Found ${createDiscussionItems.length} create-discussion item(s)`
  );

  // If in staged mode, emit step summary instead of creating discussions
  if (process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Create Discussions Preview\n\n";
    summaryContent +=
      "The following discussions would be created if staged mode was disabled:\n\n";

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

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    console.log("📝 Discussion creation preview written to step summary");
    return;
  }

  // Get repository ID and discussion categories using GraphQL API
  let discussionCategories = [];
  let repositoryId = null;
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

    repositoryId = queryResult.repository.id;
    discussionCategories =
      queryResult.repository.discussionCategories.nodes || [];
    console.log(
      "Available categories:",
      discussionCategories.map(cat => ({ name: cat.name, id: cat.id }))
    );
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);

    // Special handling for repositories without discussions enabled or GraphQL errors
    if (
      errorMessage.includes("Not Found") ||
      errorMessage.includes("not found") ||
      errorMessage.includes("Could not resolve to a Repository")
    ) {
      console.log(
        "⚠ Cannot create discussions: Discussions are not enabled for this repository"
      );
      console.log(
        "Consider enabling discussions in repository settings if you want to create discussions automatically"
      );
      return; // Exit gracefully without creating discussions
    }

    core.error(`Failed to get discussion categories: ${errorMessage}`);
    throw error;
  }

  // Determine category ID
  let categoryId = process.env.GITHUB_AW_DISCUSSION_CATEGORY_ID;
  if (!categoryId && discussionCategories.length > 0) {
    // Default to the first category if none specified
    categoryId = discussionCategories[0].id;
    console.log(
      `No category-id specified, using default category: ${discussionCategories[0].name} (${categoryId})`
    );
  }
  if (!categoryId) {
    core.error(
      "No discussion category available and none specified in configuration"
    );
    throw new Error("Discussion category is required but not available");
  }

  if (!repositoryId) {
    core.error("Repository ID is required for creating discussions");
    throw new Error("Repository ID is required but not available");
  }

  const createdDiscussions = [];

  // Process each create-discussion item
  for (let i = 0; i < createDiscussionItems.length; i++) {
    const createDiscussionItem = createDiscussionItems[i];
    console.log(
      `Processing create-discussion item ${i + 1}/${createDiscussionItems.length}:`,
      {
        title: createDiscussionItem.title,
        bodyLength: createDiscussionItem.body.length,
      }
    );

    // Extract title and body from the JSON item
    let title = createDiscussionItem.title
      ? createDiscussionItem.title.trim()
      : "";
    let bodyLines = createDiscussionItem.body.split("\n");

    // If no title was found, use the body content as title (or a default)
    if (!title) {
      title = createDiscussionItem.body || "Agent Output";
    }

    // Apply title prefix if provided via environment variable
    const titlePrefix = process.env.GITHUB_AW_DISCUSSION_TITLE_PREFIX;
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }

    // Add AI disclaimer with workflow run information
    const runId = context.runId;
    const runUrl = context.payload.repository
      ? `${context.payload.repository.html_url}/actions/runs/${runId}`
      : `https://github.com/actions/runs/${runId}`;
    bodyLines.push(
      ``,
      ``,
      `> Generated by Agentic Workflow [Run](${runUrl})`,
      ""
    );

    // Prepare the body content
    const body = bodyLines.join("\n").trim();

    console.log("Creating discussion with title:", title);
    console.log("Category ID:", categoryId);
    console.log("Body length:", body.length);

    try {
      // Create the discussion using GraphQL API with parameterized mutation
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

      console.log(
        "Created discussion #" + discussion.number + ": " + discussion.url
      );
      createdDiscussions.push(discussion);

      // Set output for the last created discussion (for backward compatibility)
      if (i === createDiscussionItems.length - 1) {
        core.setOutput("discussion_number", discussion.number);
        core.setOutput("discussion_url", discussion.url);
      }
    } catch (error) {
      core.error(
        `✗ Failed to create discussion "${title}": ${error instanceof Error ? error.message : String(error)}`
      );
      throw error;
    }
  }

  // Write summary for all created discussions
  if (createdDiscussions.length > 0) {
    let summaryContent = "\n\n## GitHub Discussions\n";
    for (const discussion of createdDiscussions) {
      summaryContent += `- Discussion #${discussion.number}: [${discussion.title}](${discussion.url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  console.log(
    `Successfully created ${createdDiscussions.length} discussion(s)`
  );
}
await main();
