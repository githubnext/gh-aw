// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const https = require("https");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Return the MIME type for a given file extension.
 * @param {string} fileName - File name (extension is used)
 * @returns {string} MIME type string
 */
function getMimeType(fileName) {
  const ext = path.extname(fileName).toLowerCase();
  switch (ext) {
    case ".png":
      return "image/png";
    case ".jpg":
    case ".jpeg":
      return "image/jpeg";
    case ".gif":
      return "image/gif";
    case ".webp":
      return "image/webp";
    case ".svg":
      return "image/svg+xml";
    case ".pdf":
      return "application/pdf";
    default:
      return "application/octet-stream";
  }
}

/**
 * Parse the ACTIONS_RUNTIME_TOKEN JWT to extract the GitHub Actions backend IDs.
 * The `scp` claim contains: "Actions.Results:{workflowRunBackendId}:{workflowJobRunBackendId}"
 * @param {string} token - JWT token from ACTIONS_RUNTIME_TOKEN env var
 * @returns {{ workflowRunBackendId: string, workflowJobRunBackendId: string }}
 */
function getBackendIdsFromToken(token) {
  const parts = token.split(".");
  if (parts.length < 2) {
    throw new Error("Invalid JWT token format");
  }
  // Base64url decode the payload
  const payload = Buffer.from(parts[1], "base64url").toString("utf8");
  const decoded = JSON.parse(payload);
  const scp = decoded.scp;
  if (!scp || typeof scp !== "string") {
    throw new Error("JWT token missing scp claim");
  }
  const prefix = "Actions.Results:";
  if (!scp.startsWith(prefix)) {
    throw new Error(`JWT scp claim does not start with expected prefix '${prefix}'`);
  }
  const ids = scp.slice(prefix.length).split(":");
  if (ids.length < 2) {
    throw new Error("JWT scp claim does not contain expected backend IDs");
  }
  return {
    workflowRunBackendId: ids[0],
    workflowJobRunBackendId: ids[1],
  };
}

/**
 * Perform an HTTPS POST request with a JSON body and return the parsed JSON response.
 * @param {string} url - Full HTTPS URL
 * @param {string} bearerToken - Authorization bearer token
 * @param {object} body - Request body (will be JSON-stringified)
 * @returns {Promise<object>} Parsed JSON response body
 */
function httpsPostJson(url, bearerToken, body) {
  return new Promise((resolve, reject) => {
    const bodyStr = JSON.stringify(body);
    const parsedUrl = new URL(url);
    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || 443,
      path: parsedUrl.pathname + parsedUrl.search,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Content-Length": Buffer.byteLength(bodyStr),
        Authorization: `Bearer ${bearerToken}`,
      },
    };
    const req = https.request(options, res => {
      let data = "";
      res.on("data", chunk => (data += chunk));
      res.on("end", () => {
        if (res.statusCode && res.statusCode >= 200 && res.statusCode < 300) {
          try {
            resolve(JSON.parse(data));
          } catch (e) {
            reject(new Error(`Failed to parse response JSON: ${data}`));
          }
        } else {
          reject(new Error(`HTTP ${res.statusCode}: ${data}`));
        }
      });
    });
    req.on("error", reject);
    req.write(bodyStr);
    req.end();
  });
}

/**
 * Upload file content to a blob storage URL using HTTP PUT.
 * @param {string} signedUrl - Pre-signed upload URL
 * @param {Buffer} fileContent - File data to upload
 * @param {string} mimeType - MIME content type
 * @param {string} sha256Hash - Hex SHA-256 of fileContent
 * @returns {Promise<void>}
 */
function uploadToBlobStorage(signedUrl, fileContent, mimeType, sha256Hash) {
  return new Promise((resolve, reject) => {
    const parsedUrl = new URL(signedUrl);
    // Azure Blob Storage uses PUT; the scheme may be http or https
    const lib = parsedUrl.protocol === "https:" ? https : require("http");
    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || (parsedUrl.protocol === "https:" ? 443 : 80),
      path: parsedUrl.pathname + parsedUrl.search,
      method: "PUT",
      headers: {
        "Content-Type": mimeType,
        "Content-Length": fileContent.length,
        "x-ms-blob-type": "BlockBlob",
        "x-ms-blob-content-type": mimeType,
        // Include SHA-256 checksum so Azure validates integrity
        "x-ms-meta-sha256": sha256Hash,
      },
    };
    const req = lib.request(options, res => {
      let data = "";
      res.on("data", chunk => (data += chunk));
      res.on("end", () => {
        if (res.statusCode && res.statusCode >= 200 && res.statusCode < 300) {
          resolve();
        } else {
          reject(new Error(`Blob upload HTTP ${res.statusCode}: ${data}`));
        }
      });
    });
    req.on("error", reject);
    req.write(fileContent);
    req.end();
  });
}

