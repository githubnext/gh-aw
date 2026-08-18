// @ts-check
/// <reference types="@actions/github-script" />

const { closeOlderEntities } = require("./close_older_entities.cjs");

/**
 * Create a search adapter that places excludeNumber between leading and trailing args.
 * @param {(github: any, owner: string, repo: string, workflowId: string, ...args: any[]) => Promise<Array<any>>} searchFn - Search function
 * @param {Array<any>} [leadingArgs] - Arguments to place before excludeNumber
 * @param {Array<any>} [trailingArgs] - Arguments to place after excludeNumber
 * @returns {(github: any, owner: string, repo: string, workflowId: string, excludeNumber: number) => Promise<Array<any>>}
 */
function createCloseOlderSearchAdapter(searchFn, leadingArgs = [], trailingArgs = []) {
  return (github, owner, repo, workflowId, excludeNumber) => searchFn(github, owner, repo, workflowId, ...leadingArgs, excludeNumber, ...trailingArgs);
}

/**
 * Run close-older flow using an entity descriptor.
 * @param {object} params - Close-older parameters
 * @param {any} params.github - GitHub API instance
 * @param {string} params.owner - Repository owner
 * @param {string} params.repo - Repository name
 * @param {string} params.workflowId - Workflow ID marker
 * @param {any} params.newEntity - Newly created entity
 * @param {string} params.workflowName - Workflow name
 * @param {string} params.runUrl - Workflow run URL
 * @param {string} params.entityType - Entity type name
 * @param {string} params.entityTypePlural - Entity type plural name
 * @param {(params: any) => string} params.getCloseMessage - Closing message renderer
 * @param {(github: any, owner: string, repo: string, workflowId: string, excludeNumber: number) => Promise<Array<any>>} params.searchOlderEntities - Search function
 * @param {(github: any, owner: string, repo: string, entityId: any, message: string) => Promise<any>} params.addComment - Add comment function
 * @param {(github: any, owner: string, repo: string, entityId: any) => Promise<any>} params.closeEntity - Close entity function
 * @param {number} params.delayMs - Delay between API calls
 * @param {(entity: any) => any} params.getEntityId - Entity identifier selector
 * @param {(entity: any) => string} params.getEntityUrl - Entity URL selector
 * @param {(item: {number: number, url?: string, html_url?: string}) => any} params.mapClosedEntity - Closed-entity return mapper
 * @returns {Promise<Array<any>>} List of mapped closed entities
 */
async function closeOlderWithDescriptor({
  github,
  owner,
  repo,
  workflowId,
  newEntity,
  workflowName,
  runUrl,
  entityType,
  entityTypePlural,
  getCloseMessage,
  searchOlderEntities,
  addComment,
  closeEntity,
  delayMs,
  getEntityId,
  getEntityUrl,
  mapClosedEntity,
}) {
  const result = await closeOlderEntities(github, owner, repo, workflowId, newEntity, workflowName, runUrl, {
    entityType,
    entityTypePlural,
    searchOlderEntities,
    getCloseMessage,
    addComment,
    closeEntity,
    delayMs,
    getEntityId,
    getEntityUrl,
  });

  return result.map(mapClosedEntity);
}

module.exports = {
  createCloseOlderSearchAdapter,
  closeOlderWithDescriptor,
};
