// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Output Handler Manager
 *
 * This module provides a registry and dispatcher for safe output handlers.
 * Each handler can register itself for a specific message type, and the manager
 * will dispatch messages to the appropriate handler, collecting temporary ID
 * mappings along the way.
 */

/**
 * @typedef {Object} HandlerContext
 * @property {typeof import("@actions/core")} core - GitHub Actions core
 * @property {typeof import("@actions/github")} github - GitHub Actions toolkit
 * @property {ReturnType<typeof import("@actions/github").getOctokit>} github - Octokit instance
 * @property {typeof import("@actions/github").context} context - GitHub Actions context
 * @property {typeof import("@actions/exec")} exec - GitHub Actions exec
 * @property {Map<string, {repo: string, number: number}>} temporaryIdMap - Map of temporary IDs to issue/PR numbers
 */

/**
 * @typedef {Object} HandlerResult
 * @property {boolean} success - Whether the handler succeeded
 * @property {Map<string, {repo: string, number: number}>} [temporaryIds] - Temporary ID mappings created by this handler
 * @property {string} [error] - Error message if handler failed
 * @property {any} [data] - Additional data returned by the handler
 */

/**
 * @callback HandlerFunction
 * @param {any} item - The safe output item to process
 * @param {HandlerContext} context - Handler context with GitHub APIs and temporary ID map
 * @returns {Promise<HandlerResult>} Result of handler execution
 */

class SafeOutputHandlerManager {
  constructor() {
    /** @type {Map<string, HandlerFunction>} */
    this.handlers = new Map();
  }

  /**
   * Register a handler for a specific message type
   * @param {string} type - Message type (e.g., "create_issue", "create_pull_request")
   * @param {HandlerFunction} handler - Handler function
   */
  registerHandler(type, handler) {
    if (this.handlers.has(type)) {
      throw new Error(`Handler already registered for type: ${type}`);
    }
    this.handlers.set(type, handler);
  }

  /**
   * Process all items from agent output
   * @param {any[]} items - Array of safe output items
   * @param {HandlerContext} context - Handler context
   * @returns {Promise<{success: boolean, temporaryIdMap: Map<string, {repo: string, number: number}>, errors: string[]}>}
   */
  async processAll(items, context) {
    const errors = [];

    for (let i = 0; i < items.length; i++) {
      const item = items[i];

      if (!item.type) {
        context.core.warning(`Item ${i} has no type, skipping`);
        continue;
      }

      const handler = this.handlers.get(item.type);
      if (!handler) {
        context.core.info(`No handler registered for type: ${item.type}, skipping`);
        continue;
      }

      try {
        context.core.info(`Processing ${item.type} (${i + 1}/${items.length})`);
        const result = await handler(item, context);

        if (!result.success) {
          const error = `Handler for ${item.type} failed: ${result.error || "Unknown error"}`;
          context.core.error(error);
          errors.push(error);
        } else {
          // Merge temporary IDs from this handler into the global map
          if (result.temporaryIds && result.temporaryIds.size > 0) {
            for (const [tempId, value] of result.temporaryIds.entries()) {
              context.temporaryIdMap.set(tempId, value);
              context.core.info(`Registered temporary ID: ${tempId} -> ${value.repo}#${value.number}`);
            }
          }
        }
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        context.core.error(`Exception in handler for ${item.type}: ${errorMessage}`);
        errors.push(`Exception in ${item.type}: ${errorMessage}`);
      }
    }

    const success = errors.length === 0;
    return {
      success,
      temporaryIdMap: context.temporaryIdMap,
      errors,
    };
  }

  /**
   * Get list of registered handler types
   * @returns {string[]}
   */
  getRegisteredTypes() {
    return Array.from(this.handlers.keys());
  }
}

module.exports = { SafeOutputHandlerManager };
