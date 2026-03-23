// @ts-check
/// <reference types="@actions/github-script" />
"use strict";

const fs = require("fs");
const path = require("path");
const { pathToFileURL } = require("url");

/**
 * @typedef {{ name: string, path: string, patterns?: string[], context?: string }} QmdCheckout
 * @typedef {{ name?: string, type?: string, query?: string, repo?: string, min?: number, max?: number, tokenEnvVar?: string }} QmdSearch
 * @typedef {{ dbPath: string, checkouts?: QmdCheckout[], searches?: QmdSearch[] }} QmdConfig
 */

/**
 * Resolves `${ENV_VAR}` placeholders in a path string using the current process environment.
 * @param {string} p
 * @returns {string}
 */
function resolveEnvVars(p) {
  return p.replace(/\$\{([^}]+)\}/g, (_, name) => process.env[name] || "");
}

/**
 * Returns an Octokit client for the given token env var, or the default github client.
 * @param {string | undefined} tokenEnvVar
 * @returns {Promise<typeof github>}
 */
async function getClient(tokenEnvVar) {
  if (tokenEnvVar && process.env[tokenEnvVar]) {
    const { getOctokit } = await import("@actions/github");
    return getOctokit(process.env[tokenEnvVar]);
  }
  return github;
}

/**
 * Writes the step summary to $GITHUB_STEP_SUMMARY via core.summary.
 * @param {QmdConfig} config
 * @param {{ indexed: number, updated: number, unchanged: number, removed: number } | null} updateResult
 * @param {{ embedded: number } | null} embedResult
 */
async function writeSummary(config, updateResult, embedResult) {
  try {
    let md = "## qmd documentation index\n\n";

    const checkouts = config.checkouts ?? [];
    if (checkouts.length > 0) {
      md += "### Collections\n\n";
      md += "| Name | Patterns | Context |\n";
      md += "| --- | --- | --- |\n";
      for (const col of checkouts) {
        const patterns = (col.patterns || ["**/*.md"]).join(", ");
        const ctx = col.context || "-";
        md += `| ${col.name} | ${patterns} | ${ctx} |\n`;
      }
      md += "\n";
    }

    const searches = config.searches ?? [];
    if (searches.length > 0) {
      md += "### Searches\n\n";
      md += "| Name | Type | Query / Repo | Min | Max |\n";
      md += "| --- | --- | --- | --- | --- |\n";
      for (const s of searches) {
        const name = s.name || "-";
        const type = s.type || "code";
        const ref = (s.query || s.repo || "-").replace(/\|/g, "\\|");
        const min = s.min && s.min > 0 ? String(s.min) : "-";
        const max = String(s.max && s.max > 0 ? s.max : type === "issues" ? 500 : 30);
        md += `| ${name} | ${type} | ${ref} | ${min} | ${max} |\n`;
      }
      md += "\n";
    }

    if (updateResult) {
      md += "### Index stats\n\n";
      md += "| Stat | Value |\n";
      md += "| --- | --- |\n";
      md += `| Indexed | ${updateResult.indexed} |\n`;
      md += `| Updated | ${updateResult.updated} |\n`;
      md += `| Unchanged | ${updateResult.unchanged} |\n`;
      md += `| Removed | ${updateResult.removed} |\n`;
      if (embedResult) {
        md += `| Embedded | ${embedResult.embedded} |\n`;
      }
    }

    await core.summary.addRaw(md).write();
  } catch (/** @type {any} */ err) {
    core.warning(`Could not write step summary: ${err.message}`);
  }
}

/**
 * Main entry point for building the qmd documentation index.
 *
 * Reads the JSON config from the QMD_CONFIG_JSON environment variable, uses the
 * @tobilu/qmd JavaScript SDK to create a vector-search store, registers all
 * configured collections (from checkouts and GitHub searches), then calls
 * store.update() and store.embed() to index the files and save the collection.
 *
 * Called from an actions/github-script step via:
 *   const { main } = require('/tmp/gh-aw/actions/qmd_index.cjs');
 *   await main();
 */