/**
 * Upload a single file as an unzipped (archive:false) GitHub Actions artifact.
 * Uses the GitHub Actions Twirp backend API (v7, skipArchive).
 *
 * @param {string} artifactName - Name of the artifact (typically the file name)
 * @param {string} filePath - Absolute path to the file to upload
 * @param {string} mimeType - MIME type of the file
 * @param {string} resultsUrl - Value of ACTIONS_RESULTS_URL env var
 * @param {string} runtimeToken - Value of ACTIONS_RUNTIME_TOKEN env var
 * @returns {Promise<string>} The artifact ID string
 */
async function uploadArtifact(artifactName, filePath, mimeType, resultsUrl, runtimeToken) {
  const backendIds = getBackendIdsFromToken(runtimeToken);
  const baseUrl = resultsUrl.endsWith("/") ? resultsUrl : resultsUrl + "/";
  const twirpBase = `${baseUrl}twirp/github.actions.results.api.v1.ArtifactService/`;

  // Step 1: Create artifact (version 7 = unzipped / archive:false support)
  core.info(`Creating artifact: ${artifactName}`);
  const createResp = await httpsPostJson(`${twirpBase}CreateArtifact`, runtimeToken, {
    workflow_run_backend_id: backendIds.workflowRunBackendId,
    workflow_job_run_backend_id: backendIds.workflowJobRunBackendId,
    name: artifactName,
    mime_type: { value: mimeType },
    version: 7,
  });
  if (!createResp.ok) {
    throw new Error(`CreateArtifact failed: ${JSON.stringify(createResp)}`);
  }
  const signedUploadUrl = createResp.signed_upload_url;
  if (!signedUploadUrl) {
    throw new Error(`CreateArtifact returned no signed_upload_url: ${JSON.stringify(createResp)}`);
  }

  // Step 2: Upload the raw file to the signed URL (no zipping)
  const fileContent = fs.readFileSync(filePath);
  const sha256Hash = crypto.createHash("sha256").update(fileContent).digest("hex");
  core.info(`Uploading ${artifactName} (${fileContent.length} bytes) to blob storage`);
  await uploadToBlobStorage(signedUploadUrl, fileContent, mimeType, sha256Hash);

  // Step 3: Finalize the artifact to make it visible
  core.info(`Finalizing artifact: ${artifactName}`);
  const finalizeResp = await httpsPostJson(`${twirpBase}FinalizeArtifact`, runtimeToken, {
    workflow_run_backend_id: backendIds.workflowRunBackendId,
    workflow_job_run_backend_id: backendIds.workflowJobRunBackendId,
    name: artifactName,
    size: fileContent.length.toString(),
    hash: { value: `sha256:${sha256Hash}` },
  });
  if (!finalizeResp.ok) {
    throw new Error(`FinalizeArtifact failed: ${JSON.stringify(finalizeResp)}`);
  }
  const artifactId = finalizeResp.artifact_id;
  if (!artifactId) {
    throw new Error(`FinalizeArtifact returned no artifact_id: ${JSON.stringify(finalizeResp)}`);
  }
  return String(artifactId);
}

/**
 * Build the browser-accessible artifact URL for a given artifact ID.
 * @param {string} artifactId - The artifact ID returned by FinalizeArtifact
 * @returns {string} URL to view the artifact in the GitHub UI
 */
