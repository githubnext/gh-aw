#!/usr/bin/env node

import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

function parseDotenv(content) {
  /** @type {Record<string, string>} */
  const vars = {};
  for (const lineRaw of content.split(/\r?\n/)) {
    const line = lineRaw.trim();
    if (!line || line.startsWith("#")) continue;
    const eq = line.indexOf("=");
    if (eq <= 0) continue;
    const key = line.slice(0, eq).trim();
    let value = line.slice(eq + 1).trim();
    if (!key) continue;

    // Strip surrounding quotes (simple .env compatibility)
    if ((value.startsWith('"') && value.endsWith('"') && value.length >= 2) || (value.startsWith("'") && value.endsWith("'") && value.length >= 2)) {
      value = value.slice(1, -1);
    }

    // Support basic escaped newlines in quoted values.
    value = value.replaceAll("\\n", "\n");

    vars[key] = value;
  }
  return vars;
}

async function loadDotenvIfPresent(docsRoot) {
  // Node scripts do not automatically read .env files.
  // Load .env.local first, then .env (do not override real env vars).
  const candidates = [path.join(docsRoot, ".env.local"), path.join(docsRoot, ".env")];

  for (const envPath of candidates) {
    try {
      const content = await fs.readFile(envPath, "utf8");
      const vars = parseDotenv(content);
      for (const [k, v] of Object.entries(vars)) {
        if (process.env[k] === undefined) process.env[k] = v;
      }
    } catch {
      // ignore missing/invalid env file
    }
  }
}

function parseArgs(argv) {
  const args = {
    repo: process.env.PLAYGROUND_REPO || "",
    ref: process.env.PLAYGROUND_REF || "main",
    token: process.env.PLAYGROUND_TOKEN || process.env.GITHUB_TOKEN || "",
    snapshotsPath: process.env.PLAYGROUND_SNAPSHOTS_PATH || "docs/playground-snapshots",
    snapshotsMode: process.env.PLAYGROUND_SNAPSHOTS_MODE || "actions",
    snapshotsBranch: process.env.PLAYGROUND_SNAPSHOTS_BRANCH || "",
    prefix: process.env.PLAYGROUND_ID_PREFIX || "",
    mdx: process.env.PLAYGROUND_MDX || "src/content/docs/playground/index.mdx",
    workflowsDir: process.env.PLAYGROUND_WORKFLOWS_DIR || ".github/workflows",
  };

  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--repo") args.repo = argv[++i] || "";
    else if (a === "--ref") args.ref = argv[++i] || "main";
    else if (a === "--token") args.token = argv[++i] || "";
    else if (a === "--snapshots-path") args.snapshotsPath = argv[++i] || args.snapshotsPath;
    else if (a === "--snapshots-mode") args.snapshotsMode = argv[++i] || args.snapshotsMode;
    else if (a === "--snapshots-branch") args.snapshotsBranch = argv[++i] || args.snapshotsBranch;
    else if (a === "--workflows-dir") args.workflowsDir = argv[++i] || args.workflowsDir;
    else if (a === "--prefix") args.prefix = argv[++i] || args.prefix;
    else if (a === "--mdx") args.mdx = argv[++i] || args.mdx;
    else if (a === "--help" || a === "-h") {
      printHelp();
      process.exit(0);
    } else {
      throw new Error(`Unknown argument: ${a}`);
    }
  }

  return args;
}

function printHelp() {
  // Keep this intentionally short; no secrets printed.
  console.log(`Usage: npm run fetch-playground-local -- --repo owner/repo [options]

Options:
  --repo owner/repo          Required. Repo to fetch from (private is OK with token)
  --ref main                 Git ref (default: main)
  --token <PAT>              Token (or use env PLAYGROUND_TOKEN / GITHUB_TOKEN)
  --snapshots-path <path>    Repo path containing snapshots (default: docs/playground-snapshots)
  --snapshots-mode <mode>    Snapshot mode: contents|actions (default: actions)
  --snapshots-branch <name>  Branch to query in Actions mode (default: --ref)
  --workflows-dir <path>     Repo dir for workflow files (default: .github/workflows)
  --prefix <prefix>          Workflow ID prefix to fetch (default: playground-user-)
  --mdx <path>               MDX file to read IDs from (default: src/content/docs/playground/index.mdx)

Environment equivalents:
  PLAYGROUND_REPO, PLAYGROUND_REF, PLAYGROUND_TOKEN, PLAYGROUND_SNAPSHOTS_PATH, PLAYGROUND_SNAPSHOTS_MODE, PLAYGROUND_SNAPSHOTS_BRANCH,
  PLAYGROUND_WORKFLOWS_DIR, PLAYGROUND_ID_PREFIX, PLAYGROUND_MDX
`);
}

