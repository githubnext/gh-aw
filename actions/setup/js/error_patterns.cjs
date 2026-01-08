// @ts-check
/**
 * Error patterns for extracting error information from agentic workflow logs.
 * These patterns are used across all engines to detect errors, warnings, and notices.
 *
 * Each pattern includes:
 * - id: Unique identifier for the pattern
 * - pattern: Regular expression string to match log lines
 * - level_group: Capture group index (1-based) containing the error level (0 = infer from context)
 * - message_group: Capture group index (1-based) containing the error message (0 = use entire match)
 * - description: Human-readable description of what this pattern matches
 * - severity: Optional explicit severity level ("error", "warning", or empty = inferred)
 */

/**
 * Common error patterns that apply to all engines.
 * These detect standard GitHub Actions workflow commands and universal error formats.
 * @returns {Array<Object>} Array of error pattern objects
 */
function getCommonErrorPatterns() {
  return [
    // GitHub Actions workflow commands - standard error/warning/notice syntax
    {
      id: "common-gh-actions-error",
      pattern: "::(error)(?:\\s+[^:]*)?::(.+)",
      level_group: 1, // "error" is in the first capture group
      message_group: 2, // message is in the second capture group
      description: "GitHub Actions workflow command - error",
    },
    {
      id: "common-gh-actions-warning",
      pattern: "::(warning)(?:\\s+[^:]*)?::(.+)",
      level_group: 1, // "warning" is in the first capture group
      message_group: 2, // message is in the second capture group
      description: "GitHub Actions workflow command - warning",
    },
    {
      id: "common-gh-actions-notice",
      pattern: "::(notice)(?:\\s+[^:]*)?::(.+)",
      level_group: 1, // "notice" is in the first capture group
      message_group: 2, // message is in the second capture group
      description: "GitHub Actions workflow command - notice",
    },
    // Generic error/warning patterns - common log formats
    {
      id: "common-generic-error",
      pattern: "(ERROR|Error):\\s+(.+)",
      level_group: 1, // "ERROR" or "Error" is in the first capture group
      message_group: 2, // error message is in the second capture group
      description: "Generic ERROR messages",
    },
    {
      id: "common-generic-warning",
      pattern: "(WARNING|Warning):\\s+(.+)",
      level_group: 1, // "WARNING" or "Warning" is in the first capture group
      message_group: 2, // warning message is in the second capture group
      description: "Generic WARNING messages",
    },
  ];
}

/**
 * Copilot-specific error patterns for timestamped logs, command failures,
 * module errors, and permission-related issues.
 * @returns {Array<Object>} Array of Copilot-specific error pattern objects
 */
function getCopilotErrorPatterns() {
  return [
    {
      id: "copilot-timestamp-warning",
      pattern: "(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\s+\\[(WARN|WARNING)\\]\\s+(.+)",
      level_group: 2, // "WARN" or "WARNING" is in the second capture group
      message_group: 3, // warning message is in the third capture group
      description: "Copilot CLI timestamped WARNING messages",
    },
    {
      id: "copilot-bracketed-critical-error",
      pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\]\\s+(CRITICAL|ERROR):\\s+(.+)",
      level_group: 2, // "CRITICAL" or "ERROR" is in the second capture group
      message_group: 3, // error message is in the third capture group
      description: "Copilot CLI bracketed critical/error messages with timestamp",
    },
    {
      id: "copilot-bracketed-warning",
      pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\]\\s+(WARNING):\\s+(.+)",
      level_group: 2, // "WARNING" is in the second capture group
      message_group: 3, // warning message is in the third capture group
      description: "Copilot CLI bracketed warning messages with timestamp",
    },
    // Copilot CLI-specific error indicators without "ERROR:" prefix
    {
      id: "copilot-failed-command",
      pattern: "✗\\s+(.+)",
      level_group: 0,
      message_group: 1,
      description: "Copilot CLI failed command indicator",
    },
    {
      id: "copilot-command-not-found",
      pattern: "(?:command not found|not found):\\s*(.+)|(.+):\\s*(?:command not found|not found)",
      level_group: 0,
      message_group: 0,
      description: "Shell command not found error",
    },
    {
      id: "copilot-module-not-found",
      pattern: "Cannot find module\\s+['\"](.+)['\"]",
      level_group: 0,
      message_group: 1,
      description: "Node.js module not found error",
    },
    {
      id: "copilot-permission-denied-no-user",
      pattern: "Permission denied and could not request permission from user",
      level_group: 0,
      message_group: 0,
      severity: "warning",
      description: "Copilot CLI permission denied warning (user interaction required)",
    },
    // Permission-related patterns (classified as warnings, not errors)
    {
      id: "copilot-permission-denied",
      pattern: "(?i)\\berror\\b.*permission.*denied",
      level_group: 0,
      message_group: 0,
      severity: "warning",
      description: "Permission denied error (requires error context)",
    },
    {
      id: "copilot-unauthorized",
      pattern: "(?i)\\berror\\b.*unauthorized",
      level_group: 0,
      message_group: 0,
      severity: "warning",
      description: "Unauthorized access error (requires error context)",
    },
    {
      id: "copilot-forbidden",
      pattern: "(?i)\\berror\\b.*forbidden",
      level_group: 0,
      message_group: 0,
      severity: "warning",
      description: "Forbidden access error (requires error context)",
    },
  ];
}

