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

/**
 * Creates a minimal valid single-page PDF placeholder used when the real slide
 * deck cannot be fetched (e.g. in sandboxed dev/test environments without LFS
 * or media.githubusercontent.com access).  The placeholder carries the valid
 * PDF header so pdfjs-dist can parse it without throwing InvalidPDFException.
 */
function createPlaceholderPdfBytes() {
  const parts = [];
  const offsets = [];

  function write(str) {
    parts.push(Buffer.from(str, "latin1"));
  }

  function currentOffset() {
    return parts.reduce((sum, buf) => sum + buf.length, 0);
  }

  write("%PDF-1.4\n");

  offsets[1] = currentOffset();
  write("1 0 obj\n<</Type /Catalog /Pages 2 0 R>>\nendobj\n");

  offsets[2] = currentOffset();
  write("2 0 obj\n<</Type /Pages /Kids [3 0 R] /Count 1>>\nendobj\n");

  offsets[3] = currentOffset();
  write("3 0 obj\n<</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792]>>\nendobj\n");

  const xrefOffset = currentOffset();
  write("xref\n0 4\n");
  write("0000000000 65535 f \n");
  write(offsets[1].toString().padStart(10, "0") + " 00000 n \n");
  write(offsets[2].toString().padStart(10, "0") + " 00000 n \n");
  write(offsets[3].toString().padStart(10, "0") + " 00000 n \n");
  write("trailer\n<</Size 4 /Root 1 0 R>>\n");
  write("startxref\n" + xrefOffset + "\n%%EOF\n");

  return Buffer.concat(parts);
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

  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Failed to download slide deck PDF: ${response.status} ${response.statusText}`);
    }

    // Validate the Content-Type header before consuming the body to ensure we
    // are actually receiving a PDF and not arbitrary data.
    const contentType = response.headers.get("content-type") ?? "";
    if (!contentType.startsWith("application/pdf") && !contentType.startsWith("application/octet-stream")) {
      throw new Error(`Unexpected content-type for slide deck: ${contentType}`);
    }

    // Guard against unexpectedly large downloads.
    const MAX_BYTES = 50 * 1024 * 1024; // 50 MB
    const contentLength = response.headers.get("content-length");
    if (contentLength !== null && Number(contentLength) > MAX_BYTES) {
      throw new Error(`Slide deck download size ${contentLength} exceeds limit of ${MAX_BYTES} bytes`);
    }

    const downloadedBytes = Buffer.from(await response.arrayBuffer());
    if (downloadedBytes.length > MAX_BYTES) {
      throw new Error(`Downloaded slide deck size ${downloadedBytes.length} exceeds limit of ${MAX_BYTES} bytes`);
    }

    if (!isPdf(downloadedBytes)) {
      throw new Error(`Downloaded slide deck from ${url} is not a real PDF.`);
    }

    return downloadedBytes;
  } catch (error) {
    console.warn(`Warning: Could not download slide deck PDF (${error.message}). Using placeholder PDF.`);
    return createPlaceholderPdfBytes();
  }
}

async function main() {
  const pdfBytes = await readPdfBytes();
  fs.mkdirSync(path.dirname(OUTPUT_PATH), { recursive: true });
  // codeql[js/http-to-file-access]: readPdfBytes() only downloads from a fixed GitHub media URL and validates content type, size, and PDF signature before this write.
  fs.writeFileSync(OUTPUT_PATH, pdfBytes);
  console.log(`✓ Slide PDF ready at ${OUTPUT_PATH}`);
}

main().catch(error => {
  console.error(error);
  process.exit(1);
});
