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
        "The GraphQL fields used by update_project require a token with Projects access (classic PAT: scope 'project'; fine-grained PAT: Organization permission 'Projects' and access to the org). " +
        "Fix: set safe-outputs.update-project.github-token to a secret PAT that can access the target org project."
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
    throw new Error(`Invalid project input: expected string, got ${typeof projectUrl}. The "project" field is required and must be a full GitHub project URL.`);
  }

  // Parse GitHub project URL
  const urlMatch = projectUrl.match(/github\.com\/(?:users|orgs)\/[^/]+\/projects\/(\d+)/);
  if (!urlMatch) {
    throw new Error(`Invalid project URL: "${projectUrl}". The "project" field must be a full GitHub project URL (e.g., https://github.com/orgs/myorg/projects/123).`);
  }

  return urlMatch[1];
}

/**
 * Parse GitHub project URL into owner scope, owner login, and project number.
 * @param {string} projectUrl - Full GitHub project URL (required)
 * @returns {{ scope: "orgs"|"users", ownerLogin: string, projectNumber: string }}
 */
function parseProjectUrl(projectUrl) {
  if (!projectUrl || typeof projectUrl !== "string") {
    throw new Error(`Invalid project input: expected string, got ${typeof projectUrl}. The "project" field is required and must be a full GitHub project URL.`);
  }

  const match = projectUrl.match(/github\.com\/(users|orgs)\/([^/]+)\/projects\/(\d+)/);
  if (!match) {
    throw new Error(`Invalid project URL: "${projectUrl}". The "project" field must be a full GitHub project URL (e.g., https://github.com/orgs/myorg/projects/123).`);
  }

  return { scope: match[1], ownerLogin: match[2], projectNumber: match[3] };
}

/**
 * List Projects v2 accessible to the token for an org or user.
 * Used as a fallback when direct lookup by number returns null or errors.
 * @param {{ scope: "orgs"|"users", ownerLogin: string }} projectInfo
 * @returns {Promise<{ nodes: Array<{id: string, number: number, title: string, closed: boolean, url: string}>, totalCount?: number }>}
 */
async function listAccessibleProjectsV2(projectInfo) {
  const baseQuery = `projectsV2(first: 100) {
    totalCount
    nodes {
      id
      number
      title
      closed
      url
    }
    edges {
      node {
        id
        number
        title
        closed
        url
      }
    }
  }`;

  if (projectInfo.scope === "orgs") {
    const result = await github.graphql(
      `query($login: String!) {
        organization(login: $login) {
          ${baseQuery}
        }
      }`,
      { login: projectInfo.ownerLogin }
    );
    const conn = result && result.organization && result.organization.projectsV2;
    const rawNodes = conn && Array.isArray(conn.nodes) ? conn.nodes : [];
    const rawEdges = conn && Array.isArray(conn.edges) ? conn.edges : [];

    const nodeNodes = rawNodes.filter(Boolean);
    const edgeNodes = rawEdges.map(e => e && e.node).filter(Boolean);

    /** @type {Map<string, any>} */
    const unique = new Map();
    for (const n of [...nodeNodes, ...edgeNodes]) {
      if (n && typeof n.id === "string") {
        unique.set(n.id, n);
      }
    }

    return {
      nodes: Array.from(unique.values()),
      totalCount: conn && conn.totalCount,
      diagnostics: {
        rawNodesCount: rawNodes.length,
        nullNodesCount: rawNodes.length - nodeNodes.length,
        rawEdgesCount: rawEdges.length,
        nullEdgeNodesCount: rawEdges.filter(e => !e || !e.node).length,
      },
    };
  }

  const result = await github.graphql(
    `query($login: String!) {
      user(login: $login) {
        ${baseQuery}
      }
    }`,
    { login: projectInfo.ownerLogin }
  );
  const conn = result && result.user && result.user.projectsV2;
  const rawNodes = conn && Array.isArray(conn.nodes) ? conn.nodes : [];
  const rawEdges = conn && Array.isArray(conn.edges) ? conn.edges : [];

  const nodeNodes = rawNodes.filter(Boolean);
  const edgeNodes = rawEdges.map(e => e && e.node).filter(Boolean);

  /** @type {Map<string, any>} */
  const unique = new Map();
  for (const n of [...nodeNodes, ...edgeNodes]) {
    if (n && typeof n.id === "string") {
      unique.set(n.id, n);
    }
  }

  return {
    nodes: Array.from(unique.values()),
    totalCount: conn && conn.totalCount,
    diagnostics: {
      rawNodesCount: rawNodes.length,
      nullNodesCount: rawNodes.length - nodeNodes.length,
      rawEdgesCount: rawEdges.length,
      nullEdgeNodesCount: rawEdges.filter(e => !e || !e.node).length,
    },
  };
}

