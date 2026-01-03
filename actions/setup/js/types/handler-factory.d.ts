// TypeScript definitions for Safe Output Handler Factory Pattern
// This file provides type definitions for the handler factory functions

/**
 * Configuration object passed to handler main() function
 * Contains handler-specific configuration from GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
 */
interface HandlerConfig {
  /** Maximum number of items this handler should process */
  max?: number;
  /** Additional handler-specific configuration properties */
  [key: string]: any;
}

/**
 * Map of resolved temporary IDs to their actual issue/PR/discussion references
 */
interface ResolvedTemporaryIds {
  [temporaryId: string]: {
    /** Repository in format "owner/repo" */
    repo: string;
    /** Issue, PR, or discussion number */
    number: number;
  };
}

/**
 * Result object returned by message handler function when successful
 */
interface HandlerSuccessResult {
  /** Indicates the operation was successful */
  success: true;
  /** Additional result properties (number, url, temporaryId, etc.) */
  [key: string]: any;
}

/**
 * Result object returned by message handler function when failed
 */
interface HandlerErrorResult {
  /** Indicates the operation failed */
  success: false;
  /** Error message describing what went wrong */
  error: string;
  /** Additional result properties (skipped, etc.) */
  [key: string]: any;
}

/**
 * Result object returned by message handler function
 */
type HandlerResult = HandlerSuccessResult | HandlerErrorResult;

/**
 * Message handler function returned by the main() factory function
 * Processes a single safe output message
 *
 * @param message - The safe output message to process
 * @param resolvedTemporaryIds - Map of temporary IDs that have been resolved to actual issue/PR/discussion numbers
 * @returns Promise resolving to result with success status and details
 */
type MessageHandlerFunction = (message: any, resolvedTemporaryIds: ResolvedTemporaryIds) => Promise<HandlerResult>;

/**
 * Main factory function signature for safe output handlers
 * Creates and returns a message handler function configured with the provided config
 *
 * @param config - Handler configuration object
 * @returns Promise resolving to a message handler function
 */
type HandlerFactoryFunction = (config?: HandlerConfig) => Promise<MessageHandlerFunction>;

export { HandlerConfig, ResolvedTemporaryIds, HandlerSuccessResult, HandlerErrorResult, HandlerResult, MessageHandlerFunction, HandlerFactoryFunction };
