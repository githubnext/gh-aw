// @ts-check
/// <reference types="@actions/github-script" />

const { closeOlderEntities } = require("./close_older_entities.cjs");

/**
 * Close older issues/PRs using shared adapter wiring.
 * @param {object} params
 * @param {any} params.github
 * @param {string} params.owner
 * @param {string} params.repo
 * @param {string} params.workflowId
 * @param {{number: number, html_url: string}} params.newEntity
 * @param {string} params.workflowName
 * @param {string} params.runUrl
 * @param {string | undefined} params.callerWorkflowId
 * @param {string | undefined} params.closeOlderKey
 * @param {string} params.entityType
 * @param {string} params.entityTypePlural
 * @param {(github: any, owner: string, repo: string, workflowId: string, excludeNumber: number, callerWorkflowId?: string, closeOlderKey?: string) => Promise<Array<{number: number, html_url?: string}>>} params.searchOlderEntities
 * @param {(params: any) => string} params.getCloseMessage
 * @param {(github: any, owner: string, repo: string, entityId: number, message: string) => Promise<any>} params.addComment
 * @param {(github: any, owner: string, repo: string, entityId: number) => Promise<any>} params.closeEntity
 * @param {number} params.delayMs
 * @param {(params: {newEntityUrl: string, newEntityNumber: number, workflowName: string, runUrl: string}) => any} params.messageParams
 * @returns {Promise<Array<{number: number, html_url: string}>>}
 */
async function closeOlderWithAdapter(params) {
  const result = await closeOlderEntities(params.github, params.owner, params.repo, params.workflowId, params.newEntity, params.workflowName, params.runUrl, {
    entityType: params.entityType,
    entityTypePlural: params.entityTypePlural,
    // Use a closure so callerWorkflowId and closeOlderKey are forwarded to searchOlderEntities
    // without going through the closeOlderEntities extraArgs mechanism (which appends
    // excludeNumber last)
    searchOlderEntities: (gh, o, r, wid, excludeNumber) => params.searchOlderEntities(gh, o, r, wid, excludeNumber, params.callerWorkflowId, params.closeOlderKey),
    getCloseMessage: closeParams => params.getCloseMessage(params.messageParams(closeParams)),
    addComment: params.addComment,
    closeEntity: params.closeEntity,
    delayMs: params.delayMs,
    getEntityId: entity => entity.number,
    getEntityUrl: entity => entity.html_url,
  });

  return result.map(item => ({
    number: item.number,
    html_url: item.html_url || "",
  }));
}

module.exports = {
  closeOlderWithAdapter,
};
