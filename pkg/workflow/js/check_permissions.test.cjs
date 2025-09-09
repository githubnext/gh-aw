import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setOutput: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setCancelled: vi.fn(),
  setError: vi.fn(),
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
    global.context.actor = "testuser";
    global.context.repo = {
      owner: "testowner",
      repo: "testrepo",
    };

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/check_permissions.cjs"
    );
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

    expect(mockCore.setError).toHaveBeenCalledWith(
      "❌ Configuration error: Required permissions not specified. Contact repository administrator."
    );
    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).not.toHaveBeenCalled();
  });

  it("should fail job when permissions are empty", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "";

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.setError).toHaveBeenCalledWith(
      "❌ Configuration error: Required permissions not specified. Contact repository administrator."
    );
    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).not.toHaveBeenCalled();
  });

  it("should set is_team_member to true for admin permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer,write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "testuser",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'testuser' has required permissions for testowner/testrepo"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Required permissions: admin, maintainer, write"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: admin"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has admin access to repository"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should set is_team_member to true for maintain permission when maintainer is required", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "maintain" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has maintain access to repository"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should set is_team_member to true for write permission when write is required", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write,triage";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has write access to repository"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should fail the job directly for insufficient permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,maintainer";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: write"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "User permission 'write' does not meet requirements: admin, maintainer"
    );
    expect(mockCore.setError).toHaveBeenCalledWith(
      "❌ Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, maintainer"
    );

    consoleSpy.mockRestore();
  });

  it("should fail the job directly for read permission", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "read" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "Repository permission level: read"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "User permission 'read' does not meet requirements: admin, write"
    );
    expect(mockCore.setError).toHaveBeenCalledWith(
      "❌ Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin, write"
    );

    consoleSpy.mockRestore();
  });

  it("should fail the job directly on API errors", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";

    const apiError = new Error("API Error: Not Found");
    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockRejectedValue(
      apiError
    );

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith(
      "Repository permission check failed: API Error: Not Found"
    );
    expect(mockCore.setError).toHaveBeenCalledWith(
      "❌ Access denied: Only authorized users can trigger this workflow. User 'testuser' is not authorized. Required permissions: admin"
    );

    consoleSpy.mockRestore();
  });

  it("should handle different actor names correctly", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin";
    global.context.actor = "different-user";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "admin" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(
      mockGithub.rest.repos.getCollaboratorPermissionLevel
    ).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      username: "different-user",
    });

    expect(consoleSpy).toHaveBeenCalledWith(
      "Checking if user 'different-user' has required permissions for testowner/testrepo"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should handle triage permission correctly", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "admin,write,triage";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "triage" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has triage access to repository"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });

  it("should handle single permission requirement", async () => {
    process.env.GITHUB_AW_REQUIRED_ROLES = "write";

    mockGithub.rest.repos.getCollaboratorPermissionLevel.mockResolvedValue({
      data: { permission: "write" },
    });

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${checkPermissionsScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith("Required permissions: write");
    expect(consoleSpy).toHaveBeenCalledWith(
      "✅ User has write access to repository"
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_team_member", "true");

    consoleSpy.mockRestore();
  });
});
