// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

const req = createRequire(import.meta.url);
// Use createRequire so the CJS module shares the same module cache as tests that import it,
// which is required for the caching/singleton tests to work correctly.
const { lookupCheckout, loadAllCheckouts, _resetCache } = req("./checkout_manifest.cjs");

describe("checkout_manifest", () => {
  /** @type {string | null} */
  let tmpDir = null;

  beforeEach(() => {
    _resetCache();
    vi.stubEnv("RUNNER_TEMP", "");
    vi.stubEnv("GH_AW_CHECKOUT_MANIFEST", "");
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "checkout-manifest-test-"));
  });

  afterEach(() => {
    _resetCache();
    vi.unstubAllEnvs();
    if (tmpDir) {
      try {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      } catch {
        // ignore
      }
      tmpDir = null;
    }
  });

  function writeManifest(manifestPath, data) {
    const dir = path.dirname(manifestPath);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(manifestPath, JSON.stringify(data), "utf8");
  }

  describe("lookupCheckout", () => {
    it("returns null when repoSlug is null", () => {
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      expect(lookupCheckout(null)).toBeNull();
    });

    it("returns null when repoSlug is undefined", () => {
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      expect(lookupCheckout(undefined)).toBeNull();
    });

    it("returns null when repoSlug is empty string", () => {
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      expect(lookupCheckout("")).toBeNull();
    });

    it("returns null when repoSlug is only whitespace", () => {
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      expect(lookupCheckout("   ")).toBeNull();
    });

    it("returns null when manifest file is missing (no RUNNER_TEMP, no explicit path)", () => {
      // beforeEach stubs both to "" — no additional setup needed
      expect(lookupCheckout("owner/repo")).toBeNull();
    });

    it("returns null when manifest file does not exist", () => {
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      // No manifest written, ENOENT case
      expect(lookupCheckout("owner/repo")).toBeNull();
    });

    it("returns entry for known repo slug (case-insensitive)", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "checkout/owner-repo", default_branch: "main" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const result = lookupCheckout("owner/repo");
      expect(result).toEqual({ repository: "owner/repo", path: "checkout/owner-repo", default_branch: "main" });
    });

    it("looks up repo slug case-insensitively", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "checkout/owner-repo", default_branch: "main" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const result = lookupCheckout("OWNER/REPO");
      expect(result).toEqual({ repository: "owner/repo", path: "checkout/owner-repo", default_branch: "main" });
    });

    it("returns null for unknown repo slug", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "checkout/owner-repo", default_branch: "main" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const result = lookupCheckout("other/repo");
      expect(result).toBeNull();
    });

    it("uses GH_AW_CHECKOUT_MANIFEST override path", () => {
      const manifestPath = path.join(String(tmpDir), "custom-manifest.json");
      writeManifest(manifestPath, {
        "myorg/myrepo": { repository: "myorg/myrepo", path: "work/myrepo", default_branch: "develop" },
      });
      vi.stubEnv("GH_AW_CHECKOUT_MANIFEST", manifestPath);
      const result = lookupCheckout("myorg/myrepo");
      expect(result).toEqual({ repository: "myorg/myrepo", path: "work/myrepo", default_branch: "develop" });
    });

    it("falls back to repoSlug when repository field is missing", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { path: "checkout/owner-repo", default_branch: "main" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const result = lookupCheckout("owner/repo");
      expect(result?.repository).toBe("owner/repo");
      expect(result?.path).toBe("checkout/owner-repo");
    });

    it("falls back to empty strings when path and default_branch are missing", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const result = lookupCheckout("owner/repo");
      expect(result?.path).toBe("");
      expect(result?.default_branch).toBe("");
    });

    it("caches the manifest after first load", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "a", default_branch: "main" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const first = lookupCheckout("owner/repo");
      // Modify the file on disk - cached result should not change
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "CHANGED", default_branch: "main" },
      });
      const second = lookupCheckout("owner/repo");
      expect(first?.path).toBe("a");
      expect(second?.path).toBe("a");
    });

    it("_resetCache clears the cache", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "a", default_branch: "main" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      lookupCheckout("owner/repo");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "CHANGED", default_branch: "main" },
      });
      _resetCache();
      const result = lookupCheckout("owner/repo");
      expect(result?.path).toBe("CHANGED");
    });

    it("returns null when manifest contains invalid JSON", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      fs.mkdirSync(path.dirname(manifestPath), { recursive: true });
      fs.writeFileSync(manifestPath, "not json", "utf8");
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      expect(lookupCheckout("owner/repo")).toBeNull();
    });

    it("returns null when manifest root is an array", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, [{ repository: "owner/repo", path: "a", default_branch: "main" }]);
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      expect(lookupCheckout("owner/repo")).toBeNull();
    });
  });

  describe("loadAllCheckouts", () => {
    it("returns an empty Map when no manifest exists", () => {
      // beforeEach stubs both RUNNER_TEMP and GH_AW_CHECKOUT_MANIFEST to "" — no setup needed
      const map = loadAllCheckouts();
      expect(map.size).toBe(0);
    });

    it("returns all entries from the manifest", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo1": { repository: "owner/repo1", path: "a", default_branch: "main" },
        "owner/repo2": { repository: "owner/repo2", path: "b", default_branch: "dev" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const map = loadAllCheckouts();
      expect(map.size).toBe(2);
      expect(map.get("owner/repo1")).toEqual({ repository: "owner/repo1", path: "a", default_branch: "main" });
      expect(map.get("owner/repo2")).toEqual({ repository: "owner/repo2", path: "b", default_branch: "dev" });
    });

    it("normalizes keys to lowercase", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "OWNER/REPO": { repository: "OWNER/REPO", path: "a", default_branch: "main" },
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const map = loadAllCheckouts();
      expect(map.has("owner/repo")).toBe(true);
      expect(map.has("OWNER/REPO")).toBe(false);
    });

    it("skips entries that are not objects", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, {
        "owner/repo": { repository: "owner/repo", path: "a", default_branch: "main" },
        "bad/entry": "not an object",
        "null/entry": null,
      });
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const map = loadAllCheckouts();
      expect(map.size).toBe(1);
      expect(map.has("owner/repo")).toBe(true);
    });

    it("returns empty Map when manifest root is an array", () => {
      const manifestPath = path.join(String(tmpDir), "gh-aw", "safeoutputs", "checkout-manifest.json");
      writeManifest(manifestPath, [{ repository: "owner/repo", path: "a", default_branch: "main" }]);
      vi.stubEnv("RUNNER_TEMP", String(tmpDir));
      const map = loadAllCheckouts();
      expect(map.size).toBe(0);
    });
  });
});
