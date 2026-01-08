// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Missing Info Formatter Module
 *
 * This module provides functions to format missing_tool and missing_data
 * messages into HTML details sections for inclusion in safe output footers.
 */

/**
 * Escape markdown content to prevent injection
 * @param {string} text - Text to escape
 * @returns {string} Escaped text
 */
function escapeMarkdown(text) {
  if (!text) return "";
  return text.replace(/\\/g, "\\\\").replace(/`/g, "\\`").replace(/\*/g, "\\*").replace(/_/g, "\\_").replace(/\[/g, "\\[").replace(/\]/g, "\\]").replace(/\(/g, "\\(").replace(/\)/g, "\\)").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

/**
 * Format missing_tool messages into markdown list items
 * @param {Array<{tool: string, reason: string, alternatives?: string}>} missingTools - Missing tool messages
 * @returns {string} Formatted markdown list
 */
function formatMissingTools(missingTools) {
  if (!missingTools || missingTools.length === 0) return "";

  const items = missingTools.map(item => {
    let line = `- **${escapeMarkdown(item.tool)}**: ${escapeMarkdown(item.reason)}`;
    if (item.alternatives) {
      line += `\n  - *Alternatives*: ${escapeMarkdown(item.alternatives)}`;
    }
    return line;
  });

  return items.join("\n");
}

/**
 * Format missing_data messages into markdown list items
 * @param {Array<{data_type: string, reason: string, context?: string, alternatives?: string}>} missingData - Missing data messages
 * @returns {string} Formatted markdown list
 */
function formatMissingData(missingData) {
  if (!missingData || missingData.length === 0) return "";

  const items = missingData.map(item => {
    let line = `- **${escapeMarkdown(item.data_type)}**: ${escapeMarkdown(item.reason)}`;
    if (item.context) {
      line += `\n  - *Context*: ${escapeMarkdown(item.context)}`;
    }
    if (item.alternatives) {
      line += `\n  - *Alternatives*: ${escapeMarkdown(item.alternatives)}`;
    }
    return line;
  });

  return items.join("\n");
}

/**
 * Format noop messages into markdown list items
 * @param {Array<{message: string}>} noopMessages - Noop messages
 * @returns {string} Formatted markdown list
 */
function formatNoopMessages(noopMessages) {
  if (!noopMessages || noopMessages.length === 0) return "";

  const items = noopMessages.map(item => {
    return `- ${escapeMarkdown(item.message)}`;
  });

  return items.join("\n");
}

/**
 * Generate HTML details section for missing tools
 * @param {Array<{tool: string, reason: string, alternatives?: string}>} missingTools - Missing tool messages
 * @returns {string} HTML details section or empty string
 */
function generateMissingToolsSection(missingTools) {
  if (!missingTools || missingTools.length === 0) return "";

  const content = formatMissingTools(missingTools);
  return `\n\n<details>\n<summary><b>Missing Tools</b></summary>\n\n${content}\n\n</details>`;
}

/**
 * Generate HTML details section for missing data
 * @param {Array<{data_type: string, reason: string, context?: string, alternatives?: string}>} missingData - Missing data messages
 * @returns {string} HTML details section or empty string
 */
function generateMissingDataSection(missingData) {
  if (!missingData || missingData.length === 0) return "";

  const content = formatMissingData(missingData);
  return `\n\n<details>\n<summary><b>Missing Data</b></summary>\n\n${content}\n\n</details>`;
}

/**
 * Generate HTML details section for noop messages
 * @param {Array<{message: string}>} noopMessages - Noop messages
 * @returns {string} HTML details section or empty string
 */
function generateNoopMessagesSection(noopMessages) {
  if (!noopMessages || noopMessages.length === 0) return "";

  const content = formatNoopMessages(noopMessages);
  return `\n\n<details>\n<summary><b>No-Op Messages</b></summary>\n\n${content}\n\n</details>`;
}

/**
 * Generate complete missing information sections for both tools and data
 * @param {{missingTools?: Array<any>, missingData?: Array<any>, noopMessages?: Array<any>}} missings - Object containing missing tools, data, and noop messages
 * @returns {string} Combined HTML details sections
 */
function generateMissingInfoSections(missings) {
  if (!missings) return "";

  let sections = "";

  if (missings.missingTools && missings.missingTools.length > 0) {
    sections += generateMissingToolsSection(missings.missingTools);
  }

  if (missings.missingData && missings.missingData.length > 0) {
    sections += generateMissingDataSection(missings.missingData);
  }

  if (missings.noopMessages && missings.noopMessages.length > 0) {
    sections += generateNoopMessagesSection(missings.noopMessages);
  }

  return sections;
}

module.exports = {
  escapeMarkdown,
  formatMissingTools,
  formatMissingData,
  formatNoopMessages,
  generateMissingToolsSection,
  generateMissingDataSection,
  generateNoopMessagesSection,
  generateMissingInfoSections,
};
