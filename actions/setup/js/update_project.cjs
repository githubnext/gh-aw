// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Log detailed GraphQL error information
 * @param {Error & { errors?: Array<{ type?: string, message: string, path?: unknown, locations?: unknown }>, request?: unknown, data?: unknown }} error - GraphQL error
 * @param {string} operation - Operation description
 */
function logGraphQLError(error, operation) {
  core.info(`GraphQL Error during: ${operation}`);
  core.info(`Message: ${getErrorMessage(error)}`);

  const errorList = Array.isArray(error.errors) ? error.errors : [];
  const hasInsufficientScopes = errorList.some(e => e?.type === "INSUFFICIENT_SCOPES");
  const hasNotFound = errorList.some(e => e?.type === "NOT_FOUND");

  if (hasInsufficientScopes) {
    core.info(
      "This looks like a token permission problem for Projects v2. The GraphQL fields used by update_project require a token with Projects access (classic PAT: scope 'project'; fine-grained PAT: Organization permission 'Projects' and access to the org). Fix: set safe-outputs.update-project.github-token to a secret PAT that can access the target org project."
    );
  } else if (hasNotFound && /projectV2\b/.test(getErrorMessage(error))) {
    core.info(
      "GitHub returned NOT_FOUND for ProjectV2. This can mean either: (1) the project number is wrong for Projects v2, (2) the project is a classic Projects board (not Projects v2), or (3) the token does not have access to that org/user project."
    );
  }

  if (error.errors) {
    core.info(`Errors array (${error.errors.length} error(s)):`);
    error.errors.forEach((err, idx) => {
      core.info(`  [${idx + 1}] ${err.message}`);
      if (err.type) core.info(`      Type: ${err.type}`);
      if (err.path) core.info(`      Path: ${JSON.stringify(err.path)}`);
      if (err.locations) core.info(`      Locations: ${JSON.stringify(err.locations)}`);
    });
  }

  if (error.request) core.info(`Request: ${JSON.stringify(error.request, null, 2)}`);
  if (error.data) core.info(`Response data: ${JSON.stringify(error.data, null, 2)}`);
}
/**
 * Parse project number from URL
 * @param {unknown} projectUrl - Project URL
 * @returns {string} Project number
 */
function parseProjectInput(projectUrl) {
  if (!projectUrl || typeof projectUrl !== "string") {
    throw new Error(`Invalid project input: expected string, got ${typeof projectUrl}. The "project" field is required and must be a full GitHub project URL.`);
  }

  const urlMatch = projectUrl.match(/github\.com\/(?:users|orgs)\/[^/]+\/projects\/(\d+)/);
  if (!urlMatch) {
    throw new Error(`Invalid project URL: "${projectUrl}". The "project" field must be a full GitHub project URL (e.g., https://github.com/orgs/myorg/projects/123).`);
  }

  return urlMatch[1];
}

/**
 * Parse project URL into components
 * @param {unknown} projectUrl - Project URL
 * @returns {{ scope: string, ownerLogin: string, projectNumber: string }} Project info
 */
