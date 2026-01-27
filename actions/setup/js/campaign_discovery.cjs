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
 * Build scope query parts for GitHub search
 * @param {string[]} repos - List of repositories to search (owner/repo format)
 * @param {string[]} orgs - List of organizations to search
 * @returns {string[]} Array of scope parts (e.g., ["repo:owner/repo", "org:orgname"])
 */
function buildScopeParts(repos, orgs) {
  return [...(repos?.length ? repos.map(r => `repo:${r}`) : []), ...(orgs?.length ? orgs.map(o => `org:${o}`) : [])];
}

/**
 * Generic search helper for issues and PRs
 * @param {any} octokit - GitHub API client
 * @param {string} searchQuery - GitHub search query
 * @param {string} searchLabel - Label for logging (e.g., "tracker-id: workflow-1" or "label: bug")
 * @param {number} maxItems - Maximum items to discover
 * @param {number} maxPages - Maximum pages to fetch
 * @param {any} cursor - Cursor for pagination
 * @param {any} cursorData - Additional data to store in cursor
 * @returns {Promise<{items: any[], cursor: any, itemsScanned: number, pagesScanned: number}>}
 */
async function searchItems(octokit, searchQuery, searchLabel, maxItems, maxPages, cursor, cursorData) {
  const items = [];
  let itemsScanned = 0;
  let pagesScanned = 0;
  let page = cursor?.page || 1;

  while (pagesScanned < maxPages && itemsScanned < maxItems) {
    core.info(`Fetching page ${page} for ${searchLabel}`);

    const response = await octokit.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 100,
      page,
      sort: "updated",
      order: "asc",
    });

    pagesScanned++;

    if (response.data.items.length === 0) {
      core.info(`No more items found for ${searchLabel}`);
      break;
    }

    for (const item of response.data.items) {
      if (itemsScanned >= maxItems) break;
      itemsScanned++;
      const contentType = item.pull_request ? "pull_request" : "issue";
      items.push(normalizeItem(item, contentType));
    }

    if (response.data.items.length < 100) break;
    page++;
  }

  return { items, cursor: { page, ...cursorData }, itemsScanned, pagesScanned };
}

/**
 * Search for items by tracker-id across issues and PRs
 * @param {any} octokit - GitHub API client
 * @param {string} trackerId - Tracker ID to search for
 * @param {string[]} repos - List of repositories to search (owner/repo format)
 * @param {string[]} orgs - List of organizations to search
 * @param {number} maxItems - Maximum items to discover
 * @param {number} maxPages - Maximum pages to fetch
 * @param {any} cursor - Cursor for pagination
 * @returns {Promise<{items: any[], cursor: any, itemsScanned: number, pagesScanned: number}>}
 */
async function searchByTrackerId(octokit, trackerId, repos, orgs, maxItems, maxPages, cursor) {
  core.info(`Searching for tracker-id: ${trackerId} in ${repos.length} repo(s) and ${orgs.length} org(s)`);

  let searchQuery = `"gh-aw-tracker-id: ${trackerId}" type:issue`;
  const scopeParts = buildScopeParts(repos, orgs);

  if (scopeParts.length > 0) {
    const scopeQuery = scopeParts.join(" ");
    if (searchQuery.length + scopeQuery.length + 1 > 1000) {
      core.warning(`Search query length (${searchQuery.length + scopeQuery.length + 1}) approaches GitHub's ~1024 character limit. Some repos/orgs may be omitted.`);
    }
    searchQuery = `${searchQuery} ${scopeQuery}`;
    core.info(`Scoped search to: ${scopeParts.join(", ")}`);
  }

  return searchItems(octokit, searchQuery, `tracker-id: ${trackerId}`, maxItems, maxPages, cursor, { trackerId });
}

