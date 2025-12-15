const { loadAgentOutput } = require("./load_agent_output.cjs");

/**
 * Log detailed GraphQL error information
 * @param {Error} error - The error object from GraphQL
 * @param {string} operation - Description of the operation that failed
 */
function logGraphQLError(error, operation) {
  core.error(`GraphQL Error during: ${operation}`);
  core.error(`Message: ${error.message}`);
  
  const errorList = Array.isArray(error.errors) ? error.errors : [];
  const hasInsufficientScopes = errorList.some(e => e && e.type === "INSUFFICIENT_SCOPES");
  const hasNotFound = errorList.some(e => e && e.type === "NOT_FOUND");
  if (hasInsufficientScopes) {
    core.error(
      "This looks like a token permission problem for Projects v2. " +
        "The GraphQL fields used by update_project require a token with Projects access (e.g., classic PAT scope read:project/write:project). " +
        "Fix: set safe-outputs.update-project.github-token to a secret PAT with Projects permissions."
    );
  } else if (hasNotFound && /projectV2\b/.test(error.message)) {
    core.error(
      "GitHub returned NOT_FOUND for ProjectV2. This can mean either: " +
        "(1) the project number is wrong for Projects v2, " +
        "(2) the project is a classic Projects board (not Projects v2), or " +
        "(3) the token does not have access to that org/user project."
    );
  }
  
  if (error.errors) {
    core.error(`Errors array (${error.errors.length} error(s)):`);
    error.errors.forEach((err, idx) => {
      core.error(`  [${idx + 1}] ${err.message}`);
      if (err.type) core.error(`      Type: ${err.type}`);
      if (err.path) core.error(`      Path: ${JSON.stringify(err.path)}`);
      if (err.locations) core.error(`      Locations: ${JSON.stringify(err.locations)}`);
    });
  }
  
  if (error.request) {
    core.error(`Request: ${JSON.stringify(error.request, null, 2)}`);
  }
  
  if (error.data) {
    core.error(`Response data: ${JSON.stringify(error.data, null, 2)}`);
  }
}

/**
 * @typedef {Object} UpdateProjectOutput
 * @property {"update_project"} type
 * @property {string} project - Full GitHub project URL (required)
 * @property {string} [content_type] - Type of content: "issue" or "pull_request"
 * @property {number|string} [content_number] - Issue or PR number (preferred)
 * @property {number|string} [issue] - Issue number (legacy, use content_number instead)
 * @property {number|string} [pull_request] - PR number (legacy, use content_number instead)
 * @property {Object} [fields] - Custom field values to set/update (creates fields if missing)
 * @property {string} [campaign_id] - Campaign tracking ID (auto-generated if not provided)
 * @property {boolean} [create_if_missing] - Opt-in: allow creating the project board if it does not exist.
 *   Default behavior is update-only; if the project does not exist, this job will fail with instructions.
 */

/**
 * Parse project URL to extract project number
 * @param {string} projectUrl - Full GitHub project URL (required)
 * @returns {string} Extracted project number
 */
function parseProjectInput(projectUrl) {
  // Validate input
  if (!projectUrl || typeof projectUrl !== "string") {
    throw new Error(
      `Invalid project input: expected string, got ${typeof projectUrl}. The "project" field is required and must be a full GitHub project URL.`
    );
  }

  // Parse GitHub project URL
  const urlMatch = projectUrl.match(/github\.com\/(?:users|orgs)\/[^/]+\/projects\/(\d+)/);
  if (!urlMatch) {
    throw new Error(
      `Invalid project URL: "${projectUrl}". The "project" field must be a full GitHub project URL (e.g., https://github.com/orgs/myorg/projects/123).`
    );
  }

  return urlMatch[1];
}

/**
 * Generate a campaign ID from project URL
 * @param {string} projectUrl - The GitHub project URL
 * @param {string} projectNumber - The project number
 * @returns {string} Campaign ID in format: org-project-{number}-{timestamp}
 */
