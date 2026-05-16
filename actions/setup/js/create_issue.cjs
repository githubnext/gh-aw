// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");
const { sanitizeTitle, applyTitlePrefix } = require("./sanitize_title.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { generateFooterWithMessages, getDetectionCautionAlert } = require("./messages_footer.cjs");
const { getBodyHeader } = require("./messages_header.cjs");
const { generateWorkflowIdMarker, generateWorkflowCallIdMarker, generateCloseKeyMarker, normalizeCloseOlderKey } = require("./generate_footer.cjs");
const { generateHistoryUrl } = require("./generate_history_link.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { generateTemporaryId, isTemporaryId, normalizeTemporaryId, getOrGenerateTemporaryId, replaceTemporaryIdReferences } = require("./temporary_id.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { removeDuplicateTitleFromDescription } = require("./remove_duplicate_title.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");
const { withRetry, RATE_LIMIT_RETRY_CONFIG } = require("./error_recovery.cjs");
const { renderTemplateFromFile } = require("./messages_core.cjs");
const { createExpirationLine, addExpirationToFooter } = require("./ephemerals.cjs");
const { MAX_SUB_ISSUES, getSubIssueCount } = require("./sub_issue_helpers.cjs");
const { closeOlderIssues, searchOlderIssues, addIssueComment } = require("./close_older_issues.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");
const { tryEnforceArrayLimit } = require("./limit_enforcement_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { parseAllowedIssueFields, validateAllowedIssueFields } = require("./allowed_issue_fields.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { MAX_LABELS, MAX_ASSIGNEES } = require("./constants.cjs");
const { findAgent, getIssueDetails, assignAgentToIssue } = require("./assign_agent_helpers.cjs");
const { parseDeduplicateByTitle, normalizeTitleForDedup, findDuplicateByTitle } = require("./issue_title_dedup.cjs");
const MS_PER_DAY = 24 * 60 * 60 * 1000;
const ISSUE_FIELD_DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const RECENTLY_CLOSED_DEDUP_DAYS = 30;
const TITLE_DEDUP_SEARCH_PER_PAGE = 100;
const TITLE_DEDUP_MAX_SEARCH_PAGES = 2;
const TITLE_DEDUP_MIN_SEARCH_RATE_LIMIT_REMAINING = 500;

/**
 * Create a dedicated GitHub client for copilot assignment operations.
 *
 * Token precedence:
 *   1. config["github-token"] — per-handler PAT configured in the workflow frontmatter
 *   2. GH_AW_ASSIGN_TO_AGENT_TOKEN — agent token injected by the compiler as a step env var
 *   3. global github — step-level token (fallback when no agent token is available)
 *
 * @param {Object} config - Handler configuration
 * @returns {Promise<Object>} Authenticated GitHub client
 */
async function createCopilotAssignmentClient(config) {
  const token = config["github-token"] || process.env.GH_AW_ASSIGN_TO_AGENT_TOKEN;
  if (!token) {
    core.debug("No dedicated agent token configured — using step-level github client for copilot assignment");
    return github;
  }
  core.info("Using dedicated github client for copilot assignment");
  return global.getOctokit(token);
}

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_issue";

/** @type {number} Maximum number of sub-issues allowed per parent issue */
const MAX_SUB_ISSUES_PER_PARENT = MAX_SUB_ISSUES;

/** @type {number} Maximum number of parent issues to check when searching */
const MAX_PARENT_ISSUES_TO_CHECK = 10;

/**
 * Searches for an existing parent issue that can accept more sub-issues
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} markerComment - The HTML comment marker to search for
 * @returns {Promise<number|null>} - Parent issue number or null if none found
 */
async function searchForExistingParent(githubClient, owner, repo, markerComment) {
  try {
    const searchQuery = `repo:${owner}/${repo} is:issue "${markerComment}" in:body`;
    const searchResults = await githubClient.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: MAX_PARENT_ISSUES_TO_CHECK,
      sort: "created",
      order: "desc",
    });

    if (searchResults.data.total_count === 0) {
      return null;
    }

    // Check each found issue to see if it can accept more sub-issues
    for (const issue of searchResults.data.items) {
      core.info(`Found potential parent issue #${issue.number}: ${issue.title}`);

      if (issue.state !== "open") {
        core.info(`Parent issue #${issue.number} is ${issue.state}, skipping`);
        continue;
      }

      const subIssueCount = await getSubIssueCount(owner, repo, issue.number);
      if (subIssueCount === null) {
        continue; // Skip if we couldn't get the count
      }

      if (subIssueCount < MAX_SUB_ISSUES_PER_PARENT) {
        core.info(`Using existing parent issue #${issue.number} (has ${subIssueCount}/${MAX_SUB_ISSUES_PER_PARENT} sub-issues)`);
        return issue.number;
      }

      core.info(`Parent issue #${issue.number} is full (${subIssueCount}/${MAX_SUB_ISSUES_PER_PARENT} sub-issues), skipping`);
    }

    return null;
  } catch (error) {
    core.warning(`Could not search for existing parent issues: ${getErrorMessage(error)}`);
    return null;
  }
}

/**
 * Finds an existing parent issue for a group, or creates a new one if needed
 * @param {object} params - Parameters for finding/creating parent issue
 * @param {object} params.githubClient - Authenticated GitHub client
 * @param {string} params.groupId - The group identifier
 * @param {string} params.owner - Repository owner
 * @param {string} params.repo - Repository name
 * @param {string} params.titlePrefix - Title prefix to use
 * @param {string[]} params.labels - Labels to apply to parent issue
 * @param {string} params.workflowName - Workflow name
 * @param {string} params.workflowSourceURL - URL to the workflow source
 * @param {number} [params.expiresHours=0] - Hours until expiration (0 means no expiration)
 * @returns {Promise<number|null>} - Parent issue number or null if creation failed
 */
async function findOrCreateParentIssue({ githubClient, groupId, owner, repo, titlePrefix, labels, workflowName, workflowSourceURL, expiresHours = 0 }) {
  const markerComment = `<!-- gh-aw-group: ${groupId} -->`;

  // Search for existing parent issue with the group marker
  core.info(`Searching for existing parent issue for group: ${groupId}`);
  const existingParent = await searchForExistingParent(githubClient, owner, repo, markerComment);
  if (existingParent) {
    return existingParent;
  }

  // No suitable parent issue found, create a new one
  core.info(`Creating new parent issue for group: ${groupId}`);
  try {
    const template = createParentIssueTemplate(groupId, titlePrefix, workflowName, workflowSourceURL, expiresHours);
    const { data: parentIssue } = await withRetry(
      () =>
        githubClient.rest.issues.create({
          owner,
          repo,
          title: template.title,
          body: template.body,
          labels: labels,
        }),
      { initialDelayMs: 15000, maxDelayMs: 45000, jitterMs: 10000 },
      `create_parent_issue for group ${groupId}`
    );

    core.info(`Created new parent issue #${parentIssue.number}: ${parentIssue.html_url}`);
    return parentIssue.number;
  } catch (error) {
    core.error(`Failed to create parent issue: ${getErrorMessage(error)}`);
    return null;
  }
}

/**
 * Creates a parent issue template for grouping sub-issues
 * @param {string} groupId - The group identifier (workflow ID)
 * @param {string} titlePrefix - Title prefix to use
 * @param {string} workflowName - Name of the workflow
 * @param {string} workflowSourceURL - URL to the workflow source
 * @param {number} [expiresHours=0] - Hours until expiration (0 means no expiration)
 * @returns {object} - Template with title and body
 */
function createParentIssueTemplate(groupId, titlePrefix, workflowName, workflowSourceURL, expiresHours = 0) {
  // Use applyTitlePrefix to ensure proper spacing after prefix
  const title = applyTitlePrefix(`${groupId} - Issue Group`, titlePrefix);

  // Create template context
  const templateContext = {
    group_id: groupId,
    workflow_name: workflowName,
    workflow_source_url: workflowSourceURL || "#",
  };

  // Load and render the issue template
  const issueTemplatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/issue_group_parent.md`;
  let body = renderTemplateFromFile(issueTemplatePath, templateContext);

  // Add footer with workflow information
  const footer = `\n\n> Workflow: [${workflowName}](${workflowSourceURL})`;

  // Add expiration to footer if configured using ephemerals helper
  const footerWithExpiration = addExpirationToFooter(footer, expiresHours, "Parent Issue");

  body = `${body}${footerWithExpiration}`;

  return { title, body };
}

/**
 * Normalize and validate issue fields payload for create_issue.
 * Ensures fields are objects with a non-empty name and string/number value.
 * @param {any} fields
 * @returns {Array<{name: string, value: string|number}>}
 */
function normalizeIssueFields(fields) {
  if (fields == null) {
    return [];
  }
  if (!Array.isArray(fields)) {
    throw new Error(`${ERR_VALIDATION}: create_issue 'fields' must be an array of objects`);
  }

  return fields.map((field, index) => {
    if (!field || typeof field !== "object" || Array.isArray(field)) {
      throw new Error(`${ERR_VALIDATION}: create_issue 'fields[${index}]' must be an object with 'name' and 'value'`);
    }

    const name = typeof field.name === "string" ? field.name.trim() : "";
    if (!name) {
      throw new Error(`${ERR_VALIDATION}: create_issue 'fields[${index}].name' must be a non-empty string`);
    }

    if (!Object.prototype.hasOwnProperty.call(field, "value")) {
      throw new Error(`${ERR_VALIDATION}: create_issue 'fields[${index}]' is missing required 'value'`);
    }

    const value = field.value;
    if ((typeof value !== "string" && typeof value !== "number") || (typeof value === "number" && !Number.isFinite(value))) {
      throw new Error(`${ERR_VALIDATION}: create_issue 'fields[${index}].value' for "${name}" must be a string or number`);
    }

    return { name, value };
  });
}

/**
 * Resolve issue node ID from issue number.
 * Queries GraphQL for the issue node ID required by field mutations.
 * @param {Object} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {number} issueNumber
 * @returns {Promise<string>}
 */
async function resolveIssueNodeId(githubClient, owner, repo, issueNumber) {
  const result = await githubClient.graphql(
    `query($owner: String!, $repo: String!, $issueNumber: Int!) {
      repository(owner: $owner, name: $repo) {
        issue(number: $issueNumber) {
          id
        }
      }
    }`,
    { owner, repo, issueNumber }
  );

  const issueId = result?.repository?.issue?.id;
  if (!issueId) {
    throw new Error(`${ERR_VALIDATION}: could not resolve node ID for issue #${issueNumber}`);
  }
  return issueId;
}

