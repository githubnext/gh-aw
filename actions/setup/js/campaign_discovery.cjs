// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Campaign Discovery Precomputation
 *
 * Discovers campaign items (worker-created issues/PRs/discussions) by scanning
 * a predefined list of repos using tracker-id markers and/or tracker labels.
 *
 * This script runs deterministically before the agent, eliminating the need for
 * agents to perform GitHub-wide discovery during Phase 1.
 *
 * Outputs:
 * - Manifest file: ./.gh-aw/campaign.discovery.json
 * - Cursor file: in repo-memory for continuation across runs
 *
 * Features:
 * - Strict pagination budgets
 * - Durable cursor for incremental discovery
 * - Stable sorting for deterministic output
 * - Discovery via tracker-id and/or tracker-label
 */

const fs = require("fs");
const path = require("path");

/**
 * Manifest schema version
 */
const MANIFEST_VERSION = "v1";

/**
 * Default discovery budgets
 */
const DEFAULT_MAX_ITEMS = 100;
const DEFAULT_MAX_PAGES = 10;

/**
 * Parse cursor from repo-memory
 * @param {string} cursorPath - Path to cursor file in repo-memory
 * @returns {any} Parsed cursor object or null
 */
function loadCursor(cursorPath) {
  try {
    if (fs.existsSync(cursorPath)) {
      const content = fs.readFileSync(cursorPath, "utf8");
      const cursor = JSON.parse(content);
      core.info(`Loaded cursor from ${cursorPath}`);
      return cursor;
    }
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    core.warning(`Failed to load cursor from ${cursorPath}: ${err.message}`);
  }
  return null;
}

/**
 * Save cursor to repo-memory
 * @param {string} cursorPath - Path to cursor file in repo-memory
 * @param {any} cursor - Cursor object to save
 */
