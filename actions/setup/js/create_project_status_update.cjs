// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_project_status_update";

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
      "This looks like a token permission problem for Projects v2. The GraphQL fields used by create-project-status-update require a token with Projects access (classic PAT: scope 'project'; fine-grained PAT: Organization permission 'Projects' and access to the org). Fix: set safe-outputs.create-project-status-update.github-token to a secret PAT that can access the target org project."
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
    // If GraphQL returned an error (e.g., insufficient permissions/scopes), surface it
    // instead of masking it as "not found".
    core.warning(`Direct projectV2(number) query failed: ${getErrorMessage(error)}`);
    throw error;
  }

  throw new Error(`Project #${projectNumberInt} not found or not accessible for ${projectInfo.scope === "orgs" ? `org ${projectInfo.ownerLogin}` : `user ${projectInfo.ownerLogin}`}`);
}

/**
 * Validate status enum value
 * @param {unknown} status - Status value to validate
 * @returns {string} Validated status
 */
function validateStatus(status) {
  const validStatuses = ["INACTIVE", "ON_TRACK", "AT_RISK", "OFF_TRACK", "COMPLETE"];
  const statusStr = String(status || "ON_TRACK").toUpperCase();

  if (!validStatuses.includes(statusStr)) {
    core.warning(`Invalid status "${status}", using ON_TRACK. Valid values: ${validStatuses.join(", ")}`);
    return "ON_TRACK";
  }

  return statusStr;
}

/**
 * Format date to ISO 8601 (YYYY-MM-DD)
 * @param {unknown} date - Date to format (string or Date object)
 * @returns {string} Formatted date
 */
function formatDate(date) {
  if (!date) {
    return new Date().toISOString().split("T")[0];
  }

  if (typeof date === "string") {
    // If already in YYYY-MM-DD format, return as-is
    if (/^\d{4}-\d{2}-\d{2}$/.test(date)) {
      return date;
    }
    // Otherwise parse and format
    const parsed = new Date(date);
    if (isNaN(parsed.getTime())) {
      core.warning(`Invalid date "${date}", using today`);
      return new Date().toISOString().split("T")[0];
    }
    return parsed.toISOString().split("T")[0];
  }

  if (date instanceof Date) {
    return date.toISOString().split("T")[0];
  }

  core.warning(`Invalid date type ${typeof date}, using today`);
  return new Date().toISOString().split("T")[0];
}

/**
 * Main handler factory for create_project_status_update
 * Returns a message handler function that processes individual create_project_status_update messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const maxCount = config.max || 10;

  core.info(`Max count: ${maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Track created status updates for outputs
  const createdStatusUpdates = [];

  /**
   * Message handler function that processes a single create_project_status_update message
   * @param {Object} message - The create_project_status_update message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status and status update details
   */
  return async function handleCreateProjectStatusUpdate(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping create-project-status-update: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const output = message;

    // Validate required fields
    if (!output.project) {
      core.error("Missing required field: project (GitHub project URL)");
      return {
        success: false,
        error: "Missing required field: project",
      };
    }

    if (!output.body) {
      core.error("Missing required field: body (status update content)");
      return {
        success: false,
        error: "Missing required field: body",
      };
    }

    try {
      core.info(`Creating status update for project: ${output.project}`);

      // Parse project URL and resolve project ID
      const projectInfo = parseProjectUrl(output.project);
      const projectNumberInt = parseInt(projectInfo.projectNumber, 10);

      if (!Number.isFinite(projectNumberInt)) {
        throw new Error(`Invalid project number parsed from URL: ${projectInfo.projectNumber}`);
      }

      const project = await resolveProjectV2(projectInfo, projectNumberInt);
      const projectId = project.id;

      core.info(`✓ Resolved project #${project.number} (${projectInfo.ownerLogin}) (ID: ${projectId})`);

      // Validate and format inputs
      const status = validateStatus(output.status);
      const startDate = formatDate(output.start_date);
      const targetDate = formatDate(output.target_date);
      const body = String(output.body);

      core.info(`Creating status update: ${status} (${startDate} → ${targetDate})`);
      core.info(`Body preview: ${body.substring(0, 100)}${body.length > 100 ? "..." : ""}`);

      // Create the status update using GraphQL mutation
      const mutation = `
        mutation($projectId: ID!, $body: String!, $startDate: Date, $targetDate: Date, $status: ProjectV2StatusUpdateStatus!) {
          createProjectV2StatusUpdate(
            input: {
              projectId: $projectId,
              body: $body,
              startDate: $startDate,
              targetDate: $targetDate,
              status: $status
            }
          ) {
            statusUpdate {
              id
              body
              bodyHTML
              startDate
              targetDate
              status
              createdAt
            }
          }
        }
      `;

      const result = await github.graphql(mutation, {
        projectId,
        body,
        startDate,
        targetDate,
        status,
      });

      const statusUpdate = result.createProjectV2StatusUpdate.statusUpdate;

      core.info(`✓ Created status update: ${statusUpdate.id}`);
      core.info(`  Status: ${statusUpdate.status}`);
      core.info(`  Start: ${statusUpdate.startDate}`);
      core.info(`  Target: ${statusUpdate.targetDate}`);
      core.info(`  Created: ${statusUpdate.createdAt}`);

      // Track created status update
      createdStatusUpdates.push({
        id: statusUpdate.id,
        project_id: projectId,
        project_number: project.number,
        status: statusUpdate.status,
        start_date: statusUpdate.startDate,
        target_date: statusUpdate.targetDate,
        created_at: statusUpdate.createdAt,
      });

      // Set output for step
      core.setOutput("status-update-id", statusUpdate.id);
      core.setOutput("created-status-updates", JSON.stringify(createdStatusUpdates));

      return {
        success: true,
        status_update_id: statusUpdate.id,
        project_id: projectId,
        status: statusUpdate.status,
      };
    } catch (err) {
      // prettier-ignore
      const error = /** @type {Error & { errors?: Array<{ type?: string, message: string, path?: unknown, locations?: unknown }>, request?: unknown, data?: unknown }} */ (err);
      core.error(`Failed to create project status update: ${getErrorMessage(error)}`);
      logGraphQLError(error, "Creating project status update");

      return {
        success: false,
        error: getErrorMessage(error),
      };
    }
  };
}

module.exports = { main, HANDLER_TYPE };