/**
 * Fetch issue field metadata from repository.
 * Returns configured field definitions including types, options, and iterations.
 * @param {Object} githubClient
 * @param {string} owner
 * @param {string} repo
 * @returns {Promise<Array<any>>}
 */
async function fetchIssueFields(githubClient, owner, repo) {
  const result = await githubClient.graphql(
    `query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        issueFields(first: 100) {
          nodes {
            __typename
            ... on IssueField {
              id
              name
              dataType
            }
            ... on IssueFieldSingleSelect {
              id
              name
              dataType
              options {
                id
                name
              }
            }
            ... on IssueFieldIteration {
              id
              name
              dataType
              configuration {
                iterations {
                  id
                  title
                }
              }
            }
          }
        }
      }
    }`,
    { owner, repo }
  );

  return Array.isArray(result?.repository?.issueFields?.nodes) ? result.repository.issueFields.nodes.filter(Boolean) : [];
}

/**
 * Build GraphQL setIssueFieldValue mutation input from named field values.
 * Maps safe-output field names/values to typed GraphQL mutation payloads.
 * @param {Array<{name: string, value: string|number}>} requestedFields
 * @param {Array<any>} availableFields
 * @returns {Array<any>}
 */
function buildIssueFieldMutationInput(requestedFields, availableFields) {
  const availableNames = availableFields.map(field => field?.name).filter(Boolean);

  return requestedFields.map(field => {
    const matchedField = availableFields.find(available => typeof available?.name === "string" && available.name.toLowerCase() === field.name.toLowerCase());
    if (!matchedField) {
      throw new Error(`${ERR_VALIDATION}: unknown issue field "${field.name}". Available fields: ${availableNames.join(", ") || "(none)"}`);
    }

    const dataType = typeof matchedField.dataType === "string" ? matchedField.dataType.toUpperCase() : "TEXT";

    if (dataType === "NUMBER") {
      const numberValue = Number(field.value);
      if (!Number.isFinite(numberValue)) {
        throw new Error(`${ERR_VALIDATION}: issue field "${field.name}" requires a numeric value`);
      }
      return { fieldId: matchedField.id, numberValue };
    }

    if (dataType === "DATE") {
      if (typeof field.value !== "string" || !ISSUE_FIELD_DATE_PATTERN.test(field.value)) {
        throw new Error(`${ERR_VALIDATION}: issue field "${field.name}" requires a date value in YYYY-MM-DD format`);
      }
      return { fieldId: matchedField.id, dateValue: field.value };
    }

    if (dataType === "SINGLE_SELECT") {
      const options = Array.isArray(matchedField.options) ? matchedField.options : [];
      const selectedOption = options.find(option => typeof option?.name === "string" && option.name.toLowerCase() === String(field.value).toLowerCase());
      if (!selectedOption) {
        throw new Error(`${ERR_VALIDATION}: invalid option "${field.value}" for issue field "${field.name}". Available options: ${options.map(option => option.name).join(", ") || "(none)"}`);
      }
      return { fieldId: matchedField.id, singleSelectOptionId: selectedOption.id };
    }

    if (dataType === "ITERATION") {
      const iterations = matchedField?.configuration?.iterations;
      const availableIterations = Array.isArray(iterations) ? iterations : [];
      const selectedIteration = availableIterations.find(iteration => typeof iteration?.title === "string" && iteration.title.toLowerCase() === String(field.value).toLowerCase());
      if (!selectedIteration) {
        throw new Error(`${ERR_VALIDATION}: invalid iteration "${field.value}" for issue field "${field.name}". Available iterations: ${availableIterations.map(iteration => iteration.title).join(", ") || "(none)"}`);
      }
      return { fieldId: matchedField.id, singleSelectOptionId: selectedIteration.id };
    }

    return { fieldId: matchedField.id, textValue: String(field.value) };
  });
}

