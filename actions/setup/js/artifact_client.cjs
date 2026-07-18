// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @safe-outputs-exempt SEC-004: "body" references are HTTP transport payloads
 * (Twirp RPC request JSON bodies and artifact upload stream bodies), not
 * user-authored issue/PR/comment bodies.
 */

const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { Readable } = require("stream");
const { pipeline } = require("stream/promises");
const { spawnSync } = require("child_process");

const { getErrorMessage } = require("./error_helpers.cjs");

const DEFAULT_RETRY_ATTEMPTS = 5;
const RETRY_DELAY_MS = 5000;
const RESULTS_SCOPE_PREFIX = "Actions.Results:";
const TWIRP_ARTIFACT_SERVICE = "github.actions.results.api.v1.ArtifactService";
const MAX_ARTIFACTS = 1000;
const PAGE_SIZE = 100;

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

function decodeJWTPayload(token) {
  const parts = String(token || "").split(".");
  if (parts.length < 2 || !parts[1]) {
    throw new Error("failed to decode ACTIONS_RUNTIME_TOKEN payload");
  }
  const payload = parts[1].replace(/-/g, "+").replace(/_/g, "/");
  const padded = payload + "=".repeat((4 - (payload.length % 4 || 4)) % 4);
  let parsed;
  try {
    parsed = JSON.parse(Buffer.from(padded, "base64").toString("utf8"));
  } catch (err) {
    throw new Error("Failed to parse JWT payload: " + getErrorMessage(err), { cause: err });
  }
  return parsed;
}

function getBackendIdsFromRuntimeToken() {
  const token = process.env.ACTIONS_RUNTIME_TOKEN || "";
  if (!token) {
    throw new Error("ACTIONS_RUNTIME_TOKEN is required for artifact upload");
  }
  const payload = decodeJWTPayload(token);
  const scope = String(payload?.scp || "");
  for (const part of scope.split(" ")) {
    if (!part.startsWith(RESULTS_SCOPE_PREFIX)) continue;
    const ids = part.split(":");
    if (ids.length !== 3 || !ids[1] || !ids[2]) {
      break;
    }
    return {
      workflowRunBackendId: ids[1],
      workflowJobRunBackendId: ids[2],
    };
  }
  throw new Error("failed to parse Actions.Results backend IDs from ACTIONS_RUNTIME_TOKEN");
}

function getResultsServiceOrigin() {
  const url = process.env.ACTIONS_RESULTS_URL || "";
  if (!url) {
    throw new Error("ACTIONS_RESULTS_URL is required for artifact upload");
  }
  return new URL(url).origin;
}

