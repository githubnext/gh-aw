const core = require("@actions/core");
const github = require("@actions/github");

/**
 * @typedef {Object} CreateProjectOutput
 * @property {"create-project"} type
 * @property {string} title - Project title
 * @property {string} [description] - Optional project description
 */

/**
 * Creates a GitHub Projects v2 board
 * @param {CreateProjectOutput} output - The project creation output
 * @returns {Promise<void>}
 */
async function createProject(output) {
  const token = process.env.GITHUB_TOKEN;
  if (!token) {
    throw new Error("GITHUB_TOKEN environment variable is required");
  }

  const octokit = github.getOctokit(token);
  const { owner, repo } = github.context.repo;

  core.info(`Creating project: ${output.title}`);

  try {
    // Get repository ID first
    const repoQuery = `
      query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          id
        }
      }
    `;

    const repoResult = await octokit.graphql(repoQuery, {
      owner,
      repo,
    });

    const repositoryId = repoResult.repository.id;

    // Create the project
    const createMutation = `
      mutation($ownerId: ID!, $title: String!, $repositoryId: ID!) {
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
      }
    `;

    const createResult = await octokit.graphql(createMutation, {
      ownerId: repositoryId,
      title: output.title,
      repositoryId,
    });

    const project = createResult.createProjectV2.projectV2;
    core.info(`✓ Created project: ${project.title} (${project.url})`);

    // Link project to repository
    const linkMutation = `
      mutation($projectId: ID!, $repositoryId: ID!) {
        linkProjectV2ToRepository(input: {
          projectId: $projectId,
          repositoryId: $repositoryId
        }) {
          repository {
            projectsV2(first: 1) {
              nodes {
                id
                title
              }
            }
          }
        }
      }
    `;

    await octokit.graphql(linkMutation, {
      projectId: project.id,
      repositoryId,
    });

    core.info(`✓ Linked project to repository`);

    // Set output
    core.setOutput("project-id", project.id);
    core.setOutput("project-number", project.number);
    core.setOutput("project-url", project.url);
    core.setOutput("project-title", project.title);

    core.info(`Project created successfully: ${project.url}`);
  } catch (error) {
    core.error(`Failed to create project: ${error.message}`);
    throw error;
  }
}

module.exports = { createProject };