async function main() {
  const configJson = process.env.QMD_CONFIG_JSON;
  if (!configJson) {
    core.setFailed("QMD_CONFIG_JSON environment variable not set");
    return;
  }

  /** @type {QmdConfig} */
  const config = JSON.parse(configJson);

  // Load @tobilu/qmd SDK (ESM-only package) via dynamic import.
  // The package is installed into the gh-aw actions directory by a prior npm-install step.
  const qmdIndexPath = path.join(__dirname, "node_modules", "@tobilu", "qmd", "dist", "index.js");
  if (!fs.existsSync(qmdIndexPath)) {
    core.setFailed(`@tobilu/qmd not found at ${qmdIndexPath}. The 'Install @tobilu/qmd SDK' step must run first.`);
    return;
  }

  const { createStore } = /** @type {any} */ await import(pathToFileURL(qmdIndexPath).href);

  // Ensure the index directory exists.
  fs.mkdirSync(config.dbPath, { recursive: true });
  const dbPath = path.join(config.dbPath, "index.sqlite");

  // ── Build collections config from checkout entries ──────────────────────
  /** @type {Record<string, { path: string, pattern?: string, context?: Record<string, string> }>} */
  const collections = {};

  for (const checkout of config.checkouts || []) {
    const resolvedPath = resolveEnvVars(checkout.path);
    const pattern = (checkout.patterns || ["**/*.md"]).join(",");
    collections[checkout.name] = {
      path: resolvedPath,
      pattern,
      ...(checkout.context ? { context: { "/": checkout.context } } : {}),
    };
  }

  // ── Process search entries ───────────────────────────────────────────────
  const searchEntries = config.searches ?? [];
  for (let i = 0; i < searchEntries.length; i++) {
    const search = searchEntries[i];
    const collectionName = search.name || `search-${i}`;
    const searchDir = `/tmp/gh-aw/qmd-search-${i}`;
    fs.mkdirSync(searchDir, { recursive: true });

    const client = await getClient(search.tokenEnvVar);

    if (search.type === "issues") {
      const repoSlug = search.repo || process.env.GITHUB_REPOSITORY || "";
      const slugParts = repoSlug.split("/");
      if (slugParts.length < 2 || !slugParts[0] || !slugParts[1]) {
        core.setFailed(`qmd search "${collectionName}": invalid repository slug "${repoSlug}" (expected "owner/repo")`);
        return;
      }
      const [owner, repo] = slugParts;
      const maxCount = search.max && search.max > 0 ? search.max : 500;

      core.info(`Fetching issues from ${repoSlug} (max: ${maxCount})…`);

      // Paginate until we have accumulated enough issues across all pages.
      let accumulated = 0;
      const issues = await client.paginate(client.rest.issues.listForRepo, { owner, repo, state: "open", per_page: 100 }, (/** @type {{ data: any[] }} */ response, done) => {
        accumulated += response.data.length;
        if (accumulated >= maxCount) done();
        return response.data;
      });

      const slice = issues.slice(0, maxCount);
      for (const issue of slice) {
        const content = `## ${issue.number}: ${issue.title}\n\n${issue.body || ""}`;
        fs.writeFileSync(path.join(searchDir, `issue-${issue.number}.md`), content, "utf8");
      }
      core.info(`Saved ${slice.length} issues to ${searchDir}`);
    } else {
      // Code search: download matching files via GitHub REST API.
      const maxCount = search.max && search.max > 0 ? search.max : 30;
      core.info(`Searching GitHub code: "${search.query}" (max: ${maxCount})…`);

      const response = await client.rest.search.code({
        q: search.query,
        per_page: Math.min(maxCount, 100),
      });

      let downloaded = 0;
      for (const item of response.data.items) {
        const fullNameParts = item.repository.full_name.split("/");
        if (fullNameParts.length < 2) continue;
        const [owner, repo] = fullNameParts;
        try {
          const fileResp = await client.rest.repos.getContent({
            owner,
            repo,
            path: item.path,
          });
          const data = /** @type {any} */ fileResp.data;
          if (data.type === "file" && data.content) {
            const fileContent = Buffer.from(data.content, "base64").toString("utf8");
            const safeName = `${owner}-${repo}-${item.path.replace(/\//g, "-")}`;
            fs.writeFileSync(path.join(searchDir, safeName), fileContent, "utf8");
            downloaded++;
          }
        } catch (/** @type {any} */ err) {
          core.warning(`Could not download ${item.repository.full_name}/${item.path}: ${err.message}`);
        }
      }
      core.info(`Downloaded ${downloaded} files to ${searchDir}`);
    }

    // Enforce minimum result count.
    const minCount = search.min ?? 0;
    if (minCount > 0) {
      const fileCount = fs.readdirSync(searchDir).length;
      if (fileCount < minCount) {
        core.setFailed(`qmd search "${collectionName}" returned ${fileCount} results, minimum is ${minCount}`);
        return;
      }
    }

    collections[collectionName] = {
      path: searchDir,
      pattern: "**/*",
    };
  }

  // ── Create store and build index ─────────────────────────────────────────
  core.info(`Creating qmd store at ${dbPath}…`);

  const store = await createStore({ dbPath, config: { collections } });

  let updateResult = null;
  let embedResult = null;

  try {
    core.info("Indexing files (update)…");
    updateResult = await store.update({
      onProgress: (/** @type {{ collection: string, file: string, current: number, total: number }} */ info) => {
        if (info.current % 50 === 0 || info.current === info.total) {
          core.debug(`[${info.collection}] ${info.current}/${info.total}: ${info.file}`);
        }
      },
    });
    core.info(`Update complete: ${updateResult.indexed} indexed, ${updateResult.updated} updated, ` + `${updateResult.unchanged} unchanged, ${updateResult.removed} removed`);

    core.info("Generating embeddings (embed)…");
    embedResult = await store.embed({
      onProgress: (/** @type {{ current: number, total: number }} */ info) => {
        if (info.current % 20 === 0 || info.current === info.total) {
          core.debug(`Embedding ${info.current}/${info.total}`);
        }
      },
    });
    core.info(`Embed complete: ${embedResult.embedded} embedded`);
  } finally {
    await store.close();
    await writeSummary(config, updateResult, embedResult);
  }

  core.info("qmd index built successfully");
}

module.exports = { main };
