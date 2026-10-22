import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { resolvePatchBase, validateBaseBranch, validateTransportBase } = require("./resolve_patch_base.cjs");
const { _resetCache } = require("./checkout_manifest.cjs");
const { createHandlers } = require("./safe_outputs_handlers.cjs");

describe("patch base resolution", () => {
  let root;
  const repo = { owner: "side", repo: "target" };
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "patch-base-"));
    const manifest = path.join(root, "manifest.json");
    fs.writeFileSync(manifest, JSON.stringify({ "side/target": { repository: "side/target", path: "target", default_branch: "main", checked_out_ref: "feature/existing-pr" } }));
    vi.stubEnv("GH_AW_CHECKOUT_MANIFEST", manifest);
    global.context = {
      repo: { owner: "host", repo: "workflow" },
      payload: {
        pull_request: {
          number: 42,
          head: { ref: "feature/existing-pr" },
          base: { ref: "release/1", repo: { full_name: "side/target" } },
        },
      },
    };
    _resetCache();
  });
  afterEach(() => {
    _resetCache();
    fs.rmSync(root, { recursive: true, force: true });
    delete global.context;
  });
  it("keeps a known PR head checkout based on the PR's base rather than itself", async () => {
    expect(await resolvePatchBase(repo, {})).toBe("release/1");
  });
  it("lets explicit configuration override an initial checkout or triggering PR", async () => {
    expect(await resolvePatchBase(repo, { base_branch: "maintenance/2" })).toBe("maintenance/2");
  });
  it("does not borrow a different repository's PR base", async () => {
    global.context.payload.pull_request.base.repo.full_name = "host/workflow";
    expect(await resolvePatchBase(repo, {})).toBe("feature/existing-pr");
  });
  it("honors a selected branch different from the triggering PR's head", async () => {
    global.context.payload.pull_request.head.ref = "another-feature";
    expect(await resolvePatchBase(repo, {})).toBe("feature/existing-pr");
  });
  it("requires trusted policy for transported runtime bases", () => {
    expect(validateTransportBase("release/1", "main", { allowed_base_branches: ["release/*"] })).toBe("release/1");
    expect(() => validateTransportBase("release/1", "main", {})).toThrow("not authorized");
  });
  it("rejects unauthorized per-call bases before patch generation", async () => {
    vi.stubEnv("GITHUB_REPOSITORY", "host/workflow");
    const append = vi.fn();
    const handlers = createHandlers({ debug: vi.fn() }, append, { create_pull_request: { base_branch: "main", allow_empty: true } });
    const result = await handlers.createPullRequestHandler({ title: "Correct release behavior", body: "A focused release fix.", branch: "fix/release", base: "release/1" });
    expect(result.isError).toBe(true);
    expect(result.content[0].text).toContain("not authorized");
    expect(append).not.toHaveBeenCalled();
  });
  it("preserves authorization for a base prepared by an earlier output in the same run", async () => {
    vi.stubEnv("GITHUB_REPOSITORY", "host/workflow");
    const append = vi.fn();
    const handlers = createHandlers({ debug: vi.fn() }, append, { create_pull_request: { base_branch: "main", allow_empty: true } });
    const first = await handlers.createPullRequestHandler({ title: "Prepare a release fix", body: "First part of a focused fix.", branch: "fix/first" });
    expect(first.isError).not.toBe(true);
    const second = await handlers.createPullRequestHandler({ title: "Complete the release fix", body: "Second part of the focused fix.", branch: "fix/second", base: "fix/first" });
    expect(second.isError, JSON.stringify(second)).not.toBe(true);
    expect(append).toHaveBeenLastCalledWith(expect.objectContaining({ base: "fix/first", base_branch: "fix/first" }));
  });
  it.each(["HEAD", "../main", "release//1", "-main", "main.lock", "main.", "release/.hidden", "main\n", " main", "release/../main"])("rejects invalid bases without rewriting %j", branch => {
    expect(() => validateBaseBranch(branch)).toThrow("Invalid patch base");
  });
});
