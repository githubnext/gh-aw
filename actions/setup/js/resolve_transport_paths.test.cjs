// @ts-check

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import crypto from "crypto";
import fs from "fs";
import { resolveTransportPaths } from "./resolve_transport_paths.cjs";
import { getPatchPathForBranch, getPatchPathForBranchInRepo } from "./git_patch_utils.cjs";
import { getBundlePathForBranch, getBundlePathForBranchInRepo } from "./generate_git_bundle.cjs";

let branch;
const created = new Set();
beforeEach(() => {
  fs.mkdirSync("/tmp/gh-aw", { recursive: true });
  branch = `resolve-${crypto.randomUUID()}`;
});
afterEach(() => {
  for (const file of created) fs.rmSync(file, { force: true });
  created.clear();
});

function write(kind, repo = undefined) {
  const file = kind === "patch" ? (repo ? getPatchPathForBranchInRepo(branch, repo) : getPatchPathForBranch(branch)) : repo ? getBundlePathForBranchInRepo(branch, repo) : getBundlePathForBranch(branch);
  created.add(file);
  fs.writeFileSync(file, `${kind} content`);
  return file;
}

describe("resolveTransportPaths", () => {
  it.each(["-", "---", " --- "])("rejects dash-only branch %s in both patch and bundle paths", branch => {
    for (const repo of [undefined, "owner/repo"]) {
      expect(() => resolveTransportPaths({ branch }, repo)).toThrow("Invalid branch name");
      expect(() => (repo ? getPatchPathForBranchInRepo(branch, repo) : getPatchPathForBranch(branch))).toThrow("Invalid branch name");
      expect(() => (repo ? getBundlePathForBranchInRepo(branch, repo) : getBundlePathForBranch(branch))).toThrow("Invalid branch name");
    }
  });

  it("returns undefined values when message has no branch or no files exist", () => {
    expect(resolveTransportPaths({}, "owner/repo")).toEqual({ patchPath: undefined, bundlePath: undefined });
    expect(resolveTransportPaths({ branch }, "owner/repo")).toEqual({ patchPath: undefined, bundlePath: undefined });
  });

  it.each([undefined, "owner/repo"])("accepts a unique legacy namespace with target %s", repo => {
    const patchPath = write("patch");
    const bundlePath = write("bundle");
    expect(resolveTransportPaths({ branch }, repo)).toEqual({ patchPath, bundlePath });
  });

  it.each(["repo", " owner/repo ", undefined])("uses only the validated qualified slug, not raw repo %s", repo => {
    const patchPath = write("patch", "owner/repo");
    const bundlePath = write("bundle", "owner/repo");
    write("patch", "repo");
    write("bundle", "owner/different");
    expect(resolveTransportPaths({ branch, repo, patch: "/untrusted.patch", bundle: "/untrusted.bundle" }, "owner/repo")).toEqual({ patchPath, bundlePath });
  });

  it.each(["patch", "bundle"])("accepts a single qualified %s without its companion", kind => {
    const file = write(kind, "owner/repo");
    expect(resolveTransportPaths({ branch }, "owner/repo")).toEqual({
      patchPath: kind === "patch" ? file : undefined,
      bundlePath: kind === "bundle" ? file : undefined,
    });
  });

  it.each([
    ["patch", "patch"],
    ["bundle", "bundle"],
    ["patch", "bundle"],
    ["bundle", "patch"],
  ])("rejects qualified %s plus legacy %s instead of preferring or mixing them", (qualifiedKind, legacyKind) => {
    write(qualifiedKind, "owner/repo");
    write(legacyKind);
    expect(() => resolveTransportPaths({ branch }, "owner/repo")).toThrow(expect.objectContaining({ code: "ERR_TRANSPORT_INTEGRITY", operation: "resolve", reason: "ambiguous transport candidates" }));
  });

  it("rejects duplicate complete namespaces even when their contents are identical", () => {
    for (const repo of [undefined, "owner/repo"]) {
      write("patch", repo);
      write("bundle", repo);
    }
    expect(() => resolveTransportPaths({ branch }, "owner/repo")).toThrow("ambiguous transport candidates");
  });

  it("does not use a different repository's artifacts as a fallback", () => {
    write("patch", "owner/default");
    write("bundle", "owner/default");
    expect(resolveTransportPaths({ branch, repo: "other" }, "owner/other")).toEqual({ patchPath: undefined, bundlePath: undefined });
  });
});