/**
 * Summarize projects for error messages.
 * @param {Array<{id?: string, number: number, title: string, closed: boolean, url?: string}>} projects
 * @param {number} [limit]
 * @returns {string}
 */
function summarizeProjectsV2(projects, limit = 20) {
  if (!Array.isArray(projects) || projects.length === 0) {
    return "(none)";
  }

  const normalized = projects
    .filter(p => p && typeof p.number === "number" && typeof p.title === "string")
    .slice(0, limit)
    .map(p => `#${p.number} ${p.closed ? "(closed) " : ""}${p.title}`);

  return normalized.length > 0 ? normalized.join("; ") : "(none)";
}

/**
 * Summarize a projectsV2 listing call when it returned no readable projects.
 * @param {{ totalCount?: number, diagnostics?: {rawNodesCount: number, nullNodesCount: number, rawEdgesCount: number, nullEdgeNodesCount: number} }} list
 * @returns {string}
 */
function summarizeEmptyProjectsV2List(list) {
  const total = typeof list.totalCount === "number" ? list.totalCount : undefined;
  const d = list && list.diagnostics;
  const diag = d ? ` nodes=${d.rawNodesCount} (null=${d.nullNodesCount}), edges=${d.rawEdgesCount} (nullNode=${d.nullEdgeNodesCount})` : "";

  if (typeof total === "number" && total > 0) {
    return `(none; totalCount=${total} but returned 0 readable project nodes${diag}. This often indicates the token can see the org/user but lacks Projects v2 access, or the org enforces SSO and the token is not authorized.)`;
  }

  return `(none${diag})`;
}

/**
 * Resolve a Projects v2 project by URL-parsed {scope, ownerLogin, number}.
 * Hybrid strategy:
 *  - Try projectV2(number) first (fast)
 *  - Fall back to listing projectsV2(first:100) and searching (more resilient, better diagnostics)
 * @param {{ scope: "orgs"|"users", ownerLogin: string }} projectInfo
 * @param {number} projectNumberInt
 * @returns {Promise<{id: string, number: number, title?: string, url?: string}>}
 */