async function readWorkflowIdsFromMdx(mdxPath, prefix) {
  const mdx = await fs.readFile(mdxPath, "utf8");

  const ids = new Set();
  const re = /\bid\s*:\s*['"]([^'"]+)['"]/g;
  let m;
  while ((m = re.exec(mdx)) !== null) {
    const id = String(m[1] || "").trim();
    if (!id) continue;
    if (prefix && !id.startsWith(prefix)) continue;
    ids.add(id);
  }

  return [...ids].sort();
}

function runNodeScript({ scriptPath, cwd, env }) {
  const res = spawnSync(process.execPath, [scriptPath], {
    cwd,
    env: { ...process.env, ...env },
    stdio: "inherit",
  });

  if (res.error) throw res.error;
  if (typeof res.status === "number" && res.status !== 0) {
    throw new Error(`Script failed: ${path.basename(scriptPath)} (exit ${res.status})`);
  }
}

async function main() {
  const docsRoot = path.resolve(__dirname, "..");
  await loadDotenvIfPresent(docsRoot);

  const args = parseArgs(process.argv);

  if (!args.repo) {
    console.error("[playground-local] Missing --repo (or PLAYGROUND_REPO).");
    printHelp();
    process.exit(2);
  }

  if (!args.token) {
    console.error("[playground-local] Missing token. Set PLAYGROUND_TOKEN or pass --token.");
    console.error("[playground-local] For fine-grained PAT: give read access to Contents + Metadata on the repo.");
    process.exit(2);
  }

  const mdxPath = path.resolve(docsRoot, args.mdx);

  const ids = await readWorkflowIdsFromMdx(mdxPath, args.prefix);
  if (ids.length === 0) {
    if (args.prefix) {
      const fallbackIds = await readWorkflowIdsFromMdx(mdxPath, "");
      if (fallbackIds.length > 0) {
        console.warn(`[playground-local] No workflow IDs found with prefix '${args.prefix}' in ${args.mdx}. ` + `Falling back to fetching all workflows listed in that file.`);
        // eslint-disable-next-line no-param-reassign
        args.prefix = "";
        // eslint-disable-next-line no-param-reassign
        ids.length = 0;
        ids.push(...fallbackIds);
      }
    }

    if (ids.length === 0) {
      console.error(`[playground-local] No workflow IDs found in ${args.mdx}`);
      process.exit(1);
    }
  }

  const repoPaths = [];
  for (const id of ids) {
    repoPaths.push(`${args.workflowsDir.replace(/\/$/, "")}/${id}.md`);
    repoPaths.push(`${args.workflowsDir.replace(/\/$/, "")}/${id}.lock.yml`);
  }

  const workflowsScript = path.resolve(__dirname, "fetch-playground-workflows.mjs");
  const snapshotsScript = path.resolve(__dirname, "fetch-playground-snapshots.mjs");

  console.log(`[playground-local] Repo: ${args.repo}@${args.ref}`);
  console.log(`[playground-local] Workflows: ${ids.length} (prefix '${args.prefix}')`);

  runNodeScript({
    scriptPath: workflowsScript,
    cwd: docsRoot,
    env: {
      PLAYGROUND_WORKFLOWS_REPO: args.repo,
      PLAYGROUND_WORKFLOWS_REF: args.ref,
      PLAYGROUND_WORKFLOWS_TOKEN: args.token,
      PLAYGROUND_WORKFLOWS_FILES: repoPaths.join(","),
    },
  });

  runNodeScript({
    scriptPath: snapshotsScript,
    cwd: docsRoot,
    env: {
      PLAYGROUND_SNAPSHOTS_REPO: args.repo,
      PLAYGROUND_SNAPSHOTS_REF: args.ref,
      PLAYGROUND_SNAPSHOTS_PATH: args.snapshotsPath,
      PLAYGROUND_SNAPSHOTS_TOKEN: args.token,
      PLAYGROUND_SNAPSHOTS_MODE: args.snapshotsMode,
      PLAYGROUND_SNAPSHOTS_BRANCH: args.snapshotsBranch || args.ref,
      PLAYGROUND_SNAPSHOTS_WORKFLOWS_DIR: args.workflowsDir,
      PLAYGROUND_SNAPSHOTS_WORKFLOW_IDS: ids.join(","),
    },
  });

  console.log("[playground-local] Done. Start the dev server with: npm run dev");
}

main().catch(err => {
  console.error(String(err?.stack || err));
  process.exitCode = 1;
});