function saveCursor(cursorPath, cursor) {
  try {
    const dir = path.dirname(cursorPath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
    fs.writeFileSync(cursorPath, JSON.stringify(cursor, null, 2));
    core.info(`Saved cursor to ${cursorPath}`);
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    core.error(`Failed to save cursor to ${cursorPath}: ${err.message}`);
    throw err;
  }
}

/**
 * Normalize a discovered item to standard format
 * @param {any} item - Raw GitHub item (issue, PR, or discussion)
 * @param {string} contentType - Type: "issue", "pull_request", or "discussion"
 * @returns {any} Normalized item
 */
function normalizeItem(item, contentType) {
  const normalized = {
    url: item.html_url || item.url,
    content_type: contentType,
    number: item.number,
    repo: item.repository?.full_name || item.repo?.full_name || "",
    created_at: item.created_at,
    updated_at: item.updated_at,
    state: item.state,
    title: item.title,
  };

  // Add closed/merged dates
  if (item.closed_at) {
    normalized.closed_at = item.closed_at;
  }
  if (item.merged_at) {
    normalized.merged_at = item.merged_at;
  }

  return normalized;
}

/**
 * Search for items by tracker-id across issues and PRs
 * @param {any} octokit - GitHub API client
 * @param {string} trackerId - Tracker ID to search for
 * @param {string[]} repos - List of repositories to search (owner/repo format)
 * @param {number} maxItems - Maximum items to discover
 * @param {number} maxPages - Maximum pages to fetch
 * @param {any} cursor - Cursor for pagination
 * @returns {Promise<{items: any[], cursor: any, itemsScanned: number, pagesScanned: number}>}
 */
async function searchByTrackerId(octokit, trackerId, repos, maxItems, maxPages, cursor) {
  const items = [];
  let itemsScanned = 0;
  let pagesScanned = 0;

  core.info(`Searching for tracker-id: ${trackerId}`);

  // Search in issues and PRs
  // Format: "tracker-id: workflow-name" appears in issue/PR body or comments
  const searchQuery = `"tracker-id: ${trackerId}" type:issue`;

  try {
    let page = cursor?.page || 1;

    while (pagesScanned < maxPages && itemsScanned < maxItems) {
      core.info(`Fetching page ${page} for tracker-id: ${trackerId}`);

      const response = await octokit.rest.search.issuesAndPullRequests({
        q: searchQuery,
        per_page: 100,
        page: page,
        sort: "updated",
        order: "asc", // Stable ordering
      });

      pagesScanned++;

      if (response.data.items.length === 0) {
        core.info(`No more items found for tracker-id: ${trackerId}`);
        break;
      }

      for (const item of response.data.items) {
        if (itemsScanned >= maxItems) {
          break;
        }

        itemsScanned++;

        // Determine if it's a PR or issue
        const contentType = item.pull_request ? "pull_request" : "issue";
        const normalized = normalizeItem(item, contentType);
        items.push(normalized);
      }

      // Check if there are more pages
      if (response.data.items.length < 100) {
        break; // Last page
      }

      page++;
    }

    return {
      items,
      cursor: { page, trackerId },
      itemsScanned,
      pagesScanned,
    };
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    core.error(`Error searching by tracker-id: ${err.message}`);
    throw err;
  }
}

/**
 * Search for items by tracker label
 * @param {any} octokit - GitHub API client
 * @param {string} label - Label to search for
 * @param {string[]} repos - List of repositories to search (owner/repo format)
 * @param {number} maxItems - Maximum items to discover
 * @param {number} maxPages - Maximum pages to fetch
 * @param {any} cursor - Cursor for pagination
 * @returns {Promise<{items: any[], cursor: any, itemsScanned: number, pagesScanned: number}>}
 */
async function searchByLabel(octokit, label, repos, maxItems, maxPages, cursor) {
  const items = [];
  let itemsScanned = 0;
  let pagesScanned = 0;

  core.info(`Searching for label: ${label}`);

  // Build search query for label across all repos
  let searchQuery = `label:"${label}"`;

  // If specific repos are provided, add them to the query
  if (repos && repos.length > 0) {
    // GitHub search supports up to ~1024 characters in query
    // For simplicity, we'll search across all repos if the list is large
    if (repos.length <= 5) {
      const repoQuery = repos.map(r => `repo:${r}`).join(" ");
      searchQuery = `${searchQuery} ${repoQuery}`;
    }
  }

  try {
    let page = cursor?.page || 1;

    while (pagesScanned < maxPages && itemsScanned < maxItems) {
      core.info(`Fetching page ${page} for label: ${label}`);

      const response = await octokit.rest.search.issuesAndPullRequests({
        q: searchQuery,
        per_page: 100,
        page: page,
        sort: "updated",
        order: "asc", // Stable ordering
      });

      pagesScanned++;

      if (response.data.items.length === 0) {
        core.info(`No more items found for label: ${label}`);
        break;
      }

      for (const item of response.data.items) {
        if (itemsScanned >= maxItems) {
          break;
        }

        itemsScanned++;

        // Determine if it's a PR or issue
        const contentType = item.pull_request ? "pull_request" : "issue";
        const normalized = normalizeItem(item, contentType);
        items.push(normalized);
      }

      // Check if there are more pages
      if (response.data.items.length < 100) {
        break; // Last page
      }

      page++;
    }

    return {
      items,
      cursor: { page, label },
      itemsScanned,
      pagesScanned,
    };
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    core.error(`Error searching by label: ${err.message}`);
    throw err;
  }
}

/**
 * Main discovery function
 * @param {any} config - Configuration object
 * @returns {Promise<any>} Discovery manifest
 */
async function discover(config) {
  const { campaignId, workflows = [], trackerLabel = null, repos = [], maxDiscoveryItems = DEFAULT_MAX_ITEMS, maxDiscoveryPages = DEFAULT_MAX_PAGES, cursorPath = null, projectUrl = null } = config;

  core.info(`Starting campaign discovery for: ${campaignId}`);
  core.info(`Workflows: ${workflows.join(", ")}`);
  core.info(`Tracker label: ${trackerLabel || "none"}`);
  core.info(`Max items: ${maxDiscoveryItems}, Max pages: ${maxDiscoveryPages}`);

  // Load cursor if available
  let cursor = cursorPath ? loadCursor(cursorPath) : null;

  const octokit = github;
  const allItems = [];
  let totalItemsScanned = 0;
  let totalPagesScanned = 0;

  // Discover by tracker-id (one search per workflow)
  if (workflows && workflows.length > 0) {
    for (const workflow of workflows) {
      if (totalItemsScanned >= maxDiscoveryItems || totalPagesScanned >= maxDiscoveryPages) {
        core.warning(`Reached discovery budget limits. Stopping discovery.`);
        break;
      }

      const remainingItems = maxDiscoveryItems - totalItemsScanned;
      const remainingPages = maxDiscoveryPages - totalPagesScanned;

      const result = await searchByTrackerId(octokit, workflow, repos, remainingItems, remainingPages, cursor);

      allItems.push(...result.items);
      totalItemsScanned += result.itemsScanned;
      totalPagesScanned += result.pagesScanned;
      cursor = result.cursor;
    }
  }

  // Discover by tracker label (if provided)
  if (trackerLabel) {
    if (totalItemsScanned < maxDiscoveryItems && totalPagesScanned < maxDiscoveryPages) {
      const remainingItems = maxDiscoveryItems - totalItemsScanned;
      const remainingPages = maxDiscoveryPages - totalPagesScanned;

      const result = await searchByLabel(octokit, trackerLabel, repos, remainingItems, remainingPages, cursor);

      // Merge items (deduplicate by URL)
      const existingUrls = new Set(allItems.map(i => i.url));
      for (const item of result.items) {
        if (!existingUrls.has(item.url)) {
          allItems.push(item);
        }
      }

      totalItemsScanned += result.itemsScanned;
      totalPagesScanned += result.pagesScanned;
      cursor = result.cursor;
    }
  }

  // Sort items for stable ordering (by updated_at, then by number)
  allItems.sort((a, b) => {
    if (a.updated_at !== b.updated_at) {
      return a.updated_at.localeCompare(b.updated_at);
    }
    return a.number - b.number;
  });

  // Calculate summary counts
  const needsAddCount = allItems.filter(i => i.state === "open").length;
  const needsUpdateCount = allItems.filter(i => i.state === "closed" || i.merged_at).length;

  // Build manifest
  const manifest = {
    schema_version: MANIFEST_VERSION,
    campaign_id: campaignId,
    generated_at: new Date().toISOString(),
    project_url: projectUrl,
    discovery: {
      total_items: allItems.length,
      items_scanned: totalItemsScanned,
      pages_scanned: totalPagesScanned,
      max_items_budget: maxDiscoveryItems,
      max_pages_budget: maxDiscoveryPages,
      cursor: cursor,
    },
    summary: {
      needs_add_count: needsAddCount,
      needs_update_count: needsUpdateCount,
      open_count: allItems.filter(i => i.state === "open").length,
      closed_count: allItems.filter(i => i.state === "closed" && !i.merged_at).length,
      merged_count: allItems.filter(i => i.merged_at).length,
    },
    items: allItems,
  };

  // Save cursor if provided
  if (cursorPath) {
    saveCursor(cursorPath, cursor);
  }

  core.info(`Discovery complete: ${allItems.length} items found`);
  core.info(`Summary: ${needsAddCount} to add, ${needsUpdateCount} to update`);

  return manifest;
}

/**
 * Main entry point
 */
async function main() {
  try {
    // Read configuration from environment variables
    const config = {
      campaignId: process.env.GH_AW_CAMPAIGN_ID || core.getInput("campaign-id", { required: true }),
      workflows: (process.env.GH_AW_WORKFLOWS || core.getInput("workflows") || "")
        .split(",")
        .map(w => w.trim())
        .filter(w => w.length > 0),
      trackerLabel: process.env.GH_AW_TRACKER_LABEL || core.getInput("tracker-label") || null,
      repos: (process.env.GH_AW_DISCOVERY_REPOS || core.getInput("repos") || "")
        .split(",")
        .map(r => r.trim())
        .filter(r => r.length > 0),
      maxDiscoveryItems: parseInt(process.env.GH_AW_MAX_DISCOVERY_ITEMS || core.getInput("max-discovery-items") || DEFAULT_MAX_ITEMS.toString(), 10),
      maxDiscoveryPages: parseInt(process.env.GH_AW_MAX_DISCOVERY_PAGES || core.getInput("max-discovery-pages") || DEFAULT_MAX_PAGES.toString(), 10),
      cursorPath: process.env.GH_AW_CURSOR_PATH || core.getInput("cursor-path") || null,
      projectUrl: process.env.GH_AW_PROJECT_URL || core.getInput("project-url") || null,
    };

    // Validate configuration
    if (!config.campaignId) {
      throw new Error("campaign-id is required");
    }

    if (!config.workflows || config.workflows.length === 0) {
      if (!config.trackerLabel) {
        throw new Error("Either workflows or tracker-label must be provided");
      }
    }

    // Run discovery
    const manifest = await discover(config);

    // Write manifest to output file
    const outputDir = "./.gh-aw";
    const outputPath = path.join(outputDir, "campaign.discovery.json");

    if (!fs.existsSync(outputDir)) {
      fs.mkdirSync(outputDir, { recursive: true });
    }

    fs.writeFileSync(outputPath, JSON.stringify(manifest, null, 2));
    core.info(`Manifest written to ${outputPath}`);

    // Set output for GitHub Actions
    core.setOutput("manifest-path", outputPath);
    core.setOutput("needs-add-count", manifest.summary.needs_add_count);
    core.setOutput("needs-update-count", manifest.summary.needs_update_count);
    core.setOutput("total-items", manifest.discovery.total_items);

    // Log summary
    core.info(`✓ Discovery complete`);
    core.info(`  Total items: ${manifest.discovery.total_items}`);
    core.info(`  Needs add: ${manifest.summary.needs_add_count}`);
    core.info(`  Needs update: ${manifest.summary.needs_update_count}`);
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error));
    core.setFailed(`Discovery failed: ${err.message}`);
    throw err;
  }
}

module.exports = {
  main,
  discover,
  normalizeItem,
  searchByTrackerId,
  searchByLabel,
  loadCursor,
  saveCursor,
};
