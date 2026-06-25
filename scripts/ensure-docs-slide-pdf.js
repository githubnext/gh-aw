#!/usr/bin/env node

import fs from "fs";
import path from "path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ROOT = path.resolve(__dirname, "..");
const DOCS_DIR = path.join(ROOT, "docs");
const SOURCE_PATH = path.join(DOCS_DIR, "slides/github-agentic-workflows.pdf");
const OUTPUT_PATH = path.join(DOCS_DIR, "public/slides/github-agentic-workflows.pdf");
const LFS_POINTER_PREFIX = "version https://git-lfs.github.com/spec/v1";

function isPdf(buffer) {
  return buffer.subarray(0, 5).toString("utf8") === "%PDF-";
}

function getRepositoryPath() {
  try {
    const remote = execFileSync("git", ["config", "--get", "remote.origin.url"], {
      cwd: ROOT,
      encoding: "utf8",
    }).trim();
    // Support the common GitHub HTTPS and SSH remote formats:
    // https://github.com/owner/repo(.git)
    // git@github.com:owner/repo(.git)
    const match = remote.match(/github\.com[:/](?<owner>[^\/]+)\/(?<repo>[^\/.]+?)(?:\.git)?$/);
    if (match?.groups?.owner && match.groups.repo) {
      return `${match.groups.owner}/${match.groups.repo}`;
    }
  } catch {
    // Fall back to the canonical public repository path.
  }

  return "github/gh-aw";
}

function getGitRef() {
  if (process.env.GITHUB_SHA) {
    return process.env.GITHUB_SHA;
  }

  try {
    return execFileSync("git", ["rev-parse", "HEAD"], { cwd: ROOT, encoding: "utf8" }).trim();
  } catch {
    throw new Error("Unable to determine the current git ref. Set GITHUB_SHA or run this script from a git checkout.");
  }
}

async function readPdfBytes() {
  const bytes = fs.readFileSync(SOURCE_PATH);
  if (isPdf(bytes)) {
    return bytes;
  }

  if (!bytes.toString("utf8").startsWith(LFS_POINTER_PREFIX)) {
    throw new Error(`${SOURCE_PATH} is neither a PDF nor a Git LFS pointer.`);
  }

  const ref = getGitRef();
  const repositoryPath = getRepositoryPath();
  const url = `https://media.githubusercontent.com/media/${repositoryPath}/${ref}/docs/slides/github-agentic-workflows.pdf`;

  console.warn(`Detected Git LFS pointer at ${SOURCE_PATH}; downloading ${url}`);

  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to download slide deck PDF: ${response.status} ${response.statusText}`);
  }

  const downloadedBytes = Buffer.from(await response.arrayBuffer());
  if (!isPdf(downloadedBytes)) {
    throw new Error(`Downloaded slide deck from ${url} is not a real PDF.`);
  }

  return downloadedBytes;
}

async function main() {
  const pdfBytes = await readPdfBytes();
  fs.mkdirSync(path.dirname(OUTPUT_PATH), { recursive: true });
  fs.writeFileSync(OUTPUT_PATH, pdfBytes);
  console.log(`✓ Slide PDF ready at ${OUTPUT_PATH}`);
}

main().catch(error => {
  console.error(error);
  process.exit(1);
});
