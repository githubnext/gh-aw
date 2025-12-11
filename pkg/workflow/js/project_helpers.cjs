/**
 * Helper functions for GitHub Projects v2 operations
 */

/**
 * Generate a campaign ID from project name
 * @param {string} projectName - The project/campaign name
 * @returns {string} Campaign ID in format: slug-timestamp (e.g., "perf-q1-2025-a3f2b4c8")
 */
function generateCampaignId(projectName) {
  // Create slug from project name
  const slug = projectName
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .substring(0, 30);

  // Add short timestamp hash for uniqueness
  const timestamp = Date.now().toString(36).substring(0, 8);

  return `${slug}-${timestamp}`;
}

/**
 * Normalize project name by trimming whitespace
 * @param {string} projectName - The project name to normalize
 * @returns {string} Normalized project name
 */
function normalizeProjectName(projectName) {
  if (!projectName || typeof projectName !== "string") {
    throw new Error(
      `Invalid project name: expected string, got ${typeof projectName}. The "project" field is required and must be a project title.`
    );
  }
  return projectName.trim();
}

// Export for testing and use in other modules
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    generateCampaignId,
    normalizeProjectName,
  };
}
