import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the context global
const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
};

global.context = mockContext;

describe("repo_helpers", () => {
  beforeEach(() => {
    vi.resetModules();
    delete process.env.GH_AW_TARGET_REPO_SLUG;
    global.context = mockContext;
  });

  describe("parseAllowedRepos", () => {
    it("should return empty set when value is undefined", async () => {
      const { parseAllowedRepos } = await import("./repo_helpers.cjs");
      const result = parseAllowedRepos(undefined);
      expect(result.size).toBe(0);
    });

    it("should parse single repo from string", async () => {
      const { parseAllowedRepos } = await import("./repo_helpers.cjs");
      const result = parseAllowedRepos("org/repo-a");
      expect(result.size).toBe(1);
      expect(result.has("org/repo-a")).toBe(true);
    });

    it("should parse multiple repos from comma-separated string", async () => {
      const { parseAllowedRepos } = await import("./repo_helpers.cjs");
      const result = parseAllowedRepos("org/repo-a, org/repo-b, org/repo-c");
      expect(result.size).toBe(3);
      expect(result.has("org/repo-a")).toBe(true);
      expect(result.has("org/repo-b")).toBe(true);
      expect(result.has("org/repo-c")).toBe(true);
    });

    it("should parse repos from array", async () => {
      const { parseAllowedRepos } = await import("./repo_helpers.cjs");
      const result = parseAllowedRepos(["org/repo-a", "org/repo-b"]);
      expect(result.size).toBe(2);
      expect(result.has("org/repo-a")).toBe(true);
      expect(result.has("org/repo-b")).toBe(true);
    });

    it("should trim whitespace from repo names in string", async () => {
      const { parseAllowedRepos } = await import("./repo_helpers.cjs");
      const result = parseAllowedRepos("  org/repo-a  ,  org/repo-b  ");
      expect(result.has("org/repo-a")).toBe(true);
      expect(result.has("org/repo-b")).toBe(true);
    });

    it("should trim whitespace from repo names in array", async () => {
      const { parseAllowedRepos } = await import("./repo_helpers.cjs");
      const result = parseAllowedRepos(["  org/repo-a  ", "  org/repo-b  "]);
      expect(result.has("org/repo-a")).toBe(true);
      expect(result.has("org/repo-b")).toBe(true);
    });

    it("should filter out empty strings", async () => {
      const { parseAllowedRepos } = await import("./repo_helpers.cjs");
      const result = parseAllowedRepos("org/repo-a,,org/repo-b,  ,");
      expect(result.size).toBe(2);
    });
  });

  describe("getDefaultTargetRepo", () => {
    it("should return target-repo from config when provided", async () => {
      const { getDefaultTargetRepo } = await import("./repo_helpers.cjs");
      const config = { "target-repo": "config-org/config-repo" };
      const result = getDefaultTargetRepo(config);
      expect(result).toBe("config-org/config-repo");
    });

    it("should prefer config target-repo over env variable", async () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "env-org/env-repo";
      const { getDefaultTargetRepo } = await import("./repo_helpers.cjs");
      const config = { "target-repo": "config-org/config-repo" };
      const result = getDefaultTargetRepo(config);
      expect(result).toBe("config-org/config-repo");
    });

    it("should return target-repo override when set", async () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "override-org/override-repo";
      const { getDefaultTargetRepo } = await import("./repo_helpers.cjs");
      const result = getDefaultTargetRepo();
      expect(result).toBe("override-org/override-repo");
    });

    it("should fall back to context repo when no override", async () => {
      const { getDefaultTargetRepo } = await import("./repo_helpers.cjs");
      const result = getDefaultTargetRepo();
      expect(result).toBe("test-owner/test-repo");
    });
  });

  describe("validateRepo", () => {
    it("should allow default repo", async () => {
      const { validateRepo } = await import("./repo_helpers.cjs");
      const result = validateRepo("default/repo", "default/repo", new Set());
      expect(result.valid).toBe(true);
      expect(result.error).toBe(null);
    });

    it("should allow repos in allowed list", async () => {
      const { validateRepo } = await import("./repo_helpers.cjs");
      const allowedRepos = new Set(["org/repo-a", "org/repo-b"]);
      const result = validateRepo("org/repo-a", "default/repo", allowedRepos);
      expect(result.valid).toBe(true);
      expect(result.error).toBe(null);
    });

    it("should reject repos not in allowed list", async () => {
      const { validateRepo } = await import("./repo_helpers.cjs");
      const allowedRepos = new Set(["org/repo-a"]);
      const result = validateRepo("org/other-repo", "default/repo", allowedRepos);
      expect(result.valid).toBe(false);
      expect(result.error).toContain("not in the allowed-repos list");
    });

    it("should include allowed repos in error message", async () => {
      const { validateRepo } = await import("./repo_helpers.cjs");
      const allowedRepos = new Set(["org/repo-a", "org/repo-b"]);
      const result = validateRepo("org/other-repo", "default/repo", allowedRepos);
      expect(result.error).toContain("default/repo");
      expect(result.error).toContain("org/repo-a");
      expect(result.error).toContain("org/repo-b");
    });
  });

  describe("parseRepoSlug", () => {
    it("should parse valid repo slug", async () => {
      const { parseRepoSlug } = await import("./repo_helpers.cjs");
      const result = parseRepoSlug("owner/repo");
      expect(result).toEqual({ owner: "owner", repo: "repo" });
    });

    it("should return null for invalid slug without slash", async () => {
      const { parseRepoSlug } = await import("./repo_helpers.cjs");
      const result = parseRepoSlug("invalid");
      expect(result).toBeNull();
    });

    it("should return null for slug with too many slashes", async () => {
      const { parseRepoSlug } = await import("./repo_helpers.cjs");
      const result = parseRepoSlug("owner/repo/extra");
      expect(result).toBeNull();
    });

    it("should return null for empty owner", async () => {
      const { parseRepoSlug } = await import("./repo_helpers.cjs");
      const result = parseRepoSlug("/repo");
      expect(result).toBeNull();
    });

    it("should return null for empty repo", async () => {
      const { parseRepoSlug } = await import("./repo_helpers.cjs");
      const result = parseRepoSlug("owner/");
      expect(result).toBeNull();
    });
  });
});