function buildArtifactUrl(artifactId) {
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const repo = process.env.GITHUB_REPOSITORY || "owner/repo";
  const runId = process.env.GITHUB_RUN_ID || "0";
  return `${githubServer}/${repo}/actions/runs/${runId}/artifacts/${artifactId}`;
}

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    core.setOutput("upload_count", "0");
    core.setOutput("asset_url_map", "{}");
    return;
  }

  // Find all upload-asset items
  const uploadItems = result.items.filter(/** @param {any} item */ item => item.type === "upload_asset");

  if (uploadItems.length === 0) {
    core.info("No upload-asset items found in agent output");
    core.setOutput("upload_count", "0");
    core.setOutput("asset_url_map", "{}");
    return;
  }

  core.info(`Found ${uploadItems.length} upload-asset item(s)`);

  // In non-staged mode, GitHub Actions backend environment variables are required for artifact upload.
  // We validate them before processing any items to fail fast.
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";
  const resultsUrl = process.env.ACTIONS_RESULTS_URL;
  const runtimeToken = process.env.ACTIONS_RUNTIME_TOKEN;
  if (!isStaged && (!resultsUrl || !runtimeToken)) {
    core.setFailed(`${ERR_SYSTEM}: ACTIONS_RESULTS_URL and ACTIONS_RUNTIME_TOKEN are required for artifact upload`);
    return;
  }

  let uploadCount = 0;
  /** @type {Record<string, string>} Maps temporaryId -> artifact URL */
  const assetUrlMap = {};

  for (const asset of uploadItems) {
    const { fileName, sha, size, temporaryId } = asset;

    if (!fileName || !sha || !temporaryId) {
      core.setFailed(`${ERR_VALIDATION}: Invalid asset entry missing required fields (fileName, sha, temporaryId): ${JSON.stringify(asset)}`);
      return;
    }

    // Check if file exists in the downloaded assets directory
    const assetSourcePath = path.join("/tmp/gh-aw/safeoutputs/assets", fileName);
    if (!fs.existsSync(assetSourcePath)) {
      core.setFailed(`${ERR_SYSTEM}: Asset file not found: ${assetSourcePath}`);
      return;
    }

    // Verify SHA integrity
    const fileContent = fs.readFileSync(assetSourcePath);
    const computedSha = crypto.createHash("sha256").update(fileContent).digest("hex");
    if (computedSha !== sha) {
      core.setFailed(`${ERR_VALIDATION}: SHA mismatch for ${fileName}: expected ${sha}, got ${computedSha}`);
      return;
    }

    // Skip upload in staged mode
    if (isStaged) {
      core.info(`🎭 Staged mode: skipping artifact upload for ${fileName}`);
      assetUrlMap[temporaryId] = `aw://staged/${temporaryId}`;
      uploadCount++;
      continue;
    }

    // Upload the file as an unzipped artifact using the GitHub Actions backend API
    const mimeType = getMimeType(fileName);
    // Use sha-prefixed artifact name to keep uploads idempotent
    const artifactName = `aw-asset-${sha.slice(0, 16)}-${fileName}`;
    let artifactId;
    try {
      artifactId = await uploadArtifact(artifactName, assetSourcePath, mimeType, resultsUrl, runtimeToken);
    } catch (error) {
      core.setFailed(`${ERR_API}: Failed to upload artifact for ${fileName}: ${getErrorMessage(error)}`);
      return;
    }

    const artifactUrl = buildArtifactUrl(artifactId);
    assetUrlMap[temporaryId] = artifactUrl;
    uploadCount++;
    core.info(`Uploaded asset: ${fileName} (${size} bytes) → ${artifactUrl}`);
  }

  // Emit job summary
  if (uploadCount > 0) {
    if (isStaged) {
      core.summary.addRaw("## 🎭 Staged Mode: Asset Upload Preview");
    } else {
      core.summary.addRaw("## Assets").addRaw(`Successfully uploaded **${uploadCount}** asset(s) as GitHub Actions artifacts`).addRaw("");
    }
    for (const [tempId, url] of Object.entries(assetUrlMap)) {
      core.summary.addRaw(`- \`${tempId}\` → [artifact](${url})`);
    }
    await core.summary.write();
  } else {
    core.info("No new assets to upload");
  }

  core.setOutput("upload_count", uploadCount.toString());
  core.setOutput("asset_url_map", JSON.stringify(assetUrlMap));
}

module.exports = { main, getMimeType, getBackendIdsFromToken, buildArtifactUrl };
