// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 * @typedef {import('./types/handler-factory').MessageHandlerFunction} MessageHandlerFunction
 * @typedef {import('./types/handler-factory').HandlerResult} HandlerResult
 */

/**
 * @typedef {Object} CountGatedHandlerConfig
 * @property {string} handlerType - Handler type name used in log/warning messages (e.g., "add_labels")
 * @property {(config: Object, maxCount: number) => Promise<MessageHandlerFunction>} setup
 *   Setup function that receives the handler config and resolved maxCount,
 *   performs handler-specific initialization (GitHub client, logging, etc.),
 *   and returns the message handler function for processing individual items.
 */

/**
 * Creates a handler factory with shared max-count gating and processedCount state.
 *
 * This eliminates the duplicated scaffolding across safe-output handlers:
 * - Extracts maxCount from config (default 10)
 * - Maintains processedCount in closure
 * - Gates each call with the max-count check
 * - Returns standardized error envelope when limit is reached
 *
 * @param {CountGatedHandlerConfig} options - Handler configuration
 * @returns {HandlerFactoryFunction} Handler factory function compatible with the handler manager
 */
function createCountGatedHandler({ handlerType, setup }) {
  return async function main(config = {}) {
    const maxCount = config.max ?? 10;
    let processedCount = 0;

    const handleItem = await setup(config, maxCount);

    /** @type {MessageHandlerFunction} */
    return async function handler(message, resolvedTemporaryIds) {
      if (processedCount >= maxCount) {
        core.warning(`Skipping ${handlerType}: max count of ${maxCount} reached`);
        return {
          success: false,
          error: `Max count of ${maxCount} reached`,
        };
      }

      processedCount++;
      return handleItem(message, resolvedTemporaryIds);
    };
  };
}

module.exports = { createCountGatedHandler };