/**
 * Apply issue field values to a newly-created issue.
 * Resolves metadata and sends the setIssueFieldValue GraphQL mutation.
 * @param {{githubClient: Object, owner: string, repo: string, issueNumber: number, fields: Array<{name: string, value: string|number}>}} params
 * @returns {Promise<void>}
 */
async function applyIssueFields({ githubClient, owner, repo, issueNumber, fields }) {
  if (!Array.isArray(fields) || fields.length === 0) {
    return;
  }

  const issueId = await resolveIssueNodeId(githubClient, owner, repo, issueNumber);
  const availableFields = await fetchIssueFields(githubClient, owner, repo);
  const issueFields = buildIssueFieldMutationInput(fields, availableFields);

  await githubClient.graphql(
    `mutation($input: SetIssueFieldValueInput!) {
      setIssueFieldValue(input: $input) {
        issue {
          id
        }
      }
    }`,
    {
      input: {
        issueId,
        issueFields,
      },
    }
  );
}

async function searchTitleDedupIssues(githubClient, query) {
  const candidates = [];
  let fetchedItems = 0;
  let totalCount = 0;
  let sawNumericTotalCount = false;
  let fetchedPageCount = 0;

  for (let page = 1; page <= TITLE_DEDUP_MAX_SEARCH_PAGES; page += 1) {
    fetchedPageCount = page;
    const response = await githubClient.rest.search.issuesAndPullRequests({
      q: query,
      per_page: TITLE_DEDUP_SEARCH_PER_PAGE,
      page,
      sort: "updated",
      order: "desc",
    });
    const items = Array.isArray(response?.data?.items) ? response.data.items : [];
    const hasNumericTotalCount = Number.isFinite(response?.data?.total_count);
    const pageTotalCount = hasNumericTotalCount ? Number(response.data.total_count) : items.length;
    if (hasNumericTotalCount) {
      sawNumericTotalCount = true;
    }
    if (!hasNumericTotalCount) {
      core.warning(`Title dedup search response missing numeric total_count for query "${query}" (page ${page}); using page item count fallback`);
    }
    totalCount = Math.max(totalCount, pageTotalCount);
    fetchedItems += items.length;

    for (const item of items) {
      if (!item.pull_request && typeof item.title === "string") {
        candidates.push({ title: item.title });
      }
    }

    if (items.length < TITLE_DEDUP_SEARCH_PER_PAGE) {
      break;
    }
  }

  const reachedPageCap = fetchedPageCount === TITLE_DEDUP_MAX_SEARCH_PAGES;
  const fetchedFullPages = fetchedItems === fetchedPageCount * TITLE_DEDUP_SEARCH_PER_PAGE;
  const reachedPageCapWithoutCount = !sawNumericTotalCount && reachedPageCap && fetchedFullPages;

  return {
    candidates,
    fetchedItems,
    totalCount,
    truncated: totalCount > fetchedItems || reachedPageCapWithoutCount,
  };
}

/**
 * Search for existing issues that are potential title-duplicates.
 * Includes open issues and recently closed issues, with paginated search up to a capped page count.
 *
 * @param {Object} githubClient
 * @param {string} owner
 * @param {string} repo
 * @returns {Promise<Array<{title: string}>>}
 */
async function getRepoTitleDedupCandidates(githubClient, owner, repo) {
  const sinceDate = new Date(Date.now() - RECENTLY_CLOSED_DEDUP_DAYS * MS_PER_DAY).toISOString().slice(0, 10);
  const [openIssues, recentlyClosedIssues] = await Promise.all([
    searchTitleDedupIssues(githubClient, `repo:${owner}/${repo} is:issue is:open`),
    searchTitleDedupIssues(githubClient, `repo:${owner}/${repo} is:issue is:closed closed:>=${sinceDate}`),
  ]);

  if (openIssues.truncated) {
    core.warning(`Title dedup search (open issues) truncated for ${owner}/${repo}: fetched ${openIssues.fetchedItems} of ${openIssues.totalCount} results (cap ${TITLE_DEDUP_MAX_SEARCH_PAGES} pages)`);
  }
  if (recentlyClosedIssues.truncated) {
    core.warning(`Title dedup search (recently closed issues) truncated for ${owner}/${repo}: fetched ${recentlyClosedIssues.fetchedItems} of ${recentlyClosedIssues.totalCount} results (cap ${TITLE_DEDUP_MAX_SEARCH_PAGES} pages)`);
  }

  return [...openIssues.candidates, ...recentlyClosedIssues.candidates];
}

/**
 * @param {Object} githubClient
 * @param {string} owner
 * @param {string} repo
 * @returns {Promise<boolean>}
 */
