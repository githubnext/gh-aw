import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
};

const mockGithub = {
  rest: {
    repos: {
      listCollaborators: vi.fn(),
      getCollaboratorPermissionLevel: vi.fn(),
    },
    users: {
      getByUsername: vi.fn(),
    },
  },
};

global.core = mockCore;
global.github = mockGithub;

const {
  extractMentions,
  isPayloadUserBot,
  getRecentCollaborators,
  checkUserPermission,
  resolveMentionsLazily,
} = require("./resolve_mentions.cjs");

describe("resolve_mentions.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("extractMentions", () => {
    it("should extract single mention", () => {
      const mentions = extractMentions("Hello @user");
      expect(mentions).toEqual(["user"]);
    });

    it("should extract multiple mentions", () => {
      const mentions = extractMentions("Hello @user1 and @user2");
      expect(mentions).toEqual(["user1", "user2"]);
    });

    it("should deduplicate mentions (case-insensitive)", () => {
      const mentions = extractMentions("Hello @user and @USER and @User");
      expect(mentions).toEqual(["user"]);
    });

    it("should skip mentions in backticks", () => {
      const mentions = extractMentions("Hello `@user` and @realuser");
      expect(mentions).toEqual(["realuser"]);
    });

    it("should handle org/team mentions", () => {
      const mentions = extractMentions("Hello @org/team");
      expect(mentions).toEqual(["org/team"]);
    });

    it("should handle empty text", () => {
      const mentions = extractMentions("");
      expect(mentions).toEqual([]);
    });

    it("should preserve original case", () => {
      const mentions = extractMentions("Hello @UserName");
      expect(mentions).toEqual(["UserName"]);
    });
  });

  describe("isPayloadUserBot", () => {
    it("should return true for bot users", () => {
      expect(isPayloadUserBot({ login: "botuser", type: "Bot" })).toBe(true);
    });

    it("should return false for regular users", () => {
      expect(isPayloadUserBot({ login: "user", type: "User" })).toBe(false);
    });

    it("should return false for null/undefined", () => {
      expect(isPayloadUserBot(null)).toBe(false);
      expect(isPayloadUserBot(undefined)).toBe(false);
    });
  });

  describe("getRecentCollaborators", () => {
    it("should return map of allowed collaborators", async () => {
      mockGithub.rest.repos.listCollaborators.mockResolvedValue({
        data: [
          {
            login: "maintainer1",
            type: "User",
            permissions: { maintain: true, admin: false, push: false },
          },
          {
            login: "admin1",
            type: "User",
            permissions: { maintain: false, admin: true, push: false },
          },
          {
            login: "contributor1",
            type: "User",
            permissions: { maintain: false, admin: false, push: true },
          },
        ],
      });

      const result = await getRecentCollaborators("owner", "repo", mockGithub, mockCore);

      expect(result.size).toBe(3);
      expect(result.get("maintainer1")).toBe(true);
      expect(result.get("admin1")).toBe(true);
      expect(result.get("contributor1")).toBe(false); // Only push access
    });

    it("should exclude bots", async () => {
      mockGithub.rest.repos.listCollaborators.mockResolvedValue({
        data: [
          {
            login: "botuser",
            type: "Bot",
            permissions: { maintain: true, admin: false, push: false },
          },
        ],
      });

      const result = await getRecentCollaborators("owner", "repo", mockGithub, mockCore);

      expect(result.get("botuser")).toBe(false);
    });

    it("should handle API errors gracefully", async () => {
      mockGithub.rest.repos.listCollaborators.mockRejectedValue(new Error("API error"));

      const result = await getRecentCollaborators("owner", "repo", mockGithub, mockCore);

      expect(result.size).toBe(0);
      expect(mockCore.warning).toHaveBeenCalled();
    });

    it("should fetch only first page (30 items)", async () => {
      mockGithub.rest.repos.listCollaborators.mockResolvedValue({
        data: [],
      });

      await getRecentCollaborators("owner", "repo", mockGithub, mockCore);

      expect(mockGithub.rest.repos.listCollaborators).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        affiliation: "direct",
        per_page: 30,
      });
    });
  });

  describe("checkUserPermission", () => {
    it("should return true for maintainer", async () => {
      mockGithub.rest.users.getByUsername.mockResolvedValue({
        data: { login: "user", type: "User" },
      });
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "maintain" },
      });

      const result = await checkUserPermission("user", "owner", "repo", mockGithub, mockCore);

      expect(result).toBe(true);
    });

    it("should return true for admin", async () => {
      mockGithub.rest.users.getByUsername.mockResolvedValue({
        data: { login: "user", type: "User" },
      });
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      const result = await checkUserPermission("user", "owner", "repo", mockGithub, mockCore);

      expect(result).toBe(true);
    });

    it("should return false for regular contributor", async () => {
      mockGithub.rest.users.getByUsername.mockResolvedValue({
        data: { login: "user", type: "User" },
      });
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      const result = await checkUserPermission("user", "owner", "repo", mockGithub, mockCore);

      expect(result).toBe(false);
    });

    it("should return false for bots", async () => {
      mockGithub.rest.users.getByUsername.mockResolvedValue({
        data: { login: "botuser", type: "Bot" },
      });

      const result = await checkUserPermission("botuser", "owner", "repo", mockGithub, mockCore);

      expect(result).toBe(false);
    });

    it("should return false on API errors", async () => {
      mockGithub.rest.users.getByUsername.mockRejectedValue(new Error("User not found"));

      const result = await checkUserPermission("user", "owner", "repo", mockGithub, mockCore);

      expect(result).toBe(false);
    });
  });

  describe("resolveMentionsLazily", () => {
    beforeEach(() => {
      mockGithub.rest.repos.listCollaborators.mockResolvedValue({
        data: [
          {
            login: "maintainer1",
            type: "User",
            permissions: { maintain: true, admin: false, push: false },
          },
        ],
      });
    });

    it("should resolve known authors without API calls", async () => {
      const text = "Hello @author1";
      const result = await resolveMentionsLazily(text, ["author1"], "owner", "repo", mockGithub, mockCore);

      expect(result.allowedMentions).toEqual(["author1"]);
      expect(result.resolvedCount).toBe(0); // No individual API calls needed
    });

    it("should resolve cached collaborators", async () => {
      const text = "Hello @maintainer1";
      const result = await resolveMentionsLazily(text, [], "owner", "repo", mockGithub, mockCore);

      expect(result.allowedMentions).toEqual(["maintainer1"]);
      expect(result.resolvedCount).toBe(0); // Resolved from cache
    });

    it("should query individual users not in cache", async () => {
      mockGithub.rest.users.getByUsername.mockResolvedValue({
        data: { login: "newuser", type: "User" },
      });
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "maintain" },
      });

      const text = "Hello @newuser";
      const result = await resolveMentionsLazily(text, [], "owner", "repo", mockGithub, mockCore);

      expect(result.allowedMentions).toEqual(["newuser"]);
      expect(result.resolvedCount).toBe(1); // Queried individually
    });

    it("should limit to 50 mentions", async () => {
      const mentions = Array.from({ length: 60 }, (_, i) => `@user${i}`).join(" ");
      const result = await resolveMentionsLazily(mentions, [], "owner", "repo", mockGithub, mockCore);

      expect(result.totalMentions).toBe(60);
      expect(result.limitExceeded).toBe(true);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Mention limit exceeded"));
    });

    it("should preserve case in allowed mentions", async () => {
      const text = "Hello @Maintainer1";
      mockGithub.rest.repos.listCollaborators.mockResolvedValue({
        data: [
          {
            login: "maintainer1", // lowercase in API response
            type: "User",
            permissions: { maintain: true, admin: false, push: false },
          },
        ],
      });

      const result = await resolveMentionsLazily(text, [], "owner", "repo", mockGithub, mockCore);

      expect(result.allowedMentions).toEqual(["Maintainer1"]); // Original case from text
    });

    it("should log resolution stats", async () => {
      const text = "Hello @author1 @maintainer1";
      await resolveMentionsLazily(text, ["author1"], "owner", "repo", mockGithub, mockCore);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found 2 unique mentions"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Total allowed mentions"));
    });
  });
});
