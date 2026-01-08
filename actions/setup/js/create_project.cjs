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
      "This looks like a token permission problem for Projects v2. The GraphQL fields used by create_project require a token with Projects access (classic PAT: scope 'project'; fine-grained PAT: Organization permission 'Projects' and access to the org). Fix: set safe-outputs.create-project.github-token to a secret PAT that can create projects in the target org."
    );
  } else if (hasNotFound && /projectV2\b/.test(getErrorMessage(error))) {
    core.info(
      "GitHub returned NOT_FOUND for ProjectV2. This can mean either: (1) the owner does not exist, or (2) the token does not have access to that org/user."
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
 * Get owner ID for an org or user
 * @param {string} ownerType - Either "org" or "user"
 * @param {string} ownerLogin - Login name of the owner
 * @returns {Promise<string>} Owner node ID
 */
async function getOwnerId(ownerType, ownerLogin) {
  if (ownerType === "org") {
    const result = await github.graphql(
      `query($login: String!) {
        organization(login: $login) {
          id
        }
      }`,
      { login: ownerLogin }
    );
    return result.organization.id;
  } else {
    const result = await github.graphql(
      `query($login: String!) {
        user(login: $login) {
          id
        }
      }`,
      { login: ownerLogin }
    );
    return result.user.id;
  }
}

/**
 * Create a new GitHub Project V2
 * @param {string} ownerId - Owner node ID
 * @param {string} title - Project title
 * @returns {Promise<{ projectId: string, projectNumber: number, projectTitle: string, projectUrl: string }>} Created project info
 */
async function createProject(ownerId, title) {
  core.info(`Creating project with title: "${title}"`);
  
  const result = await github.graphql(
    `mutation($ownerId: ID!, $title: String!) {
      createProjectV2(input: { ownerId: $ownerId, title: $title }) {
        projectV2 {
          id
          number
          title
          url
        }
      }
    }`,
    { ownerId, title }
  );

  const project = result.createProjectV2.projectV2;
  core.info(`✓ Created project #${project.number}: ${project.title}`);
  core.info(`  URL: ${project.url}`);

  return {
    projectId: project.id,
    projectNumber: project.number,
    projectTitle: project.title,
    projectUrl: project.url,
  };
}

/**
 * Add an item to a project
 * @param {string} projectId - Project node ID
 * @param {string} contentId - Content node ID (issue, PR, etc.)
 * @returns {Promise<string>} Item ID
 */
async function addItemToProject(projectId, contentId) {
  core.info(`Adding item to project...`);
  
  const result = await github.graphql(
    `mutation($projectId: ID!, $contentId: ID!) {
      addProjectV2ItemById(input: { projectId: $projectId, contentId: $contentId }) {
        item {
          id
        }
      }
    }`,
    { projectId, contentId }
  );

  const itemId = result.addProjectV2ItemById.item.id;
  core.info(`✓ Added item to project`);

  return itemId;
}

/**
 * Get issue node ID from issue number
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<string>} Issue node ID
 */
async function getIssueNodeId(owner, repo, issueNumber) {
  const result = await github.graphql(
    `query($owner: String!, $repo: String!, $issueNumber: Int!) {
      repository(owner: $owner, name: $repo) {
        issue(number: $issueNumber) {
          id
        }
      }
    }`,
    { owner, repo, issueNumber }
  );

  return result.repository.issue.id;
}

/**
 * Main handler for create-project safe output
 */
async function handler() {
  try {
    core.info("Starting create_project handler");

    // Load agent output
    const agentOutput = await loadAgentOutput();
    core.info(`Loaded agent output, checking for create_project calls...`);

    const createProjectCalls = agentOutput.filter(output => output.type === "create_project");
    
    if (createProjectCalls.length === 0) {
      core.info("No create_project calls found in agent output");
      return;
    }

    core.info(`Found ${createProjectCalls.length} create_project call(s)`);

    // Get default target owner from environment variable if set
    const defaultTargetOwner = process.env.GH_AW_CREATE_PROJECT_TARGET_OWNER;

    for (const call of createProjectCalls) {
      const { title, owner, owner_type, item_url } = call.content;

      if (!title) {
        throw new Error("Missing required field 'title' in create_project call");
      }

      // Determine owner - use explicit owner, default, or error
      const targetOwner = owner || defaultTargetOwner;
      if (!targetOwner) {
        throw new Error("No owner specified and no default target-owner configured. Either provide 'owner' field or configure 'target-owner' in safe-outputs.create-project");
      }

      // Determine owner type (org or user)
      const ownerType = owner_type || "org"; // Default to org if not specified

      core.info(`Creating project "${title}" for ${ownerType}/${targetOwner}`);

      // Get owner ID
      const ownerId = await getOwnerId(ownerType, targetOwner);
      
      // Create the project
      const projectInfo = await createProject(ownerId, title);

      // Set outputs
      core.setOutput("project-id", projectInfo.projectId);
      core.setOutput("project-number", projectInfo.projectNumber);
      core.setOutput("project-title", projectInfo.projectTitle);
      core.setOutput("project-url", projectInfo.projectUrl);

      // If item_url is provided, add it to the project
      if (item_url) {
        core.info(`Adding item to project: ${item_url}`);
        
        // Parse item URL to get issue number
        const urlMatch = item_url.match(/github\.com\/([^/]+)\/([^/]+)\/issues\/(\d+)/);
        if (urlMatch) {
          const [, itemOwner, itemRepo, issueNumberStr] = urlMatch;
          const issueNumber = parseInt(issueNumberStr, 10);
          
          // Get issue node ID
          const contentId = await getIssueNodeId(itemOwner, itemRepo, issueNumber);
          
          // Add item to project
          const itemId = await addItemToProject(projectInfo.projectId, contentId);
          core.setOutput("item-id", itemId);
        } else {
          core.warning(`Could not parse item URL: ${item_url}`);
        }
      }

      core.info(`✓ Successfully created project: ${projectInfo.projectUrl}`);
    }

    core.info("✓ All create_project operations completed successfully");
  } catch (error) {
    logGraphQLError(error, "create_project");
    core.setFailed(`Failed to create project: ${getErrorMessage(error)}`);
    throw error;
  }
}

// Run the handler
handler().catch(error => {
  core.setFailed(getErrorMessage(error));
  process.exit(1);
});
