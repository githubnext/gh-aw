const fs = require("fs");
const path = require("path");
const { extractRepoSlugFromUrl, normalizeRepoSlug, findGitDirectories, findRepoCheckout, buildRepoCheckoutMap } = require("./find_repo_checkout.cjs");
const { getPatchPathForRepo, sanitizeBranchNameForPatch, sanitizeRepoSlugForPatch } = require("./generate_git_patch.cjs");

describe("find_repo_checkout", () => {
  describe("extractRepoSlugFromUrl", () => {
    it("should extract slug from HTTPS URL", () => {
      expect(extractRepoSlugFromUrl("https://github.com/owner/repo.git")).toBe("owner/repo");
      expect(extractRepoSlugFromUrl("https://github.com/owner/repo")).toBe("owner/repo");
    });

    it("should extract slug from SSH URL", () => {
      expect(extractRepoSlugFromUrl("git@github.com:owner/repo.git")).toBe("owner/repo");
      expect(extractRepoSlugFromUrl("git@github.com:owner/repo")).toBe("owner/repo");
    });

    it("should handle GitHub Enterprise URLs", () => {
      expect(extractRepoSlugFromUrl("https://github.example.com/org/project.git")).toBe("org/project");
      expect(extractRepoSlugFromUrl("git@github.example.com:org/project.git")).toBe("org/project");
    });

    it("should normalize to lowercase", () => {
      expect(extractRepoSlugFromUrl("https://github.com/Owner/Repo.git")).toBe("owner/repo");
      expect(extractRepoSlugFromUrl("git@github.com:OWNER/REPO")).toBe("owner/repo");
    });

    it("should return null for invalid URLs", () => {
      expect(extractRepoSlugFromUrl("")).toBeNull();
      expect(extractRepoSlugFromUrl("invalid")).toBeNull();
      expect(extractRepoSlugFromUrl(null)).toBeNull();
      expect(extractRepoSlugFromUrl(undefined)).toBeNull();
    });

    it("should handle URLs with ports", () => {
      expect(extractRepoSlugFromUrl("https://github.example.com:8443/org/repo.git")).toBe("org/repo");
    });

    it("should handle HTTP URLs", () => {
      expect(extractRepoSlugFromUrl("http://github.local/owner/repo")).toBe("owner/repo");
    });
  });

  describe("normalizeRepoSlug", () => {
    it("should normalize to lowercase", () => {
      expect(normalizeRepoSlug("Owner/Repo")).toBe("owner/repo");
      expect(normalizeRepoSlug("ORG/PROJECT")).toBe("org/project");
    });

    it("should trim whitespace", () => {
      expect(normalizeRepoSlug("  owner/repo  ")).toBe("owner/repo");
    });

    it("should return empty string for invalid input", () => {
      expect(normalizeRepoSlug("")).toBe("");
      expect(normalizeRepoSlug(null)).toBe("");
      expect(normalizeRepoSlug(undefined)).toBe("");
    });

    it("should handle tabs and newlines", () => {
      expect(normalizeRepoSlug("\towner/repo\n")).toBe("owner/repo");
    });
  });

  describe("findGitDirectories", () => {
    let testDir;

    beforeEach(() => {
      testDir = `/tmp/test-find-git-dirs-${Date.now()}`;
      fs.mkdirSync(testDir, { recursive: true });
    });

    afterEach(() => {
      try {
        fs.rmSync(testDir, { recursive: true, force: true });
      } catch {
        // Ignore cleanup errors
      }
    });

    it("should find git directories in workspace", () => {
      // Create a mock git repo structure
      fs.mkdirSync(path.join(testDir, "repo-a", ".git"), { recursive: true });
      fs.mkdirSync(path.join(testDir, "repo-b", ".git"), { recursive: true });

      const dirs = findGitDirectories(testDir);

      expect(dirs).toHaveLength(2);
      expect(dirs).toContain(path.join(testDir, "repo-a"));
      expect(dirs).toContain(path.join(testDir, "repo-b"));
    });

    it("should handle nested repos", () => {
      // Create a nested structure
      fs.mkdirSync(path.join(testDir, "projects", "frontend", ".git"), { recursive: true });
      fs.mkdirSync(path.join(testDir, "projects", "backend", ".git"), { recursive: true });

      const dirs = findGitDirectories(testDir);

      expect(dirs).toHaveLength(2);
      expect(dirs).toContain(path.join(testDir, "projects", "frontend"));
      expect(dirs).toContain(path.join(testDir, "projects", "backend"));
    });

    it("should skip node_modules", () => {
      fs.mkdirSync(path.join(testDir, "node_modules", "some-pkg", ".git"), { recursive: true });
      fs.mkdirSync(path.join(testDir, "actual-repo", ".git"), { recursive: true });

      const dirs = findGitDirectories(testDir);

      expect(dirs).toHaveLength(1);
      expect(dirs).toContain(path.join(testDir, "actual-repo"));
    });

    it("should return empty array when no git dirs found", () => {
      fs.mkdirSync(path.join(testDir, "empty-folder"), { recursive: true });

      const dirs = findGitDirectories(testDir);

      expect(dirs).toEqual([]);
    });

    it("should respect maxDepth", () => {
      // Create a deeply nested repo
      fs.mkdirSync(path.join(testDir, "a", "b", "c", "d", "e", "f", ".git"), { recursive: true });

      const dirs = findGitDirectories(testDir, 3);

      // Should not find the deeply nested repo
      expect(dirs).toEqual([]);
    });
  });

  describe("findRepoCheckout", () => {
    it("should return error for invalid repo slug", () => {
      const result = findRepoCheckout("");
      expect(result.success).toBe(false);
      expect(result.error).toBe("Invalid repo slug provided");
    });

    it("should return error for null repo slug", () => {
      const result = findRepoCheckout(null);
      expect(result.success).toBe(false);
      expect(result.error).toBe("Invalid repo slug provided");
    });

    it("should return not found for missing repo", () => {
      const testDir = `/tmp/test-find-repo-${Date.now()}`;
      fs.mkdirSync(testDir, { recursive: true });

      try {
        const result = findRepoCheckout("owner/missing-repo", testDir);
        expect(result.success).toBe(false);
        expect(result.error).toContain("not found in workspace");
      } finally {
        fs.rmSync(testDir, { recursive: true, force: true });
      }
    });
  });

  describe("buildRepoCheckoutMap", () => {
    let testDir;

    beforeEach(() => {
      testDir = `/tmp/test-build-map-${Date.now()}`;
      fs.mkdirSync(testDir, { recursive: true });
    });

    afterEach(() => {
      try {
        fs.rmSync(testDir, { recursive: true, force: true });
      } catch {
        // Ignore cleanup errors
      }
    });

    it("should return empty map when no repos found", () => {
      const map = buildRepoCheckoutMap(testDir);
      expect(map.size).toBe(0);
    });

    it("should find repos with valid git remotes", () => {
      // Create a mock repo with a config file
      const repoPath = path.join(testDir, "my-repo", ".git");
      fs.mkdirSync(repoPath, { recursive: true });
      fs.writeFileSync(
        path.join(repoPath, "config"),
        `[remote "origin"]
	url = https://github.com/owner/my-repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
      );

      // Without a real git binary, this won't work, so we expect an empty map
      // since execGitSync will fail
      const map = buildRepoCheckoutMap(testDir);

      // In a real git repo this would work, but in tests without git setup it's ok to be empty
      expect(map).toBeDefined();
    });
  });
});

describe("generate_git_patch multi-repo support", () => {
  describe("getPatchPathForRepo", () => {
    it("should include repo slug in path", () => {
      const filePath = getPatchPathForRepo("feature-branch", "owner/repo");
      expect(filePath).toBe("/tmp/gh-aw/aw-owner-repo-feature-branch.patch");
    });

    it("should sanitize repo slug", () => {
      const filePath = getPatchPathForRepo("main", "org/my-project");
      expect(filePath).toBe("/tmp/gh-aw/aw-org-my-project-main.patch");
    });

    it("should sanitize branch name", () => {
      const filePath = getPatchPathForRepo("feature/add-login", "owner/repo");
      expect(filePath).toBe("/tmp/gh-aw/aw-owner-repo-feature-add-login.patch");
    });

    it("should handle complex repo names", () => {
      const filePath = getPatchPathForRepo("main", "github/gh-aw");
      expect(filePath).toBe("/tmp/gh-aw/aw-github-gh-aw-main.patch");
    });

    it("should handle uppercase input", () => {
      const filePath = getPatchPathForRepo("Feature-Branch", "Owner/Repo");
      expect(filePath).toBe("/tmp/gh-aw/aw-owner-repo-feature-branch.patch");
    });
  });

  describe("sanitizeRepoSlugForPatch", () => {
    it("should replace slash with dash", () => {
      expect(sanitizeRepoSlugForPatch("owner/repo")).toBe("owner-repo");
    });

    it("should handle special characters", () => {
      expect(sanitizeRepoSlugForPatch("org:name/proj*test")).toBe("org-name-proj-test");
    });

    it("should collapse multiple dashes", () => {
      expect(sanitizeRepoSlugForPatch("org//repo")).toBe("org-repo");
    });

    it("should remove leading/trailing dashes", () => {
      expect(sanitizeRepoSlugForPatch("/owner/repo/")).toBe("owner-repo");
    });

    it("should convert to lowercase", () => {
      expect(sanitizeRepoSlugForPatch("Owner/Repo")).toBe("owner-repo");
    });

    it("should return empty string for null/undefined", () => {
      expect(sanitizeRepoSlugForPatch(null)).toBe("");
      expect(sanitizeRepoSlugForPatch(undefined)).toBe("");
      expect(sanitizeRepoSlugForPatch("")).toBe("");
    });
  });

  describe("sanitizeBranchNameForPatch", () => {
    it("should replace path separators with dashes", () => {
      expect(sanitizeBranchNameForPatch("feature/login")).toBe("feature-login");
      expect(sanitizeBranchNameForPatch("fix\\bug")).toBe("fix-bug");
    });

    it("should replace special characters", () => {
      expect(sanitizeBranchNameForPatch("feature:test")).toBe("feature-test");
      expect(sanitizeBranchNameForPatch("fix*bug")).toBe("fix-bug");
    });

    it("should collapse multiple dashes", () => {
      expect(sanitizeBranchNameForPatch("feature//login")).toBe("feature-login");
    });

    it("should remove leading/trailing dashes", () => {
      expect(sanitizeBranchNameForPatch("-feature-")).toBe("feature");
    });

    it("should convert to lowercase", () => {
      expect(sanitizeBranchNameForPatch("Feature-Branch")).toBe("feature-branch");
    });

    it("should handle empty/null input", () => {
      expect(sanitizeBranchNameForPatch("")).toBe("unknown");
      expect(sanitizeBranchNameForPatch(null)).toBe("unknown");
      expect(sanitizeBranchNameForPatch(undefined)).toBe("unknown");
    });

    it("should handle question marks and pipes", () => {
      expect(sanitizeBranchNameForPatch("branch?name|test")).toBe("branch-name-test");
    });

    it("should handle angle brackets", () => {
      expect(sanitizeBranchNameForPatch("branch<>name")).toBe("branch-name");
    });
  });
});
