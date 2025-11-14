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
 * Parse project input to extract project number from URL or return project name
 * @param {string} projectInput - Project URL, number, or name
 * @returns {{projectNumber: string|null, projectName: string}} Extracted project number (if URL) and name
 */
function parseProjectInput(projectInput) {
  // Try to parse as GitHub project URL
  const urlMatch = projectInput.match(/github\.com\/(?:users|orgs)\/[^/]+\/projects\/(\d+)/);
  if (urlMatch) {
    return {
      projectNumber: urlMatch[1],
      projectName: null,
    };
  }

  // Otherwise treat as project name or number
  return {
    projectNumber: /^\d+$/.test(projectInput) ? projectInput : null,
    projectName: /^\d+$/.test(projectInput) ? null : projectInput,
  };
}

/**
 * Generate a campaign ID from project name
 * @param {string} projectName - The project/campaign name
 * @returns {string} Campaign ID in format: slug-timestamp (e.g., "perf-q1-2025-a3f2b4c8")
 */
function generateCampaignId(projectName) {
  // Create slug from project name
  const slug = projectName
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
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

  // Parse project input to extract number from URL or use name
  const { projectNumber: parsedProjectNumber, projectName: parsedProjectName } = parseProjectInput(output.project);
  core.info(`Parsed project input: ${output.project} -> number=${parsedProjectNumber}, name=${parsedProjectName}`);

  // Generate or use provided campaign ID
  const displayName = parsedProjectName || parsedProjectNumber || output.project;
  const campaignId = output.campaign_id || generateCampaignId(displayName);
  core.info(`Campaign ID: ${campaignId}`);
  core.info(`Managing project: ${output.project}`);

  // Check for custom token with projects permissions and create authenticated client
  let githubClient = github;
  if (process.env.PROJECT_GITHUB_TOKEN) {
    core.info(`✓ Using custom PROJECT_GITHUB_TOKEN for project operations`);
    // Create new Octokit instance with the custom token
    const { Octokit } = require("@octokit/rest");
    const octokit = new Octokit({
      auth: process.env.PROJECT_GITHUB_TOKEN,
      baseUrl: process.env.GITHUB_API_URL || "https://api.github.com",
    });
    // Wrap in the same interface as github-script provides
    githubClient = {
      graphql: octokit.graphql.bind(octokit),
      rest: octokit.rest,
    };
  } else {
    core.info(`ℹ Using default GITHUB_TOKEN (may not have project creation permissions)`);
  }

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

    core.info(`Owner type: ${ownerType}, Owner ID: ${ownerId}`);

    // Step 2: Find existing project or create it
    let projectId;
    let projectNumber;
    let existingProject = null;

    // Search for projects at the owner level (user/org)
    // Note: repository.projectsV2 doesn't reliably return user-owned projects even when linked
    core.info(`Searching ${ownerType.toLowerCase()} projects...`);

    const ownerQuery =
      ownerType === "User"
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

    const ownerProjectsResult = await githubClient.graphql(ownerQuery, { login: owner });

    const ownerProjects =
      ownerType === "User" ? ownerProjectsResult.user.projectsV2.nodes : ownerProjectsResult.organization.projectsV2.nodes;

    // Search by project number if extracted from URL, otherwise by name
    existingProject = ownerProjects.find(p => {
      if (parsedProjectNumber) {
        return p.number.toString() === parsedProjectNumber;
      }
      return p.title === parsedProjectName;
    });

    // If found at owner level, ensure it's linked to the repository
    if (existingProject) {
      core.info(`✓ Found project "${existingProject.title}" (#${existingProject.number})`);

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
        const projectDisplay = parsedProjectNumber ? `project #${parsedProjectNumber}` : `project "${parsedProjectName}"`;
        const manualUrl = `https://github.com/users/${owner}/projects/new`;
        core.error(
          `❌ Cannot find ${projectDisplay} on user account.\n\n` +
            `GitHub Actions cannot create projects on user accounts due to permission restrictions.\n\n` +
            `📋 To fix this:\n` +
            `  1. Verify the project exists and is accessible\n` +
            `  2. If it doesn't exist, create it at: ${manualUrl}\n` +
            `  3. Ensure it's linked to this repository\n` +
            `  4. Provide a valid PROJECT_GITHUB_TOKEN with 'project' scope\n` +
            `  5. Re-run this workflow\n\n` +
            `The workflow will then be able to add issues/PRs to the existing project.`
        );
        throw new Error(
          `Cannot find ${projectDisplay} on user account. Please verify it exists and you have the correct token permissions.`
        );
      }

      // Create new project (organization only)
      core.info(`Creating new project: ${output.project}`);

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
          ownerId: ownerId, // Use owner ID (org/user), not repository ID
          title: output.project,
        }
      );

      const newProject = createResult.createProjectV2.projectV2;
      projectId = newProject.id;
      projectNumber = newProject.number;

      // Link project to repository
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
    // Support both old format (issue/pull_request) and new format (content_type/content_number)
    const contentNumber = output.content_number || output.issue || output.pull_request;
    if (contentNumber) {
      const contentType =
        output.content_type === "pull_request"
          ? "PullRequest"
          : output.content_type === "issue"
            ? "Issue"
            : output.issue
              ? "Issue"
              : "PullRequest";

      core.info(`Adding/updating ${contentType} #${contentNumber} on project board`);

      // Get content ID
      const contentQuery =
        contentType === "Issue"
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

      const contentResult = await githubClient.graphql(contentQuery, {
        owner,
        repo,
        number: contentNumber,
      });

      const contentId = contentType === "Issue" ? contentResult.repository.issue.id : contentResult.repository.pullRequest.id;

      // Check if item already exists on board
      const existingItemsResult = await githubClient.graphql(
        `query($projectId: ID!) {
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
        { projectId }
      );

      const existingItem = existingItemsResult.node.items.nodes.find(item => item.content && item.content.id === contentId);

      let itemId;
      if (existingItem) {
        itemId = existingItem.id;
        core.info(`✓ Item already on board`);
      } else {
        // Add item to board
        const addResult = await githubClient.graphql(
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
          await githubClient.rest.issues.addLabels({
            owner,
            repo,
            issue_number: contentNumber,
            labels: [campaignLabel],
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
        const fieldsResult = await githubClient.graphql(
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
          let field = projectFields.find(f => f.name.toLowerCase() === fieldName.toLowerCase());
          if (!field) {
            core.info(`Field "${fieldName}" not found, attempting to create it...`);

            // Try to create the field - determine type based on field name or value
            const isTextField =
              fieldName.toLowerCase() === "classification" || (typeof fieldValue === "string" && fieldValue.includes("|"));

            if (isTextField) {
              // Create text field
              try {
                const createFieldResult = await githubClient.graphql(
                  `mutation($projectId: ID!, $name: String!, $dataType: ProjectV2CustomFieldType!) {
                    createProjectV2Field(input: {
                      projectId: $projectId,
                      name: $name,
                      dataType: $dataType
                    }) {
                      projectV2Field {
                        ... on ProjectV2Field {
                          id
                          name
                        }
                      }
                    }
                  }`,
                  {
                    projectId,
                    name: fieldName,
                    dataType: "TEXT",
                  }
                );
                field = createFieldResult.createProjectV2Field.projectV2Field;
                core.info(`✓ Created text field "${fieldName}"`);
              } catch (createError) {
                core.warning(`Failed to create field "${fieldName}": ${createError.message}`);
                continue;
              }
            } else {
              // Create single select field with the provided value as an option
              try {
                const createFieldResult = await githubClient.graphql(
                  `mutation($projectId: ID!, $name: String!, $dataType: ProjectV2CustomFieldType!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {
                    createProjectV2Field(input: {
                      projectId: $projectId,
                      name: $name,
                      dataType: $dataType,
                      singleSelectOptions: $options
                    }) {
                      projectV2Field {
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
                  }`,
                  {
                    projectId,
                    name: fieldName,
                    dataType: "SINGLE_SELECT",
                    options: [{ name: String(fieldValue), description: "", color: "GRAY" }],
                  }
                );
                field = createFieldResult.createProjectV2Field.projectV2Field;
                core.info(`✓ Created single select field "${fieldName}" with option "${fieldValue}"`);
              } catch (createError) {
                core.warning(`Failed to create field "${fieldName}": ${createError.message}`);
                continue;
              }
            }
          }

          // Handle different field types
          let valueToSet;
          if (field.options) {
            // Single select field - find option ID
            let option = field.options.find(o => o.name === fieldValue);
            if (!option) {
              // Option doesn't exist, try to create it
              core.info(`Option "${fieldValue}" not found for field "${fieldName}", attempting to create it...`);
              try {
                // Build options array with existing options plus the new one
                const allOptions = [
                  ...field.options.map(o => ({ name: o.name, description: "" })),
                  { name: String(fieldValue), description: "" },
                ];

                const createOptionResult = await githubClient.graphql(
                  `mutation($projectId: ID!, $fieldId: ID!, $fieldName: String!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {
                    updateProjectV2Field(input: {
                      projectId: $projectId,
                      fieldId: $fieldId,
                      name: $fieldName,
                      singleSelectOptions: $options
                    }) {
                      projectV2Field {
                        ... on ProjectV2SingleSelectField {
                          id
                          options {
                            id
                            name
                          }
                        }
                      }
                    }
                  }`,
                  {
                    projectId,
                    fieldId: field.id,
                    fieldName: field.name,
                    options: allOptions,
                  }
                );
                // Find the newly created option
                const updatedField = createOptionResult.updateProjectV2Field.projectV2Field;
                option = updatedField.options.find(o => o.name === fieldValue);
                field = updatedField; // Update field reference with new options
                core.info(`✓ Created option "${fieldValue}" for field "${fieldName}"`);
              } catch (createError) {
                core.warning(`Failed to create option "${fieldValue}": ${createError.message}`);
                continue;
              }
            }
            if (option) {
              valueToSet = { singleSelectOptionId: option.id };
            } else {
              core.warning(`Could not get option ID for "${fieldValue}" in field "${fieldName}"`);
              continue;
            }
          } else {
            // Text, number, or date field
            valueToSet = { text: String(fieldValue) };
          }

          await githubClient.graphql(
            `mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: ProjectV2FieldValue!) {
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
      const usingCustomToken = !!process.env.PROJECT_GITHUB_TOKEN;
      core.error(
        `Failed to manage project: ${error.message}\n\n` +
          `💡 Troubleshooting:\n` +
          `  1. Create the project manually first at https://github.com/orgs/${owner}/projects/new\n` +
          `     Then the workflow can add items to it automatically.\n\n` +
          `  2. Or, add a Personal Access Token (PAT) with 'project' permissions:\n` +
          `     - Create a PAT at https://github.com/settings/tokens/new?scopes=project\n` +
          `     - Add it as a secret named PROJECT_GITHUB_TOKEN\n` +
          `     - Pass it to the workflow: PROJECT_GITHUB_TOKEN: \${{ secrets.PROJECT_GITHUB_TOKEN }}\n\n` +
          `  3. Ensure the workflow has 'projects: write' permission.\n\n` +
          `${usingCustomToken ? "⚠️  Note: Already using PROJECT_GITHUB_TOKEN but still getting permission error." : "📝 Currently using default GITHUB_TOKEN (no project create permissions)."}`
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

  const updateProjectItems = result.items.filter(item => item.type === "update_project");
  if (updateProjectItems.length === 0) {
    core.info("No update-project items found in agent output");
    return;
  }

  core.info(`Processing ${updateProjectItems.length} update_project items`);

  // Process all update_project items
  for (let i = 0; i < updateProjectItems.length; i++) {
    const output = updateProjectItems[i];
    core.info(
      `\n[${i + 1}/${updateProjectItems.length}] Processing item: ${output.content_type || "project"} #${output.content_number || output.issue || output.pull_request || "N/A"}`
    );
    try {
      await updateProject(output);
    } catch (error) {
      core.error(`Failed to process item ${i + 1}: ${error.message}`);
      // Continue processing remaining items even if one fails
    }
  }

  core.info(`\n✓ Completed processing ${updateProjectItems.length} items`);
})();