/**
 * Search for items by tracker label
 * @param {any} octokit - GitHub API client
 * @param {string} label - Label to search for
 * @param {string[]} repos - List of repositories to search (owner/repo format)
 * @param {string[]} orgs - List of organizations to search
 * @param {number} maxItems - Maximum items to discover
 * @param {number} maxPages - Maximum pages to fetch
 * @param {any} cursor - Cursor for pagination
 * @returns {Promise<{items: any[], cursor: any, itemsScanned: number, pagesScanned: number}>}
 */
async function searchByLabel(octokit, label, repos, orgs, maxItems, maxPages, cursor) {
  core.info(`Searching for label: ${label} in ${repos.length} repo(s) and ${orgs.length} org(s)`);

  let searchQuery = `label:"${label}"`;
  const scopeParts = buildScopeParts(repos, orgs);

  if (scopeParts.length > 0) {
    const scopeQuery = scopeParts.join(" ");
    if (searchQuery.length + scopeQuery.length + 1 > 1000) {
      core.warning(`Search query length (${searchQuery.length + scopeQuery.length + 1}) approaches GitHub's ~1024 character limit. Some repos/orgs may be omitted.`);
    }
    searchQuery = `${searchQuery} ${scopeQuery}`;
    core.info(`Scoped search to: ${scopeParts.join(", ")}`);
  }

  return searchItems(octokit, searchQuery, `label: ${label}`, maxItems, maxPages, cursor, { label });
}

/**
 * Discover security alerts for a repository
 * @param {any} octokit - GitHub API client
 * @param {string[]} repos - List of repositories to search (owner/repo format)
 * @returns {Promise<any>} Security alerts summary
 */
async function discoverSecurityAlerts(octokit, repos) {
  if (!repos || repos.length === 0) {
    core.warning("No repos specified for security alert discovery");
    return null;
  }

  const alerts = {
    code_scanning: { total: 0, by_severity: {}, by_state: {}, items: /** @type {any[]} */ ([]) },
    secret_scanning: { total: 0, by_state: {}, items: /** @type {any[]} */ ([]) },
    dependabot: { total: 0, by_severity: {}, by_state: {}, items: /** @type {any[]} */ ([]) },
  };

  // Discover alerts for each repository
  for (const repoFullName of repos) {
    const [owner, repo] = repoFullName.split("/");
    if (!owner || !repo) {
      core.warning(`Invalid repo format: ${repoFullName}. Expected owner/repo`);
      continue;
    }

    core.info(`Discovering security alerts for ${repoFullName}...`);

    // Code Scanning Alerts
    try {
      const response = await octokit.rest.codeScanning.listAlertsForRepo({
        owner,
        repo,
        per_page: 100,
        state: "open",
      });

      const codeAlerts = response.data;
      alerts.code_scanning.total += codeAlerts.length;

      for (const alert of codeAlerts) {
        const severity = alert.rule?.severity || "unknown";
        alerts.code_scanning.by_severity[severity] = (alerts.code_scanning.by_severity[severity] || 0) + 1;
        alerts.code_scanning.by_state[alert.state] = (alerts.code_scanning.by_state[alert.state] || 0) + 1;

        alerts.code_scanning.items.push({
          type: "code_scanning",
          number: alert.number,
          url: alert.html_url,
          state: alert.state,
          severity: severity,
          rule_id: alert.rule?.id,
          rule_description: alert.rule?.description,
          created_at: alert.created_at,
          updated_at: alert.updated_at,
          repository: repoFullName,
        });
      }

      core.info(`  Code scanning: ${codeAlerts.length} open alerts`);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      core.warning(`Failed to fetch code scanning alerts for ${repoFullName}: ${err.message}`);
    }

    // Secret Scanning Alerts
    try {
      const response = await octokit.rest.secretScanning.listAlertsForRepo({
        owner,
        repo,
        per_page: 100,
        state: "open",
      });

      const secretAlerts = response.data;
      alerts.secret_scanning.total += secretAlerts.length;

      for (const alert of secretAlerts) {
        alerts.secret_scanning.by_state[alert.state] = (alerts.secret_scanning.by_state[alert.state] || 0) + 1;

        alerts.secret_scanning.items.push({
          type: "secret_scanning",
          number: alert.number,
          url: alert.html_url,
          state: alert.state,
          secret_type: alert.secret_type,
          created_at: alert.created_at,
          updated_at: alert.updated_at,
          repository: repoFullName,
        });
      }

      core.info(`  Secret scanning: ${secretAlerts.length} open alerts`);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      core.warning(`Failed to fetch secret scanning alerts for ${repoFullName}: ${err.message}`);
    }

    // Dependabot Alerts
    try {
      const response = await octokit.rest.dependabot.listAlertsForRepo({
        owner,
        repo,
        per_page: 100,
        state: "open",
      });

      const dependabotAlerts = response.data;
      alerts.dependabot.total += dependabotAlerts.length;

      for (const alert of dependabotAlerts) {
        const severity = alert.security_advisory?.severity || "unknown";
        alerts.dependabot.by_severity[severity] = (alerts.dependabot.by_severity[severity] || 0) + 1;
        alerts.dependabot.by_state[alert.state] = (alerts.dependabot.by_state[alert.state] || 0) + 1;

        alerts.dependabot.items.push({
          type: "dependabot",
          number: alert.number,
          url: alert.html_url,
          state: alert.state,
          severity: severity,
          package_name: alert.security_advisory?.package?.name,
          created_at: alert.created_at,
          updated_at: alert.updated_at,
          repository: repoFullName,
        });
      }

      core.info(`  Dependabot: ${dependabotAlerts.length} open alerts`);
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));
      core.warning(`Failed to fetch Dependabot alerts for ${repoFullName}: ${err.message}`);
    }
  }

  // Log summary
  const totalAlerts = alerts.code_scanning.total + alerts.secret_scanning.total + alerts.dependabot.total;
  core.info(`✓ Security alert discovery complete: ${totalAlerts} total alerts`);
  core.info(`  Code scanning: ${alerts.code_scanning.total}`);
  core.info(`  Secret scanning: ${alerts.secret_scanning.total}`);
  core.info(`  Dependabot: ${alerts.dependabot.total}`);

  return alerts;
}

