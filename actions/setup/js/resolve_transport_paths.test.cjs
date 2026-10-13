// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import { createRequire } from "module";
const require = createRequire(import.meta.url);
const { resolveTransportPaths } = require("./resolve_transport_paths.cjs");
const { withClaimExecution, claimArtifactPath } = require("./work_queue_claim_scope.cjs");

let root;
beforeEach(() => {
  root = fs.mkdtempSync(path.join(process.cwd(), ".artifact-resolver-"));
});
afterEach(() => fs.rmSync(root, { recursive: true, force: true }));
const write = (relative, content = "artifact") => {
  const filename = path.join(root, relative);
  fs.mkdirSync(path.dirname(filename), { recursive: true });
  fs.writeFileSync(filename, content);
  return filename;
};
const resolve = (message, config = {}) => resolveTransportPaths(message, "owner/repo", { root, config });

describe("resolveTransportPaths", () => {
  it("returns no transport for missing branch", () => expect(resolve({})).toMatchObject({ patchPath: undefined, bundlePath: undefined }));
  it.each(["", "artifact", path.join("artifact", "nested")])("resolves actual downloaded layout %s", directory => {
    const filename = write(path.join(directory, "aw-owner-repo-feature.patch"));
    expect(resolve({ branch: "feature", repo: "repo" }).patchPath).toBe(filename);
  });
  it("accepts the unqualified single-repository producer filename", () => {
    const filename = write("aw-feature.patch");
    expect(resolve({ branch: "feature" }).patchPath).toBe(filename);
  });
  it.each(["", "nested"])("rejects bare repo alias plus benign qualified patch plus alternate violating patch (%s)", directory => {
    write("aw-owner-repo-feature.patch", "benign src/allowed.js patch");
    write(path.join(directory, "aw-repo-feature.patch"), "policy-violating .github/workflows/alternate.yml patch");
    expect(() => resolve({ branch: "feature", repo: "repo" }, { allowed_files: ["src/**"] })).toThrow("CTR-033: ambiguous");
  });
  it("rejects duplicate basenames across directories", () => {
    write(path.join("one", "aw-owner-repo-feature.patch"));
    write(path.join("two", "aw-owner-repo-feature.patch"));
    expect(() => resolve({ branch: "feature" })).toThrow("CTR-033: ambiguous");
  });
  it("authorizes repositories before looking up artifacts", () => {
    write("aw-owner-other-feature.patch");
    expect(() => resolve({ branch: "feature", repo: "other" })).toThrow("CTR-033: artifact repository is not authorized");
  });
  it("rejects conflicting patch and bundle aliases", () => {
    write("aw-owner-repo-feature.patch");
    write("aw-feature.bundle");
    expect(() => resolve({ branch: "feature" })).toThrow("CTR-033: patch preview and bundle have conflicting identities");
  });
  it("rejects a bundle under an explicit am policy", () => {
    write("aw-owner-repo-feature.patch");
    write("aw-owner-repo-feature.bundle");
    expect(() => resolve({ branch: "feature" }, { patch_format: "am" })).toThrow("CTR-033: bundle conflicts");
  });
  it("requires both bundle and preview under bundle policy", () => {
    write("aw-owner-repo-feature.bundle");
    expect(() => resolve({ branch: "feature" }, { patch_format: "bundle" })).toThrow("CTR-033: bundle transport requires");
  });
  it("rejects hardlinked extracted files", () => {
    const original = write("aw-owner-repo-feature.patch");
    fs.linkSync(original, path.join(root, "another.patch"));
    expect(() => resolve({ branch: "feature" })).toThrow("CTR-033: artifact must be a bounded regular file");
  });
  it("rejects directory junctions escaping staging", () => {
    const outside = fs.mkdtempSync(path.join(process.cwd(), ".artifact-outside-"));
    try {
      fs.symlinkSync(outside, path.join(root, "linked"), "junction");
      expect(() => resolve({ branch: "feature" })).toThrow("CTR-033: linked extracted entry");
    } finally {
      fs.rmSync(outside, { recursive: true, force: true });
    }
  });
  it("resolves nested claim scope without selecting a foreign claim's same basename", async () => {
    const assignment = {
      version: 3,
      dispatch_id: "dispatch",
      request_id: "request",
      commit_id: "commit",
      policy_epoch: "epoch",
      pool: "pool",
      worker_profile: "worker",
      claims: [
        { handle: "one", claim_id: "claim-one", work_id: "work-one", work: {}, result_refs: [] },
        { handle: "two", claim_id: "claim-two", work_id: "work-two", work: {}, result_refs: [] },
      ],
    };
    const own = path.join("nested", claimArtifactPath("", "one", assignment), "aw-owner-repo-feature.patch");
    const foreign = path.join("nested", claimArtifactPath("", "two", assignment), "aw-owner-repo-feature.patch");
    const expected = write(own);
    write(foreign, "foreign claim");
    await withClaimExecution({ assignment, claim_handle: "one" }, () => {
      expect(resolve({ branch: "feature" }).patchPath).toBe(expected);
    });
  });
});
