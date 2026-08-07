// @ts-check
/// <reference types="@actions/github-script" />

const { closeOlderEntities } = require("./close_older_entities.cjs");
const { searchOlderEntitiesByMarker } = require("./close_older_search_helpers.cjs");

/**
 * Adapt a generic close-older message context to an entity-specific message
 * builder that expects `new<Entity>Url` / `new<Entity>Number` keys.
 *
 * @param {string} entityKey - PascalCase entity key (e.g. "Issue", "PullRequest", "Discussion")
 * @param {(ctx: any) => string} getMessage - Entity-specific message builder
 * @returns {(params: {newEntityUrl: string, newEntityNumber: number, workflowName: string, runUrl: string}) => string}
 */
function adaptCloseMessage(entityKey, getMessage) {
  return params =>
    getMessage({
      [`new${entityKey}Url`]: params.newEntityUrl,
      [`new${entityKey}Number`]: params.newEntityNumber,
      workflowName: params.workflowName,
      runUrl: params.runUrl,
    });
}

/**
 * Create an entity-specific `closeOlder*` handler around the shared
 * {@link closeOlderEntities} orchestrator.
 *
 * The returned handler wires the entity configuration into the orchestrator and
 * maps the generic result shape back to the entity-specific one (`html_url` for
 * REST entities, `url` for GraphQL entities).
 *
 * @param {object} config
 * @param {string} config.entityType - Entity type name for logging (e.g. "issue")
 * @param {string} config.entityTypePlural - Plural form (e.g. "issues")
 * @param {string} config.entityKey - PascalCase entity key used by the message builder (e.g. "Issue")
 * @param {"url"|"html_url"} config.urlKey - Result key holding the entity URL
 * @param {(ctx: any) => string} config.getMessage - Entity-specific close message builder
 * @param {(github: any, owner: string, repo: string, entityId: any, message: string) => Promise<any>} config.addComment - Comment helper
 * @param {(github: any, owner: string, repo: string, entityId: any) => Promise<any>} config.closeEntity - Close helper
 * @param {number} config.delayMs - Delay between API operations in milliseconds
 * @param {(entity: any) => any} [config.getEntityId] - Extract the entity ID used for API calls (defaults to `entity.number`)
 * @returns {(github: any, owner: string, repo: string, workflowId: string, newEntity: any, workflowName: string, runUrl: string, searchOlderEntities: (...args: any[]) => Promise<Array<any>>, ...extraArgs: any[]) => Promise<Array<any>>}
 */
function createCloseOlderHandler({ entityType, entityTypePlural, entityKey, urlKey, getMessage, addComment, closeEntity, delayMs, getEntityId }) {
  return async function closeOlder(github, owner, repo, workflowId, newEntity, workflowName, runUrl, searchOlderEntities, ...extraArgs) {
    const result = await closeOlderEntities(
      github,
      owner,
      repo,
      workflowId,
      newEntity,
      workflowName,
      runUrl,
      {
        entityType,
        entityTypePlural,
        searchOlderEntities,
        getCloseMessage: adaptCloseMessage(entityKey, getMessage),
        addComment,
        closeEntity,
        delayMs,
        getEntityId: getEntityId || (entity => entity.number),
        getEntityUrl: entity => entity[urlKey],
      },
      ...extraArgs
    );

    // Map to the entity-specific return type
    return result.map(item => ({
      number: item.number,
      [urlKey]: item[urlKey] || "",
    }));
  };
}

/**
 * Shared REST search for issue-like entities (issues and pull requests). Both
 * use the same `search.issuesAndPullRequests` endpoint and result shape; only
 * the qualifier and the cross-entity exclusion differ.
 *
 * @param {object} params
 * @param {any} params.github - GitHub REST API instance
 * @param {string} params.owner - Repository owner
 * @param {string} params.repo - Repository name
 * @param {string} params.workflowId - Workflow ID to match in the marker
 * @param {number} params.excludeNumber - Entity number to exclude (the newly created one)
 * @param {boolean} params.isPullRequest - Whether pull requests (true) or issues (false) are searched
 * @param {string} [params.callerWorkflowId] - Optional calling workflow identity
 * @param {string} [params.closeOlderKey] - Optional explicit deduplication key
 * @param {Set<number>} [params.additionalExcludeNumbers] - Optional additional entity numbers to exclude
 * @returns {Promise<Array<{number: number, title: string, html_url: string, labels: Array<{name: string}>, created_at: string}>>} Matching entities
 */
async function searchOlderIssueLikeEntities({ github, owner, repo, workflowId, excludeNumber, isPullRequest, callerWorkflowId, closeOlderKey, additionalExcludeNumbers }) {
  const counterKey = isPullRequest ? "issueCount" : "pullRequestCount";
  const counterLabel = isPullRequest ? "Excluded issues" : "Excluded pull requests";

  return searchOlderEntitiesByMarker({
    owner,
    repo,
    workflowId,
    excludeNumber,
    entityType: isPullRequest ? "pull request" : "issue",
    callerWorkflowId,
    closeOlderKey,
    additionalExcludeNumbers,
    entityQualifier: isPullRequest ? "is:pr" : "is:issue",
    executeSearch: searchQuery =>
      github.rest.search.issuesAndPullRequests({
        q: searchQuery,
        per_page: 50,
      }),
    getItems: result => result?.data?.items,
    mapItem: item => ({
      number: item.number,
      title: item.title,
      html_url: item.html_url,
      labels: item.labels || [],
      created_at: item.created_at,
    }),
    additionalFilter: (item, extra) => {
      if (Boolean(item.pull_request) !== isPullRequest) {
        extra[counterKey] = (extra[counterKey] || 0) + 1;
        return false;
      }
      return true;
    },
    extraLabels: [[counterKey, counterLabel]],
  });
}

module.exports = {
  adaptCloseMessage,
  createCloseOlderHandler,
  searchOlderIssueLikeEntities,
};