async function shouldSkipRepoTitleDedupSearch(githubClient, owner, repo) {
  try {
    const response = await githubClient.rest.rateLimit.get();
    const rawRemaining = response?.data?.resources?.search?.remaining;
    const remaining = Number(rawRemaining);
    if (!Number.isFinite(remaining)) {
      core.warning(`Could not determine search rate limit remaining for ${owner}/${repo}; proceeding with repo-level title dedup search`);
      return false;
    }
    if (remaining <= TITLE_DEDUP_MIN_SEARCH_RATE_LIMIT_REMAINING) {
      core.warning(`Skipping repo-level title dedup search for ${owner}/${repo}: search rate limit remaining is ${remaining} (threshold <= ${TITLE_DEDUP_MIN_SEARCH_RATE_LIMIT_REMAINING})`);
      return true;
    }
  } catch (error) {
    core.warning(`Could not check search rate limit before title dedup search: ${getErrorMessage(error)} — proceeding with repo-level dedup search`);
  }

  return false;
}

/**
 * Main handler factory for create_issue
 * Returns a message handler function that processes individual create_issue messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const envLabels = config.labels ? (Array.isArray(config.labels) ? config.labels : config.labels.split(",")).map(label => String(label).trim()).filter(Boolean) : [];
  const allowedIssueFields = parseAllowedIssueFields(config.allowed_fields);
  const envAssignees = config.assignees ? (Array.isArray(config.assignees) ? config.assignees : config.assignees.split(",")).map(assignee => String(assignee).trim()).filter(Boolean) : [];
  const titlePrefix = config.title_prefix ?? "";
  const expiresHours = config.expires ? parseInt(String(config.expires), 10) : 0;
  const maxCount = config.max ?? 10;
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const groupEnabled = parseBoolTemplatable(config.group, false);
  const closeOlderIssuesEnabled = parseBoolTemplatable(config.close_older_issues, false);
  const groupByDayEnabled = parseBoolTemplatable(config.group_by_day, false);
  let deduplicateByTitle;
  try {
    deduplicateByTitle = parseDeduplicateByTitle(config.deduplicate_by_title);
  } catch (error) {
    throw new Error(`${ERR_VALIDATION}: ${getErrorMessage(error)}`);
  }
  const rawCloseOlderKey = config.close_older_key ? String(config.close_older_key) : "";
  const closeOlderKey = rawCloseOlderKey ? normalizeCloseOlderKey(rawCloseOlderKey) : "";
  if (rawCloseOlderKey && !closeOlderKey) {
    throw new Error(`${ERR_VALIDATION}: close-older-key "${rawCloseOlderKey}" is invalid: it must contain at least one alphanumeric character after normalization`);
  }
  const includeFooter = parseBoolTemplatable(config.footer, true);

  // Create an authenticated GitHub client. Uses config["github-token"] when set
  // (for cross-repository operations), otherwise falls back to the step-level github.
  const githubClient = await createAuthenticatedGitHubClient(config);

  // Check if copilot assignment is enabled
  const assignCopilot = process.env.GH_AW_ASSIGN_COPILOT === "true";

  // Lazily-initialised client for copilot assignment (only allocated when needed).
  // Uses GH_AW_ASSIGN_TO_AGENT_TOKEN (agent token preference chain) when available,
  // otherwise falls back to the step-level github object.
  /** @type {Object|null} */
  let copilotClient = null;

  // Check if we're in staged mode
  const isStaged = isStagedMode(config);

  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }
  if (envLabels.length > 0) {
    core.info(`Default labels: ${envLabels.join(", ")}`);
  }
  if (envAssignees.length > 0) {
    core.info(`Default assignees: ${envAssignees.join(", ")}`);
  }
  if (allowedIssueFields.length > 0 && !allowedIssueFields.includes("*")) {
    core.info(`Allowed issue fields: ${allowedIssueFields.join(", ")}`);
  }
  if (titlePrefix) {
    core.info(`Title prefix: ${titlePrefix}`);
  }
  if (expiresHours > 0) {
    core.info(`Issues expire after: ${expiresHours} hours`);
  }
  core.info(`Max count: ${maxCount}`);
  if (groupEnabled) {
    core.info(`Issue grouping enabled: issues will be grouped as sub-issues`);
  }
  if (closeOlderIssuesEnabled) {
    core.info(`Close older issues enabled: older issues with same workflow-id marker will be closed`);
    if (closeOlderKey) {
      core.info(`  Using explicit close-older-key: "${closeOlderKey}"`);
    }
  }
  if (groupByDayEnabled) {
    core.info(`Group-by-day mode enabled: if an open issue was already created today, new content will be posted as a comment`);
    if (!closeOlderKey && !process.env.GH_AW_WORKFLOW_ID) {
      core.warning(`Group-by-day mode has no effect: neither close-older-key nor GH_AW_WORKFLOW_ID is set — issues cannot be searched`);
    }
  }
  if (deduplicateByTitle.enabled) {
    const mode = deduplicateByTitle.maxDistance === 0 ? "exact title match" : `Levenshtein distance <= ${deduplicateByTitle.maxDistance}`;
    core.info(`Title deduplication enabled (${mode})`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Track created issues for outputs
  const createdIssues = [];

  // Track seen issue titles by repo for within-run deduplication
  /** @type {Map<string, Array<{title: string, normalizedTitle: string}>>} */
  const createdTitlesByRepo = new Map();
  /** @type {Map<string, Promise<Array<{title: string}>>>} */
  const repoTitleDedupCandidatesCache = new Map();
  let skipRepoLevelSearch = false;

  /**
   * @param {string} repo
   * @param {string} seenTitle
   * @param {string} seenNormalizedTitle
   * @returns {void}
   */
  function recordSeenTitle(repo, seenTitle, seenNormalizedTitle) {
    const titles = createdTitlesByRepo.get(repo) || [];
    titles.push({ title: seenTitle, normalizedTitle: seenNormalizedTitle });
    createdTitlesByRepo.set(repo, titles);
  }

  // Map to track temporary_id -> {repo, number} relationships across messages
  const temporaryIdMap = new Map();

  // Cache for parent issue per group ID
  const parentIssueCache = new Map();

  // Extract triggering context for footer generation
  const triggeringIssueNumber = context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber = context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;
  const parentIssueNumber = context.payload?.issue?.number;

  /**
   * Message handler function that processes a single create_issue message
   * @param {Object} message - The create_issue message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status and issue details
   */
  return async function handleCreateIssue(message, resolvedTemporaryIds) {
    // Merge external resolved temp IDs with our local map
    if (resolvedTemporaryIds) {
      for (const [tempId, resolved] of Object.entries(resolvedTemporaryIds)) {
        if (!temporaryIdMap.has(tempId)) {
          temporaryIdMap.set(tempId, resolved);
        }
      }
    }

    // Resolve and validate target repository
    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "issue");
    if (!repoResult.success) {
      core.warning(`Skipping issue: ${repoResult.error}`);
      return {
        success: false,
        error: repoResult.error,
      };
    }
    const { repo: qualifiedItemRepo, repoParts } = repoResult;

    // Get or generate the temporary ID for this issue
    const tempIdResult = getOrGenerateTemporaryId(message, "issue");
    if (tempIdResult.error) {
      core.warning(`Skipping issue: ${tempIdResult.error}`);
      return {
        success: false,
        error: tempIdResult.error,
      };
    }
    // At this point, temporaryId is guaranteed to be a string (not null)
    const temporaryId = /** @type {string} */ tempIdResult.temporaryId;
    core.info(`Processing create_issue: title=${message.title}, bodyLength=${message.body?.length ?? 0}, temporaryId=${temporaryId}, repo=${qualifiedItemRepo}`);

    // Resolve parent: check if it's a temporary ID reference
    let effectiveParentIssueNumber;
    let effectiveParentRepo = qualifiedItemRepo; // Default to same repo
    if (message.parent !== undefined) {
      // Strip # prefix if present to allow flexible temporary ID format
      const parentStr = String(message.parent).trim();
      const parentWithoutHash = parentStr.startsWith("#") ? parentStr.substring(1) : parentStr;

      if (isTemporaryId(parentWithoutHash)) {
        // It's a temporary ID, look it up in the map
        const resolvedParent = temporaryIdMap.get(normalizeTemporaryId(parentWithoutHash));
        if (resolvedParent) {
          effectiveParentIssueNumber = resolvedParent.number;
          effectiveParentRepo = resolvedParent.repo;
          core.info(`Resolved parent temporary ID '${message.parent}' to ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
        } else {
          core.warning(`Parent temporary ID '${message.parent}' not found in map. Ensure parent issue is created before sub-issues.`);
        }
      } else {
        // Check if it looks like a malformed temporary ID
        if (parentWithoutHash.startsWith("aw_")) {
          core.warning(`Invalid temporary ID format for parent: '${message.parent}'. Temporary IDs must be in format 'aw_' followed by 3 to 12 alphanumeric characters (A-Za-z0-9). Example: 'aw_abc' or 'aw_Test123'`);
        } else {
          // It's a real issue number
          const parsed = parseInt(parentWithoutHash, 10);
          if (!isNaN(parsed)) {
            effectiveParentIssueNumber = parsed;
          } else {
            core.warning(`Invalid parent value: ${message.parent}. Expected either a valid temporary ID (format: aw_XXXXXXXXXXXX where X is a hex digit) or a numeric issue number.`);
          }
        }
      }
    } else {
      // Only use context parent if we're in the same repo as context
      const contextRepo = `${context.repo.owner}/${context.repo.repo}`;
      if (qualifiedItemRepo === contextRepo) {
        effectiveParentIssueNumber = parentIssueNumber;
      }
    }

    // Build labels array
    const labels = [...envLabels, ...(Array.isArray(message.labels) ? message.labels : [])]
      .filter(Boolean)
      .map(label => String(label).trim())
      .filter(Boolean)
      .map(label => sanitizeLabelContent(label))
      .filter(Boolean)
      .map(label => (label.length > 64 ? label.substring(0, 64) : label))
      .filter((label, index, arr) => arr.indexOf(label) === index);

    // Build assignees array (merge config default assignees with message-specific assignees)
    let assignees = [...envAssignees, ...(Array.isArray(message.assignees) ? message.assignees : [])]
      .filter(Boolean)
      .map(assignee => String(assignee).trim())
      .filter(Boolean)
      .filter((assignee, index, arr) => arr.indexOf(assignee) === index);

    let issueFields;
    try {
      issueFields = normalizeIssueFields(message.fields);
      validateAllowedIssueFields(issueFields, allowedIssueFields);
    } catch (error) {
      return { success: false, error: getErrorMessage(error) };
    }

    // Check if copilot is in the assignees list
    const hasCopilot = assignees.includes("copilot");

    // Filter out "copilot" from assignees - it will be assigned separately using GraphQL
    // Copilot is not a valid GitHub user and must be assigned via the agent assignment API
    assignees = assignees.filter(assignee => assignee !== "copilot");

    // Enforce max limits on labels and assignees before API calls
    const labelsLimitResult = tryEnforceArrayLimit(labels, MAX_LABELS, "labels");
    if (!labelsLimitResult.success) {
      core.warning(`Issue limit exceeded: ${labelsLimitResult.error}`);
      return { success: false, error: labelsLimitResult.error };
    }

    const assigneesLimitResult = tryEnforceArrayLimit(assignees, MAX_ASSIGNEES, "assignees");
    if (!assigneesLimitResult.success) {
      core.warning(`Issue limit exceeded: ${assigneesLimitResult.error}`);
      return { success: false, error: assigneesLimitResult.error };
    }

    let title = message.title?.trim() ?? "";

    // Replace temporary ID references in the body using already-created issues
    let processedBody = replaceTemporaryIdReferences(message.body ?? "", temporaryIdMap, qualifiedItemRepo);

    // Remove duplicate title from description if it starts with a header matching the title
    processedBody = removeDuplicateTitleFromDescription(title, processedBody);

    // Sanitize body content to neutralize @mentions, URLs, and other security risks
    processedBody = sanitizeContent(processedBody);

    const bodyLines = processedBody.split("\n");

    if (!title) {
      title = message.body ?? "Agent Output";
    }

    // Sanitize title for Unicode security and remove any duplicate prefixes
    title = sanitizeTitle(title, titlePrefix);

    // Apply title prefix (only if it doesn't already exist)
    title = applyTitlePrefix(title, titlePrefix);

    const normalizedTitle = normalizeTitleForDedup(title);

    if (message._dropped_duplicate_by_title === true) {
      const existingTitle = typeof message._duplicate_title === "string" ? message._duplicate_title : title;
      const distance = typeof message._duplicate_distance === "number" ? message._duplicate_distance : 0;
      core.warning(`Dropping duplicate create_issue from MCP pre-check in ${qualifiedItemRepo}: "${title}" (matched "${existingTitle}", distance=${distance})`);
      return {
        success: true,
        dropped_duplicate: true,
        dedup_source: "mcp-precheck",
        title,
        duplicate_of_title: existingTitle,
        duplicate_distance: distance,
      };
    }

    if (deduplicateByTitle.enabled) {
      const withinRunCandidates = createdTitlesByRepo.get(qualifiedItemRepo) || [];
      const withinRunDuplicate = findDuplicateByTitle(normalizedTitle, withinRunCandidates, deduplicateByTitle.maxDistance);
      if (withinRunDuplicate) {
        core.warning(`Dropping duplicate create_issue (within-run) in ${qualifiedItemRepo}: "${title}" (matched "${withinRunDuplicate.title}", distance=${withinRunDuplicate.distance})`);
        return {
          success: true,
          dropped_duplicate: true,
          dedup_source: "within-run",
          title,
          duplicate_of_title: withinRunDuplicate.title,
          duplicate_distance: withinRunDuplicate.distance,
        };
      }

      try {
        const repoCacheKey = `${repoParts.owner}/${repoParts.repo}`;
        if (!repoTitleDedupCandidatesCache.has(repoCacheKey) && !skipRepoLevelSearch) {
          skipRepoLevelSearch = await shouldSkipRepoTitleDedupSearch(githubClient, repoParts.owner, repoParts.repo);
          if (!skipRepoLevelSearch) {
            const dedupCandidatesPromise = getRepoTitleDedupCandidates(githubClient, repoParts.owner, repoParts.repo);
            dedupCandidatesPromise.catch(() => {
              if (repoTitleDedupCandidatesCache.get(repoCacheKey) === dedupCandidatesPromise) {
                repoTitleDedupCandidatesCache.delete(repoCacheKey);
              }
            });
            repoTitleDedupCandidatesCache.set(repoCacheKey, dedupCandidatesPromise);
          }
        }

        const repoCandidatesPromise = repoTitleDedupCandidatesCache.get(repoCacheKey);
        if (repoCandidatesPromise) {
          const repoCandidates = await repoCandidatesPromise;
          const repoDuplicate = findDuplicateByTitle(normalizedTitle, repoCandidates, deduplicateByTitle.maxDistance);
          if (repoDuplicate) {
            recordSeenTitle(qualifiedItemRepo, title, normalizedTitle);
            core.warning(`Dropping duplicate create_issue (repo-level) in ${qualifiedItemRepo}: "${title}" (matched "${repoDuplicate.title}", distance=${repoDuplicate.distance})`);
            return {
              success: true,
              dropped_duplicate: true,
              dedup_source: "repo-level",
              title,
              duplicate_of_title: repoDuplicate.title,
              duplicate_distance: repoDuplicate.distance,
            };
          }
        }
      } catch (error) {
        core.warning(`Title deduplication search failed: ${getErrorMessage(error)} — proceeding with issue creation`);
      }
    }

    // Add parent reference
    if (effectiveParentIssueNumber) {
      core.info("Detected issue context, parent issue " + effectiveParentRepo + "#" + effectiveParentIssueNumber);
      // Use full repo reference if cross-repo, short reference if same repo
      if (effectiveParentRepo === qualifiedItemRepo) {
        bodyLines.push(`Related to #${effectiveParentIssueNumber}`);
      } else {
        bodyLines.push(`Related to ${effectiveParentRepo}#${effectiveParentIssueNumber}`);
      }
    }

    const workflowName = process.env.GH_AW_WORKFLOW_NAME ?? "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE ?? "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL ?? "";
    const workflowId = process.env.GH_AW_WORKFLOW_ID ?? "";
    // GH_AW_CALLER_WORKFLOW_ID is set at compile time to `github.repository/<workflow-id>`.
    // When multiple workflows call the same reusable workflow via workflow_call they all
    // share the same GH_AW_WORKFLOW_ID. We embed a separate gh-aw-workflow-call-id marker
    // with the caller's identity so close-older-issues can distinguish callers precisely.
    const callerWorkflowId = process.env.GH_AW_CALLER_WORKFLOW_ID ?? "";
    const runUrl = buildWorkflowRunUrl(context, context.repo);

    // Inject body header before user content (unshifted first, so caution will appear before it)
    const bodyHeader = getBodyHeader({ workflowName, runUrl });
    if (bodyHeader) {
      bodyLines.unshift(...bodyHeader.split("\n"), "");
    }

    // Inject CAUTION at top of body if threat detection warning was raised
    // (unshifted after header so it appears first in the final output)
    const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
    if (detectionCaution) {
      bodyLines.unshift(...detectionCaution.split("\n"), "");
    }

    // Add tracker-id comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    // Generate footer and add expiration using helper
    // When footer is disabled, only add XML markers (no visible footer content)
    if (includeFooter) {
      const historyUrl = generateHistoryUrl({
        owner: repoParts.owner,
        repo: repoParts.repo,
        itemType: "issue",
        workflowCallId: callerWorkflowId,
        workflowId,
        serverUrl: context.serverUrl,
      });
      const footer = addExpirationToFooter(
        generateFooterWithMessages(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber, historyUrl, { skipDetectionCaution: true }).trimEnd(),
        expiresHours,
        "Issue"
      );
      bodyLines.push(``, ``, footer);
    }

    // Add standalone workflow-id marker for searchability (consistent with comments)
    // Always add XML markers even when footer is disabled
    if (workflowId) {
      bodyLines.push(``, generateWorkflowIdMarker(workflowId));
    }
    // Add workflow-call-id marker when available to allow close-older-issues to
    // distinguish callers that share the same reusable workflow (and GH_AW_WORKFLOW_ID)
    if (callerWorkflowId) {
      bodyLines.push(generateWorkflowCallIdMarker(callerWorkflowId));
    }
    // Add explicit close-key marker when a custom deduplication key is provided
    if (closeOlderKey) {
      bodyLines.push(generateCloseKeyMarker(closeOlderKey));
    }

    bodyLines.push("");
    const body = bodyLines.join("\n").trim();

    // Reserve a max-count slot synchronously before any async pre-creation work.
    // There is no await between check and increment, so concurrent invocations
    // cannot interleave between these two operations.
    if (processedCount >= maxCount) {
      core.warning(`Skipping create_issue: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }
    processedCount++;

    // Group-by-day check: if enabled, search for an existing open issue created today.
    // When found, post the new content as a comment on the existing issue instead of
    // creating a duplicate. This groups multiple same-day runs into a single issue.
    // The reserved max-count slot is released when posting as a comment.
    if (groupByDayEnabled && (closeOlderKey || workflowId)) {
      const today = new Date().toISOString().split("T")[0]; // YYYY-MM-DD (UTC)
      try {
        const existingIssues = await searchOlderIssues(
          githubClient,
          repoParts.owner,
          repoParts.repo,
          workflowId,
          0, // no issue to exclude — this is a pre-creation check
          callerWorkflowId,
          closeOlderKey
        );
        const todayIssue = existingIssues.find(issue => {
          const createdDate = issue.created_at ? String(issue.created_at).split("T")[0] : "";
          return createdDate === today;
        });
        if (todayIssue) {
          core.info(`Group-by-day: found open issue #${todayIssue.number} created today (${today}) — posting new content as a comment`);
          const comment = await addIssueComment(githubClient, repoParts.owner, repoParts.repo, todayIssue.number, body);
          core.info(`Posted content as comment ${comment.html_url} on issue #${todayIssue.number}`);
          // No issue was created (content was grouped into a comment), so free
          // the reserved slot for subsequent create_issue calls.
          processedCount--;
          return {
            success: true,
            grouped: true,
            existingIssueNumber: todayIssue.number,
            existingIssueUrl: todayIssue.html_url,
            commentUrl: comment.html_url,
          };
        }
      } catch (error) {
        // Log but do not abort — fall through to normal creation
        core.warning(`Group-by-day pre-check failed: ${getErrorMessage(error)} — proceeding with issue creation`);
      }
    }

    core.info(`Creating issue in ${qualifiedItemRepo} with title: ${title}`);
    core.info(`Labels: ${labels.join(", ")}`);
    if (assignees.length > 0) {
      core.info(`Assignees: ${assignees.join(", ")}`);
    }
    if (issueFields.length > 0) {
      core.info(`Issue fields: ${issueFields.map(field => field.name).join(", ")}`);
    }
    core.info(`Body length: ${body.length}`);

    // If in staged mode, preview the issue without creating it
    if (isStaged) {
      logStagedPreviewInfo(`Would create issue in ${qualifiedItemRepo} with title: ${title}`);
      if (deduplicateByTitle.enabled) {
        recordSeenTitle(qualifiedItemRepo, title, normalizedTitle);
      }
      // Return success with staged flag and preview info
      return {
        success: true,
        staged: true,
        previewInfo: {
          repo: qualifiedItemRepo,
          title,
          labels,
          assignees,
          fields: issueFields,
          bodyLength: body.length,
          temporaryId,
        },
      };
    }

    try {
      const { data: issue } = await withRetry(
        () =>
          githubClient.rest.issues.create({
            owner: repoParts.owner,
            repo: repoParts.repo,
            title,
            body,
            labels,
            assignees,
          }),
        RATE_LIMIT_RETRY_CONFIG,
        `create_issue in ${qualifiedItemRepo}`
      );

      core.info(`Created issue ${qualifiedItemRepo}#${issue.number}: ${issue.html_url}`);
      createdIssues.push({ ...issue, _repo: qualifiedItemRepo });
      if (deduplicateByTitle.enabled) {
        recordSeenTitle(qualifiedItemRepo, title, normalizedTitle);
      }

      if (issueFields.length > 0) {
        try {
          await applyIssueFields({
            githubClient,
            owner: repoParts.owner,
            repo: repoParts.repo,
            issueNumber: issue.number,
            fields: issueFields,
          });
          core.info(`Applied ${issueFields.length} issue field(s) to ${qualifiedItemRepo}#${issue.number}`);
        } catch (error) {
          const fieldError = getErrorMessage(error);
          core.error(`✗ Failed to apply issue fields on ${qualifiedItemRepo}#${issue.number}: ${fieldError}`);
          return {
            success: false,
            error: `Issue ${qualifiedItemRepo}#${issue.number} was created, but issue fields could not be applied: ${fieldError}`,
          };
        }
      }

      // Store the mapping of temporary_id -> {repo, number}
      // temporaryId is guaranteed to be non-null because we checked tempIdResult.error above
      const normalizedTempId = normalizeTemporaryId(String(temporaryId));
      temporaryIdMap.set(normalizedTempId, { repo: qualifiedItemRepo, number: issue.number });
      core.info(`Stored temporary ID mapping: ${temporaryId} -> ${qualifiedItemRepo}#${issue.number}`);

      // Assign copilot directly using agent helpers when enabled (similar to assign_to_agent.cjs pattern)
      if (hasCopilot && assignCopilot) {
        // Lazily allocate the dedicated copilot client on first use
        if (!copilotClient) {
          copilotClient = await createCopilotAssignmentClient(config);
        }
        core.info(`Assigning copilot coding agent to issue #${issue.number} in ${qualifiedItemRepo}...`);
        try {
          const agentId = await findAgent(repoParts.owner, repoParts.repo, "copilot", copilotClient);
          if (!agentId) {
            core.warning(`copilot coding agent is not available for ${qualifiedItemRepo}`);
          } else {
            const issueDetails = await getIssueDetails(repoParts.owner, repoParts.repo, issue.number, copilotClient);
            if (!issueDetails) {
              core.warning(`Failed to get issue details for copilot assignment of issue #${issue.number}`);
            } else if (issueDetails.currentAssignees.some(a => a.id === agentId)) {
              core.info(`copilot is already assigned to issue #${issue.number}`);
            } else {
              const assigned = await assignAgentToIssue(issueDetails.issueId, agentId, issueDetails.currentAssignees, "copilot", null, null, null, null, null, null, copilotClient);
              if (assigned) {
                core.info(`Successfully assigned copilot coding agent to issue #${issue.number}`);
              } else {
                core.warning(`Failed to assign copilot to issue #${issue.number}`);
              }
            }
          }
        } catch (error) {
          core.warning(`Failed to assign copilot to issue #${issue.number}: ${getErrorMessage(error)}`);
        }
      }

      // Close older issues if enabled
      if (closeOlderIssuesEnabled) {
        if (workflowId || closeOlderKey) {
          const searchKey = closeOlderKey ? `close-older-key: ${closeOlderKey}` : `workflow-id: ${workflowId}`;
          core.info(`Attempting to close older issues for ${qualifiedItemRepo}#${issue.number} using ${searchKey}`);
          try {
            const closedIssues = await closeOlderIssues(github, repoParts.owner, repoParts.repo, workflowId, { number: issue.number, html_url: issue.html_url }, workflowName, runUrl, callerWorkflowId, closeOlderKey);
            if (closedIssues.length > 0) {
              core.info(`Closed ${closedIssues.length} older issue(s)`);
            }
          } catch (error) {
            // Log error but don't fail the workflow
            core.warning(`Failed to close older issues: ${getErrorMessage(error)}`);
          }
        } else {
          core.warning("Close older issues enabled but GH_AW_WORKFLOW_ID environment variable not set - skipping");
        }
      }
      if (groupEnabled && !effectiveParentIssueNumber) {
        // Use workflow name as the group ID
        const groupId = workflowName;
        core.info(`Grouping enabled - finding or creating parent issue for group: ${groupId}`);

        // Check cache first
        let groupParentNumber = parentIssueCache.get(groupId);

        if (!groupParentNumber) {
          // Not in cache, find or create parent
          // Parent issue expires 1 day (24 hours) after sub-issues
          const parentExpiresHours = expiresHours > 0 ? expiresHours + 24 : 0;
          groupParentNumber = await findOrCreateParentIssue({
            githubClient: githubClient,
            groupId,
            owner: repoParts.owner,
            repo: repoParts.repo,
            titlePrefix,
            labels,
            workflowName,
            workflowSourceURL,
            expiresHours: parentExpiresHours,
          });

          if (groupParentNumber) {
            // Cache the parent issue number for this group
            parentIssueCache.set(groupId, groupParentNumber);
          }
        }

        if (groupParentNumber) {
          effectiveParentIssueNumber = groupParentNumber;
          effectiveParentRepo = qualifiedItemRepo;
          core.info(`Using parent issue #${effectiveParentIssueNumber} for group: ${groupId}`);
        } else {
          core.warning(`Failed to find or create parent issue for group: ${groupId}`);
        }
      }

      // Sub-issue linking only works within the same repository
      if (effectiveParentIssueNumber && effectiveParentRepo === qualifiedItemRepo) {
        core.info(`Attempting to link issue #${issue.number} as sub-issue of #${effectiveParentIssueNumber}`);
        try {
          // First, get the node IDs for both parent and child issues
          core.info(`Fetching node ID for parent issue #${effectiveParentIssueNumber}...`);
          const getIssueNodeIdQuery = `
            query($owner: String!, $repo: String!, $issueNumber: Int!) {
              repository(owner: $owner, name: $repo) {
                issue(number: $issueNumber) {
                  id
                }
              }
            }
          `;

          // Get parent issue node ID
          const parentResult = await githubClient.graphql(getIssueNodeIdQuery, {
            owner: repoParts.owner,
            repo: repoParts.repo,
            issueNumber: effectiveParentIssueNumber,
          });
          const parentNodeId = parentResult.repository.issue.id;
          core.info(`Parent issue node ID: ${parentNodeId}`);

          // Get child issue node ID
          core.info(`Fetching node ID for child issue #${issue.number}...`);
          const childResult = await githubClient.graphql(getIssueNodeIdQuery, {
            owner: repoParts.owner,
            repo: repoParts.repo,
            issueNumber: issue.number,
          });
          const childNodeId = childResult.repository.issue.id;
          core.info(`Child issue node ID: ${childNodeId}`);

          // Link the child issue as a sub-issue of the parent
          core.info(`Executing addSubIssue mutation...`);
          const addSubIssueMutation = `
            mutation($issueId: ID!, $subIssueId: ID!) {
              addSubIssue(input: {
                issueId: $issueId,
                subIssueId: $subIssueId
              }) {
                subIssue {
                  id
                  number
                }
              }
            }
          `;

          await githubClient.graphql(addSubIssueMutation, {
            issueId: parentNodeId,
            subIssueId: childNodeId,
          });

          core.info("✓ Successfully linked issue #" + issue.number + " as sub-issue of #" + effectiveParentIssueNumber);
        } catch (error) {
          core.info(`Warning: Could not link sub-issue to parent: ${getErrorMessage(error)}`);
          core.info(`Error details: ${error instanceof Error ? error.stack : String(error)}`);
          // Fallback: add a comment if sub-issue linking fails
          try {
            core.info(`Attempting fallback: adding comment to parent issue #${effectiveParentIssueNumber}...`);
            await githubClient.rest.issues.createComment({
              owner: repoParts.owner,
              repo: repoParts.repo,
              issue_number: effectiveParentIssueNumber,
              body: `Created related issue: #${issue.number}`,
            });
            core.info("✓ Added comment to parent issue #" + effectiveParentIssueNumber + " (sub-issue linking not available)");
          } catch (commentError) {
            core.info(`Warning: Could not add comment to parent issue: ${commentError instanceof Error ? commentError.message : String(commentError)}`);
          }
        }
      } else if (effectiveParentIssueNumber && effectiveParentRepo !== qualifiedItemRepo) {
        core.info(`Skipping sub-issue linking: parent is in different repository (${effectiveParentRepo})`);
      }

      // Return result with temporary ID mapping info
      return {
        success: true,
        repo: qualifiedItemRepo,
        number: issue.number,
        url: issue.html_url,
        temporaryId: temporaryId,
        _repo: qualifiedItemRepo, // For tracking in the closure
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      if (errorMessage.includes("Issues has been disabled in this repository")) {
        core.info(`⚠ Cannot create issue "${title}" in ${qualifiedItemRepo}: Issues are disabled for this repository`);
        core.info("Consider enabling issues in repository settings if you want to create issues automatically");
        return {
          success: false,
          error: "Issues disabled for repository",
        };
      }
      core.error(`✗ Failed to create issue "${title}" in ${qualifiedItemRepo}: ${errorMessage}`);
      return {
        success: false,
        error: errorMessage,
      };
    }
  };
}

module.exports = { main, createParentIssueTemplate, searchForExistingParent, getSubIssueCount };