/**
 * Main discovery function
 * @param {any} config - Configuration object
 * @returns {Promise<any>} Discovery manifest
 */
async function discover(config) {
  const { campaignId, workflows = [], trackerLabel = null, repos = [], orgs = [], maxDiscoveryItems = DEFAULT_MAX_ITEMS, maxDiscoveryPages = DEFAULT_MAX_PAGES, cursorPath = null, projectUrl = null } = config;

  core.info(`Starting campaign discovery for: ${campaignId}`);
  core.info(`Workflows: ${workflows.join(", ")}`);
  core.info(`Tracker label: ${trackerLabel || "none"}`);
  core.info(`Repos: ${repos.join(", ")}`);
  core.info(`Orgs: ${orgs.join(", ")}`);
  core.info(`Max items: ${maxDiscoveryItems}, Max pages: ${maxDiscoveryPages}`);

  // Load cursor if available
  let cursor = cursorPath ? loadCursor(cursorPath) : null;

  const octokit = github;
  const allItems = [];
  let totalItemsScanned = 0;
  let totalPagesScanned = 0;

  // Generate campaign-specific label
  const campaignLabel = `z_campaign_${campaignId.toLowerCase().replace(/[_\s]+/g, "-")}`;

  // Primary discovery: Search by campaign-specific label (most reliable)
  core.info(`Primary discovery: Searching by campaign-specific label: ${campaignLabel}`);
  const labelResult = await searchByLabel(octokit, campaignLabel, repos, orgs, maxDiscoveryItems, maxDiscoveryPages, cursor).catch(err => {
    core.warning(`Campaign-specific label discovery failed: ${err instanceof Error ? err.message : String(err)}`);
    return { items: [], itemsScanned: 0, pagesScanned: 0, cursor };
  });

  allItems.push(...labelResult.items);
  totalItemsScanned += labelResult.itemsScanned;
  totalPagesScanned += labelResult.pagesScanned;
  cursor = labelResult.cursor;
  core.info(`Campaign-specific label discovery found ${labelResult.items.length} item(s)`);

  // Secondary discovery: Search by generic "agentic-campaign" label
  if (allItems.length === 0 || totalItemsScanned < maxDiscoveryItems) {
    core.info(`Secondary discovery: Searching by generic agentic-campaign label...`);
    const remainingItems = maxDiscoveryItems - totalItemsScanned;
    const remainingPages = maxDiscoveryPages - totalPagesScanned;

    const genericResult = await searchByLabel(octokit, "agentic-campaign", repos, orgs, remainingItems, remainingPages, cursor).catch(err => {
      core.warning(`Generic label discovery failed: ${err instanceof Error ? err.message : String(err)}`);
      return { items: [], itemsScanned: 0, pagesScanned: 0, cursor };
    });

    // Merge items (deduplicate by URL)
    const existingUrls = new Set(allItems.map(i => i.url));
    const newItems = genericResult.items.filter(item => !existingUrls.has(item.url));
    allItems.push(...newItems);

    totalItemsScanned += genericResult.itemsScanned;
    totalPagesScanned += genericResult.pagesScanned;
    cursor = genericResult.cursor;
    core.info(`Generic label discovery found ${newItems.length} item(s)`);
  }

  // Fallback: Search GitHub API by tracker-id (if still no items)
  if (allItems.length === 0 && workflows?.length && totalItemsScanned < maxDiscoveryItems && totalPagesScanned < maxDiscoveryPages) {
    core.info(`No items found via labels. Searching GitHub API by tracker-id...`);

    for (const workflow of workflows) {
      if (totalItemsScanned >= maxDiscoveryItems || totalPagesScanned >= maxDiscoveryPages) {
        core.warning(`Reached discovery budget limits. Stopping discovery.`);
        break;
      }

      const result = await searchByTrackerId(octokit, workflow, repos, orgs, maxDiscoveryItems - totalItemsScanned, maxDiscoveryPages - totalPagesScanned, cursor);

      allItems.push(...result.items);
      totalItemsScanned += result.itemsScanned;
      totalPagesScanned += result.pagesScanned;
      cursor = result.cursor;
    }
  }

  // Legacy discovery by tracker label (if provided and still needed)
  if (trackerLabel && (allItems.length === 0 || totalItemsScanned < maxDiscoveryItems)) {
    if (totalItemsScanned < maxDiscoveryItems && totalPagesScanned < maxDiscoveryPages) {
      const result = await searchByLabel(octokit, trackerLabel, repos, orgs, maxDiscoveryItems - totalItemsScanned, maxDiscoveryPages - totalPagesScanned, cursor);

      // Merge items (deduplicate by URL)
      const existingUrls = new Set(allItems.map(i => i.url));
      const newItems = result.items.filter(item => !existingUrls.has(item.url));
      allItems.push(...newItems);

      totalItemsScanned += result.itemsScanned;
      totalPagesScanned += result.pagesScanned;
      cursor = result.cursor;
    }
  }

  // Sort items for stable ordering (by updated_at, then by number)
  allItems.sort((a, b) => a.updated_at.localeCompare(b.updated_at) || a.number - b.number);

  // Calculate summary counts
  const openItems = allItems.filter(i => i.state === "open");
  const closedItems = allItems.filter(i => i.state === "closed" && !i.merged_at);
  const mergedItems = allItems.filter(i => i.merged_at);
  const needsAddCount = openItems.length;
  const needsUpdateCount = closedItems.length + mergedItems.length;

  // Determine if budget was exhausted
  const itemsBudgetExhausted = totalItemsScanned >= maxDiscoveryItems;
  const pagesBudgetExhausted = totalPagesScanned >= maxDiscoveryPages;
  const budgetExhausted = itemsBudgetExhausted || pagesBudgetExhausted;
  const exhaustedReason = budgetExhausted ? (itemsBudgetExhausted ? "max_items_reached" : "max_pages_reached") : null;

  // Security alert discovery (for security-focused campaigns)
  let securityAlerts = null;
  if (campaignId.toLowerCase().includes("security")) {
    core.info("Security-focused campaign detected - discovering security alerts...");
    securityAlerts = await discoverSecurityAlerts(octokit, repos);
  }

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
      budget_exhausted: budgetExhausted,
      exhausted_reason: exhaustedReason,
      cursor: cursor,
    },
    summary: {
      needs_add_count: needsAddCount,
      needs_update_count: needsUpdateCount,
      open_count: openItems.length,
      closed_count: closedItems.length,
      merged_count: mergedItems.length,
    },
    items: allItems,
  };

  // Add security alerts to manifest if discovered
  if (securityAlerts) {
    manifest.security_alerts = securityAlerts;
  }

  // Save cursor if provided
  if (cursorPath) {
    saveCursor(cursorPath, cursor);
  }

  core.info(`Discovery complete: ${allItems.length} items found`);
  core.info(`Budget utilization: ${totalItemsScanned}/${maxDiscoveryItems} items, ${totalPagesScanned}/${maxDiscoveryPages} pages`);

  if (budgetExhausted) {
    const message = allItems.length === 0 ? `Discovery budget exhausted with 0 items found. Consider increasing budget limits in governance configuration.` : `Discovery stopped at budget limit. Use cursor for continuation in next run.`;
    allItems.length === 0 ? core.warning(message) : core.info(message);
  }

  core.info(`Summary: ${needsAddCount} to add, ${needsUpdateCount} to update`);

  if (securityAlerts) {
    const totalSecurityAlerts = securityAlerts.code_scanning.total + securityAlerts.secret_scanning.total + securityAlerts.dependabot.total;
    core.info(`Security alerts: ${totalSecurityAlerts} total (${securityAlerts.code_scanning.total} code scanning, ${securityAlerts.secret_scanning.total} secret scanning, ${securityAlerts.dependabot.total} dependabot)`);
  }

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
      orgs: (process.env.GH_AW_DISCOVERY_ORGS || core.getInput("orgs") || "")
        .split(",")
        .map(o => o.trim())
        .filter(o => o.length > 0),
      maxDiscoveryItems: parseInt(process.env.GH_AW_MAX_DISCOVERY_ITEMS || core.getInput("max-discovery-items") || DEFAULT_MAX_ITEMS.toString(), 10),
      maxDiscoveryPages: parseInt(process.env.GH_AW_MAX_DISCOVERY_PAGES || core.getInput("max-discovery-pages") || DEFAULT_MAX_PAGES.toString(), 10),
      cursorPath: process.env.GH_AW_CURSOR_PATH || core.getInput("cursor-path") || null,
      projectUrl: process.env.GH_AW_PROJECT_URL || core.getInput("project-url") || null,
    };

    // Validate configuration
    if (!config.campaignId) {
      throw new Error("campaign-id is required");
    }

    // RUNTIME GUARD: Campaigns MUST be scoped
    if (!config.repos?.length && !config.orgs?.length) {
      throw new Error("campaigns MUST be scoped: GH_AW_DISCOVERY_REPOS or GH_AW_DISCOVERY_ORGS is required. Configure scope in the campaign spec.");
    }

    if (!config.workflows?.length && !config.trackerLabel) {
      throw new Error("Either workflows or tracker-label must be provided");
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
  searchItems,
  loadCursor,
  saveCursor,
  buildScopeParts,
};
