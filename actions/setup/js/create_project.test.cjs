// @ts-check
import { describe, it, expect, beforeAll, beforeEach, vi } from "vitest";

// Mock loadAgentOutput BEFORE importing create_project
vi.mock("./load_agent_output.cjs", () => ({
  loadAgentOutput: vi.fn(),
}));

const { loadAgentOutput } = await import("./load_agent_output.cjs");

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
    process.env.GH_AW_CREATE_PROJECT_TARGET_OWNER = "";
  });

  it("should create a project with explicit title and owner", async () => {
    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
          title: "Test Campaign",
          owner: "test-org",
          owner_type: "org",
        },
      ],
    });

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

    await main();

    expect(mockGithub.graphql).toHaveBeenCalledTimes(2);
    expect(mockCore.setOutput).toHaveBeenCalledWith("project-id", "PVT_test123");
    expect(mockCore.setOutput).toHaveBeenCalledWith("project-number", 42);
    expect(mockCore.setOutput).toHaveBeenCalledWith("project-title", "Test Campaign");
    expect(mockCore.setOutput).toHaveBeenCalledWith("project-url", "https://github.com/orgs/test-org/projects/42");
  });

  it("should generate title from issue context when not provided", async () => {
    mockContext.payload = {
      issue: {
        title: "Testing Campaigns",
        number: 123,
      },
    };

    process.env.GH_AW_CREATE_PROJECT_TARGET_OWNER = "test-org";

    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
          owner_type: "org",
        },
      ],
    });

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

    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Campaign: Testing Campaigns"));
  });

  it("should add item to project when item_url is provided", async () => {
    process.env.GH_AW_CREATE_PROJECT_TARGET_OWNER = "test-org";

    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
          title: "Test Project",
          item_url: "https://github.com/test-owner/test-repo/issues/9707",
        },
      ],
    });

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

    await main();

    expect(mockGithub.graphql).toHaveBeenCalledTimes(4);
    expect(mockCore.setOutput).toHaveBeenCalledWith("item-id", "PVTI_test123");
  });

  it("should handle user owner type", async () => {
    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
          title: "User Project",
          owner: "test-user",
          owner_type: "user",
        },
      ],
    });

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

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("project-url", "https://github.com/users/test-user/projects/45");
  });

  it("should fail when no owner specified and no default configured", async () => {
    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
          title: "Test Project",
        },
      ],
    });

    await expect(main()).rejects.toThrow();
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("No owner specified"));
  });

  it("should fail when title cannot be generated", async () => {
    mockContext.payload = {}; // No issue context
    process.env.GH_AW_CREATE_PROJECT_TARGET_OWNER = "test-org";

    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
        },
      ],
    });

    await expect(main()).rejects.toThrow();
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Missing required field 'title'"));
  });

  it("should handle GraphQL errors gracefully", async () => {
    process.env.GH_AW_CREATE_PROJECT_TARGET_OWNER = "test-org";

    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
          title: "Test Project",
        },
      ],
    });

    mockGithub.graphql.mockRejectedValueOnce(new Error("github-actions[bot] does not have permission to create projects"));

    await expect(main()).rejects.toThrow();
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to create project"));
  });

  it("should warn when item_url format is invalid", async () => {
    process.env.GH_AW_CREATE_PROJECT_TARGET_OWNER = "test-org";

    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [
        {
          type: "create_project",
          title: "Test Project",
          item_url: "invalid-url",
        },
      ],
    });

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

    await main();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not parse item URL"));
  });

  it("should return early when no items to process", async () => {
    loadAgentOutput.mockReturnValueOnce({
      success: true,
      items: [],
    });

    await main();

    expect(mockCore.info).toHaveBeenCalledWith("No create_project items found in agent output");
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should return early when loadAgentOutput fails", async () => {
    loadAgentOutput.mockReturnValueOnce({
      success: false,
    });

    await main();

    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });
});
