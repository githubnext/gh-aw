const { loadAgentOutput } = require("./load_agent_output.cjs");
function logGraphQLError(error, operation) {
  (core.info(`GraphQL Error during: ${operation}`), core.info(`Message: ${error.message}`));
  const errorList = Array.isArray(error.errors) ? error.errors : [],
    hasInsufficientScopes = errorList.some(e => e && "INSUFFICIENT_SCOPES" === e.type),
    hasNotFound = errorList.some(e => e && "NOT_FOUND" === e.type);
  (hasInsufficientScopes
    ? core.info(
        "This looks like a token permission problem for Projects v2. The GraphQL fields used by update_project require a token with Projects access (classic PAT: scope 'project'; fine-grained PAT: Organization permission 'Projects' and access to the org). Fix: set safe-outputs.update-project.github-token to a secret PAT that can access the target org project."
      )
    : hasNotFound &&
      /projectV2\b/.test(error.message) &&
      core.info(
        "GitHub returned NOT_FOUND for ProjectV2. This can mean either: (1) the project number is wrong for Projects v2, (2) the project is a classic Projects board (not Projects v2), or (3) the token does not have access to that org/user project."
      ),
    error.errors &&
      (core.info(`Errors array (${error.errors.length} error(s)):`),
      error.errors.forEach((err, idx) => {
        (core.info(`  [${idx + 1}] ${err.message}`),
          err.type && core.info(`      Type: ${err.type}`),
          err.path && core.info(`      Path: ${JSON.stringify(err.path)}`),
          err.locations && core.info(`      Locations: ${JSON.stringify(err.locations)}`));
      })),
    error.request && core.info(`Request: ${JSON.stringify(error.request, null, 2)}`),
    error.data && core.info(`Response data: ${JSON.stringify(error.data, null, 2)}`));
}
function parseProjectInput(projectUrl) {
  if (!projectUrl || "string" != typeof projectUrl) throw new Error(`Invalid project input: expected string, got ${typeof projectUrl}. The "project" field is required and must be a full GitHub project URL.`);
  const urlMatch = projectUrl.match(/github\.com\/(?:users|orgs)\/[^/]+\/projects\/(\d+)/);
  if (!urlMatch) throw new Error(`Invalid project URL: "${projectUrl}". The "project" field must be a full GitHub project URL (e.g., https://github.com/orgs/myorg/projects/123).`);
  return urlMatch[1];
}
function parseProjectUrl(projectUrl) {
  if (!projectUrl || "string" != typeof projectUrl) throw new Error(`Invalid project input: expected string, got ${typeof projectUrl}. The "project" field is required and must be a full GitHub project URL.`);
  const match = projectUrl.match(/github\.com\/(users|orgs)\/([^/]+)\/projects\/(\d+)/);
  if (!match) throw new Error(`Invalid project URL: "${projectUrl}". The "project" field must be a full GitHub project URL (e.g., https://github.com/orgs/myorg/projects/123).`);
  return { scope: match[1], ownerLogin: match[2], projectNumber: match[3] };
}
async function listAccessibleProjectsV2(projectInfo) {
  const baseQuery =
    "projectsV2(first: 100) {\n    totalCount\n    nodes {\n      id\n      number\n      title\n      closed\n      url\n    }\n    edges {\n      node {\n        id\n        number\n        title\n        closed\n        url\n      }\n    }\n  }";
  if ("orgs" === projectInfo.scope) {
    const result = await github.graphql(`query($login: String!) {\n        organization(login: $login) {\n          ${baseQuery}\n        }\n      }`, { login: projectInfo.ownerLogin }),
      conn = result && result.organization && result.organization.projectsV2,
      rawNodes = conn && Array.isArray(conn.nodes) ? conn.nodes : [],
      rawEdges = conn && Array.isArray(conn.edges) ? conn.edges : [],
      nodeNodes = rawNodes.filter(Boolean),
      edgeNodes = rawEdges.map(e => e && e.node).filter(Boolean),
      unique = new Map();
    for (const n of [...nodeNodes, ...edgeNodes]) n && "string" == typeof n.id && unique.set(n.id, n);
    return {
      nodes: Array.from(unique.values()),
      totalCount: conn && conn.totalCount,
      diagnostics: { rawNodesCount: rawNodes.length, nullNodesCount: rawNodes.length - nodeNodes.length, rawEdgesCount: rawEdges.length, nullEdgeNodesCount: rawEdges.filter(e => !e || !e.node).length },
    };
  }
  const result = await github.graphql(`query($login: String!) {\n      user(login: $login) {\n        ${baseQuery}\n      }\n    }`, { login: projectInfo.ownerLogin }),
    conn = result && result.user && result.user.projectsV2,
    rawNodes = conn && Array.isArray(conn.nodes) ? conn.nodes : [],
    rawEdges = conn && Array.isArray(conn.edges) ? conn.edges : [],
    nodeNodes = rawNodes.filter(Boolean),
    edgeNodes = rawEdges.map(e => e && e.node).filter(Boolean),
    unique = new Map();
  for (const n of [...nodeNodes, ...edgeNodes]) n && "string" == typeof n.id && unique.set(n.id, n);
  return {
    nodes: Array.from(unique.values()),
    totalCount: conn && conn.totalCount,
    diagnostics: { rawNodesCount: rawNodes.length, nullNodesCount: rawNodes.length - nodeNodes.length, rawEdgesCount: rawEdges.length, nullEdgeNodesCount: rawEdges.filter(e => !e || !e.node).length },
  };
}
function summarizeProjectsV2(projects, limit = 20) {
  if (!Array.isArray(projects) || 0 === projects.length) return "(none)";
  const normalized = projects
    .filter(p => p && "number" == typeof p.number && "string" == typeof p.title)
    .slice(0, limit)
    .map(p => `#${p.number} ${p.closed ? "(closed) " : ""}${p.title}`);
  return normalized.length > 0 ? normalized.join("; ") : "(none)";
}
function summarizeEmptyProjectsV2List(list) {
  const total = "number" == typeof list.totalCount ? list.totalCount : void 0,
    d = list && list.diagnostics,
    diag = d ? ` nodes=${d.rawNodesCount} (null=${d.nullNodesCount}), edges=${d.rawEdgesCount} (nullNode=${d.nullEdgeNodesCount})` : "";
  return "number" == typeof total && total > 0
    ? `(none; totalCount=${total} but returned 0 readable project nodes${diag}. This often indicates the token can see the org/user but lacks Projects v2 access, or the org enforces SSO and the token is not authorized.)`
    : `(none${diag})`;
}
async function resolveProjectV2(projectInfo, projectNumberInt) {
  try {
    if ("orgs" === projectInfo.scope) {
      const direct = await github.graphql(
          "query($login: String!, $number: Int!) {\n          organization(login: $login) {\n            projectV2(number: $number) {\n              id\n              number\n              title\n              url\n            }\n          }\n        }",
          { login: projectInfo.ownerLogin, number: projectNumberInt }
        ),
        project = direct && direct.organization && direct.organization.projectV2;
      if (project) return project;
    } else {
      const direct = await github.graphql(
          "query($login: String!, $number: Int!) {\n          user(login: $login) {\n            projectV2(number: $number) {\n              id\n              number\n              title\n              url\n            }\n          }\n        }",
          { login: projectInfo.ownerLogin, number: projectNumberInt }
        ),
        project = direct && direct.user && direct.user.projectV2;
      if (project) return project;
    }
  } catch (error) {
    core.warning(`Direct projectV2(number) query failed; falling back to projectsV2 list search: ${error.message}`);
  }
  const list = await listAccessibleProjectsV2(projectInfo),
    nodes = Array.isArray(list.nodes) ? list.nodes : [],
    found = nodes.find(p => p && "number" == typeof p.number && p.number === projectNumberInt);
  if (found) return found;
  const summary = nodes.length > 0 ? summarizeProjectsV2(nodes) : summarizeEmptyProjectsV2List(list),
    total = "number" == typeof list.totalCount ? ` (totalCount=${list.totalCount})` : "",
    who = "orgs" === projectInfo.scope ? `org ${projectInfo.ownerLogin}` : `user ${projectInfo.ownerLogin}`;
  throw new Error(`Project #${projectNumberInt} not found or not accessible for ${who}.${total} Accessible Projects v2: ${summary}`);
}
function generateCampaignId(projectUrl, projectNumber) {
  const urlMatch = projectUrl.match(/github\.com\/(users|orgs)\/([^/]+)\/projects/);
  return `${`${urlMatch ? urlMatch[2] : "project"}-project-${projectNumber}`
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .substring(0, 30)}-${Date.now().toString(36).substring(0, 8)}`;
}
async function updateProject(output) {
  const { owner, repo } = context.repo,
    projectInfo = parseProjectUrl(output.project),
    projectNumberFromUrl = projectInfo.projectNumber,
    campaignId = output.campaign_id;
  try {
    let repoResult;
    (core.info(`Looking up project #${projectNumberFromUrl} from URL: ${output.project}`), core.info("[1/5] Fetching repository information..."));
    try {
      repoResult = await github.graphql(
        "query($owner: String!, $repo: String!) {\n          repository(owner: $owner, name: $repo) {\n            id\n            owner {\n              id\n              __typename\n            }\n          }\n        }",
        { owner, repo }
      );
    } catch (error) {
      throw (logGraphQLError(error, "Fetching repository information"), error);
    }
    const repositoryId = repoResult.repository.id,
      ownerType = repoResult.repository.owner.__typename;
    core.info(`✓ Repository: ${owner}/${repo} (${ownerType})`);
    try {
      const viewerResult = await github.graphql("query {\n          viewer {\n            login\n          }\n        }");
      viewerResult && viewerResult.viewer && viewerResult.viewer.login && core.info(`✓ Authenticated as: ${viewerResult.viewer.login}`);
    } catch (viewerError) {
      core.warning(`Could not resolve token identity (viewer.login): ${viewerError.message}`);
    }
    let projectId;
    core.info(`[2/5] Resolving project from URL (scope=${projectInfo.scope}, login=${projectInfo.ownerLogin}, number=${projectNumberFromUrl})...`);
    let resolvedProjectNumber = projectNumberFromUrl;
    try {
      const projectNumberInt = parseInt(projectNumberFromUrl, 10);
      if (!Number.isFinite(projectNumberInt)) throw new Error(`Invalid project number parsed from URL: ${projectNumberFromUrl}`);
      const project = await resolveProjectV2(projectInfo, projectNumberInt);
      ((projectId = project.id), (resolvedProjectNumber = String(project.number)), core.info(`✓ Resolved project #${resolvedProjectNumber} (${projectInfo.ownerLogin}) (ID: ${projectId})`));
    } catch (error) {
      throw (logGraphQLError(error, "Resolving project from URL"), error);
    }
    core.info("[3/5] Linking project to repository...");
    try {
      await github.graphql(
        "mutation($projectId: ID!, $repositoryId: ID!) {\n          linkProjectV2ToRepository(input: {\n            projectId: $projectId,\n            repositoryId: $repositoryId\n          }) {\n            repository {\n              id\n            }\n          }\n        }",
        { projectId, repositoryId }
      );
    } catch (linkError) {
      (linkError.message && linkError.message.includes("already linked")) || (logGraphQLError(linkError, "Linking project to repository"), core.warning(`Could not link project: ${linkError.message}`));
    }
    (core.info("✓ Project linked to repository"), core.info("[4/5] Processing content (issue/PR/draft) if specified..."));
    const hasContentNumber = void 0 !== output.content_number && null !== output.content_number,
      hasIssue = void 0 !== output.issue && null !== output.issue,
      hasPullRequest = void 0 !== output.pull_request && null !== output.pull_request,
      values = [];
    if (
      (hasContentNumber && values.push({ key: "content_number", value: output.content_number }),
      hasIssue && values.push({ key: "issue", value: output.issue }),
      hasPullRequest && values.push({ key: "pull_request", value: output.pull_request }),
      values.length > 1)
    ) {
      const uniqueValues = [...new Set(values.map(v => String(v.value)))],
        list = values.map(v => `${v.key}=${v.value}`).join(", "),
        descriptor = uniqueValues.length > 1 ? "different values" : `same value "${uniqueValues[0]}"`;
      core.warning(`Multiple content number fields (${descriptor}): ${list}. Using priority content_number > issue > pull_request.`);
    }
    (hasIssue && core.warning('Field "issue" deprecated; use "content_number" instead.'), hasPullRequest && core.warning('Field "pull_request" deprecated; use "content_number" instead.'));

    if ("draft_issue" === output.content_type) {
      values.length > 0 && core.warning('content_number/issue/pull_request is ignored when content_type is "draft_issue".');
      const draftTitle = "string" == typeof output.draft_title ? output.draft_title.trim() : "";
      if (!draftTitle) throw new Error('Invalid draft_title. When content_type is "draft_issue", draft_title is required and must be a non-empty string.');
      const draftBody = "string" == typeof output.draft_body ? output.draft_body : void 0;
      const itemId = (
        await github.graphql(
          "mutation($projectId: ID!, $title: String!, $body: String) {\n          addProjectV2DraftIssue(input: {\n            projectId: $projectId,\n            title: $title,\n            body: $body\n          }) {\n            projectItem {\n              id\n            }\n          }\n        }",
          { projectId, title: draftTitle, body: draftBody }
        )
      ).addProjectV2DraftIssue.projectItem.id;

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
                core.warning(`Failed to create field "${fieldName}": ${createError.message}`);
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
                core.warning(`Failed to create field "${fieldName}": ${createError.message}`);
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
                core.warning(`Failed to create option "${fieldValue}": ${createError.message}`);
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
            ? "query($owner: String!, $repo: String!, $number: Int!) {\n            repository(owner: $owner, name: $repo) {\n              issue(number: $number) {\n                id\n                createdAt\n                closedAt\n              }\n            }\n          }"
            : "query($owner: String!, $repo: String!, $number: Int!) {\n            repository(owner: $owner, name: $repo) {\n              pullRequest(number: $number) {\n                id\n                createdAt\n                closedAt\n              }\n            }\n          }",
        contentResult = await github.graphql(contentQuery, { owner, repo, number: contentNumber }),
        contentData = "Issue" === contentType ? contentResult.repository.issue : contentResult.repository.pullRequest,
        contentId = contentData.id,
        createdAt = contentData.createdAt,
        closedAt = contentData.closedAt,
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
      if (existingItem) ((itemId = existingItem.id), core.info("✓ Item already on board"));
      else {
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
            core.warning(`Failed to add campaign label: ${labelError.message}`);
          }
        }
      }
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
                core.warning(`Failed to create field "${fieldName}": ${createError.message}`);
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
                core.warning(`Failed to create field "${fieldName}": ${createError.message}`);
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
                core.warning(`Failed to create option "${fieldValue}": ${createError.message}`);
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
    if (error.message && error.message.includes("does not have permission to create projects")) {
      const usingCustomToken = !!process.env.GH_AW_PROJECT_GITHUB_TOKEN;
      core.error(
        `Failed to manage project: ${error.message}\n\nTroubleshooting:\n  • Create the project manually at https://github.com/orgs/${owner}/projects/new.\n  • Or supply a PAT (classic with project + repo scopes, or fine-grained with Projects: Read+Write) via GH_AW_PROJECT_GITHUB_TOKEN.\n  • Or use a GitHub App with Projects: Read+Write permission.\n  • Ensure the workflow grants projects: write.\n\n` +
          (usingCustomToken ? "GH_AW_PROJECT_GITHUB_TOKEN is set but lacks access." : "Using default GITHUB_TOKEN - this cannot access Projects v2 API. You must configure GH_AW_PROJECT_GITHUB_TOKEN.")
      );
    } else core.error(`Failed to manage project: ${error.message}`);
    throw error;
  }
}
async function main() {
  const result = loadAgentOutput();
  if (!result.success) return;
  const updateProjectItems = result.items.filter(item => "update_project" === item.type);
  if (0 !== updateProjectItems.length)
    for (let i = 0; i < updateProjectItems.length; i++) {
      const output = updateProjectItems[i];
      try {
        await updateProject(output);
      } catch (error) {
        (core.error(`Failed to process item ${i + 1}`), logGraphQLError(error, `Processing update_project item ${i + 1}`));
      }
    }
}

module.exports = { updateProject, parseProjectInput, generateCampaignId, main };
