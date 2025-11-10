const core = require("@actions/core");
const github = require("@actions/github");

/**
 * @typedef {Object} UpdateProjectItemOutput
 * @property {"update-project-item"} type
 * @property {string} project - Project title or number
 * @property {"issue"|"pull_request"} content_type - Type of content
 * @property {number} content_number - Issue/PR number
 * @property {Object} fields - Custom field values to update
 */

/**
 * Updates an item in a GitHub Projects v2 board
 * @param {UpdateProjectItemOutput} output - The update item output
 * @returns {Promise<void>}
 */
async function updateProjectItem(output) {
  const token = process.env.GITHUB_TOKEN;
  if (!token) {
    throw new Error("GITHUB_TOKEN environment variable is required");
  }

  const octokit = github.getOctokit(token);
  const { owner, repo } = github.context.repo;

  core.info(`Updating ${output.content_type} #${output.content_number} in project: ${output.project}`);

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

    // Get issue or PR ID
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

    const contentId = output.content_type === "issue" ? contentResult.repository.issue.id : contentResult.repository.pullRequest.id;

    core.info(`Found ${output.content_type} #${output.content_number}: ${contentId}`);

    // Find the item in the project
    const itemQuery = `
      query($projectId: ID!, $contentId: ID!) {
        node(id: $projectId) {
          ... on ProjectV2 {
            items(first: 100) {
              nodes {
                id
                content {
                  ... on Issue {
                    id
                  }
                  ... on PullRequest {
                    id
                  }
                }
              }
            }
          }
        }
      }
    `;

    const itemResult = await octokit.graphql(itemQuery, {
      projectId: project.id,
      contentId,
    });

    const items = itemResult.node.items.nodes;
    const item = items.find(i => i.content && i.content.id === contentId);

    if (!item) {
      throw new Error(`${output.content_type} #${output.content_number} not found in project`);
    }

    core.info(`Found item in project: ${item.id}`);

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
        itemId: item.id,
        fieldId: field.id,
        value,
      });

      core.info(`  ✓ Updated field: ${fieldName} = ${fieldValue}`);
    }

    // Set output
    core.setOutput("item-id", item.id);
    core.setOutput("project-id", project.id);
    core.info(`Item updated successfully`);
  } catch (error) {
    core.error(`Failed to update project item: ${error.message}`);
    throw error;
  }
}

module.exports = { updateProjectItem };
