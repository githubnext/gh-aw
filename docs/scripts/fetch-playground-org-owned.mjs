#!/usr/bin/env node

import fs from "node:fs/promises";
import path from "node:path";

const outDir = path.resolve("src/assets/playground-workflows/org-owned");

const MAX_FILES = Number(process.env.PLAYGROUND_ORG_WORKFLOWS_MAX_FILES || 25);
const MAX_FILE_BYTES = Number(process.env.PLAYGROUND_ORG_WORKFLOWS_MAX_FILE_BYTES || 1024 * 1024);
const MAX_TOTAL_BYTES = Number(process.env.PLAYGROUND_ORG_WORKFLOWS_MAX_TOTAL_BYTES || 3 * 1024 * 1024);

const SAFE_BASENAME = /^[a-z0-9][a-z0-9._-]{0,200}$/;

async function main() {
  // Comma-separated list of repo-relative file paths to copy into the docs bundle.
  const filesCsv = process.env.PLAYGROUND_ORG_WORKFLOWS_FILES || "";

  const files = filesCsv
    .split(",")
    .map(s => s.trim())
    .filter(Boolean);

  if (files.length === 0) {
    console.warn("[playground-org-owned] PLAYGROUND_ORG_WORKFLOWS_FILES not set; skipping.");
    return;
  }

  if (files.length > MAX_FILES) {
    throw new Error(`[playground-org-owned] Refusing to copy ${files.length} files (max ${MAX_FILES}).`);
  }

  // Script runs with CWD=docs/, so ".." is repo root.
  const repoRoot = path.resolve("..");

  await fs.mkdir(outDir, { recursive: true });

  console.log(`[playground-org-owned] Copying ${files.length} file(s) from repo into ${outDir}`);

  let totalBytes = 0;
  for (const repoPath of files) {
    const srcPath = path.resolve(repoRoot, repoPath);
    const basename = path.posix.basename(repoPath);

    if (!SAFE_BASENAME.test(basename)) {
      throw new Error(`[playground-org-owned] Refusing unsafe filename: ${basename}`);
    }

    const bytes = await fs.readFile(srcPath);

    if (bytes.length > MAX_FILE_BYTES) {
      throw new Error(`[playground-org-owned] Refusing oversized file ${basename} (${bytes.length} bytes; max ${MAX_FILE_BYTES}).`);
    }

    totalBytes += bytes.length;
    if (totalBytes > MAX_TOTAL_BYTES) {
      throw new Error(`[playground-org-owned] Refusing files total ${totalBytes} bytes (max ${MAX_TOTAL_BYTES}).`);
    }

    const destPath = path.join(outDir, basename);
    await fs.writeFile(destPath, bytes);
    console.log(`[playground-org-owned] Wrote ${basename}`);
  }
}

main().catch(err => {
  console.error(String(err?.stack || err));
  process.exitCode = 1;
});
