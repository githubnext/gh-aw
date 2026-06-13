// @ts-check
/**
 * Integration tests for DefaultArtifactClient using the live GitHub Actions artifact APIs.
 *
 * These tests are skipped when the required environment variables are absent
 * (i.e. outside of a real GitHub Actions job). When running inside GHA the
 * runner provides ACTIONS_RUNTIME_TOKEN and ACTIONS_RESULTS_URL automatically,
 * and GITHUB_TOKEN is injected via the workflow step's `env:` block.
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
const { DefaultArtifactClient } = req("./artifact_client.cjs");

// These are set automatically by the GitHub Actions runner for the current job.
const HAVE_ACTIONS_RUNTIME = !!(process.env.ACTIONS_RUNTIME_TOKEN && process.env.ACTIONS_RESULTS_URL);
// GITHUB_TOKEN is injected via env: in the workflow step.
const HAVE_GITHUB_TOKEN = !!(process.env.GITHUB_TOKEN || process.env.GH_TOKEN);

/**
 * Build findBy options from standard GHA environment variables.
 * @returns {{ token: string, repositoryOwner: string, repositoryName: string, workflowRunId: string }}
 */
function makeFindBy() {
  const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN || "";
  const [owner = "", repo = ""] = (process.env.GITHUB_REPOSITORY || "/").split("/");
  const workflowRunId = process.env.GITHUB_RUN_ID || "";
  return { token, repositoryOwner: owner, repositoryName: repo, workflowRunId };
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

  it("uploads then downloads an artifact end-to-end using live APIs", async () => {
    if (!HAVE_ACTIONS_RUNTIME) {
      console.log("Skipping live artifact test — ACTIONS_RUNTIME_TOKEN / ACTIONS_RESULTS_URL not set.\n" + "This test only runs inside a GitHub Actions job.");
      return;
    }
    if (!HAVE_GITHUB_TOKEN) {
      console.log("Skipping live artifact test — GITHUB_TOKEN not set.\n" + "Pass it via env: GITHUB_TOKEN in the workflow step.");
      return;
    }

    const client = new DefaultArtifactClient();

    // ── 1. Prepare a small test file ───────────────────────────────────────
    const testFile = path.join(tmpDir, "hello.txt");
    const expectedContent = `gh-aw artifact integration test\nrun=${process.env.GITHUB_RUN_ID}\n`;
    fs.writeFileSync(testFile, expectedContent, "utf8");

    // Use run ID in the name so parallel runs don't collide.
    const runId = process.env.GITHUB_RUN_ID || "local";
    const artifactName = `gh-aw-artifact-integration-${runId}`;

    // ── 2. Upload ───────────────────────────────────────────────────────────
    console.log(`Uploading artifact "${artifactName}" …`);
    const uploadResult = await client.uploadArtifact(artifactName, [testFile], tmpDir, { skipArchive: true });

    expect(uploadResult.id).toBeTypeOf("number");
    expect(uploadResult.size).toBeGreaterThan(0);
    expect(uploadResult.digest).toMatch(/^[0-9a-f]{64}$/);
    console.log(`  ✅ uploaded — id=${uploadResult.id} size=${uploadResult.size} digest=${uploadResult.digest}`);

    // ── 3. List artifacts and verify the upload appears ────────────────────
    console.log("Listing artifacts for current workflow run …");
    const listResult = await client.listArtifacts({ findBy: makeFindBy() });

    expect(Array.isArray(listResult.artifacts)).toBe(true);
    const uploaded = listResult.artifacts.find(a => a.id === uploadResult.id);
    expect(uploaded, `artifact id=${uploadResult.id} should appear in list`).toBeDefined();
    expect(uploaded?.name).toBe(artifactName);
    expect(uploaded?.size).toBeGreaterThan(0);
    console.log(`  ✅ found artifact "${uploaded?.name}" in list`);

    // ── 4. Download ─────────────────────────────────────────────────────────
    const downloadDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-artifact-live-dl-"));
    try {
      console.log(`Downloading artifact id=${uploadResult.id} …`);
      const downloadResult = await client.downloadArtifact(uploadResult.id, {
        path: downloadDir,
        skipDecompress: true,
        findBy: makeFindBy(),
      });

      expect(downloadResult.downloadPath).toBe(downloadDir);
      expect(downloadResult.digestMismatch).toBe(false);

      const downloadedFiles = fs.readdirSync(downloadDir);
      expect(downloadedFiles.length).toBeGreaterThan(0);
      console.log(`  ✅ downloaded ${downloadedFiles.length} file(s): ${downloadedFiles.join(", ")}`);

      // ── 5. Verify content matches what was uploaded ──────────────────────
      const downloadedFile = path.join(downloadDir, downloadedFiles[0]);
      const downloadedContent = fs.readFileSync(downloadedFile, "utf8");
      expect(downloadedContent).toBe(expectedContent);
      console.log("  ✅ downloaded content matches uploaded content");
    } finally {
      fs.rmSync(downloadDir, { recursive: true, force: true });
    }
  });
});
