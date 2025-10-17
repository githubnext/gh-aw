import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  setCancelled: vi.fn(),
  setError: vi.fn(),

  // Input/state functions (less commonly used but included for completeness)
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
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
  eventName: "issues",
  actor: "testuser",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("check_permissions.cjs", () => {
  let checkPermissionsScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = process.env.GITHUB_AW_REQUIRED_ROLES;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.actor = "testuser";
    global.context.repo = {
      owner: "testowner",
      repo: "testrepo",
    };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "check_permissions.cjs");
    checkPermissionsScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv !== undefined) {
      process.env.GITHUB_AW_REQUIRED_ROLES = originalEnv;
    } else {
      delete process.env.GITHUB_AW_REQUIRED_ROLES;
    }
  });

  it("should fail job when no permissions specified", async () => {
    delete process.env.GITHUB_AW_REQUIRED_ROLES;

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith(
      "❌ Configuration error: Required permissions not specified. Contact repository administrator."
    );
    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
  });

  it("should fail job when permissions are empty", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "";

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith(
      "❌ Configuration error: Required permissions not specified. Contact repository administrator."
    );
    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
  });

  it("should skip validation for safe events", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.eventName = "workflow_dispatch";

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("✅ Event workflow_dispatch does not require validation");
    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });
  it("should pass validation for admin permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer,write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'testuser' has required permissions for testowner/testrepo");
    expect(mockCore.info).toHaveBeenCalledWith("Required permissions: admin, maintainer, write");
    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: admin");
    expect(mockCore.info).toHaveBeenCalledWith("✅ User has admin access to repository");

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should pass validation for maintain permission when maintainer is required", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "maintain" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("✅ User has maintain access to repository");

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should pass validation for write permission when write is required", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write,triage";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository");

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should fail the job for insufficient permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: write");
    expect(mockCore.warning).toHaveBeenCalledWith("User permission 'write' does not meet requirements: admin, maintainer");
    expect(mockCore.warning).toHaveBeenCalledWith(
      "Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, maintainer"
    );
  });

  it("should fail the job for read permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "read" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Repository permission level: read");
    expect(mockCore.warning).toHaveBeenCalledWith("User permission 'read' does not meet requirements: admin, write");
    expect(mockCore.warning).toHaveBeenCalledWith(
      "Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, write"
    );
  });

  it("should fail the job on API errors", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";

    const apiError = new Error("API Error: Not Found");
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(apiError);

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith("Repository permission check failed: API Error: Not Found");
  });

  it("should handle different actor names correctly", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.actor = "different-user";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "different-user",
    });

    expect(mockCore.info).toHaveBeenCalledWith("Checking if user 'different-user' has required permissions for testowner/testrepo");

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should handle triage permission correctly", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write,triage";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "triage" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("✅ User has triage access to repository");

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should handle single permission requirement", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Required permissions: write");
    expect(mockCore.info).toHaveBeenCalledWith("✅ User has write access to repository");

    // Should not call any error or warning methods
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should skip validation for workflow_run events", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.eventName = "workflow_run";

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("✅ Event workflow_run does not require validation");
    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
  });

  it("should skip validation for schedule events", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.eventName = "schedule";

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("✅ Event schedule does not require validation");
    expect(mockGithub.rest.repos.getCollaboratorPermissionLevel).not.toHaveBeenCalled();
  });
});
