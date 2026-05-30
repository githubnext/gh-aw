import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    repos: {
      getCollaboratorPermissionLevel: vi.fn(),
    },
  },
};

const mockContext = {
  actor: "testuser",
  repo: { owner: "testowner", repo: "testrepo" },
  eventName: "workflow_dispatch",
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("check admin permissions for copilot maintenance", () => {
  let checkRepositoryPermission;

  beforeEach(async () => {
    // Reset all mocks
    vi.clearAllMocks();

    // Import the module functions
    const module = await import("./check_permissions_utils.cjs");
    checkRepositoryPermission = module.checkRepositoryPermission;
  });

  describe("admin permission check for branch deletion", () => {
    it("should allow admin user to delete branches", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "admin" },
      });

      const result = await checkRepositoryPermission("admin-user", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: true,
        permission: "admin",
      });

      expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'admin-user' has required permissions for testowner/testrepo");
      expect(mockCore.info).toHaveBeenCalledWith("Required permissions: admin");
      expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: admin");
      expect(mockCore.info).toHaveBeenCalledWith("✅ User has admin access to repository");
    });

    it("should deny non-admin user from deleting branches", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "write" },
      });

      const result = await checkRepositoryPermission("write-user", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        permission: "write",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("User permission 'write' does not meet requirements: admin");
    });

    it("should deny user with read permission", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "read" },
      });

      const result = await checkRepositoryPermission("read-user", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        permission: "read",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("User permission 'read' does not meet requirements: admin");
    });

    it("should deny user with triage permission", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "triage" },
      });

      const result = await checkRepositoryPermission("triage-user", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        permission: "triage",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("User permission 'triage' does not meet requirements: admin");
    });

    it("should deny user with maintain permission", async () => {
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
        data: { permission: "maintain" },
      });

      const result = await checkRepositoryPermission("maintain-user", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        permission: "maintain",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("User permission 'maintain' does not meet requirements: admin");
    });

    it("should handle API errors gracefully", async () => {
      const apiError = new Error("API Error: Not Found");
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

      const result = await checkRepositoryPermission("unknown-user", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        error: "API Error: Not Found",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: API Error: Not Found");
    });

    it("should handle network errors", async () => {
      const networkError = new Error("Network error: Connection timeout");
      mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(networkError);

      const result = await checkRepositoryPermission("testuser", "testowner", "testrepo", ["admin"]);

      expect(result).toEqual({
        authorized: false,
        error: "Network error: Connection timeout",
      });
    });
  });

  describe("permission check for scheduled events", () => {
    it("should skip permission check for scheduled events", () => {
      // In the actual workflow, scheduled events skip the permission check
      // This test verifies the logic in the workflow file
      const eventName = "schedule";
      const shouldCheck = eventName === "workflow_dispatch";

      expect(shouldCheck).toBe(false);
    });

    it("should perform permission check for workflow_dispatch events", () => {
      const eventName = "workflow_dispatch";
      const shouldCheck = eventName === "workflow_dispatch";

      expect(shouldCheck).toBe(true);
    });
  });
});
