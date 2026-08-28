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
const MAX_PDF_BYTES = 50 * 1024 * 1024; // 50 MB
const TRUSTED_MEDIA_ORIGIN = "https://media.githubusercontent.com";
const SLIDE_DECK_RELATIVE_PATH = "docs/slides/github-agentic-workflows.pdf";

export function isPdf(buffer) {
  return buffer.subarray(0, 5).toString("utf8") === "%PDF-";
}

export function validatePdfBytes(buffer, sourceDescription) {
  if (!Buffer.isBuffer(buffer)) {
    throw new Error(`${sourceDescription} did not produce a Buffer.`);
  }
  if (buffer.length === 0) {
    throw new Error(`${sourceDescription} is empty.`);
  }
  if (buffer.length > MAX_PDF_BYTES) {
    throw new Error(`${sourceDescription} size ${buffer.length} exceeds limit of ${MAX_PDF_BYTES} bytes`);
  }
  if (!isPdf(buffer)) {
    throw new Error(`${sourceDescription} is not a real PDF.`);
  }
  return buffer;
}

export function isAllowedPdfContentType(contentType) {
  const mediaType = contentType.split(";", 1)[0].trim().toLowerCase();
  return mediaType === "application/pdf" || mediaType === "application/octet-stream";
}

export function validateSlideDeckResponse(response) {
  const contentType = response.headers.get("content-type") ?? "";
  if (!isAllowedPdfContentType(contentType)) {
    throw new Error(`Unexpected content-type for slide deck: ${contentType}`);
  }

  const contentLength = response.headers.get("content-length");
  if (contentLength !== null) {
    const normalizedContentLength = contentLength.trim();
    if (!/^\d+$/.test(normalizedContentLength)) {
      throw new Error(`Unexpected content-length for slide deck: ${contentLength}`);
    }
    const expectedBytes = Number(normalizedContentLength);
    if (!Number.isSafeInteger(expectedBytes) || expectedBytes > MAX_PDF_BYTES) {
      throw new Error(`Slide deck download size ${contentLength} exceeds limit of ${MAX_PDF_BYTES} bytes`);
    }
  }
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

export function buildSlideDeckUrl(repositoryPath, ref) {
  // Validate each URL component before interpolating into the request URL.
  // getGitRef() always returns a 40-character hex commit SHA (from GITHUB_SHA
  // or `git rev-parse HEAD`).
  const safeSHAPattern = /^[0-9a-f]{40}$/i;
  if (!safeSHAPattern.test(ref)) {
    throw new Error(`Unsafe git ref value: ${ref}`);
  }
  // Repository path must be exactly "owner/repo" with no dot-only path components.
  const safeRepoPattern = /^[a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+$/;
  if (!safeRepoPattern.test(repositoryPath) || repositoryPath.split("/").some(p => /^[.]+$/.test(p))) {
    throw new Error(`Unsafe repository path value: ${repositoryPath}`);
  }

  const url = new URL(`/media/${repositoryPath}/${ref}/${SLIDE_DECK_RELATIVE_PATH}`, TRUSTED_MEDIA_ORIGIN);
  if (url.origin !== TRUSTED_MEDIA_ORIGIN) {
    throw new Error(`Unsafe slide deck origin: ${url.origin}`);
  }
  return url.toString();
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
    return validatePdfBytes(bytes, SOURCE_PATH);
  }

  if (!bytes.toString("utf8").startsWith(LFS_POINTER_PREFIX)) {
    throw new Error(`${SOURCE_PATH} is neither a PDF nor a Git LFS pointer.`);
  }

  const ref = getGitRef();
  const repositoryPath = getRepositoryPath();
  const url = buildSlideDeckUrl(repositoryPath, ref);

  console.warn(`Detected Git LFS pointer at ${SOURCE_PATH}; downloading ${url}`);

  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Failed to download slide deck PDF: ${response.status} ${response.statusText}`);
    }

    validateSlideDeckResponse(response);

    const downloadedBytes = Buffer.from(await response.arrayBuffer());
    return validatePdfBytes(downloadedBytes, `Downloaded slide deck from ${url}`);
  } catch (error) {
    console.warn(`Warning: Could not download slide deck PDF (${error.message}). Using placeholder PDF.`);
    return validatePdfBytes(createPlaceholderPdfBytes(), "Placeholder slide deck");
  }
}

async function main() {
  const pdfBytes = await readPdfBytes();
  fs.mkdirSync(path.dirname(OUTPUT_PATH), { recursive: true });
  // codeql[js/http-to-file-access]: readPdfBytes() only ever returns bytes from a
  // fixed, hardcoded GitHub media URL (constructed from validated repo/ref values)
  // after checking the response content-type, enforcing a size limit, and
  // verifying the PDF file signature — or a locally generated placeholder PDF.
  // Neither path writes arbitrary/unvalidated network data to disk.
  fs.writeFileSync(OUTPUT_PATH, pdfBytes);
  console.log(`✓ Slide PDF ready at ${OUTPUT_PATH}`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === __filename) {
  main().catch(error => {
    console.error(error);
    process.exit(1);
  });
}