async function twirpRequest(method, body) {
  const runtimeToken = process.env.ACTIONS_RUNTIME_TOKEN || "";
  if (!runtimeToken) {
    throw new Error("ACTIONS_RUNTIME_TOKEN is required for artifact upload");
  }
  const url = new URL(`/twirp/${TWIRP_ARTIFACT_SERVICE}/${method}`, getResultsServiceOrigin()).toString();

  let lastError;
  for (let attempt = 1; attempt <= DEFAULT_RETRY_ATTEMPTS; attempt++) {
    try {
      const response = await fetch(url, {
        method: "POST",
        headers: {
          Authorization: "Bearer " + runtimeToken,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
      });

      if (response.ok) {
        return await response.json();
      }

      const responseBody = await response.text();
      const retryable = response.status >= 500 || response.status === 429;
      if (!retryable || attempt === DEFAULT_RETRY_ATTEMPTS) {
        throw new Error(`artifact twirp ${method} failed (${response.status}): ${responseBody || response.statusText}`);
      }
      await sleep(RETRY_DELAY_MS);
    } catch (error) {
      lastError = error;
      if (attempt === DEFAULT_RETRY_ATTEMPTS) {
        break;
      }
      await sleep(RETRY_DELAY_MS);
    }
  }

  throw lastError || new Error(`artifact twirp ${method} failed`);
}

function artifactListFilterLatest(artifacts) {
  const sorted = [...artifacts].sort((a, b) => Number(b.id || 0) - Number(a.id || 0));
  const seen = new Set();
  /** @type {any[]} */
  const latest = [];
  for (const artifact of sorted) {
    if (seen.has(artifact.name)) continue;
    seen.add(artifact.name);
    latest.push(artifact);
  }
  return latest;
}

function parseFilenameFromContentDisposition(contentDisposition) {
  if (!contentDisposition) return "artifact";
  const filenameStar = contentDisposition.match(/filename\*\s*=\s*UTF-8''([^;\r\n]*)/i);
  const filenamePlain = contentDisposition.match(/(?<!\*)filename\s*=\s*['"]?([^;\r\n"']*)['"]?/i);
  const raw = filenameStar?.[1] || filenamePlain?.[1];
  if (!raw) return "artifact";
  return path.basename(decodeURIComponent(raw.trim())) || "artifact";
}

function isZipResponse(url, contentType) {
  const mime = String(contentType || "")
    .split(";", 1)[0]
    .trim()
    .toLowerCase();
  if (mime === "application/zip" || mime === "application/x-zip-compressed" || mime === "application/zip-compressed") {
    return true;
  }
  try {
    return new URL(url).pathname.toLowerCase().endsWith(".zip");
  } catch {
    return false;
  }
}

async function streamToFile(response, filePath) {
  const nodeStream = Readable.fromWeb(response.body);
  const hash = crypto.createHash("sha256");
  nodeStream.on("data", chunk => hash.update(chunk));
  await pipeline(nodeStream, fs.createWriteStream(filePath));
  return hash.digest("hex");
}

function ensureZipAvailable() {
  const result = spawnSync("zip", ["-v"], { stdio: "ignore" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error("zip command is required to upload artifacts (for example: apt-get install zip)");
  }
}

function ensureUnzipAvailable() {
  const result = spawnSync("unzip", ["-v"], { stdio: "ignore" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error("unzip command is required to download artifacts (for example: apt-get install unzip)");
  }
}

function createZipFromFiles(files, rootDirectory, outputPath) {
  ensureZipAvailable();
  const relativeFiles = files.map(file => path.relative(rootDirectory, file));
  const invalid = relativeFiles.find(rel => !rel || rel.startsWith("..") || path.isAbsolute(rel));
  if (invalid) {
    throw new Error(`all upload artifact files must be under rootDirectory (invalid path: ${invalid})`);
  }
  const result = spawnSync("zip", ["-q", "-r", outputPath, ...relativeFiles], {
    cwd: rootDirectory,
    encoding: "utf8",
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`zip command failed: ${result.stderr || result.stdout || "unknown error"}`);
  }
}

async function uploadFileToSignedURL(filePath, signedUploadURL, contentType) {
  const stats = fs.statSync(filePath);
  const response = await fetch(signedUploadURL, {
    method: "PUT",
    headers: {
      "Content-Type": contentType,
      "Content-Length": String(stats.size),
      "x-ms-blob-type": "BlockBlob",
    },
    body: fs.createReadStream(filePath),
    duplex: "half",
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(`artifact blob upload failed (${response.status}): ${body || response.statusText}`);
  }
  return stats.size;
}

async function hashFile(filePath) {
  const hash = crypto.createHash("sha256");
  const nodeStream = fs.createReadStream(filePath);
  await pipeline(nodeStream, async function* (source) {
    for await (const chunk of source) {
      hash.update(chunk);
    }
  });
  return hash.digest("hex");
}

function formatRetentionTimestamp(retentionDays) {
  if (!Number.isFinite(retentionDays) || retentionDays <= 0) {
    return "";
  }
  return new Date(Date.now() + retentionDays * 24 * 60 * 60 * 1000).toISOString();
}

class DefaultArtifactClient {
  async listArtifacts(options = {}) {
    const findBy = options.findBy;
    if (!findBy?.token || !findBy?.repositoryOwner || !findBy?.repositoryName || !findBy?.workflowRunId) {
      throw new Error("listArtifacts requires findBy.token, findBy.repositoryOwner, findBy.repositoryName, and findBy.workflowRunId");
    }

    const serverUrl = process.env.GITHUB_API_URL || "https://api.github.com";
    /** @type {Array<{id:number,name:string,size:number,createdAt?:Date,digest?:string}>} */
    const artifacts = [];

    let page = 1;
    const maxPages = Math.ceil(MAX_ARTIFACTS / PAGE_SIZE);
    for (; page <= maxPages; page++) {
      const url = new URL(`/repos/${findBy.repositoryOwner}/${findBy.repositoryName}/actions/runs/${findBy.workflowRunId}/artifacts`, serverUrl);
      url.searchParams.set("per_page", String(PAGE_SIZE));
      url.searchParams.set("page", String(page));
      const response = await fetch(url.toString(), {
        headers: {
          Authorization: "Bearer " + findBy.token,
          Accept: "application/vnd.github+json",
          "User-Agent": "gh-aw-artifact-client",
        },
      });
      if (!response.ok) {
        throw new Error(`failed to list artifacts (${response.status}): ${await response.text()}`);
      }
      /** @type {any} */
      const payload = await response.json();
      const pageArtifacts = Array.isArray(payload?.artifacts) ? payload.artifacts : [];
      for (const item of pageArtifacts) {
        artifacts.push({
          id: Number(item.id),
          name: String(item.name || ""),
          size: Number(item.size_in_bytes || 0),
          createdAt: item.created_at ? new Date(item.created_at) : undefined,
          digest: typeof item.digest === "string" ? item.digest : undefined,
        });
      }
      if (pageArtifacts.length < PAGE_SIZE) {
        break;
      }
    }

    return {
      artifacts: options.latest ? artifactListFilterLatest(artifacts) : artifacts,
    };
  }

  async downloadArtifact(artifactId, options = {}) {
    const findBy = options.findBy;
    if (!findBy?.token || !findBy?.repositoryOwner || !findBy?.repositoryName) {
      throw new Error("downloadArtifact requires findBy.token, findBy.repositoryOwner, and findBy.repositoryName");
    }

    const destination = options.path || process.env.GITHUB_WORKSPACE || process.cwd();
    try {
      fs.mkdirSync(destination, { recursive: true });
    } catch (err) {
      throw new Error(`Failed to create directory ${destination}: ${String(err)}`, { cause: err });
    }

    const apiUrl = new URL(`/repos/${findBy.repositoryOwner}/${findBy.repositoryName}/actions/artifacts/${artifactId}/zip`, process.env.GITHUB_API_URL || "https://api.github.com");
    const redirectResponse = await fetch(apiUrl.toString(), {
      headers: {
        Authorization: "Bearer " + findBy.token,
        Accept: "application/vnd.github+json",
        "User-Agent": "gh-aw-artifact-client",
      },
      redirect: "manual",
    });
    if (![301, 302, 303, 307, 308].includes(redirectResponse.status)) {
      throw new Error(`unable to download artifact: unexpected status ${redirectResponse.status}`);
    }
    const location = redirectResponse.headers.get("location");
    if (!location) {
      throw new Error("unable to download artifact: missing redirect location");
    }

    const blobResponse = await fetch(location);
    if (!blobResponse.ok) {
      throw new Error(`artifact blob download failed (${blobResponse.status})`);
    }

    let digest;
    const contentType = blobResponse.headers.get("content-type") || "";
    const zipLike = isZipResponse(location, contentType);
    if (zipLike && !options.skipDecompress) {
      ensureUnzipAvailable();
      const tempZip = path.join(fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-artifact-download-")), "artifact.zip");
      digest = await streamToFile(blobResponse, tempZip);
      const unzipResult = spawnSync("unzip", ["-q", tempZip, "-d", destination], { encoding: "utf8" });
      if (unzipResult.error) {
        throw unzipResult.error;
      }
      if (unzipResult.status !== 0) {
        throw new Error(`unzip failed: ${unzipResult.stderr || unzipResult.stdout || "unknown error"}`);
      }
    } else {
      const fileName = parseFilenameFromContentDisposition(blobResponse.headers.get("content-disposition") || "");
      const outputPath = path.join(destination, fileName);
      digest = await streamToFile(blobResponse, outputPath);
    }

    const computed = digest ? `sha256:${digest}` : "";
    return {
      downloadPath: destination,
      digestMismatch: !!(computed && options.expectedHash && options.expectedHash !== computed),
    };
  }

  async uploadArtifact(name, files, rootDirectory, options = {}) {
    if (!Array.isArray(files) || files.length === 0) {
      throw new Error("uploadArtifact requires at least one file");
    }

    let artifactName = String(name || "").trim();
    let uploadPath = "";
    let contentType = "application/zip";

    if (options.skipArchive) {
      if (files.length !== 1) {
        throw new Error("skipArchive option is only supported when uploading a single file");
      }
      uploadPath = files[0];
      contentType = "application/octet-stream";
    } else {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-artifact-upload-"));
      uploadPath = path.join(tmpDir, `${artifactName || "artifact"}.zip`);
      createZipFromFiles(files, rootDirectory, uploadPath);
    }

    const { workflowRunBackendId, workflowJobRunBackendId } = getBackendIdsFromRuntimeToken();
    const createRequest = {
      workflowRunBackendId,
      workflowJobRunBackendId,
      name: artifactName,
      version: 7,
      mimeType: contentType,
    };
    const expiresAt = formatRetentionTimestamp(options.retentionDays);
    if (expiresAt) {
      createRequest.expiresAt = expiresAt;
    }

    /** @type {any} */
    const createResponse = await twirpRequest("CreateArtifact", createRequest);
    const signedUploadUrl = createResponse?.signedUploadUrl || createResponse?.signed_upload_url;
    if (!createResponse?.ok || !signedUploadUrl) {
      throw new Error("CreateArtifact returned an invalid response");
    }

    const uploadSize = await uploadFileToSignedURL(uploadPath, signedUploadUrl, contentType);
    const sha256 = await hashFile(uploadPath);

    const finalizeRequest = {
      workflowRunBackendId,
      workflowJobRunBackendId,
      name: artifactName,
      size: String(uploadSize),
      hash: `sha256:${sha256}`,
    };
    /** @type {any} */
    const finalizeResponse = await twirpRequest("FinalizeArtifact", finalizeRequest);
    if (!finalizeResponse?.ok) {
      throw new Error("FinalizeArtifact returned an invalid response");
    }

    return {
      id: Number(finalizeResponse.artifactId ?? finalizeResponse.artifact_id ?? 0) || undefined,
      size: uploadSize,
      digest: sha256,
    };
  }
}

module.exports = {
  DefaultArtifactClient,
};
