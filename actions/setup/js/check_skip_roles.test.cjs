// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

describe("check_skip_roles", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let checkSkipRoles;

  beforeEach(async () => {
    // Mock @actions/core
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
    };

    // Mock @actions/github
    mockGithub = {
      rest: {
        repos: {
          getCollaboratorPermissionLevel: vi.fn(),
        },
      },
    };

    // Mock context
    mockContext = {
      eventName: "issues",
      actor: "test-user",
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
    };

    // Setup global mocks
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;

    // Reset environment variables
    delete process.env.GH_AW_SKIP_ROLES;

    // Reload the module to get fresh instance
    vi.resetModules();
    checkSkipRoles = await import("./check_skip_roles.cjs");
  });

  describe("no skip-roles configured", () => {
    it("should allow workflow to proceed when skip-roles is not set", async () => {
      // Don't set GH_AW_SKIP_ROLES

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "no_skip_roles");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No skip-roles configured"));
    });

    it("should allow workflow to proceed when skip-roles is empty", async () => {
      process.env.GH_AW_SKIP_ROLES = "";

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "no_skip_roles");
    });
  });

  describe("safe events", () => {
    it("should allow workflow for schedule events", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockContext.eventName = "schedule";

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "safe_event");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("safe event"));
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
    });

    it("should allow workflow for merge_group events", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockContext.eventName = "merge_group";

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "safe_event");
      expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
    });
  });

  describe("actor has skip-role", () => {
    it("should skip workflow when actor is admin", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "admin",
          role_name: "admin",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_should_be_skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "admin");
      expect(mockCore.setOutput).toHaveBeenCalledWith("error_message", expect.stringContaining("Workflow skipped"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("has role 'admin'"));
    });

    it("should skip workflow when actor has write permission", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "write",
          role_name: "write",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_should_be_skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "write");
    });

    it("should skip workflow when actor has maintain permission", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,maintain,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "maintain",
          role_name: "maintain",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_should_be_skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "maintain");
    });
  });

  describe("actor does not have skip-role", () => {
    it("should allow workflow when actor has triage permission", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "triage",
          role_name: "triage",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_not_in_skip_roles");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "triage");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not have any skip-roles"));
    });

    it("should allow workflow when actor has read permission", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "read",
          role_name: "read",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_not_in_skip_roles");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "read");
    });

    it("should allow workflow when actor has no repository access", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "none",
          role_name: "none",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_not_in_skip_roles");
      expect(mockCore.setOutput).toHaveBeenCalledWith("user_permission", "none");
    });
  });

  describe("API error handling", () => {
    it("should allow workflow to proceed on API error", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(new Error("API rate limit exceeded"));

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "api_error");
      expect(mockCore.setOutput).toHaveBeenCalledWith("error_message", expect.stringContaining("Skip-roles check failed"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not verify user permissions"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Allowing workflow to proceed due to API error"));
    });

    it("should allow workflow on 404 error (user not found)", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      const error = new Error("Not Found");
      error.status = 404;
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(error);

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "api_error");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not verify user permissions"));
    });
  });

  describe("edge cases", () => {
    it("should handle single skip-role value", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "admin",
          role_name: "admin",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_should_be_skipped");
    });

    it("should handle whitespace in skip-roles list", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin, write, maintain";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "write",
          role_name: "write",
        },
      });

      await checkSkipRoles.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_should_be_skipped");
    });

    it("should handle case-sensitive role matching", async () => {
      process.env.GH_AW_SKIP_ROLES = "admin,write";
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: {
          permission: "ADMIN", // Testing uppercase vs lowercase
          role_name: "admin",
        },
      });

      await checkSkipRoles.main();

      // Should not match due to case sensitivity (permission is "ADMIN" not "admin")
      // Since GitHub API returns lowercase in practice, this is edge case testing
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_roles_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "user_not_in_skip_roles");
    });
  });
});
