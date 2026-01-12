// @ts-check
import { describe, it, expect, beforeAll, beforeEach, vi } from "vitest";

let main;

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setOutput: vi.fn(),
};

const mockGithub = {
  graphql: vi.fn(),
};

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  payload: {},
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

beforeAll(async () => {
  const mod = await import("./create_project.cjs");
  main = mod.main;
});

describe("create_project", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockContext.payload = {};
  });

  it("should create a project with explicit title and owner", async () => {
    mockGithub.graphql
      .mockResolvedValueOnce({
        // Get owner ID
        organization: {
          id: "MDEyOk9yZ2FuaXphdGlvbjg5NjE1ODgy",
        },
      })
      .mockResolvedValueOnce({
        // Create project
        createProjectV2: {
          projectV2: {
            id: "PVT_test123",
            number: 42,
            title: "Test Campaign",
            url: "https://github.com/orgs/test-org/projects/42",
          },
        },
      });

    const handler = await main({ max: 1, target_owner: "" });

    const result = await handler(
      {
        title: "Test Campaign",
        owner: "test-org",
        owner_type: "org",
      },
      {}
    );

    expect(result.success).toBe(true);
    expect(result.projectId).toBe("PVT_test123");
    expect(result.projectNumber).toBe(42);
    expect(result.projectTitle).toBe("Test Campaign");
    expect(result.projectUrl).toBe("https://github.com/orgs/test-org/projects/42");

    expect(mockGithub.graphql).toHaveBeenCalledTimes(2);
  });

  it("should generate title from issue context when not provided", async () => {
    mockContext.payload = {
      issue: {
        title: "Testing Campaigns",
        number: 123,
      },
    };

    mockGithub.graphql
      .mockResolvedValueOnce({
        organization: {
          id: "MDEyOk9yZ2FuaXphdGlvbjg5NjE1ODgy",
        },
      })
      .mockResolvedValueOnce({
        createProjectV2: {
          projectV2: {
            id: "PVT_test456",
            number: 43,
            title: "Campaign: Testing Campaigns",
            url: "https://github.com/orgs/test-org/projects/43",
          },
        },
      });

    const handler = await main({ max: 1, target_owner: "test-org" });

    const result = await handler(
      {
        owner_type: "org",
      },
      {}
    );

    expect(result.success).toBe(true);
    expect(result.projectTitle).toBe("Campaign: Testing Campaigns");
  });

  it("should add item to project when item_url is provided", async () => {
    mockGithub.graphql
      .mockResolvedValueOnce({
        organization: {
          id: "MDEyOk9yZ2FuaXphdGlvbjg5NjE1ODgy",
        },
      })
      .mockResolvedValueOnce({
        createProjectV2: {
          projectV2: {
            id: "PVT_test789",
            number: 44,
            title: "Test Project",
            url: "https://github.com/orgs/test-org/projects/44",
          },
        },
      })
      .mockResolvedValueOnce({
        // Get issue node ID
        repository: {
          issue: {
            id: "I_kwDOPc1QR87irmxQ",
          },
        },
      })
      .mockResolvedValueOnce({
        // Add item to project
        addProjectV2ItemById: {
          item: {
            id: "PVTI_test123",
          },
        },
      });

    const handler = await main({ max: 1, target_owner: "test-org" });

    const result = await handler(
      {
        title: "Test Project",
        item_url: "https://github.com/test-owner/test-repo/issues/9707",
      },
      {}
    );

    expect(result.success).toBe(true);
    expect(result.itemId).toBe("PVTI_test123");
    expect(mockGithub.graphql).toHaveBeenCalledTimes(4);
  });

  it("should handle user owner type", async () => {
    mockGithub.graphql
      .mockResolvedValueOnce({
        // Get user ID (not organization)
        user: {
          id: "MDQ6VXNlcjg5NjE1ODgy",
        },
      })
      .mockResolvedValueOnce({
        createProjectV2: {
          projectV2: {
            id: "PVT_user123",
            number: 45,
            title: "User Project",
            url: "https://github.com/users/test-user/projects/45",
          },
        },
      });

    const handler = await main({ max: 1, target_owner: "" });

    const result = await handler(
      {
        title: "User Project",
        owner: "test-user",
        owner_type: "user",
      },
      {}
    );

    expect(result.success).toBe(true);
    expect(result.projectUrl).toBe("https://github.com/users/test-user/projects/45");
  });

  it("should fail when no owner specified and no default configured", async () => {
    const handler = await main({ max: 1, target_owner: "" });

    const result = await handler(
      {
        title: "Test Project",
      },
      {}
    );

    expect(result.success).toBe(false);
    expect(result.error).toContain("No owner specified");
  });

  it("should fail when title cannot be generated", async () => {
    mockContext.payload = {}; // No issue context

    const handler = await main({ max: 1, target_owner: "test-org" });

    const result = await handler({}, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Missing required field 'title'");
  });

  it("should respect max count limit", async () => {
    mockGithub.graphql
      .mockResolvedValueOnce({
        organization: {
          id: "MDEyOk9yZ2FuaXphdGlvbjg5NjE1ODgy",
        },
      })
      .mockResolvedValueOnce({
        createProjectV2: {
          projectV2: {
            id: "PVT_first",
            number: 1,
            title: "First Project",
            url: "https://github.com/orgs/test-org/projects/1",
          },
        },
      });

    const handler = await main({ max: 1, target_owner: "test-org" });

    // First call should succeed
    const result1 = await handler(
      {
        title: "First Project",
      },
      {}
    );
    expect(result1.success).toBe(true);

    // Second call should fail due to max limit
    const result2 = await handler(
      {
        title: "Second Project",
      },
      {}
    );
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count of 1 reached");
  });

  it("should handle GraphQL errors gracefully", async () => {
    mockGithub.graphql.mockRejectedValueOnce(new Error("github-actions[bot] does not have permission to create projects"));

    const handler = await main({ max: 1, target_owner: "test-org" });

    const result = await handler(
      {
        title: "Test Project",
      },
      {}
    );

    expect(result.success).toBe(false);
    expect(result.error).toBeTruthy();
  });

  it("should warn when item_url format is invalid", async () => {
    mockGithub.graphql
      .mockResolvedValueOnce({
        organization: {
          id: "MDEyOk9yZ2FuaXphdGlvbjg5NjE1ODgy",
        },
      })
      .mockResolvedValueOnce({
        createProjectV2: {
          projectV2: {
            id: "PVT_test",
            number: 50,
            title: "Test Project",
            url: "https://github.com/orgs/test-org/projects/50",
          },
        },
      });

    const handler = await main({ max: 1, target_owner: "test-org" });

    const result = await handler(
      {
        title: "Test Project",
        item_url: "invalid-url",
      },
      {}
    );

    expect(result.success).toBe(true);
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not parse item URL"));
  });
});
