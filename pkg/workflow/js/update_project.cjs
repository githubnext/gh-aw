const core = require("@actions/core");
const github = require("@actions/github");

/**
 * @typedef {Object} UpdateProjectOutput
 * @property {"update_project"} type
 * @property {string} project - Project title or number
 * @property {number} [issue] - Issue number to add/update on the board
 * @property {number} [pull_request] - PR number to add/update on the board
 * @property {Object} [fields] - Custom field values to set/update
 * @property {Object} [fields_schema] - Define custom fields when creating a new project
 */

/**
 * Smart project board management - handles create/add/update automatically
 * @param {UpdateProjectOutput} output - The update output
 * @returns {Promise<void>}
 */
async function updateProject(output) {
  const token = process.env.GITHUB_TOKEN;
  if (!token) {
    throw new Error("GITHUB_TOKEN environment variable is required");
  }

  const octokit = github.getOctokit(token);
  const { owner, repo } = github.context.repo;

  core.info(`Managing project: ${output.project}`);

  try {
    // Step 1: Get repository ID
    const repoResult = await octokit.graphql(
      `query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          id
        }
      }`,
      { owner, repo }
    );
    const repositoryId = repoResult.repository.id;

    // Step 2: Find existing project or create it
    let projectId;
    let projectNumber;
    
    // Try to find existing project by title
    const existingProjectsResult = await octokit.graphql(
      `query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          projectsV2(first: 100) {
            nodes {
              id
              title
              number
            }
          }
        }
      }`,
      { owner, repo }
    );

    const existingProject = existingProjectsResult.repository.projectsV2.nodes.find(
      p => p.title === output.project || p.number.toString() === output.project.toString()
    );

    if (existingProject) {
      // Project exists
      projectId = existingProject.id;
      projectNumber = existingProject.number;
      core.info(`✓ Found existing project: ${output.project} (#${projectNumber})`);
    } else {
      // Create new project
      core.info(`Creating new project: ${output.project}`);
      const createResult = await octokit.graphql(
        `mutation($ownerId: ID!, $title: String!) {
          createProjectV2(input: {
            ownerId: $ownerId,
            title: $title
          }) {
            projectV2 {
              id
              title
              url
              number
            }
          }
        }`,
        { ownerId: repositoryId, title: output.project }
      );

      const newProject = createResult.createProjectV2.projectV2;
      projectId = newProject.id;
      projectNumber = newProject.number;

      // Link project to repository
      await octokit.graphql(
        `mutation($projectId: ID!, $repositoryId: ID!) {
          linkProjectV2ToRepository(input: {
            projectId: $projectId,
            repositoryId: $repositoryId
          }) {
            repository {
              id
            }
          }
        }`,
        { projectId, repositoryId }
      );

      core.info(`✓ Created and linked project: ${newProject.title} (${newProject.url})`);
      core.setOutput("project-id", projectId);
      core.setOutput("project-number", projectNumber);
      core.setOutput("project-url", newProject.url);
    }

    // Step 3: If issue or PR specified, add/update it on the board
    if (output.issue || output.pull_request) {
      const contentType = output.issue ? "Issue" : "PullRequest";
      const contentNumber = output.issue || output.pull_request;

      core.info(`Adding/updating ${contentType} #${contentNumber} on project board`);

      // Get content ID
      const contentQuery = output.issue
        ? `query($owner: String!, $repo: String!, $number: Int!) {
            repository(owner: $owner, name: $repo) {
              issue(number: $number) {
                id
              }
            }
          }`
        : `query($owner: String!, $repo: String!, $number: Int!) {
            repository(owner: $owner, name: $repo) {
              pullRequest(number: $number) {
                id
              }
            }
          }`;

      const contentResult = await octokit.graphql(contentQuery, {
        owner,
        repo,
        number: contentNumber,
      });

      const contentId = output.issue
        ? contentResult.repository.issue.id
        : contentResult.repository.pullRequest.id;

      // Check if item already exists on board
      const existingItemsResult = await octokit.graphql(
        `query($projectId: ID!, $contentId: ID!) {
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
        }`,
        { projectId, contentId }
      );

      const existingItem = existingItemsResult.node.items.nodes.find(
        item => item.content && item.content.id === contentId
      );

      let itemId;
      if (existingItem) {
        itemId = existingItem.id;
        core.info(`✓ Item already on board`);
      } else {
        // Add item to board
        const addResult = await octokit.graphql(
          `mutation($projectId: ID!, $contentId: ID!) {
            addProjectV2ItemById(input: {
              projectId: $projectId,
              contentId: $contentId
            }) {
              item {
                id
              }
            }
          }`,
          { projectId, contentId }
        );
        itemId = addResult.addProjectV2ItemById.item.id;
        core.info(`✓ Added ${contentType} #${contentNumber} to project board`);
      }

      // Step 4: Update custom fields if provided
      if (output.fields && Object.keys(output.fields).length > 0) {
        core.info(`Updating custom fields...`);
        
        // Get project fields
        const fieldsResult = await octokit.graphql(
          `query($projectId: ID!) {
            node(id: $projectId) {
              ... on ProjectV2 {
                fields(first: 20) {
                  nodes {
                    ... on ProjectV2Field {
                      id
                      name
                    }
                    ... on ProjectV2SingleSelectField {
                      id
                      name
                      options {
                        id
                        name
                      }
                    }
                  }
                }
              }
            }
          }`,
          { projectId }
        );

        const projectFields = fieldsResult.node.fields.nodes;

        // Update each specified field
        for (const [fieldName, fieldValue] of Object.entries(output.fields)) {
          const field = projectFields.find(f => f.name.toLowerCase() === fieldName.toLowerCase());
          if (!field) {
            core.warning(`Field "${fieldName}" not found in project`);
            continue;
          }

          // Handle different field types
          let valueToSet;
          if (field.options) {
            // Single select field - find option ID
            const option = field.options.find(o => o.name === fieldValue);
            if (option) {
              valueToSet = { singleSelectOptionId: option.id };
            } else {
              core.warning(`Option "${fieldValue}" not found for field "${fieldName}"`);
              continue;
            }
          } else {
            // Text, number, or date field
            valueToSet = { text: String(fieldValue) };
          }

          await octokit.graphql(
            `mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: ProjectV2FieldValue!) {
              updateProjectV2ItemFieldValue(input: {
                projectId: $projectId,
                itemId: $itemId,
                fieldId: $field.id,
                value: $value
              }) {
                projectV2Item {
                  id
                }
              }
            }`,
            {
              projectId,
              itemId,
              fieldId: field.id,
              value: valueToSet,
            }
          );

          core.info(`✓ Updated field "${fieldName}" = "${fieldValue}"`);
        }
      }

      core.setOutput("item-id", itemId);
    }

    core.info(`✓ Project management completed successfully`);
  } catch (error) {
    core.error(`Failed to manage project: ${error.message}`);
    throw error;
  }
}

module.exports = { updateProject };
