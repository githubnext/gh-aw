// @ts-check
import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { extractFilenamesFromPatch, checkForManifestFiles } = require("./manifest_file_helpers.cjs");

describe("manifest_file_helpers", () => {
  describe("extractFilenamesFromPatch", () => {
    it("should return empty array for empty patch", () => {
      expect(extractFilenamesFromPatch("")).toEqual([]);
      expect(extractFilenamesFromPatch(null)).toEqual([]);
      expect(extractFilenamesFromPatch(undefined)).toEqual([]);
    });

    it("should extract a single filename", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
--- a/src/index.js
+++ b/src/index.js
@@ -1 +1 @@
-old
+new
`;
      expect(extractFilenamesFromPatch(patch)).toEqual(["index.js"]);
    });

    it("should extract basename only (no directory path)", () => {
      const patch = `diff --git a/path/to/deep/package.json b/path/to/deep/package.json
index abc..def 100644
--- a/path/to/deep/package.json
+++ b/path/to/deep/package.json
`;
      expect(extractFilenamesFromPatch(patch)).toEqual(["package.json"]);
    });

    it("should extract multiple filenames", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
diff --git a/package.json b/package.json
index abc..def 100644
diff --git a/README.md b/README.md
index abc..def 100644
`;
      const result = extractFilenamesFromPatch(patch);
      expect(result).toContain("index.js");
      expect(result).toContain("package.json");
      expect(result).toContain("README.md");
      expect(result).toHaveLength(3);
    });

    it("should deduplicate filenames", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
diff --git a/lib/index.js b/lib/index.js
index abc..def 100644
`;
      const result = extractFilenamesFromPatch(patch);
      expect(result).toEqual(["index.js"]);
    });

    it("should handle files at root (no directory)", () => {
      const patch = `diff --git a/package.json b/package.json
index abc..def 100644
`;
      expect(extractFilenamesFromPatch(patch)).toEqual(["package.json"]);
    });
  });

  describe("checkForManifestFiles", () => {
    it("should return false for empty patch", () => {
      const result = checkForManifestFiles("", ["package.json"]);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should return false for empty manifest files list", () => {
      const patch = `diff --git a/package.json b/package.json\n`;
      const result = checkForManifestFiles(patch, []);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should return false for null manifest files list", () => {
      const patch = `diff --git a/package.json b/package.json\n`;
      const result = checkForManifestFiles(patch, null);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should detect package.json as a manifest file", () => {
      const patch = `diff --git a/package.json b/package.json
index abc..def 100644
--- a/package.json
+++ b/package.json
@@ -1 +1 @@
-{"name": "old"}
+{"name": "new"}
`;
      const result = checkForManifestFiles(patch, ["package.json", "go.mod"]);
      expect(result.hasManifestFiles).toBe(true);
      expect(result.manifestFilesFound).toContain("package.json");
    });

    it("should detect manifest files in nested directories", () => {
      const patch = `diff --git a/nested/path/go.mod b/nested/path/go.mod
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["go.mod", "go.sum"]);
      expect(result.hasManifestFiles).toBe(true);
      expect(result.manifestFilesFound).toContain("go.mod");
    });

    it("should not detect non-manifest files", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
diff --git a/README.md b/README.md
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["package.json", "go.mod", "requirements.txt"]);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should return all manifest files found", () => {
      const patch = `diff --git a/package.json b/package.json
index abc..def 100644
diff --git a/package-lock.json b/package-lock.json
index abc..def 100644
diff --git a/src/index.js b/src/index.js
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["package.json", "package-lock.json", "yarn.lock"]);
      expect(result.hasManifestFiles).toBe(true);
      expect(result.manifestFilesFound).toContain("package.json");
      expect(result.manifestFilesFound).toContain("package-lock.json");
      expect(result.manifestFilesFound).toHaveLength(2);
    });

    it("should match by filename only, not partial name", () => {
      const patch = `diff --git a/src/my-package.json b/src/my-package.json
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["package.json"]);
      expect(result.hasManifestFiles).toBe(false);
    });
  });
});
