const { loadAgentOutput } = require("./load_agent_output.cjs");

/**
 * @typedef {Object} UpdateProjectOutput
 * @property {"update_project"} type
 * @property {string} project - Project title or number
 * @property {number} [issue] - Issue number to add/update on the board
 * @property {number} [pull_request] - PR number to add/update on the board
 * @property {Object} [fields] - Custom field values to set/update
 * @property {Object} [fields_schema] - Define custom fields when creating a new project
 * @property {string} [campaign_id] - Campaign tracking ID (auto-generated if not provided)
 */

/**
 * Generate a campaign ID from project name
 * @param {string} projectName - The project/campaign name
 * @returns {string} Campaign ID in format: slug-timestamp (e.g., "perf-q1-2025-a3f2b4c8")
 */
function generateCampaignId(projectName) {
  // Create slug from project name
  const slug = projectName
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .substring(0, 30);
  
  // Add short timestamp hash for uniqueness
  const timestamp = Date.now().toString(36).substring(0, 8);
  
  return `${slug}-${timestamp}`;
}

/**
 * Smart project board management - handles create/add/update automatically
 * @param {UpdateProjectOutput} output - The update output
 * @returns {Promise<void>}
 */
async function updateProject(output) {
  // In actions/github-script, 'github' and 'context' are already available
  const { owner, repo } = context.repo;

  // Generate or use provided campaign ID
  const campaignId = output.campaign_id || generateCampaignId(output.project);
  core.info(`Campaign ID: ${campaignId}`);
  core.info(`Managing project: ${output.project}`);

  try {
    // Step 1: Get repository and owner IDs
    const repoResult = await github.graphql(
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
    
    core.info(`Owner type: ${ownerType}, Owner ID: ${ownerId}`);
    
    // Step 2: Find existing project or create it
    let projectId;
    let projectNumber;
    let existingProject = null;
    
    // Search for projects at the owner level (user/org)
    // Note: repository.projectsV2 doesn't reliably return user-owned projects even when linked
    core.info(`Searching ${ownerType.toLowerCase()} projects...`);
    
    const ownerQuery = ownerType === "User"
        ? `query($login: String!) {
            user(login: $login) {
              projectsV2(first: 100) {
                nodes {
                  id
                  title
                  number
                }
              }
            }
          }`
        : `query($login: String!) {
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

    const ownerProjectsResult = await github.graphql(ownerQuery, { login: owner });
    
    const ownerProjects = ownerType === "User" 
      ? ownerProjectsResult.user.projectsV2.nodes
      : ownerProjectsResult.organization.projectsV2.nodes;
    
    core.info(`Found ${ownerProjects.length} ${ownerType.toLowerCase()} projects`);
    ownerProjects.forEach(p => {
      core.info(`  - "${p.title}" (#${p.number})`);
    });
    
    existingProject = ownerProjects.find(
      p => p.title === output.project || p.number.toString() === output.project.toString()
    );
    
    // If found at owner level, ensure it's linked to the repository
    if (existingProject) {
      core.info(`✓ Found project "${existingProject.title}" (#${existingProject.number})`);
      
      try {
        await github.graphql(
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
        core.info(`✓ Ensured project is linked to repository`);
      } catch (linkError) {
        // Project might already be linked, that's okay
        if (linkError.message && linkError.message.includes("already linked")) {
          core.info(`✓ Project already linked to repository`);
        } else {
          core.warning(`Could not link project to repository: ${linkError.message}`);
        }
      }
    }

    if (existingProject) {
      // Project exists
      projectId = existingProject.id;
      projectNumber = existingProject.number;
      core.info(`✓ Using project: ${output.project} (#${projectNumber})`);
    } else {
      // Check if owner is a User before attempting to create
      if (ownerType === "User") {
        const manualUrl = `https://github.com/users/${owner}/projects/new`;
        core.error(
          `❌ Cannot create project "${output.project}" on user account.\n\n` +
          `GitHub Actions cannot create projects on user accounts due to permission restrictions.\n\n` +
          `📋 To fix this:\n` +
          `  1. Go to: ${manualUrl}\n` +
          `  2. Create a project named "${output.project}"\n` +
          `  3. Link it to this repository\n` +
          `  4. Re-run this workflow\n\n` +
          `The workflow will then be able to add issues/PRs to the existing project.`
        );
        throw new Error(`Cannot create project on user account. Please create it manually at ${manualUrl}`);
      }
      
      // Create new project (organization only)
      core.info(`Creating new project: ${output.project}`);
      
      const createResult = await github.graphql(
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
          ownerId: ownerId,  // Use owner ID (org/user), not repository ID
          title: output.project
        }
      );

      const newProject = createResult.createProjectV2.projectV2;
      projectId = newProject.id;
      projectNumber = newProject.number;

      // Link project to repository
      await github.graphql(
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
      core.info(`✓ Campaign ID stored in project: ${campaignId}`);
      core.setOutput("project-id", projectId);
      core.setOutput("project-number", projectNumber);
      core.setOutput("project-url", newProject.url);
      core.setOutput("campaign-id", campaignId);
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

      const contentResult = await github.graphql(contentQuery, {
        owner,
        repo,
        number: contentNumber,
      });

      const contentId = output.issue
        ? contentResult.repository.issue.id
        : contentResult.repository.pullRequest.id;

      // Check if item already exists on board
      const existingItemsResult = await github.graphql(
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
        const addResult = await github.graphql(
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
        
        // Add campaign label to issue/PR
        try {
          const campaignLabel = `campaign:${campaignId}`;
          await github.rest.issues.addLabels({
            owner,
            repo,
            issue_number: contentNumber,
            labels: [campaignLabel]
          });
          core.info(`✓ Added campaign label: ${campaignLabel}`);
        } catch (labelError) {
          core.warning(`Failed to add campaign label: ${labelError.message}`);
        }
      }

      // Step 4: Update custom fields if provided
      if (output.fields && Object.keys(output.fields).length > 0) {
        core.info(`Updating custom fields...`);
        
        // Get project fields
        const fieldsResult = await github.graphql(
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

          await github.graphql(
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
    // Provide helpful error messages for common permission issues
    if (error.message && error.message.includes("does not have permission to create projects")) {
      core.error(
        `Failed to manage project: ${error.message}\n\n` +
        `💡 Troubleshooting:\n` +
        `  - If this is a User account, GitHub Actions cannot create projects. Use an Organization repository instead.\n` +
        `  - Or, create the project manually first, then the workflow can add items to it.\n` +
        `  - Ensure the workflow has 'projects: write' permission in the workflow file.`
      );
    } else {
      core.error(`Failed to manage project: ${error.message}`);
    }
    throw error;
  }
}

(async () => {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const updateProjectItems = result.items.filter(
    (item) => item.type === "update_project"
  );
  if (updateProjectItems.length === 0) {
    core.info("No update-project items found in agent output");
    return;
  }

  // Process the first update_project item
  const output = updateProjectItems[0];
  await updateProject(output);
})();
