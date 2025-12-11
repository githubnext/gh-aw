const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateCampaignId, normalizeProjectName } = require("./project_helpers.cjs");

/**
 * @typedef {Object} CreateProjectOutput
 * @property {"create_project"} type
 * @property {string} project - Project title to create
 * @property {string} [campaign_id] - Campaign tracking ID (auto-generated if not provided)
 */

/**
 * Create a new GitHub Project v2
 * @param {CreateProjectOutput} output - The create output
 * @returns {Promise<void>}
 */
async function createProject(output) {
  // In actions/github-script, 'github' and 'context' are already available
  const { owner, repo } = context.repo;

  // Normalize and validate project name
  const normalizedProjectName = normalizeProjectName(output.project);

  const campaignId = output.campaign_id || generateCampaignId(normalizedProjectName);

  // Use the github client that's already configured with the token via github-token parameter
  const githubClient = github;

  try {
    // Step 1: Get repository and owner IDs
    const repoResult = await githubClient.graphql(
      `query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          id
          owner {
            id
            __typename
          }
        }
      }`,
      { owner, repo }
    );
    const repositoryId = repoResult.repository.id;
    const ownerId = repoResult.repository.owner.id;
    const ownerType = repoResult.repository.owner.__typename;

    // Step 2: Check if owner is a User
    if (ownerType === "User") {
      core.error(`Cannot create projects on user accounts. Create the project manually at https://github.com/users/${owner}/projects/new.`);
      throw new Error(`Cannot create project "${output.project}" on user account.`);
    }

    // Step 3: Check if project already exists
    const ownerQuery = `query($login: String!) {
      organization(login: $login) {
        projectsV2(first: 100) {
          nodes {
            id
            title
            number
          }
        }
      }
    }`;

    const ownerProjectsResult = await githubClient.graphql(ownerQuery, { login: owner });
    const ownerProjects = ownerProjectsResult.organization.projectsV2.nodes;

    const existingProject = ownerProjects.find(p => p.title === normalizedProjectName);
    if (existingProject) {
      core.info(`✓ Project already exists: ${existingProject.title} (#${existingProject.number})`);
      core.setOutput("project-id", existingProject.id);
      core.setOutput("project-number", existingProject.number);
      core.setOutput("campaign-id", campaignId);

      // Link project to repository if not already linked
      try {
        await githubClient.graphql(
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
          { projectId: existingProject.id, repositoryId }
        );
      } catch (linkError) {
        if (!linkError.message || !linkError.message.includes("already linked")) {
          core.warning(`Could not link project: ${linkError.message}`);
        }
      }
      return;
    }

    // Step 4: Create new project (organization only)
    const createResult = await githubClient.graphql(
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
      {
        ownerId: ownerId,
        title: normalizedProjectName,
      }
    );

    const newProject = createResult.createProjectV2.projectV2;

    // Step 5: Link project to repository
    await githubClient.graphql(
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
      { projectId: newProject.id, repositoryId }
    );

    core.info(`✓ Created project: ${newProject.title}`);
    core.setOutput("project-id", newProject.id);
    core.setOutput("project-number", newProject.number);
    core.setOutput("project-url", newProject.url);
    core.setOutput("campaign-id", campaignId);
  } catch (error) {
    // Provide helpful error messages for common permission issues
    if (error.message && error.message.includes("does not have permission to create projects")) {
      core.error(
        `Failed to create project: ${error.message}\n\n` +
          `Troubleshooting:\n` +
          `  • Create the project manually at https://github.com/orgs/${owner}/projects/new.\n` +
          `  • Or supply a PAT with project scope via github-token configuration.\n` +
          `  • Ensure the workflow grants projects: write.\n`
      );
    } else {
      core.error(`Failed to create project: ${error.message}`);
    }
    throw error;
  }
}

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const createProjectItems = result.items.filter(item => item.type === "create_project");
  if (createProjectItems.length === 0) {
    return;
  }

  // Process all create_project items
  for (let i = 0; i < createProjectItems.length; i++) {
    const output = createProjectItems[i];
    try {
      await createProject(output);
    } catch (error) {
      core.error(`Failed to process item ${i + 1}: ${error.message}`);
      // Continue processing remaining items even if one fails
    }
  }
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = { createProject, main };
}

// Run automatically in GitHub Actions (module undefined) or when executed directly via Node
if (typeof module === "undefined" || require.main === module) {
  main();
}