function parseProjectUrl(projectUrl) {
  if (!projectUrl || typeof projectUrl !== "string") {
    throw new Error(`Invalid project input: expected string, got ${typeof projectUrl}. The "project" field is required and must be a full GitHub project URL.`);
  }

  const match = projectUrl.match(/github\.com\/(users|orgs)\/([^/]+)\/projects\/(\d+)/);
  if (!match) {
    throw new Error(`Invalid project URL: "${projectUrl}". The "project" field must be a full GitHub project URL (e.g., https://github.com/orgs/myorg/projects/123).`);
  }

  return {
    scope: match[1],
    ownerLogin: match[2],
    projectNumber: match[3],
  };
}
/**
 * List accessible Projects v2 for org or user
 * @param {{ scope: string, ownerLogin: string, projectNumber: string }} projectInfo - Project info
 * @returns {Promise<{ nodes: Array<{ id: string, number: number, title: string, closed?: boolean, url: string }>, totalCount?: number, diagnostics: { rawNodesCount: number, nullNodesCount: number, rawEdgesCount: number, nullEdgeNodesCount: number } }>} List result
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

  const query =
    projectInfo.scope === "orgs"
      ? `query($login: String!) {
        organization(login: $login) {
          ${baseQuery}
        }
      }`
      : `query($login: String!) {
      user(login: $login) {
        ${baseQuery}
      }
    }`;

  const result = await github.graphql(query, { login: projectInfo.ownerLogin });
  const conn = projectInfo.scope === "orgs" ? result?.organization?.projectsV2 : result?.user?.projectsV2;

  const rawNodes = Array.isArray(conn?.nodes) ? conn.nodes : [];
  const rawEdges = Array.isArray(conn?.edges) ? conn.edges : [];
  const nodeNodes = rawNodes.filter(Boolean);
  const edgeNodes = rawEdges.map(e => e?.node).filter(Boolean);

  const unique = new Map();
  for (const n of [...nodeNodes, ...edgeNodes]) {
    if (n && typeof n.id === "string") {
      unique.set(n.id, n);
    }
  }

  return {
    nodes: Array.from(unique.values()),
    totalCount: conn?.totalCount,
    diagnostics: {
      rawNodesCount: rawNodes.length,
      nullNodesCount: rawNodes.length - nodeNodes.length,
      rawEdgesCount: rawEdges.length,
      nullEdgeNodesCount: rawEdges.filter(e => !e || !e.node).length,
    },
  };
}
/**
 * Summarize list of projects
 * @param {Array<{ number: number, title: string, closed?: boolean }>} projects - Projects list
 * @param {number} [limit=20] - Max number to show
 * @returns {string} Summary string
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
 * Summarize empty projects list with diagnostics
 * @param {{ totalCount?: number, diagnostics?: { rawNodesCount: number, nullNodesCount: number, rawEdgesCount: number, nullEdgeNodesCount: number } }} list - List result
 * @returns {string} Summary string
 */
function summarizeEmptyProjectsV2List(list) {
  const total = typeof list.totalCount === "number" ? list.totalCount : undefined;
  const d = list?.diagnostics;
  const diag = d ? ` nodes=${d.rawNodesCount} (null=${d.nullNodesCount}), edges=${d.rawEdgesCount} (nullNode=${d.nullEdgeNodesCount})` : "";

  if (typeof total === "number" && total > 0) {
    return `(none; totalCount=${total} but returned 0 readable project nodes${diag}. This often indicates the token can see the org/user but lacks Projects v2 access, or the org enforces SSO and the token is not authorized.)`;
  }

  return `(none${diag})`;
}
/**
 * Resolve a project by number
 * @param {{ scope: string, ownerLogin: string, projectNumber: string }} projectInfo - Project info
 * @param {number} projectNumberInt - Project number
 * @returns {Promise<{ id: string, number: number, title: string, url: string }>} Project details
 */
async function resolveProjectV2(projectInfo, projectNumberInt) {
  try {
    const query =
      projectInfo.scope === "orgs"
        ? `query($login: String!, $number: Int!) {
          organization(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`
        : `query($login: String!, $number: Int!) {
          user(login: $login) {
            projectV2(number: $number) {
              id
              number
              title
              url
            }
          }
        }`;

    const direct = await github.graphql(query, {
      login: projectInfo.ownerLogin,
      number: projectNumberInt,
    });

    const project = projectInfo.scope === "orgs" ? direct?.organization?.projectV2 : direct?.user?.projectV2;

    if (project) return project;
  } catch (error) {
    core.warning(`Direct projectV2(number) query failed; falling back to projectsV2 list search: ${getErrorMessage(error)}`);
  }

  const list = await listAccessibleProjectsV2(projectInfo);
  const nodes = Array.isArray(list.nodes) ? list.nodes : [];
  const found = nodes.find(p => p && typeof p.number === "number" && p.number === projectNumberInt);

  if (found) return found;

  const summary = nodes.length > 0 ? summarizeProjectsV2(nodes) : summarizeEmptyProjectsV2List(list);
  const total = typeof list.totalCount === "number" ? ` (totalCount=${list.totalCount})` : "";
  const who = projectInfo.scope === "orgs" ? `org ${projectInfo.ownerLogin}` : `user ${projectInfo.ownerLogin}`;

  throw new Error(`Project #${projectNumberInt} not found or not accessible for ${who}.${total} Accessible Projects v2: ${summary}`);
}
/**
 * Generate a campaign ID for the project
 * @param {string} projectUrl - Project URL
 * @param {string} projectNumber - Project number
 * @returns {string} Campaign ID
 */
