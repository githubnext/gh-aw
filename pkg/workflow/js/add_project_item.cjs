const core = require("@actions/core");
const github = require("@actions/github");

/**
 * @typedef {Object} AddProjectItemOutput
 * @property {"add-project-item"} type
 * @property {string} project - Project title or number
 * @property {"issue"|"pull_request"|"draft"} content_type - Type of content to add
 * @property {number} [content_number] - Issue/PR number (required for issue/pull_request)
 * @property {string} [title] - Title for draft items (required for draft)
 * @property {string} [body] - Body text for draft items (optional for draft)
 * @property {Object} [fields] - Custom field values to set
 */

/**
 * Adds an item to a GitHub Projects v2 board
 * @param {AddProjectItemOutput} output - The add item output
 * @returns {Promise<void>}
 */
async function addProjectItem(output) {
  const token = process.env.GITHUB_TOKEN;
  if (!token) {
    throw new Error("GITHUB_TOKEN environment variable is required");
  }

  const octokit = github.getOctokit(token);
  const { owner, repo } = github.context.repo;

  core.info(`Adding item to project: ${output.project}`);

  try {
    // Find project by title or number
    const projectQuery = `
      query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          projectsV2(first: 100) {
            nodes {
              id
              title
              number
            }
          }
        }
      }
    `;

    const projectResult = await octokit.graphql(projectQuery, {
      owner,
      repo,
    });

    const projects = projectResult.repository.projectsV2.nodes;
    const projectNumber = parseInt(output.project);
    const project = projects.find(p => p.title === output.project || (Number.isInteger(projectNumber) && p.number === projectNumber));

    if (!project) {
      throw new Error(`Project not found: ${output.project}`);
    }

    core.info(`Found project: ${project.title} (#${project.number})`);

    let contentId;

    // Handle different content types
    if (output.content_type === "draft") {
      // Create draft issue
      const draftMutation = `
        mutation($projectId: ID!, $title: String!, $body: String) {
          addProjectV2DraftIssue(input: {
            projectId: $projectId,
            title: $title,
            body: $body
          }) {
            projectItem {
              id
              content {
                ... on DraftIssue {
                  id
                  title
                }
              }
            }
          }
        }
      `;

      const draftResult = await octokit.graphql(draftMutation, {
        projectId: project.id,
        title: output.title || "Untitled",
        body: output.body || "",
      });

      const itemId = draftResult.addProjectV2DraftIssue.projectItem.id;
      core.info(`✓ Added draft item: ${output.title}`);

      // Set output
      core.setOutput("item-id", itemId);
      core.setOutput("project-id", project.id);
      core.info(`Draft item added successfully`);
      return;
    } else {
      // Get issue or PR ID
      if (!output.content_number) {
        throw new Error(`content_number is required for ${output.content_type}`);
      }

      const contentQuery = `
        query($owner: String!, $repo: String!, $number: Int!) {
          repository(owner: $owner, name: $repo) {
            ${output.content_type === "issue" ? "issue(number: $number) { id }" : "pullRequest(number: $number) { id }"}
          }
        }
      `;

      const contentResult = await octokit.graphql(contentQuery, {
        owner,
        repo,
        number: output.content_number,
      });

      contentId = output.content_type === "issue" ? contentResult.repository.issue.id : contentResult.repository.pullRequest.id;

      core.info(`Found ${output.content_type} #${output.content_number}: ${contentId}`);
    }

    // Add item to project
    const addMutation = `
      mutation($projectId: ID!, $contentId: ID!) {
        addProjectV2ItemById(input: {
          projectId: $projectId,
          contentId: $contentId
        }) {
          item {
            id
          }
        }
      }
    `;

    const addResult = await octokit.graphql(addMutation, {
      projectId: project.id,
      contentId,
    });

    const itemId = addResult.addProjectV2ItemById.item.id;
    core.info(`✓ Added ${output.content_type} #${output.content_number} to project`);

    // Update custom fields if provided
    if (output.fields && Object.keys(output.fields).length > 0) {
      core.info(`Updating custom fields...`);

      // Get project fields
      const fieldsQuery = `
        query($projectId: ID!) {
          node(id: $projectId) {
            ... on ProjectV2 {
              fields(first: 100) {
                nodes {
                  ... on ProjectV2Field {
                    id
                    name
                    dataType
                  }
                  ... on ProjectV2SingleSelectField {
                    id
                    name
                    dataType
                    options {
                      id
                      name
                    }
                  }
                }
              }
            }
          }
        }
      `;

      const fieldsResult = await octokit.graphql(fieldsQuery, {
        projectId: project.id,
      });

      const fields = fieldsResult.node.fields.nodes;

      // Update each field
      for (const [fieldName, fieldValue] of Object.entries(output.fields)) {
        const field = fields.find(f => f.name === fieldName);
        if (!field) {
          core.warning(`Field not found: ${fieldName}`);
          continue;
        }

        let value;
        if (field.dataType === "SINGLE_SELECT" && field.options) {
          const option = field.options.find(o => o.name === fieldValue);
          if (!option) {
            core.warning(`Option not found for field ${fieldName}: ${fieldValue}`);
            continue;
          }
          value = { singleSelectOptionId: option.id };
        } else {
          value = { text: String(fieldValue) };
        }

        const updateMutation = `
          mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: ProjectV2FieldValue!) {
            updateProjectV2ItemFieldValue(input: {
              projectId: $projectId,
              itemId: $itemId,
              fieldId: $fieldId,
              value: $value
            }) {
              projectV2Item {
                id
              }
            }
          }
        `;

        await octokit.graphql(updateMutation, {
          projectId: project.id,
          itemId,
          fieldId: field.id,
          value,
        });

        core.info(`  ✓ Updated field: ${fieldName} = ${fieldValue}`);
      }
    }

    // Set output
    core.setOutput("item-id", itemId);
    core.setOutput("project-id", project.id);
    core.info(`Item added successfully`);
  } catch (error) {
    core.error(`Failed to add project item: ${error.message}`);
    throw error;
  }
}

module.exports = { addProjectItem };
