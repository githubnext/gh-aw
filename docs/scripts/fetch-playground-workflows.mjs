#!/usr/bin/env node

import fs from "node:fs/promises";
import path from "node:path";

const repo = process.env.PLAYGROUND_WORKFLOWS_REPO; // "owner/repo"
const ref = process.env.PLAYGROUND_WORKFLOWS_REF || "main";
const token = process.env.PLAYGROUND_WORKFLOWS_TOKEN || process.env.GITHUB_TOKEN;

// Comma-separated list of repo-relative file paths to fetch.
// Example:
//   .github/workflows/playground-user-project-update-draft.md,
//   .github/workflows/playground-user-project-update-draft.lock.yml
const filesCsv = process.env.PLAYGROUND_WORKFLOWS_FILES || "";

const outDir = path.resolve("src/assets/playground-workflows/user-owned");

const MAX_FILES = Number(process.env.PLAYGROUND_WORKFLOWS_MAX_FILES || 25);
const MAX_FILE_BYTES = Number(process.env.PLAYGROUND_WORKFLOWS_MAX_FILE_BYTES || 1024 * 1024);
const MAX_TOTAL_BYTES = Number(process.env.PLAYGROUND_WORKFLOWS_MAX_TOTAL_BYTES || 3 * 1024 * 1024);

const SAFE_BASENAME = /^[a-z0-9][a-z0-9._-]{0,200}$/;

function headerAuth() {
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

async function ghJson(url) {
  const res = await fetch(url, {
    headers: {
      Accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2022-11-28",
      ...headerAuth(),
    },
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`GitHub API ${res.status} ${res.statusText}: ${text}`);
  }

  return res.json();
}

async function verifyRepoAccess() {
  // This call is intentionally simple: it helps distinguish
  // (a) missing file path from (b) token lacking access to the private repo.
  const url = `https://api.github.com/repos/${repo}`;
  const res = await fetch(url, {
    headers: {
      Accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2022-11-28",
      ...headerAuth(),
    },
  });

  if (res.ok) return;

  const body = await res.text().catch(() => "");
  if (res.status === 404) {
    throw new Error(
      `[playground-workflows] Cannot access repo '${repo}'. GitHub returned 404 for the repo endpoint.\n` + `This usually means the token is missing access to the private repo (or the repo name/ref is wrong).\n` + `Response: ${body}`
    );
  }

  throw new Error(`[playground-workflows] Repo access check failed (${res.status} ${res.statusText}).\nResponse: ${body}`);
}

async function download(url) {
  const res = await fetch(url, { headers: { ...headerAuth() } });
  if (!res.ok) throw new Error(`Download failed ${res.status} ${res.statusText}: ${url}`);
  return Buffer.from(await res.arrayBuffer());
}

async function main() {
  if (!repo) {
    console.warn("[playground-workflows] PLAYGROUND_WORKFLOWS_REPO not set; skipping fetch.");
    return;
  }

  await verifyRepoAccess();

  const files = filesCsv
    .split(",")
    .map(s => s.trim())
    .filter(Boolean);

  if (files.length === 0) {
    console.warn("[playground-workflows] PLAYGROUND_WORKFLOWS_FILES not set; skipping fetch.");
    return;
  }

  if (files.length > MAX_FILES) {
    throw new Error(`[playground-workflows] Refusing to fetch ${files.length} files (max ${MAX_FILES}).`);
  }

  await fs.mkdir(outDir, { recursive: true });

  console.log(`[playground-workflows] Fetching ${repo}@${ref} (${files.length} files)`);

  let totalBytes = 0;
  for (const repoPath of files) {
    const url = `https://api.github.com/repos/${repo}/contents/${repoPath.split("/").map(encodeURIComponent).join("/")}?ref=${encodeURIComponent(ref)}`;
    let info;
    try {
      info = await ghJson(url);
    } catch (err) {
      const msg = String(err?.message || err);
      if (msg.includes("GitHub API 404")) {
        throw new Error(`[playground-workflows] File not found at '${repoPath}' (ref '${ref}').\n` + `If the repo is private and you expected this file to exist, double-check token permissions and the path.\n` + `Original error: ${msg}`);
      }
      throw err;
    }

    if (!info || typeof info !== "object" || info.type !== "file" || typeof info.download_url !== "string") {
      throw new Error(`[playground-workflows] Unexpected contents API response for ${repoPath}`);
    }

    const basename = path.posix.basename(repoPath);
    if (!SAFE_BASENAME.test(basename)) {
      throw new Error(`[playground-workflows] Refusing unsafe filename: ${basename}`);
    }

    const bytes = await download(info.download_url);

    if (bytes.length > MAX_FILE_BYTES) {
      throw new Error(`[playground-workflows] Refusing oversized file ${basename} (${bytes.length} bytes; max ${MAX_FILE_BYTES}).`);
    }

    totalBytes += bytes.length;
    if (totalBytes > MAX_TOTAL_BYTES) {
      throw new Error(`[playground-workflows] Refusing files total ${totalBytes} bytes (max ${MAX_TOTAL_BYTES}).`);
    }

    await fs.writeFile(path.join(outDir, basename), bytes);
    console.log(`[playground-workflows] Wrote ${basename}`);
  }
}

main().catch(err => {
  console.error(String(err?.stack || err));
  process.exitCode = 1;
});