/**
 * Codex-specific error patterns for Rust log format with timestamps.
 * @returns {Array<Object>} Array of Codex-specific error pattern objects
 */
function getCodexErrorPatterns() {
  return [
    // Rust format patterns (without brackets, with milliseconds and Z timezone)
    {
      id: "codex-rust-error",
      pattern: "(\\d{4}-\\d{2}-\\d{2}T[\\d:.]+Z)\\s+(ERROR)\\s+(.+)",
      level_group: 2, // "ERROR" is in the second capture group
      message_group: 3, // error message is in the third capture group
      description: "Codex ERROR messages with timestamp",
    },
    {
      id: "codex-rust-warning",
      pattern: "(\\d{4}-\\d{2}-\\d{2}T[\\d:.]+Z)\\s+(WARN|WARNING)\\s+(.+)",
      level_group: 2, // "WARN" or "WARNING" is in the second capture group
      message_group: 3, // warning message is in the third capture group
      description: "Codex warning messages with timestamp",
    },
  ];
}

/**
 * Claude-specific error patterns.
 * Claude uses common GitHub Actions workflow commands for error reporting.
 * @returns {Array<Object>} Array of Claude-specific error pattern objects (empty - uses common patterns only)
 */
function getClaudeErrorPatterns() {
  // Claude uses common GitHub Actions workflow commands for error reporting
  // No engine-specific log formats to parse
  return [];
}

/**
 * Get all error patterns for a specific engine.
 * @param {string} engineId - The engine identifier ("copilot", "codex", "claude", or "custom")
 * @returns {Array<Object>} Array of error pattern objects for the specified engine
 */
function getErrorPatternsForEngine(engineId) {
  const common = getCommonErrorPatterns();

  switch (engineId) {
    case "copilot":
      return [...common, ...getCopilotErrorPatterns()];
    case "codex":
      return [...common, ...getCodexErrorPatterns()];
    case "claude":
      return [...common, ...getClaudeErrorPatterns()];
    case "custom":
      // Custom engine uses common patterns by default
      // Additional patterns can be provided via environment variable
      return common;
    default:
      // Unknown engine - return common patterns
      return common;
  }
}

/**
 * Load custom error patterns from environment variable.
 * This allows custom engines to define their own error patterns.
 * @returns {Array<Object>} Array of custom error pattern objects from environment variable
 */
function getCustomErrorPatternsFromEnv() {
  const customPatternsEnv = process.env.GH_AW_CUSTOM_ERROR_PATTERNS;
  if (!customPatternsEnv) {
    return [];
  }

  try {
    const patterns = JSON.parse(customPatternsEnv);
    if (!Array.isArray(patterns)) {
      console.error("GH_AW_CUSTOM_ERROR_PATTERNS must be a JSON array");
      return [];
    }
    return patterns;
  } catch (e) {
    console.error(`Failed to parse GH_AW_CUSTOM_ERROR_PATTERNS as JSON: ${e instanceof Error ? e.message : String(e)}`);
    return [];
  }
}

module.exports = {
  getCommonErrorPatterns,
  getCopilotErrorPatterns,
  getCodexErrorPatterns,
  getClaudeErrorPatterns,
  getErrorPatternsForEngine,
  getCustomErrorPatternsFromEnv,
};
