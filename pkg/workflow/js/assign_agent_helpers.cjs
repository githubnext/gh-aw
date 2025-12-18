const AGENT_LOGIN_NAMES = { copilot: "copilot-swe-agent" };
function getAgentName(assignee) {
  const normalized = assignee.startsWith("@") ? assignee.slice(1) : assignee;
  return AGENT_LOGIN_NAMES[normalized] ? normalized : null;
}
async function getAvailableAgentLogins(owner, repo) {
  try {
    const response = await github.graphql(
        "\n    query($owner: String!, $repo: String!) {\n      repository(owner: $owner, name: $repo) {\n        suggestedActors(first: 100, capabilities: CAN_BE_ASSIGNED) {\n          nodes { ... on Bot { login __typename } }\n        }\n      }\n    }\n  ",
        { owner, repo }
      ),
      actors = response.repository?.suggestedActors?.nodes || [],
      knownValues = Object.values(AGENT_LOGIN_NAMES),
      available = [];
    for (const actor of actors) actor && actor.login && knownValues.includes(actor.login) && available.push(actor.login);
    return available.sort();
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return (core.debug(`Failed to list available agent logins: ${msg}`), []);
  }
}
async function findAgent(owner, repo, agentName) {
  try {
    const actors = (
        await github.graphql(
          "\n    query($owner: String!, $repo: String!) {\n      repository(owner: $owner, name: $repo) {\n        suggestedActors(first: 100, capabilities: CAN_BE_ASSIGNED) {\n          nodes {\n            ... on Bot {\n              id\n              login\n              __typename\n            }\n          }\n        }\n      }\n    }\n  ",
          { owner, repo }
        )
      ).repository.suggestedActors.nodes,
      loginName = AGENT_LOGIN_NAMES[agentName];
    if (!loginName) return (core.error(`Unknown agent: ${agentName}. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`), null);
    for (const actor of actors) if (actor.login === loginName) return actor.id;
    const available = actors.filter(a => a && a.login && Object.values(AGENT_LOGIN_NAMES).includes(a.login)).map(a => a.login);
    return (
      core.warning(`${agentName} coding agent (${loginName}) is not available as an assignee for this repository`),
      available.length > 0 ? core.info(`Available assignable coding agents: ${available.join(", ")}`) : core.info("No coding agents are currently assignable in this repository."),
      "copilot" === agentName && core.info("Please visit https://docs.github.com/en/copilot/using-github-copilot/using-copilot-coding-agent-to-work-on-tasks/about-assigning-tasks-to-copilot"),
      null
    );
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    return (core.error(`Failed to find ${agentName} agent: ${errorMessage}`), null);
  }
}
async function getIssueDetails(owner, repo, issueNumber) {
  try {
    const issue = (
      await github.graphql(
        "\n    query($owner: String!, $repo: String!, $issueNumber: Int!) {\n      repository(owner: $owner, name: $repo) {\n        issue(number: $issueNumber) {\n          id\n          assignees(first: 100) {\n            nodes {\n              id\n            }\n          }\n        }\n      }\n    }\n  ",
        { owner, repo, issueNumber }
      )
    ).repository.issue;
    if (!issue || !issue.id) return (core.error("Could not get issue data"), null);
    const currentAssignees = issue.assignees.nodes.map(assignee => assignee.id);
    return { issueId: issue.id, currentAssignees };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    return (core.error(`Failed to get issue details: ${errorMessage}`), null);
  }
}
async function assignAgentToIssue(issueId, agentId, currentAssignees, agentName) {
  const actorIds = [agentId];
  for (const assigneeId of currentAssignees) assigneeId !== agentId && actorIds.push(assigneeId);
  try {
    (core.info("Using built-in github object for mutation"), core.debug(`GraphQL mutation with variables: assignableId=${issueId}, actorIds=${JSON.stringify(actorIds)}`));
    const response = await github.graphql(
      "\n    mutation($assignableId: ID!, $actorIds: [ID!]!) {\n      replaceActorsForAssignable(input: {\n        assignableId: $assignableId,\n        actorIds: $actorIds\n      }) {\n        __typename\n      }\n    }\n  ",
      { assignableId: issueId, actorIds }
    );
    return !!(response && response.replaceActorsForAssignable && response.replaceActorsForAssignable.__typename) || (core.error("Unexpected response from GitHub API"), !1);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    try {
      if ((core.debug(`Raw GraphQL error message: ${errorMessage}`), error && "object" == typeof error)) {
        const details = {};
        (error.errors && (details.errors = error.errors),
          error.response && (details.response = error.response),
          error.data && (details.data = error.data),
          Array.isArray(error.errors) && (details.compactMessages = error.errors.map(e => e.message).filter(Boolean)));
        const serialized = JSON.stringify(details, (_k, v) => v, 2);
        if (serialized && "{}" !== serialized) {
          (core.debug(`Raw GraphQL error details: ${serialized}`), core.error("Raw GraphQL error details (for troubleshooting):"));
          for (const line of serialized.split(/\n/)) line.trim() && core.error(line);
        }
      }
    } catch (loggingErr) {
      core.debug(`Failed to serialize GraphQL error details: ${loggingErr instanceof Error ? loggingErr.message : String(loggingErr)}`);
    }
    if (errorMessage.includes("Resource not accessible by personal access token") || errorMessage.includes("Resource not accessible by integration") || errorMessage.includes("Insufficient permissions to assign")) {
      core.info("Primary mutation replaceActorsForAssignable forbidden. Attempting fallback addAssigneesToAssignable...");
      try {
        const fallbackMutation =
          "\n          mutation($assignableId: ID!, $assigneeIds: [ID!]!) {\n            addAssigneesToAssignable(input: {\n              assignableId: $assignableId,\n              assigneeIds: $assigneeIds\n            }) {\n              clientMutationId\n            }\n          }\n        ";
        (core.info("Using built-in github object for fallback mutation"), core.debug(`Fallback GraphQL mutation with variables: assignableId=${issueId}, assigneeIds=[${agentId}]`));
        const fallbackResp = await github.graphql(fallbackMutation, { assignableId: issueId, assigneeIds: [agentId] });
        if (fallbackResp && fallbackResp.addAssigneesToAssignable) return (core.info(`Fallback succeeded: agent '${agentName}' added via addAssigneesToAssignable.`), !0);
        core.warning("Fallback mutation returned unexpected response; proceeding with permission guidance.");
      } catch (fallbackError) {
        const fbMsg = fallbackError instanceof Error ? fallbackError.message : String(fallbackError);
        core.error(`Fallback addAssigneesToAssignable failed: ${fbMsg}`);
      }
      logPermissionError(agentName);
    } else core.error(`Failed to assign ${agentName}: ${errorMessage}`);
    return !1;
  }
}
function logPermissionError(agentName) {
  (core.error(`Failed to assign ${agentName}: Insufficient permissions`),
    core.error(""),
    core.error("Assigning Copilot agents requires:"),
    core.error("  1. All four workflow permissions:"),
    core.error("     - actions: write"),
    core.error("     - contents: write"),
    core.error("     - issues: write"),
    core.error("     - pull-requests: write"),
    core.error(""),
    core.error("  2. A classic PAT with 'repo' scope OR fine-grained PAT with explicit Write permissions above:"),
    core.error("     (Fine-grained PATs must grant repository access + write for Issues, Pull requests, Contents, Actions)"),
    core.error(""),
    core.error("  3. Repository settings:"),
    core.error("     - Actions must have write permissions"),
    core.error("     - Go to: Settings > Actions > General > Workflow permissions"),
    core.error("     - Select: 'Read and write permissions'"),
    core.error(""),
    core.error("  4. Organization/Enterprise settings:"),
    core.error("     - Check if your org restricts bot assignments"),
    core.error("     - Verify Copilot is enabled for your repository"),
    core.error(""),
    core.info("For more information, see: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr"));
}
function generatePermissionErrorSummary() {
  let content = "\n### ⚠️ Permission Requirements\n\n";
  return (
    (content += "Assigning Copilot agents requires **ALL** of these permissions:\n\n"),
    (content += "```yaml\n"),
    (content += "permissions:\n"),
    (content += "  actions: write\n"),
    (content += "  contents: write\n"),
    (content += "  issues: write\n"),
    (content += "  pull-requests: write\n"),
    (content += "```\n\n"),
    (content += "**Token capability note:**\n"),
    (content += "- Current token (PAT or GITHUB_TOKEN) lacks assignee mutation capability for this repository.\n"),
    (content += "- Both `replaceActorsForAssignable` and fallback `addAssigneesToAssignable` returned FORBIDDEN/Resource not accessible.\n"),
    (content += "- This typically means bot/user assignment requires an elevated OAuth or GitHub App installation token.\n\n"),
    (content += "**Recommended remediation paths:**\n"),
    (content += "1. Create & install a GitHub App with: Issues/Pull requests/Contents/Actions (write) → use installation token in job.\n"),
    (content += "2. Manual assignment: add the agent through the UI until broader token support is available.\n"),
    (content += "3. Open a support ticket referencing failing mutation `replaceActorsForAssignable` and repository slug.\n\n"),
    (content += "**Why this failed:** Fine-grained and classic PATs can update issue title (verified) but not modify assignees in this environment.\n\n"),
    (content += "📖 Reference: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr (general agent docs)\n"),
    "\n### ⚠️ Permission Requirements\n\nAssigning Copilot agents requires **ALL** of these permissions:\n\n```yaml\npermissions:\n  actions: write\n  contents: write\n  issues: write\n  pull-requests: write\n```\n\n**Token capability note:**\n- Current token (PAT or GITHUB_TOKEN) lacks assignee mutation capability for this repository.\n- Both `replaceActorsForAssignable` and fallback `addAssigneesToAssignable` returned FORBIDDEN/Resource not accessible.\n- This typically means bot/user assignment requires an elevated OAuth or GitHub App installation token.\n\n**Recommended remediation paths:**\n1. Create & install a GitHub App with: Issues/Pull requests/Contents/Actions (write) → use installation token in job.\n2. Manual assignment: add the agent through the UI until broader token support is available.\n3. Open a support ticket referencing failing mutation `replaceActorsForAssignable` and repository slug.\n\n**Why this failed:** Fine-grained and classic PATs can update issue title (verified) but not modify assignees in this environment.\n\n📖 Reference: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/create-a-pr (general agent docs)\n"
  );
}
async function assignAgentToIssueByName(owner, repo, issueNumber, agentName) {
  if (!AGENT_LOGIN_NAMES[agentName]) {
    const error = `Agent "${agentName}" is not supported. Supported agents: ${Object.keys(AGENT_LOGIN_NAMES).join(", ")}`;
    return (core.warning(error), { success: !1, error });
  }
  try {
    core.info(`Looking for ${agentName} coding agent...`);
    const agentId = await findAgent(owner, repo, agentName);
    if (!agentId) {
      const error = `${agentName} coding agent is not available for this repository`,
        available = await getAvailableAgentLogins(owner, repo);
      return { success: !1, error: available.length > 0 ? `${error} (available agents: ${available.join(", ")})` : error };
    }
    (core.info(`Found ${agentName} coding agent (ID: ${agentId})`), core.info("Getting issue details..."));
    const issueDetails = await getIssueDetails(owner, repo, issueNumber);
    return issueDetails
      ? (core.info(`Issue ID: ${issueDetails.issueId}`),
        issueDetails.currentAssignees.includes(agentId)
          ? (core.info(`${agentName} is already assigned to issue #${issueNumber}`), { success: !0 })
          : (core.info(`Assigning ${agentName} coding agent to issue #${issueNumber}...`),
            (await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, agentName))
              ? (core.info(`Successfully assigned ${agentName} coding agent to issue #${issueNumber}`), { success: !0 })
              : { success: !1, error: `Failed to assign ${agentName} via GraphQL` }))
      : { success: !1, error: "Failed to get issue details" };
  } catch (error) {
    return { success: !1, error: error instanceof Error ? error.message : String(error) };
  }
}
module.exports = { AGENT_LOGIN_NAMES, getAgentName, getAvailableAgentLogins, findAgent, getIssueDetails, assignAgentToIssue, logPermissionError, generatePermissionErrorSummary, assignAgentToIssueByName };
