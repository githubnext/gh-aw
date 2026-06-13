// @ts-check
/**
 * Integration tests for DefaultArtifactClient using the live GitHub Actions artifact APIs.
 *
 * These tests are skipped when the required environment variables are absent
 * (i.e. outside of a real GitHub Actions job). When running inside GHA the
 * runner provides ACTIONS_RUNTIME_TOKEN and ACTIONS_RESULTS_URL automatically,
 * and GITHUB_TOKEN is injected via the workflow step's `env:` block.
 *
 * Each test cross-validates our custom DefaultArtifactClient against the official
 * @actions/artifact SDK:
 *   - Test 1: upload with our client → list + download with @actions/artifact SDK
 *   - Test 2: upload with @actions/artifact SDK → list + download with our client
 *
 * Run locally against a real GHA environment with:
 *   cd actions/setup/js && npm run test:js-integration-artifact
 */

import { afterAll, beforeAll, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
/** @type {{ DefaultArtifactClient: any }} */
const { DefaultArtifactClient: OurArtifactClient } = req("./artifact_client.cjs");

// Official @actions/artifact SDK — imported as ESM since the package is ESM-only.
const sdkModule = await import("@actions/artifact");
/** @type {import("@actions/artifact").DefaultArtifactClient} */
const SdkArtifactClientClass = sdkModule.DefaultArtifactClient;

// These are set automatically by the GitHub Actions runner for the current job.
const HAVE_ACTIONS_RUNTIME = !!(process.env.ACTIONS_RUNTIME_TOKEN && process.env.ACTIONS_RESULTS_URL);
// GITHUB_TOKEN is injected via env: in the workflow step.
const HAVE_GITHUB_TOKEN = !!(process.env.GITHUB_TOKEN || process.env.GH_TOKEN);

/** @returns {string} */
function getToken() {
  return process.env.GITHUB_TOKEN || process.env.GH_TOKEN || "";
}

/** @returns {number} */
function getWorkflowRunId() {
  return Number(process.env.GITHUB_RUN_ID || "0");
}

/** @returns {{ repositoryOwner: string, repositoryName: string }} */
function getRepoInfo() {
  const [owner = "", repositoryName = ""] = (process.env.GITHUB_REPOSITORY || "/").split("/");
  return { repositoryOwner: owner, repositoryName };
}

/**
 * Build findBy options for our custom client (token + run + repo).
 * @returns {{ token: string, repositoryOwner: string, repositoryName: string, workflowRunId: string }}
 */
function makeOurFindBy() {
  const { repositoryOwner, repositoryName } = getRepoInfo();
  return { token: getToken(), repositoryOwner, repositoryName, workflowRunId: String(getWorkflowRunId()) };
}

/**
 * Build findBy options for the official SDK (same fields but workflowRunId is a number).
 * @returns {{ token: string, repositoryOwner: string, repositoryName: string, workflowRunId: number }}
 */
function makeSdkFindBy() {
  const { repositoryOwner, repositoryName } = getRepoInfo();
  return { token: getToken(), repositoryOwner, repositoryName, workflowRunId: getWorkflowRunId() };
}

describe("DefaultArtifactClient live API integration", () => {
  /** @type {string} */
  let tmpDir;

  beforeAll(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-artifact-live-"));
  });

  afterAll(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("our client upload is consumable by the official @actions/artifact SDK", async () => {
    if (!HAVE_ACTIONS_RUNTIME) {
      console.log(`Skipping live artifact test — ACTIONS_RUNTIME_TOKEN / ACTIONS_RESULTS_URL not set.\nThis test only runs inside a GitHub Actions job.`);
      return;
    }
    if (!HAVE_GITHUB_TOKEN) {
      console.log(`Skipping live artifact test — GITHUB_TOKEN not set.\nPass it via env: GITHUB_TOKEN in the workflow step.`);
      return;
    }

    const ourClient = new OurArtifactClient();
    const sdkClient = new SdkArtifactClientClass();

    // ── 1. Prepare a small test file ───────────────────────────────────────
    const testFile = path.join(tmpDir, "our-upload.txt");
    const runId = process.env.GITHUB_RUN_ID || "local";
    const expectedContent = `gh-aw artifact integration test — custom upload\nrun=${runId}\n`;
    fs.writeFileSync(testFile, expectedContent, "utf8");

    const artifactName = `gh-aw-custom-upload-${runId}`;

    // ── 2. Upload with our client ───────────────────────────────────────────
    console.log(`[custom→sdk] Uploading "${artifactName}" with our client …`);
    const uploadResult = await ourClient.uploadArtifact(artifactName, [testFile], tmpDir, { skipArchive: true });

    expect(uploadResult.id).toBeTypeOf("number");
    expect(uploadResult.size).toBeGreaterThan(0);
    console.log(`  ✅ uploaded — id=${uploadResult.id} size=${uploadResult.size}`);

    // ── 3. List with official SDK and verify artifact appears ──────────────
    console.log(`[custom→sdk] Listing artifacts with SDK …`);
    const sdkListResult = await sdkClient.listArtifacts({ findBy: makeSdkFindBy() });

    expect(Array.isArray(sdkListResult.artifacts)).toBe(true);
    const found = sdkListResult.artifacts.find(a => a.id === uploadResult.id);
    expect(found, `artifact id=${uploadResult.id} should appear in SDK list`).toBeDefined();
    expect(found?.name).toBe(artifactName);
    expect(found?.size).toBeGreaterThan(0);
    console.log(`  ✅ SDK list found artifact "${found?.name}"`);

    // ── 4. Download with official SDK ──────────────────────────────────────
    const sdkDownloadDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-sdk-dl-"));
    try {
      console.log(`[custom→sdk] Downloading with SDK …`);
      const sdkDownloadResult = await sdkClient.downloadArtifact(uploadResult.id, {
        path: sdkDownloadDir,
        skipDecompress: true,
        findBy: makeSdkFindBy(),
      });

      expect(sdkDownloadResult.downloadPath).toBeDefined();

      const downloadedFiles = fs.readdirSync(sdkDownloadDir);
      expect(downloadedFiles.length).toBeGreaterThan(0);
      console.log(`  ✅ SDK downloaded ${downloadedFiles.length} file(s): ${downloadedFiles.join(", ")}`);

      const downloadedContent = fs.readFileSync(path.join(sdkDownloadDir, downloadedFiles[0]), "utf8");
      expect(downloadedContent).toBe(expectedContent);
      console.log("  ✅ SDK-downloaded content matches uploaded content");
    } finally {
      fs.rmSync(sdkDownloadDir, { recursive: true, force: true });
    }
  });

  it("official @actions/artifact SDK upload is consumable by our client", async () => {
    if (!HAVE_ACTIONS_RUNTIME) {
      console.log(`Skipping live artifact test — ACTIONS_RUNTIME_TOKEN / ACTIONS_RESULTS_URL not set.\nThis test only runs inside a GitHub Actions job.`);
      return;
    }
    if (!HAVE_GITHUB_TOKEN) {
      console.log(`Skipping live artifact test — GITHUB_TOKEN not set.\nPass it via env: GITHUB_TOKEN in the workflow step.`);
      return;
    }

    const ourClient = new OurArtifactClient();
    const sdkClient = new SdkArtifactClientClass();

    // ── 1. Prepare a small test file ───────────────────────────────────────
    const testFile = path.join(tmpDir, "sdk-upload.txt");
    const runId = process.env.GITHUB_RUN_ID || "local";
    const expectedContent = `gh-aw artifact integration test — SDK upload\nrun=${runId}\n`;
    fs.writeFileSync(testFile, expectedContent, "utf8");

    // Note: when skipArchive is not set the SDK creates a zip; we name the
    // artifact explicitly. (When skipArchive is true the SDK ignores the name
    // parameter and uses the file's basename instead.)
    const artifactName = `gh-aw-sdk-upload-${runId}`;

    // ── 2. Upload with official SDK ────────────────────────────────────────
    console.log(`[sdk→custom] Uploading "${artifactName}" with SDK …`);
    const sdkUploadResult = await sdkClient.uploadArtifact(artifactName, [testFile], tmpDir);

    expect(sdkUploadResult.id).toBeTypeOf("number");
    expect(sdkUploadResult.size).toBeGreaterThan(0);
    console.log(`  ✅ SDK uploaded — id=${sdkUploadResult.id} size=${sdkUploadResult.size}`);

    // ── 3. List with our client and verify artifact appears ────────────────
    console.log(`[sdk→custom] Listing artifacts with our client …`);
    const ourListResult = await ourClient.listArtifacts({ findBy: makeOurFindBy() });

    expect(Array.isArray(ourListResult.artifacts)).toBe(true);
    const found = ourListResult.artifacts.find(a => a.id === sdkUploadResult.id);
    expect(found, `artifact id=${sdkUploadResult.id} should appear in our client list`).toBeDefined();
    expect(found?.name).toBe(artifactName);
    expect(found?.size).toBeGreaterThan(0);
    console.log(`  ✅ our client list found artifact "${found?.name}"`);

    // ── 4. Download with our client (SDK creates a zip by default) ─────────
    const ourDownloadDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-our-dl-"));
    try {
      console.log(`[sdk→custom] Downloading with our client …`);
      const ourDownloadResult = await ourClient.downloadArtifact(sdkUploadResult.id, {
        path: ourDownloadDir,
        findBy: makeOurFindBy(),
      });

      expect(ourDownloadResult.downloadPath).toBe(ourDownloadDir);

      const downloadedFiles = fs.readdirSync(ourDownloadDir);
      expect(downloadedFiles.length).toBeGreaterThan(0);
      console.log(`  ✅ our client downloaded ${downloadedFiles.length} file(s): ${downloadedFiles.join(", ")}`);

      // The SDK uploaded a zip containing sdk-upload.txt; after unzip the file
      // should be present with its original content.
      const downloadedFile = downloadedFiles.find(f => f === "sdk-upload.txt");
      expect(downloadedFile, "sdk-upload.txt should be present after unzip").toBeDefined();
      const downloadedContent = fs.readFileSync(path.join(ourDownloadDir, downloadedFile), "utf8");
      expect(downloadedContent).toBe(expectedContent);
      console.log("  ✅ our-client-downloaded content matches SDK-uploaded content");
    } finally {
      fs.rmSync(ourDownloadDir, { recursive: true, force: true });
    }
  });
});