async function resolveProjectV2(projectInfo, projectNumberInt) {
  // Fast path: direct lookup by number
  try {
    if (projectInfo.scope === "orgs") {
      const direct = await github.graphql(
        `query($login: String!, $number: Int!) {
          organization(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`,
        { login: projectInfo.ownerLogin, number: projectNumberInt }
      );
      const project = direct && direct.organization && direct.organization.projectV2;
      if (project) {
        return project;
      }
    } else {
      const direct = await github.graphql(
        `query($login: String!, $number: Int!) {
          user(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`,
        { login: projectInfo.ownerLogin, number: projectNumberInt }
      );
      const project = direct && direct.user && direct.user.projectV2;
      if (project) {
        return project;
      }
    }
  } catch (error) {
    core.warning(`Direct projectV2(number) query failed; falling back to projectsV2 list search: ${error.message}`);
  }

  // Fallback: list accessible projects and find by number
  const list = await listAccessibleProjectsV2(projectInfo);
  const nodes = Array.isArray(list.nodes) ? list.nodes : [];
  const found = nodes.find(p => p && typeof p.number === "number" && p.number === projectNumberInt);
  if (found) {
    return found;
  }

  const summary = nodes.length > 0 ? summarizeProjectsV2(nodes) : summarizeEmptyProjectsV2List(list);
  const total = typeof list.totalCount === "number" ? ` (totalCount=${list.totalCount})` : "";
  const who = projectInfo.scope === "orgs" ? `org ${projectInfo.ownerLogin}` : `user ${projectInfo.ownerLogin}`;
  throw new Error(`Project #${projectNumberInt} not found or not accessible for ${who}.${total} Accessible Projects v2: ${summary}`);
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
  const projectInfo = parseProjectUrl(output.project);
  const projectNumberFromUrl = projectInfo.projectNumber;
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

    // Helpful diagnostic: log which account this token belongs to.
    // This is safe to log (no secrets) and helps debug permission mismatches between local runs and Actions.
    try {
      const viewerResult = await github.graphql(
        `query {
          viewer {
            login
          }
        }`
      );
      if (viewerResult && viewerResult.viewer && viewerResult.viewer.login) {
        core.info(`✓ Authenticated as: ${viewerResult.viewer.login}`);
      }
    } catch (viewerError) {
      core.warning(`Could not resolve token identity (viewer.login): ${viewerError.message}`);
    }

    // Step 2: Resolve project using org/user + number parsed from URL
    // Note: GitHub GraphQL `resource(url:)` does not support Projects v2 URLs.
    core.info(`[2/5] Resolving project from URL (scope=${projectInfo.scope}, login=${projectInfo.ownerLogin}, number=${projectNumberFromUrl})...`);
    let projectId;
    let resolvedProjectNumber = projectNumberFromUrl;
    try {
      const projectNumberInt = parseInt(projectNumberFromUrl, 10);
      if (!Number.isFinite(projectNumberInt)) {
        throw new Error(`Invalid project number parsed from URL: ${projectNumberFromUrl}`);
      }

      const project = await resolveProjectV2(projectInfo, projectNumberInt);
      projectId = project.id;
      resolvedProjectNumber = String(project.number);
      core.info(`✓ Resolved project #${resolvedProjectNumber} (${projectInfo.ownerLogin}) (ID: ${projectId})`);
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

      const sanitizedContentNumber = rawContentNumber === undefined || rawContentNumber === null ? "" : typeof rawContentNumber === "number" ? rawContentNumber.toString() : String(rawContentNumber).trim();

      if (!sanitizedContentNumber) {
        core.warning("Content number field provided but empty; skipping project item update.");
      } else if (!/^\d+$/.test(sanitizedContentNumber)) {
        throw new Error(`Invalid content number "${rawContentNumber}". Provide a positive integer.`);
      } else {
        contentNumber = Number.parseInt(sanitizedContentNumber, 10);
      }
    }
    if (contentNumber !== null) {
      const contentType = output.content_type === "pull_request" ? "PullRequest" : output.content_type === "issue" ? "Issue" : output.issue ? "Issue" : "PullRequest";

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
          await github.rest.issues.addLabels({
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
                        color
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
            const isTextField = fieldName.toLowerCase() === "classification" || (typeof fieldValue === "string" && fieldValue.includes("|"));

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
                const allOptions = [...field.options.map(o => ({ name: o.name, description: "", color: o.color || "GRAY" })), { name: String(fieldValue), description: "", color: "GRAY" }];

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
          `  • Or supply a PAT (classic with project + repo scopes, or fine-grained with Projects: Read+Write) via GH_AW_PROJECT_GITHUB_TOKEN.\n` +
          `  • Or use a GitHub App with Projects: Read+Write permission.\n` +
          `  • Ensure the workflow grants projects: write.\n\n` +
          `${usingCustomToken ? "GH_AW_PROJECT_GITHUB_TOKEN is set but lacks access." : "Using default GITHUB_TOKEN - this cannot access Projects v2 API. You must configure GH_AW_PROJECT_GITHUB_TOKEN."}`
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
