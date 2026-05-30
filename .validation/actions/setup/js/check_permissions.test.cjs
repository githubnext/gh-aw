import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = {
    debug: vi.fn(),
    info: vi.fn(),
    notice: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
    setFailed: vi.fn(),
    setOutput: vi.fn(),
    exportVariable: vi.fn(),
    setSecret: vi.fn(),
    setCancelled: vi.fn(),
    setError: vi.fn(),
    getInput: vi.fn(),
    getBooleanInput: vi.fn(),
    getMultilineInput: vi.fn(),
    getState: vi.fn(),
    saveState: vi.fn(),
    startGroup: vi.fn(),
    endGroup: vi.fn(),
    group: vi.fn(),
    addPath: vi.fn(),
    setCommandEcho: vi.fn(),
    isDebug: vi.fn().mockReturnValue(!1),
    getIDToken: vi.fn(),
    toPlatformPath: vi.fn(),
    toPosixPath: vi.fn(),
    toWin32Path: vi.fn(),
    summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
  },
  mockGithub = { rest: { repos: { getCollaboratorPermissionLevel: vi.fn() } } },
  mockContext = { eventName: "issues", actor: "testuser", repo: { owner: "testowner", repo: "testrepo" } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("check_permissions.cjs", () => {
    let checkPermissionsScript, originalEnv;
    (beforeEach(() => {
      (vi.clearAllMocks(), (originalEnv = process.env.GH_AW_REQUIRED_ROLES), (global.context.eventName = "issues"), (global.context.actor = "testuser"), (global.context.repo = { owner: "testowner", repo: "testrepo" }));
      const scriptPath = path.join(process.cwd(), "check_permissions.cjs");
      checkPermissionsScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        void 0 !== originalEnv ? (process.env.GH_AW_REQUIRED_ROLES = originalEnv) : delete process.env.GH_AW_REQUIRED_ROLES;
      }),
      it("should fail job when no permissions specified", async () => {
        (delete process.env.GH_AW_REQUIRED_ROLES,
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.error).toHaveBeenCalledWith("❌ Configuration error: Required permissions not specified. Contact repository administrator."),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled());
      }),
      it("should fail job when permissions are empty", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = ""),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.error).toHaveBeenCalledWith("❌ Configuration error: Required permissions not specified. Contact repository administrator."),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled());
      }),
      it("should skip validation for safe events", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin"),
          (global.context.eventName = "workflow_dispatch"),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Event workflow_dispatch does not require validation"),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled(),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should skip validation for merge_group events", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin"),
          (global.context.eventName = "merge_group"),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Event merge_group does not require validation"),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled(),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should pass validation for admin permission", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin,maintainer,write"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "admin" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", username: "testuser" }),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'testuser' has required permissions for testowner/testrepo"),
          expect(mockCore.info).toHaveBeenCalledWith("Required permissions: admin, maintainer, write"),
          expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: admin"),
          expect(mockCore.info).toHaveBeenCalledWith("✅ User has admin access to repository"),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should pass validation for maintain permission when maintainer is required", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin,maintainer"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "maintain" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("✅ User has maintain access to repository"),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should pass validation for write permission when write is required", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin,write,triage"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "write" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository"),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should fail the job for insufficient permission", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin,maintainer"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "write" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: write"),
          expect(mockCore.warning).toHaveBeenCalledWith("User permission 'write' does not meet requirements: admin, maintainer"),
          expect(mockCore.warning).toHaveBeenCalledWith("Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, maintainer"));
      }),
      it("should fail the job for read permission", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin,write"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "read" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: read"),
          expect(mockCore.warning).toHaveBeenCalledWith("User permission 'read' does not meet requirements: admin, write"),
          expect(mockCore.warning).toHaveBeenCalledWith("Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, write"));
      }),
      it("should fail the job on API errors", async () => {
        process.env.GH_AW_REQUIRED_ROLES = "admin";
        const apiError = new Error("API Error: Not Found");
        (mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: API Error: Not Found"));
      }),
      it("should handle different actor names correctly", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin"),
          (global.context.actor = "different-user"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "admin" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", username: "different-user" }),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'different-user' has required permissions for testowner/testrepo"),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should handle triage permission correctly", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin,write,triage"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "triage" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("✅ User has triage access to repository"),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should handle single permission requirement", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "write"),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "write" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Required permissions: write"),
          expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository"),
          expect(mockCore.error).not.toHaveBeenCalled(),
          expect(mockCore.warning).not.toHaveBeenCalled());
      }),
      it("should skip validation for schedule events", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin"),
          (global.context.eventName = "schedule"),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Event schedule does not require validation"),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled());
      }),
      it("should correctly extract owner and repo from context.repo", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin"),
          (global.context.eventName = "issues"),
          (global.context.repo = { owner: "custom-owner", repo: "custom-repo" }),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "admin" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({ owner: "custom-owner", repo: "custom-repo", username: "testuser" }),
          expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'testuser' has required permissions for custom-owner/custom-repo"));
      }),
      it("should handle context with different repo names correctly", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "write"),
          (global.context.eventName = "pull_request"),
          (global.context.actor = "contributor"),
          (global.context.repo = { owner: "org-name", repo: "project-name" }),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "write" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({ owner: "org-name", repo: "project-name", username: "contributor" }),
          expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository"));
      }),
      it("should correctly destructure context properties in safe event", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "admin"),
          (global.context.eventName = "workflow_dispatch"),
          (global.context.actor = "dispatch-user"),
          (global.context.repo = { owner: "test-org", repo: "test-repo" }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("✅ Event workflow_dispatch does not require validation"),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled());
      }),
      it("should handle repo names with hyphens and underscores", async () => {
        ((process.env.GH_AW_REQUIRED_ROLES = "maintainer"),
          (global.context.eventName = "push"),
          (global.context.actor = "test-user"),
          (global.context.repo = { owner: "my-org", repo: "my_test-repo" }),
          mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({ data: { permission: "maintain" } }),
          await eval(`(async () => { ${checkPermissionsScript}; await main(); })()`),
          expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({ owner: "my-org", repo: "my_test-repo", username: "test-user" }),
          expect(mockCore.info).toHaveBeenCalledWith("✅ User has maintain access to repository"));
      }));
  }));
