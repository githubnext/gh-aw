// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import crypto from "crypto";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const scopePath = require.resolve("./work_queue_claim_scope.cjs");
const resourcePath = require.resolve("./work_queue_effect_resource.cjs");
const clientPath = require.resolve("./work_queue_effect_client.cjs");
const modulePath = require.resolve("./work_queue_upload_assets.cjs");

/** @param {string} p @param {any} exports */
function stub(p, exports) {
  require.cache[p] = /** @type {any} */ { id: p, filename: p, loaded: true, exports };
}

describe("work_queue_upload_assets", () => {
  let root, saved, handle, scope, authorizeCalls, effectCalls, git, client;

  beforeEach(() => {
    saved = {
      scope: require.cache[scopePath],
      resource: require.cache[resourcePath],
      client: require.cache[clientPath],
      mod: require.cache[modulePath],
      stagedMode: process.env.GH_AW_SAFE_OUTPUTS_STAGED,
    };
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    root = fs.mkdtempSync(path.join(os.tmpdir(), "assets-"));
    handle = "claim-1";
    authorizeCalls = [];
    effectCalls = [];
    scope = {
      assertClaimAuthorized: async (m, o) => (authorizeCalls.push(o), m),
      currentClaimHandle: () => handle,
      claimArtifactPath: (dir, h) => (dir ? dir : `ns-${h}`),
      assertClaimArtifactFile: () => {},
      claimIdentity: () => ({ id: "x" }),
      assertClaimIdentity: () => {},
      receiptMatchesClaim: r => !!r,
      withClaimResourceEffects: async (resources, names, fn) => (effectCalls.push({ resources, names }), fn()),
      createClaimResourceVerification: p => p,
      withClaimResourceVerification: async (proof, fn) => fn(),
    };
    stub(scopePath, scope);
    stub(resourcePath, { resolveRepositoryTarget: async (_c, t) => ({ ...t, kind: "repo" }) });
    stub(clientPath, { wrapClaimEffectClient: c => c });
    delete require.cache[modulePath];
    git = {
      getRef: async () => ({ data: { object: { sha: "head1" } } }),
      getCommit: async () => ({ data: { tree: { sha: "tree0" } } }),
      createBlob: async () => ({ data: { sha: "blob1" } }),
      createTree: async () => ({ data: { sha: "tree1" } }),
      createCommit: async () => ({ data: { sha: "commit1" } }),
      updateRef: async () => ({}),
      createRef: async () => ({}),
    };
    client = { rest: { git } };
  });

  afterEach(() => {
    for (const [p, k] of [
      [scopePath, "scope"],
      [resourcePath, "resource"],
      [clientPath, "client"],
      [modulePath, "mod"],
    ]) {
      if (saved[k]) require.cache[p] = saved[k];
      else delete require.cache[p];
    }
    if (saved.stagedMode === undefined) delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    else process.env.GH_AW_SAFE_OUTPUTS_STAGED = saved.stagedMode;
    fs.rmSync(root, { recursive: true, force: true });
  });

  const load = () => require("./work_queue_upload_assets.cjs");
  const stage = (source, content) => {
    const ext = path.extname(source).toLowerCase();
    fs.writeFileSync(path.join(root, crypto.createHash("sha256").update(source).digest("hex") + ext), content);
  };

  it("rejects creation without a Claim factory", async () => {
    handle = undefined;
    await expect(load().main({}, client)).rejects.toThrow("immutable Claim factory");
  });

  it("rejects an invalid repository", async () => {
    await expect(load().main({ "target-repo": "bad repo" }, client)).rejects.toThrow("trusted repository destination");
  });

  it("returns a staged preview without writing", async () => {
    const handler = await load().main({ staged: true, "assets-dir": root, "target-repo": "o/r" }, client);
    expect(await handler({ path: "a.png" })).toEqual({ success: true, staged: true, claim_handle: "claim-1" });
    expect(authorizeCalls[0]).toEqual({ requireCompletion: false });
  });

  it("rejects conflicting explicit repositories", async () => {
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({ repo: "o/other", path: "a.png" })).rejects.toThrow("conflicts with its trusted adapter");
  });

  it("rejects a missing source path", async () => {
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({})).rejects.toThrow("original staged source path");
  });

  it("rejects disallowed extensions", async () => {
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({ path: "a.exe" })).rejects.toThrow("extension is outside");
  });

  it("enforces the per-Claim maximum", async () => {
    stage("a.png", "x");
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r", max: 1 }, client);
    await handler({ path: "a.png" });
    await expect(handler({ path: "a.png" })).rejects.toThrow("per-Claim maximum");
  });

  it("rejects files above the size limit", async () => {
    stage("a.png", Buffer.alloc(2048));
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r", "max-size": 1 }, client);
    await expect(handler({ path: "a.png" })).rejects.toThrow("bounded isolated regular file");
  });

  it("rejects a mismatched declared digest", async () => {
    stage("a.png", "data");
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({ path: "a.png", sha: "deadbeef" })).rejects.toThrow("differs from its declared digest");
  });

  it("publishes to an existing branch via updateRef", async () => {
    stage("a.PNG", "data");
    const sha = crypto.createHash("sha256").update("data").digest("hex");
    let updated;
    git.updateRef = async args => (updated = args);
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r", branch: "assets/test" }, client);
    const result = await handler({ path: "a.PNG", sha });
    expect(result.success).toBe(true);
    expect(result.sha).toBe(sha);
    expect(result.path).toBe(`claims/ns-claim-1/${sha}.png`);
    expect(result.commit).toBe("commit1");
    expect(updated).toMatchObject({ sha: "commit1", force: false, ref: expect.stringMatching(/^heads\/assets\/test/) });
    expect(effectCalls).toHaveLength(1);
  });

  it("creates a new assets/ branch on 404", async () => {
    stage("a.png", "data");
    git.getRef = async () => Promise.reject({ status: 404 });
    let created;
    git.createRef = async args => (created = args);
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r", branch: "assets/new" }, client);
    await handler({ path: "a.png" });
    expect(created.ref).toMatch(/^refs\/heads\/assets\/new/);
    expect(created.force).toBeUndefined();
  });

  it("rejects creating a branch outside the assets/ namespace", async () => {
    stage("a.png", "data");
    git.getRef = async () => Promise.reject({ status: 404 });
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r", branch: "feature/not-assets" }, client);
    await expect(handler({ path: "a.png" })).rejects.toThrow("approved assets/ namespace");
  });

  it("propagates non-404 getRef errors", async () => {
    stage("a.png", "data");
    git.getRef = async () => Promise.reject({ status: 500, message: "boom" });
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({ path: "a.png" })).rejects.toMatchObject({ status: 500 });
  });

  it("fails when native endpoints are missing", async () => {
    stage("a.png", "data");
    delete git.createBlob;
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({ path: "a.png" })).rejects.toThrow("native read and write endpoints");
  });

  it("fails when the native parent commit endpoint is missing", async () => {
    stage("a.png", "data");
    delete git.getCommit;
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({ path: "a.png" })).rejects.toThrow("native parent commit endpoint");
  });

  it("fails when the native branch write endpoint is missing", async () => {
    stage("a.png", "data");
    delete git.updateRef;
    const handler = await load().main({ "assets-dir": root, "target-repo": "o/r" }, client);
    await expect(handler({ path: "a.png" })).rejects.toThrow("native branch write endpoint");
  });

  it("verifyAssetDelivery returns unverified without a receipt", async () => {
    expect(await load().verifyAssetDelivery({ claim: {}, result: {}, github: {} })).toEqual({ verified: false });
    expect(await load().verifyAssetDelivery({ claim: {}, result: undefined, github: {} })).toEqual({ verified: false });
  });
});
