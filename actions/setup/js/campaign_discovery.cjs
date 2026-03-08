// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Campaign Discovery
 *
 * This module precomputes campaign discovery data by:
 * 1. Searching for issues/PRs labeled with the tracker label in specified repos
 * 2. Respecting pagination budget (max items, max pages)
 * 3. Writing discovery results to a JSON file for the AI agent to consume
 * 4. Setting step outputs with summary information
 *
 * Environment variables:
 *   GH_AW_CAMPAIGN_ID          - Campaign identifier (required)
 *   GH_AW_TRACKER_LABEL        - Label used to track campaign items (required)
 *   GH_AW_DISCOVERY_REPOS      - Comma-separated list of "owner/repo" to search (required)
 *   GH_AW_CURSOR_PATH          - Path to cursor JSON file for pagination state (optional)
 *   GH_AW_MAX_DISCOVERY_ITEMS  - Maximum items to discover per run (default: 50)
 *   GH_AW_MAX_DISCOVERY_PAGES  - Maximum pages to search per repo (default: 3)
 *   GH_AW_PROJECT_URL          - URL of the GitHub Project board (optional)
 *   GH_AW_WORKFLOWS            - Comma-separated list of associated workflow names (optional)
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_CONFIG } = require("./error_codes.cjs");

/** Default output directory for temporary gh-aw files */
const GH_AW_TMP_DIR = "/tmp/gh-aw";

/** Output filename for discovery results */
const DISCOVERY_OUTPUT_FILENAME = "campaign-discovery.json";

/**
 * Main entry point for campaign discovery
 * @returns {Promise<void>}
 */
async function main() {
  const campaignId = process.env.GH_AW_CAMPAIGN_ID;
  const trackerLabel = process.env.GH_AW_TRACKER_LABEL;
  const discoveryRepos = process.env.GH_AW_DISCOVERY_REPOS;
  const cursorPath = process.env.GH_AW_CURSOR_PATH;
  const maxItemsStr = process.env.GH_AW_MAX_DISCOVERY_ITEMS ?? "50";
  const maxPagesStr = process.env.GH_AW_MAX_DISCOVERY_PAGES ?? "3";
  const projectUrl = process.env.GH_AW_PROJECT_URL;
  const workflowsEnv = process.env.GH_AW_WORKFLOWS;

  if (!campaignId) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_CAMPAIGN_ID not set`);
    return;
  }

  if (!trackerLabel) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_TRACKER_LABEL not set`);
    return;
  }

  if (!discoveryRepos) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_DISCOVERY_REPOS not set`);
    return;
  }

  const maxItems = parseInt(maxItemsStr, 10);
  const maxPages = parseInt(maxPagesStr, 10);

  if (isNaN(maxItems) || maxItems < 1) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_MAX_DISCOVERY_ITEMS must be a positive integer, got "${maxItemsStr}"`);
    return;
  }

  if (isNaN(maxPages) || maxPages < 1) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_MAX_DISCOVERY_PAGES must be a positive integer, got "${maxPagesStr}"`);
    return;
  }

  core.info(`Campaign Discovery: ${campaignId}`);
  core.info(`Tracker label: ${trackerLabel}`);
  core.info(`Discovery repos: ${discoveryRepos}`);
  core.info(`Max items: ${maxItems}, Max pages: ${maxPages}`);

  // Parse workflows list
  const workflows = workflowsEnv
    ? workflowsEnv
        .split(",")
        .map(w => w.trim())
        .filter(w => w)
    : [];

  // Parse repos list
  const repos = discoveryRepos
    .split(",")
    .map(r => r.trim())
    .filter(r => r);

  // Search for issues/PRs with tracker label across all discovery repos
  /** @type {Array<{id: number, number: number, title: string, state: string, html_url: string, created_at: string, updated_at: string, labels: string[], repo: string, is_pr: boolean}>} */
  const allItems = [];

  for (const repoPath of repos) {
    const parts = repoPath.split("/");
    if (parts.length !== 2) {
      core.warning(`Invalid repo format: "${repoPath}", expected "owner/repo" — skipping`);
      continue;
    }

    const [owner, repo] = parts;
    core.info(`Searching ${owner}/${repo} for issues with label: ${trackerLabel}`);

    let page = 1;

    while (page <= maxPages && allItems.length < maxItems) {
      const perPage = Math.min(30, maxItems - allItems.length);

      try {
        const response = await github.rest.issues.listForRepo({
          owner,
          repo,
          labels: trackerLabel,
          state: "all",
          sort: "updated",
          direction: "desc",
          per_page: perPage,
          page,
        });

        if (response.data.length === 0) {
          core.info(`No more items on page ${page} for ${owner}/${repo}`);
          break;
        }

        for (const item of response.data) {
          if (allItems.length >= maxItems) break;
          allItems.push({
            id: item.id,
            number: item.number,
            title: item.title,
            state: item.state,
            html_url: item.html_url,
            created_at: item.created_at,
            updated_at: item.updated_at,
            labels: item.labels.map(l => (typeof l === "string" ? l : (l.name ?? ""))),
            repo: `${owner}/${repo}`,
            is_pr: !!item.pull_request,
          });
        }

        core.info(`Page ${page}: fetched ${response.data.length} items (total so far: ${allItems.length})`);

        if (response.data.length < perPage) {
          break; // No more pages available
        }

        page++;
      } catch (err) {
        core.error(`${ERR_API}: Failed to search ${owner}/${repo} page ${page}: ${getErrorMessage(err)}`);
        break;
      }
    }
  }

  core.info(`Discovery complete: found ${allItems.length} tracked items`);

  // Write discovery results to a file for the AI agent to consume
  /** @type {{campaign_id: string, timestamp: string, tracker_label: string, project_url: string | undefined, workflows: string[], items_count: number, items: typeof allItems}} */
  const discoveryData = {
    campaign_id: campaignId,
    timestamp: new Date().toISOString(),
    tracker_label: trackerLabel,
    project_url: projectUrl,
    workflows,
    items_count: allItems.length,
    items: allItems,
  };

  const discoveryOutputPath = path.join(GH_AW_TMP_DIR, DISCOVERY_OUTPUT_FILENAME);

  try {
    fs.mkdirSync(GH_AW_TMP_DIR, { recursive: true });
    fs.writeFileSync(discoveryOutputPath, JSON.stringify(discoveryData, null, 2));
    core.info(`Wrote discovery data to ${discoveryOutputPath}`);
  } catch (err) {
    core.warning(`Failed to write discovery data to ${discoveryOutputPath}: ${getErrorMessage(err)}`);
  }

  // Set step outputs for downstream steps or summary
  core.setOutput("items_count", String(allItems.length));
  core.setOutput("campaign_id", campaignId);
  core.setOutput("discovery_file", discoveryOutputPath);

  core.info(`✓ Campaign discovery complete: ${allItems.length} item(s) found for campaign "${campaignId}"`);
  if (cursorPath) {
    core.info(`Note: Cursor path configured at ${cursorPath} (managed by repo-memory)`);
  }
}

module.exports = { main };
