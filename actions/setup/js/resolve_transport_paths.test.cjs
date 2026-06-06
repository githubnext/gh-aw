// @ts-check

import { describe, it, expect, beforeEach } from "vitest";
import fs from "fs";

import { resolveTransportPaths } from "./resolve_transport_paths.cjs";

beforeEach(() => {
  // git_patch_utils.cjs hardcodes /tmp/gh-aw
  fs.mkdirSync("/tmp/gh-aw", { recursive: true });
});

describe("resolveTransportPaths", () => {
  it("returns undefined values when message has no branch", () => {
    const result = resolveTransportPaths({}, "owner/repo");
    expect(result.patchPath).toBeUndefined();
    expect(result.bundlePath).toBeUndefined();
  });

  it("recovers single-repo patch path from branch when not present on the message", () => {
    const branch = `resolve-test-${Date.now()}`;
    const expectedPatch = `/tmp/gh-aw/aw-${branch}.patch`;
    const expectedBundle = `/tmp/gh-aw/aw-${branch}.bundle`;
    fs.writeFileSync(expectedPatch, "patch-content");
    fs.writeFileSync(expectedBundle, "bundle-content");
    try {
      const result = resolveTransportPaths({ branch }, undefined);
      expect(result.patchPath).toBe(expectedPatch);
      expect(result.bundlePath).toBe(expectedBundle);
    } finally {
      fs.rmSync(expectedPatch, { force: true });
      fs.rmSync(expectedBundle, { force: true });
    }
  });

  it("prefers multi-repo path from message.repo when that file exists", () => {
    const branch = `resolve-multi-${Date.now()}`;
    const repo = "owner/repo";
    const sanitizedRepo = "owner-repo";
    const expectedPatch = `/tmp/gh-aw/aw-${sanitizedRepo}-${branch}.patch`;
    fs.writeFileSync(expectedPatch, "patch-content");
    try {
      const result = resolveTransportPaths({ branch, repo }, undefined);
      expect(result.patchPath).toBe(expectedPatch);
    } finally {
      fs.rmSync(expectedPatch, { force: true });
    }
  });

  it("falls back to defaultTargetRepo when message.repo is absent", () => {
    const branch = `resolve-default-${Date.now()}`;
    const repo = "owner/defaultrepo";
    const sanitizedRepo = "owner-defaultrepo";
    const expectedPatch = `/tmp/gh-aw/aw-${sanitizedRepo}-${branch}.patch`;
    fs.writeFileSync(expectedPatch, "patch-content");
    try {
      const result = resolveTransportPaths({ branch }, repo);
      expect(result.patchPath).toBe(expectedPatch);
    } finally {
      fs.rmSync(expectedPatch, { force: true });
    }
  });

  it("falls back to single-repo path when neither multi-repo candidate exists", () => {
    const branch = `resolve-fallback-${Date.now()}`;
    const expectedPatch = `/tmp/gh-aw/aw-${branch}.patch`;
    fs.writeFileSync(expectedPatch, "patch-content");
    try {
      const result = resolveTransportPaths({ branch, repo: "owner/nonexistent" }, "owner/alsoNo");
      expect(result.patchPath).toBe(expectedPatch);
    } finally {
      fs.rmSync(expectedPatch, { force: true });
    }
  });

  it("returns undefined when no candidate file exists on disk", () => {
    const branch = `resolve-missing-${Date.now()}`;
    const result = resolveTransportPaths({ branch }, "owner/repo");
    expect(result.patchPath).toBeUndefined();
    expect(result.bundlePath).toBeUndefined();
  });

  it("recovers bundle path independently of patch path", () => {
    const branch = `resolve-bundle-only-${Date.now()}`;
    const expectedBundle = `/tmp/gh-aw/aw-${branch}.bundle`;
    fs.writeFileSync(expectedBundle, "bundle-content");
    try {
      const result = resolveTransportPaths({ branch }, undefined);
      expect(result.patchPath).toBeUndefined();
      expect(result.bundlePath).toBe(expectedBundle);
    } finally {
      fs.rmSync(expectedBundle, { force: true });
    }
  });
});