function generateCampaignId(projectUrl, projectNumber) {
  const urlMatch = projectUrl.match(/github\.com\/(users|orgs)\/([^/]+)\/projects/);
  const base = `${urlMatch ? urlMatch[2] : "project"}-project-${projectNumber}`
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .substring(0, 30);
  const timestamp = Date.now().toString(36).substring(0, 8);
  return `${base}-${timestamp}`;
}
/**
 * Update a GitHub Project v2
 * @param {any} output - Safe output configuration
 * @returns {Promise<void>}
 */
async function updateProject(output) {
  const { owner, repo } = context.repo;
  const projectInfo = parseProjectUrl(output.project);
  const projectNumberFromUrl = projectInfo.projectNumber;
  const campaignId = output.campaign_id;

  try {
    core.info(`Looking up project #${projectNumberFromUrl} from URL: ${output.project}`);
    core.info("[1/4] Fetching repository information...");

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
    } catch (err) {
      // prettier-ignore
      const error = /** @type {Error & { errors?: Array<{ type?: string, message: string, path?: unknown, locations?: unknown }>, request?: unknown, data?: unknown }} */ (err);
      logGraphQLError(error, "Fetching repository information");
      throw error;
    }

    const repositoryId = repoResult.repository.id;
    const ownerType = repoResult.repository.owner.__typename;
    core.info(`✓ Repository: ${owner}/${repo} (${ownerType})`);

    try {
      const viewerResult = await github.graphql(`query {
          viewer {
            login
          }
        }`);
      if (viewerResult?.viewer?.login) {
        core.info(`✓ Authenticated as: ${viewerResult.viewer.login}`);
      }
    } catch (viewerError) {
      core.warning(`Could not resolve token identity (viewer.login): ${getErrorMessage(viewerError)}`);
    }

    let projectId;
    core.info(`[2/4] Resolving project from URL (scope=${projectInfo.scope}, login=${projectInfo.ownerLogin}, number=${projectNumberFromUrl})...`);
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
    } catch (err) {
      // prettier-ignore
      const error = /** @type {Error & { errors?: Array<{ type?: string, message: string, path?: unknown, locations?: unknown }>, request?: unknown, data?: unknown }} */ (err);
      logGraphQLError(error, "Resolving project from URL");
      throw error;
    }

    core.info("[3/4] Processing content (issue/PR/draft) if specified...");
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

    if (hasIssue) core.warning('Field "issue" deprecated; use "content_number" instead.');
    if (hasPullRequest) core.warning('Field "pull_request" deprecated; use "content_number" instead.');

    if (output.content_type === "draft_issue") {
      if (values.length > 0) {
        core.warning('content_number/issue/pull_request is ignored when content_type is "draft_issue".');
      }

      const draftTitle = typeof output.draft_title === "string" ? output.draft_title.trim() : "";
      if (!draftTitle) {
        throw new Error('Invalid draft_title. When content_type is "draft_issue", draft_title is required and must be a non-empty string.');
      }

      const draftBody = typeof output.draft_body === "string" ? output.draft_body : undefined;
      const result = await github.graphql(
        `mutation($projectId: ID!, $title: String!, $body: String) {
          addProjectV2DraftIssue(input: {
            projectId: $projectId,
            title: $title,
            body: $body
          }) {
            projectItem {
              id
            }
          }
        }`,
        { projectId, title: draftTitle, body: draftBody }
      );
      const itemId = result.addProjectV2DraftIssue.projectItem.id;

      const fieldsToUpdate = output.fields ? { ...output.fields } : {};
      if (Object.keys(fieldsToUpdate).length > 0) {
        const projectFields = (
          await github.graphql(
            "query($projectId: ID!) {\n            node(id: $projectId) {\n              ... on ProjectV2 {\n                fields(first: 20) {\n                  nodes {\n                    ... on ProjectV2Field {\n                      id\n                      name\n                      dataType\n                    }\n                    ... on ProjectV2SingleSelectField {\n                      id\n                      name\n                      dataType\n                      options {\n                        id\n                        name\n                        color\n                      }\n                    }\n                  }\n                }\n              }\n            }\n          }",
            { projectId }
          )
        ).node.fields.nodes;
        for (const [fieldName, fieldValue] of Object.entries(fieldsToUpdate)) {
          const normalizedFieldName = fieldName
            .split(/[\s_-]+/)
            .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
            .join(" ");
          let valueToSet,
            field = projectFields.find(f => f.name.toLowerCase() === normalizedFieldName.toLowerCase());
          if (!field)
            if ("classification" === fieldName.toLowerCase() || ("string" == typeof fieldValue && fieldValue.includes("|")))
              try {
                field = (
                  await github.graphql(
                    "mutation($projectId: ID!, $name: String!, $dataType: ProjectV2CustomFieldType!) {\n                    createProjectV2Field(input: {\n                      projectId: $projectId,\n                      name: $name,\n                      dataType: $dataType\n                    }) {\n                      projectV2Field {\n                        ... on ProjectV2Field {\n                          id\n                          name\n                        }\n                        ... on ProjectV2SingleSelectField {\n                          id\n                          name\n                          options { id name }\n                        }\n                      }\n                    }\n                  }",
                    { projectId, name: normalizedFieldName, dataType: "TEXT" }
                  )
                ).createProjectV2Field.projectV2Field;
              } catch (createError) {
                core.warning(`Failed to create field "${fieldName}": ${getErrorMessage(createError)}`);
                continue;
              }
            else
              try {
                field = (
                  await github.graphql(
                    "mutation($projectId: ID!, $name: String!, $dataType: ProjectV2CustomFieldType!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {\n                    createProjectV2Field(input: {\n                      projectId: $projectId,\n                      name: $name,\n                      dataType: $dataType,\n                      singleSelectOptions: $options\n                    }) {\n                      projectV2Field {\n                        ... on ProjectV2SingleSelectField {\n                          id\n                          name\n                          options { id name }\n                        }\n                        ... on ProjectV2Field {\n                          id\n                          name\n                        }\n                      }\n                    }\n                  }",
                    { projectId, name: normalizedFieldName, dataType: "SINGLE_SELECT", options: [{ name: String(fieldValue), description: "", color: "GRAY" }] }
                  )
                ).createProjectV2Field.projectV2Field;
              } catch (createError) {
                core.warning(`Failed to create field "${fieldName}": ${getErrorMessage(createError)}`);
                continue;
              }
          if (field.dataType === "DATE") valueToSet = { date: String(fieldValue) };
          else if (field.options) {
            let option = field.options.find(o => o.name === fieldValue);
            if (!option)
              try {
                const allOptions = [...field.options.map(o => ({ name: o.name, description: "", color: o.color || "GRAY" })), { name: String(fieldValue), description: "", color: "GRAY" }],
                  updatedField = (
                    await github.graphql(
                      "mutation($fieldId: ID!, $fieldName: String!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {\n                    updateProjectV2Field(input: {\n                      fieldId: $fieldId,\n                      name: $fieldName,\n                      singleSelectOptions: $options\n                    }) {\n                      projectV2Field {\n                        ... on ProjectV2SingleSelectField {\n                          id\n                          options {\n                            id\n                            name\n                          }\n                        }\n                      }\n                    }\n                  }",
                      { fieldId: field.id, fieldName: field.name, options: allOptions }
                    )
                  ).updateProjectV2Field.projectV2Field;
                ((option = updatedField.options.find(o => o.name === fieldValue)), (field = updatedField));
              } catch (createError) {
                core.warning(`Failed to create option "${fieldValue}": ${getErrorMessage(createError)}`);
                continue;
              }
            if (!option) {
              core.warning(`Could not get option ID for "${fieldValue}" in field "${fieldName}"`);
              continue;
            }
            valueToSet = { singleSelectOptionId: option.id };
          } else valueToSet = { text: String(fieldValue) };
          await github.graphql(
            "mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: ProjectV2FieldValue!) {\n              updateProjectV2ItemFieldValue(input: {\n                projectId: $projectId,\n                itemId: $itemId,\n                fieldId: $fieldId,\n                value: $value\n              }) {\n                projectV2Item {\n                  id\n                }\n              }\n            }",
            { projectId, itemId, fieldId: field.id, value: valueToSet }
          );
        }
      }

      core.setOutput("item-id", itemId);
      return;
    }
    let contentNumber = null;
    if (hasContentNumber || hasIssue || hasPullRequest) {
      const rawContentNumber = hasContentNumber ? output.content_number : hasIssue ? output.issue : output.pull_request,
        sanitizedContentNumber = null == rawContentNumber ? "" : "number" == typeof rawContentNumber ? rawContentNumber.toString() : String(rawContentNumber).trim();
      if (sanitizedContentNumber) {
        if (!/^\d+$/.test(sanitizedContentNumber)) throw new Error(`Invalid content number "${rawContentNumber}". Provide a positive integer.`);
        contentNumber = Number.parseInt(sanitizedContentNumber, 10);
      } else core.warning("Content number field provided but empty; skipping project item update.");
    }
    if (null !== contentNumber) {
      const contentType = "pull_request" === output.content_type ? "PullRequest" : "issue" === output.content_type || output.issue ? "Issue" : "PullRequest",
        contentQuery =
          "Issue" === contentType
            ? "query($owner: String!, $repo: String!, $number: Int!) {\n            repository(owner: $owner, name: $repo) {\n              issue(number: $number) {\n                id\n              }\n            }\n          }"
            : "query($owner: String!, $repo: String!, $number: Int!) {\n            repository(owner: $owner, name: $repo) {\n              pullRequest(number: $number) {\n                id\n              }\n            }\n          }",
        contentResult = await github.graphql(contentQuery, { owner, repo, number: contentNumber }),
        contentData = "Issue" === contentType ? contentResult.repository.issue : contentResult.repository.pullRequest,
        contentId = contentData.id,
        existingItem = await (async function (projectId, contentId) {
          let hasNextPage = !0,
            endCursor = null;
          for (; hasNextPage; ) {
            const result = await github.graphql(
                "query($projectId: ID!, $after: String) {\n              node(id: $projectId) {\n                ... on ProjectV2 {\n                  items(first: 100, after: $after) {\n                    nodes {\n                      id\n                      content {\n                        ... on Issue {\n                          id\n                        }\n                        ... on PullRequest {\n                          id\n                        }\n                      }\n                    }\n                    pageInfo {\n                      hasNextPage\n                      endCursor\n                    }\n                  }\n                }\n              }\n            }",
                { projectId, after: endCursor }
              ),
              found = result.node.items.nodes.find(item => item.content && item.content.id === contentId);
            if (found) return found;
            ((hasNextPage = result.node.items.pageInfo.hasNextPage), (endCursor = result.node.items.pageInfo.endCursor));
          }
          return null;
        })(projectId, contentId);
      let itemId;
      let existingItemFields = {};
      if (existingItem) {
        itemId = existingItem.id;
        core.info("✓ Item already on board");

        // Fetch existing field values for this item
        try {
          const itemDetails = await github.graphql(
            `query($itemId: ID!) {
              node(id: $itemId) {
                ... on ProjectV2Item {
                  fieldValues(first: 20) {
                    nodes {
                      ... on ProjectV2ItemFieldTextValue {
                        text
                        field {
                          ... on ProjectV2Field {
                            name
                          }
                        }
                      }
                      ... on ProjectV2ItemFieldSingleSelectValue {
                        name
                        field {
                          ... on ProjectV2SingleSelectField {
                            name
                          }
                        }
                      }
                      ... on ProjectV2ItemFieldDateValue {
                        date
                        field {
                          ... on ProjectV2Field {
                            name
                          }
                        }
                      }
                    }
                  }
                }
              }
            }`,
            { itemId }
          );

          // Parse existing field values
          const fieldValues = itemDetails.node?.fieldValues?.nodes || [];
          for (const fieldValue of fieldValues) {
            if (!fieldValue || !fieldValue.field) continue;
            const fieldName = fieldValue.field.name;
            if (fieldValue.text !== undefined) {
              existingItemFields[fieldName] = fieldValue.text;
            } else if (fieldValue.name !== undefined) {
              existingItemFields[fieldName] = fieldValue.name;
            } else if (fieldValue.date !== undefined) {
              existingItemFields[fieldName] = fieldValue.date;
            }
          }
          core.info(`Existing fields: ${JSON.stringify(existingItemFields)}`);
        } catch (fetchError) {
          core.warning(`Failed to fetch existing item fields: ${getErrorMessage(fetchError)}`);
        }
      } else {
        itemId = (
          await github.graphql(
            "mutation($projectId: ID!, $contentId: ID!) {\n            addProjectV2ItemById(input: {\n              projectId: $projectId,\n              contentId: $contentId\n            }) {\n              item {\n                id\n              }\n            }\n          }",
            { projectId, contentId }
          )
        ).addProjectV2ItemById.item.id;
        if (campaignId) {
          try {
            await github.rest.issues.addLabels({ owner, repo, issue_number: contentNumber, labels: [`campaign:${campaignId}`] });
          } catch (labelError) {
            core.warning(`Failed to add campaign label: ${getErrorMessage(labelError)}`);
          }
        }
      }
      const fieldsToUpdate = output.fields ? { ...output.fields } : {};

      // Backfill missing required fields for existing items
      if (existingItem && Object.keys(existingItemFields).length > 0) {
        const requiredDefaults = {
          "Campaign Id": campaignId || output.campaign_id || "unknown",
          "Worker Workflow": "unknown",
          Repository: `${owner}/${repo}`,
          Priority: "Medium",
          Size: "Medium",
        };

        for (const [fieldName, defaultValue] of Object.entries(requiredDefaults)) {
          // Check if field is missing or empty in existing item
          const existingValue = existingItemFields[fieldName];
          const hasExistingValue = existingValue && existingValue.trim() !== "";

          // Check if field is being updated in this operation
          const isBeingUpdated = Object.keys(fieldsToUpdate).some(key => {
            const normalized = key
              .split(/[\s_-]+/)
              .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
              .join(" ");
            return normalized === fieldName;
          });

          // If field is missing and not being updated, add it to fieldsToUpdate
          if (!hasExistingValue && !isBeingUpdated) {
            // Convert field name to snake_case for fieldsToUpdate
            const snakeCaseFieldName = fieldName.toLowerCase().replace(/\s+/g, "_");
            fieldsToUpdate[snakeCaseFieldName] = defaultValue;
            core.info(`Backfilling missing field "${fieldName}" with default value: ${defaultValue}`);
          }
        }
      }
      if (Object.keys(fieldsToUpdate).length > 0) {
        const projectFields = (
          await github.graphql(
            "query($projectId: ID!) {\n            node(id: $projectId) {\n              ... on ProjectV2 {\n                fields(first: 20) {\n                  nodes {\n                    ... on ProjectV2Field {\n                      id\n                      name\n                      dataType\n                    }\n                    ... on ProjectV2SingleSelectField {\n                      id\n                      name\n                      dataType\n                      options {\n                        id\n                        name\n                        color\n                      }\n                    }\n                  }\n                }\n              }\n            }\n          }",
            { projectId }
          )
        ).node.fields.nodes;
        for (const [fieldName, fieldValue] of Object.entries(fieldsToUpdate)) {
          const normalizedFieldName = fieldName
            .split(/[\s_-]+/)
            .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
            .join(" ");
          let valueToSet,
            field = projectFields.find(f => f.name.toLowerCase() === normalizedFieldName.toLowerCase());
          if (!field)
            if ("classification" === fieldName.toLowerCase() || ("string" == typeof fieldValue && fieldValue.includes("|")))
              try {
                field = (
                  await github.graphql(
                    "mutation($projectId: ID!, $name: String!, $dataType: ProjectV2CustomFieldType!) {\n                    createProjectV2Field(input: {\n                      projectId: $projectId,\n                      name: $name,\n                      dataType: $dataType\n                    }) {\n                      projectV2Field {\n                        ... on ProjectV2Field {\n                          id\n                          name\n                        }\n                        ... on ProjectV2SingleSelectField {\n                          id\n                          name\n                          options { id name }\n                        }\n                      }\n                    }\n                  }",
                    { projectId, name: normalizedFieldName, dataType: "TEXT" }
                  )
                ).createProjectV2Field.projectV2Field;
              } catch (createError) {
                core.warning(`Failed to create field "${fieldName}": ${getErrorMessage(createError)}`);
                continue;
              }
            else
              try {
                field = (
                  await github.graphql(
                    "mutation($projectId: ID!, $name: String!, $dataType: ProjectV2CustomFieldType!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {\n                    createProjectV2Field(input: {\n                      projectId: $projectId,\n                      name: $name,\n                      dataType: $dataType,\n                      singleSelectOptions: $options\n                    }) {\n                      projectV2Field {\n                        ... on ProjectV2SingleSelectField {\n                          id\n                          name\n                          options { id name }\n                        }\n                        ... on ProjectV2Field {\n                          id\n                          name\n                        }\n                      }\n                    }\n                  }",
                    { projectId, name: normalizedFieldName, dataType: "SINGLE_SELECT", options: [{ name: String(fieldValue), description: "", color: "GRAY" }] }
                  )
                ).createProjectV2Field.projectV2Field;
              } catch (createError) {
                core.warning(`Failed to create field "${fieldName}": ${getErrorMessage(createError)}`);
                continue;
              }
          // Check dataType first to properly handle DATE fields before checking for options
          // This prevents date fields from being misidentified as single-select fields
          if (field.dataType === "DATE") {
            // Date fields use ProjectV2FieldValue input type with date property
            // The date value must be in ISO 8601 format (YYYY-MM-DD) with no time component
            // Unlike other field types that may require IDs, date fields accept the date string directly
            valueToSet = { date: String(fieldValue) };
          } else if (field.options) {
            let option = field.options.find(o => o.name === fieldValue);
            if (!option)
              try {
                const allOptions = [...field.options.map(o => ({ name: o.name, description: "", color: o.color || "GRAY" })), { name: String(fieldValue), description: "", color: "GRAY" }],
                  updatedField = (
                    await github.graphql(
                      "mutation($fieldId: ID!, $fieldName: String!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {\n                    updateProjectV2Field(input: {\n                      fieldId: $fieldId,\n                      name: $fieldName,\n                      singleSelectOptions: $options\n                    }) {\n                      projectV2Field {\n                        ... on ProjectV2SingleSelectField {\n                          id\n                          options {\n                            id\n                            name\n                          }\n                        }\n                      }\n                    }\n                  }",
                      { fieldId: field.id, fieldName: field.name, options: allOptions }
                    )
                  ).updateProjectV2Field.projectV2Field;
                ((option = updatedField.options.find(o => o.name === fieldValue)), (field = updatedField));
              } catch (createError) {
                core.warning(`Failed to create option "${fieldValue}": ${getErrorMessage(createError)}`);
                continue;
              }
            if (!option) {
              core.warning(`Could not get option ID for "${fieldValue}" in field "${fieldName}"`);
              continue;
            }
            valueToSet = { singleSelectOptionId: option.id };
          } else valueToSet = { text: String(fieldValue) };
          await github.graphql(
            "mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: ProjectV2FieldValue!) {\n              updateProjectV2ItemFieldValue(input: {\n                projectId: $projectId,\n                itemId: $itemId,\n                fieldId: $fieldId,\n                value: $value\n              }) {\n                projectV2Item {\n                  id\n                }\n              }\n            }",
            { projectId, itemId, fieldId: field.id, value: valueToSet }
          );
        }
      }

      core.setOutput("item-id", itemId);
    }
  } catch (error) {
    if (getErrorMessage(error) && getErrorMessage(error).includes("does not have permission to create projects")) {
      const usingCustomToken = !!process.env.GH_AW_PROJECT_GITHUB_TOKEN;
      core.error(
        `Failed to manage project: ${getErrorMessage(error)}\n\nTroubleshooting:\n  • Create the project manually at https://github.com/orgs/${owner}/projects/new.\n  • Or supply a PAT (classic with project + repo scopes, or fine-grained with Projects: Read+Write) via GH_AW_PROJECT_GITHUB_TOKEN.\n  • Or use a GitHub App with Projects: Read+Write permission.\n  • Ensure the workflow grants projects: write.\n\n` +
          (usingCustomToken ? "GH_AW_PROJECT_GITHUB_TOKEN is set but lacks access." : "Using default GITHUB_TOKEN - this cannot access Projects v2 API. You must configure GH_AW_PROJECT_GITHUB_TOKEN.")
      );
    } else {
      core.error(`Failed to manage project: ${getErrorMessage(error)}`);
    }
    throw error;
  }
}

/**
 * Main entry point
 */
async function main() {
  const result = loadAgentOutput();
  if (!result.success) return;

  const updateProjectItems = result.items.filter(item => item.type === "update_project");
  if (updateProjectItems.length === 0) return;

  for (let i = 0; i < updateProjectItems.length; i++) {
    const output = updateProjectItems[i];
    try {
      await updateProject(output);
    } catch (err) {
      // prettier-ignore
      const error = /** @type {Error & { errors?: Array<{ type?: string, message: string, path?: unknown, locations?: unknown }>, request?: unknown, data?: unknown }} */ (err);
      core.error(`Failed to process item ${i + 1}`);
      logGraphQLError(error, `Processing update_project item ${i + 1}`);
    }
  }
}

module.exports = { updateProject, parseProjectInput, generateCampaignId, main };