function generateCampaignId(projectUrl, projectNumber) {
  // Extract org/user name from URL for the slug
  const urlMatch = projectUrl.match(/github\.com\/(users|orgs)\/([^/]+)\/projects/);
  const orgName = urlMatch ? urlMatch[2] : "project";

  const slug = `${orgName}-project-${projectNumber}`
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

  // Parse project URL to get project number
  const projectNumberFromUrl = parseProjectInput(output.project);
  const campaignId = output.campaign_id || generateCampaignId(output.project, projectNumberFromUrl);

  try {
    core.info(`Looking up project #${projectNumberFromUrl} from URL: ${output.project}`);
    
    // Step 1: Get repository and owner IDs
    core.info("[1/5] Fetching repository information...");
    let repoResult;
    try {
      repoResult = await github.graphql(
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
    } catch (error) {
      logGraphQLError(error, "Fetching repository information");
      throw error;
    }
    const repositoryId = repoResult.repository.id;
    const ownerType = repoResult.repository.owner.__typename;
    core.info(`✓ Repository: ${owner}/${repo} (${ownerType})`);

    // Step 2: Resolve project from exact URL
    core.info(`[2/5] Resolving project from URL...`);
    let projectId;
    let resolvedProjectNumber = projectNumberFromUrl;
    try {
      const resourceResult = await github.graphql(
        `query($url: URI!) {
          resource(url: $url) {
            __typename
            ... on ProjectV2 {
              id
              number
              title
              url
              owner {
                __typename
                ... on Organization { login }
                ... on User { login }
              }
            }
          }
        }`,
        { url: output.project }
      );

      const resource = resourceResult && resourceResult.resource;
      if (!resource) {
        core.error(`Cannot resolve project URL: ${output.project}`);
        throw new Error("Project URL could not be resolved (not found or not accessible to the token).");
      }

      if (resource.__typename !== "ProjectV2") {
        core.error(`Project URL did not resolve to a Projects v2 board. Resolved type: ${resource.__typename}`);
        throw new Error(
          `Project URL must point to a GitHub Projects v2 board, but resolved to: ${resource.__typename}.`
        );
      }

      projectId = resource.id;
      resolvedProjectNumber = String(resource.number);
      const ownerLogin = resource.owner && resource.owner.login ? resource.owner.login : "(unknown)";
      core.info(`✓ Resolved project #${resolvedProjectNumber} (${ownerLogin}) (ID: ${projectId})`);
    } catch (error) {
      logGraphQLError(error, "Resolving project from URL");
      throw error;
    }

    // Ensure project is linked to the repository
    core.info("[3/5] Linking project to repository...");
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
        { projectId, repositoryId }
      );
    } catch (linkError) {
      if (!linkError.message || !linkError.message.includes("already linked")) {
        logGraphQLError(linkError, "Linking project to repository");
        core.warning(`Could not link project: ${linkError.message}`);
      }
    }
    core.info("✓ Project linked to repository");

    // Step 3: If issue or PR specified, add/update it on the board
    core.info("[4/5] Processing content (issue/PR) if specified...");
    // Support both old format (issue/pull_request) and new format (content_type/content_number)
    // Validate mutually exclusive content_number/issue/pull_request fields
    const hasContentNumber = output.content_number !== undefined && output.content_number !== null;
    const hasIssue = output.issue !== undefined && output.issue !== null;
    const hasPullRequest = output.pull_request !== undefined && output.pull_request !== null;
    const values = [];
    if (hasContentNumber) values.push({ key: "content_number", value: output.content_number });
    if (hasIssue) values.push({ key: "issue", value: output.issue });
    if (hasPullRequest) values.push({ key: "pull_request", value: output.pull_request });
    if (values.length > 1) {
      const uniqueValues = [...new Set(values.map(v => String(v.value)))];
      const list = values.map(v => `${v.key}=${v.value}`).join(", ");
      const descriptor = uniqueValues.length > 1 ? "different values" : `same value "${uniqueValues[0]}"`;
      core.warning(`Multiple content number fields (${descriptor}): ${list}. Using priority content_number > issue > pull_request.`);
    }
    if (hasIssue) {
      core.warning('Field "issue" deprecated; use "content_number" instead.');
    }
    if (hasPullRequest) {
      core.warning('Field "pull_request" deprecated; use "content_number" instead.');
    }
    let contentNumber = null;
    if (hasContentNumber || hasIssue || hasPullRequest) {
      const rawContentNumber = hasContentNumber ? output.content_number : hasIssue ? output.issue : output.pull_request;

      const sanitizedContentNumber =
        rawContentNumber === undefined || rawContentNumber === null
          ? ""
          : typeof rawContentNumber === "number"
            ? rawContentNumber.toString()
            : String(rawContentNumber).trim();

      if (!sanitizedContentNumber) {
        core.warning("Content number field provided but empty; skipping project item update.");
      } else if (!/^\d+$/.test(sanitizedContentNumber)) {
        throw new Error(`Invalid content number "${rawContentNumber}". Provide a positive integer.`);
      } else {
        contentNumber = Number.parseInt(sanitizedContentNumber, 10);
      }
    }
    if (contentNumber !== null) {
      const contentType =
        output.content_type === "pull_request"
          ? "PullRequest"
          : output.content_type === "issue"
            ? "Issue"
            : output.issue
              ? "Issue"
              : "PullRequest";

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

      const contentResult = await github.graphql(contentQuery, {
        owner,
        repo,
        number: contentNumber,
      });

      const contentId = contentType === "Issue" ? contentResult.repository.issue.id : contentResult.repository.pullRequest.id;

      // Check if item already exists on board (handle pagination)
      async function findExistingProjectItem(projectId, contentId) {
        let hasNextPage = true;
        let endCursor = null;
        while (hasNextPage) {
          const result = await github.graphql(
            `query($projectId: ID!, $after: String) {
              node(id: $projectId) {
                ... on ProjectV2 {
                  items(first: 100, after: $after) {
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
                    pageInfo {
                      hasNextPage
                      endCursor
                    }
                  }
                }
              }
            }`,
            { projectId, after: endCursor }
          );
          const items = result.node.items.nodes;
          const found = items.find(item => item.content && item.content.id === contentId);
          if (found) {
            return found;
          }
          hasNextPage = result.node.items.pageInfo.hasNextPage;
          endCursor = result.node.items.pageInfo.endCursor;
        }
        return null;
      }

      const existingItem = await findExistingProjectItem(projectId, contentId);

      let itemId;
      if (existingItem) {
        itemId = existingItem.id;
        core.info("✓ Item already on board");
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

        // Add campaign label to issue/PR
        try {
          await githubClient.rest.issues.addLabels({
            owner,
            repo,
            issue_number: contentNumber,
            labels: [`campaign:${campaignId}`],
          });
        } catch (labelError) {
          core.warning(`Failed to add campaign label: ${labelError.message}`);
        }
      }

      // Step 4: Update custom fields if provided
      if (output.fields && Object.keys(output.fields).length > 0) {
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
          // Normalize field names: capitalize first letter of each word for consistency
          const normalizedFieldName = fieldName
            .split(/[\s_-]+/)
            .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
            .join(" ");

          let field = projectFields.find(f => f.name.toLowerCase() === normalizedFieldName.toLowerCase());
          if (!field) {
            // Try to create the field - determine type based on field name or value
            const isTextField =
              fieldName.toLowerCase() === "classification" || (typeof fieldValue === "string" && fieldValue.includes("|"));

            if (isTextField) {
              // Create text field
              try {
                const createFieldResult = await github.graphql(
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
                        ... on ProjectV2SingleSelectField {
                          id
                          name
                          options { id name }
                        }
                      }
                    }
                  }`,
                  {
                    projectId,
                    name: normalizedFieldName,
                    dataType: "TEXT",
                  }
                );
                field = createFieldResult.createProjectV2Field.projectV2Field;
              } catch (createError) {
                core.warning(`Failed to create field "${fieldName}": ${createError.message}`);
                continue;
              }
            } else {
              // Create single select field with the provided value as an option
              try {
                const createFieldResult = await github.graphql(
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
                          options { id name }
                        }
                        ... on ProjectV2Field {
                          id
                          name
                        }
                      }
                    }
                  }`,
                  {
                    projectId,
                    name: normalizedFieldName,
                    dataType: "SINGLE_SELECT",
                    options: [{ name: String(fieldValue), description: "", color: "GRAY" }],
                  }
                );
                field = createFieldResult.createProjectV2Field.projectV2Field;
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
              try {
                // Build options array with existing options plus the new one
                const allOptions = [
                  ...field.options.map(o => ({ name: o.name, description: "" })),
                  { name: String(fieldValue), description: "" },
                ];

                const createOptionResult = await github.graphql(
                  `mutation($fieldId: ID!, $fieldName: String!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {
                    updateProjectV2Field(input: {
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
                    fieldId: field.id,
                    fieldName: field.name,
                    options: allOptions,
                  }
                );
                // Find the newly created option
                const updatedField = createOptionResult.updateProjectV2Field.projectV2Field;
                option = updatedField.options.find(o => o.name === fieldValue);
                field = updatedField; // Update field reference with new options
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

          await github.graphql(
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
        }
      }

      core.setOutput("item-id", itemId);
    }
  } catch (error) {
    // Provide helpful error messages for common permission issues
    if (error.message && error.message.includes("does not have permission to create projects")) {
      const usingCustomToken = !!process.env.GH_AW_PROJECT_GITHUB_TOKEN;
      core.error(
        `Failed to manage project: ${error.message}\n\n` +
          `Troubleshooting:\n` +
          `  • Create the project manually at https://github.com/orgs/${owner}/projects/new.\n` +
          `  • Or supply a PAT with project scope via GH_AW_PROJECT_GITHUB_TOKEN.\n` +
          `  • Ensure the workflow grants projects: write.\n\n` +
          `${usingCustomToken ? "GH_AW_PROJECT_GITHUB_TOKEN is set but lacks access." : "Using default GITHUB_TOKEN without project create rights."}`
      );
    } else {
      core.error(`Failed to manage project: ${error.message}`);
    }
    throw error;
  }
}

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const updateProjectItems = result.items.filter(item => item.type === "update_project");
  if (updateProjectItems.length === 0) {
    return;
  }

  // Process all update_project items
  for (let i = 0; i < updateProjectItems.length; i++) {
    const output = updateProjectItems[i];
    try {
      await updateProject(output);
    } catch (error) {
      core.error(`Failed to process item ${i + 1}`);
      logGraphQLError(error, `Processing update_project item ${i + 1}`);
      // Continue processing remaining items even if one fails
    }
  }
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = { updateProject, parseProjectInput, generateCampaignId, main };
}

// Run automatically in GitHub Actions (module undefined) or when executed directly via Node
if (typeof module === "undefined" || require.main === module) {
  main();
}
