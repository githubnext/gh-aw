import { describe, it, expect, beforeEach, vi } from "vitest";

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

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  eventName: "issues",
  payload: {
    issue: {
      number: 123,
    },
  },
};

const mockGraphql = vi.fn();

const mockGithub = {
  rest: {
    issues: {
      get: vi.fn(),
    },
  },
  graphql: mockGraphql,
};

global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("set_issue_type (Handler Factory Architecture)", () => {
  let handler;

  const issueNodeId = "I_kwDOABCD123456";
  const bugTypeId = "IT_kwDOABCD_bug";
  const featureTypeId = "IT_kwDOABCD_feature";

  const mockIssueTypesQuery = {
    repository: {
      issueTypes: {
        nodes: [
          { id: bugTypeId, name: "Bug" },
          { id: featureTypeId, name: "Feature" },
        ],
      },
    },
  };

  beforeEach(async () => {
    vi.clearAllMocks();

    mockGithub.rest.issues.get.mockResolvedValue({ data: { node_id: issueNodeId } });
    mockGraphql.mockImplementation(query => {
      if (query.includes("issueTypes")) {
        return Promise.resolve(mockIssueTypesQuery);
      }
      if (query.includes("updateIssue")) {
        return Promise.resolve({ updateIssue: { issue: { id: issueNodeId } } });
      }
      return Promise.resolve({});
    });

    const { main } = require("./set_issue_type.cjs");
    handler = await main({ max: 5 });
  });

  it("should return a function from main()", async () => {
    const { main } = require("./set_issue_type.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should set issue type successfully", async () => {
    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "Bug",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.issue_number).toBe(42);
    expect(result.issue_type).toBe("Bug");
    expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
    });
    expect(mockGraphql).toHaveBeenCalledWith(expect.stringContaining("updateIssue"), expect.objectContaining({ issueId: issueNodeId, typeId: bugTypeId }));
  });

  it("should clear issue type when issue_type is empty string", async () => {
    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.issue_type).toBe("");
    // Should call mutation with null typeId to clear
    expect(mockGraphql).toHaveBeenCalledWith(expect.stringContaining("updateIssue"), expect.objectContaining({ issueId: issueNodeId, typeId: null }));
    // Should NOT fetch issue types when clearing
    expect(mockGraphql).not.toHaveBeenCalledWith(expect.stringContaining("issueTypes"), expect.anything());
  });

  it("should use context issue number when issue_number not provided", async () => {
    const message = {
      type: "set_issue_type",
      issue_type: "Bug",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.issue_number).toBe(123); // from context.payload.issue.number
    expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 123,
    });
  });

  it("should validate against allowed types list", async () => {
    const { main } = require("./set_issue_type.cjs");
    const handlerWithAllowed = await main({
      max: 5,
      allowed: ["Bug", "Feature"],
    });

    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "Bug",
    };

    const result = await handlerWithAllowed(message, {});
    expect(result.success).toBe(true);
  });

  it("should reject type not in allowed list", async () => {
    const { main } = require("./set_issue_type.cjs");
    const handlerWithAllowed = await main({
      max: 5,
      allowed: ["Bug", "Feature"],
    });

    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "Task",
    };

    const result = await handlerWithAllowed(message, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("is not in the allowed list");
  });

  it("should allow clearing type even with allowed list configured", async () => {
    const { main } = require("./set_issue_type.cjs");
    const handlerWithAllowed = await main({
      max: 5,
      allowed: ["Bug", "Feature"],
    });

    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "",
    };

    const result = await handlerWithAllowed(message, {});
    expect(result.success).toBe(true);
  });

  it("should return error when issue type not found in repository", async () => {
    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "NonExistentType",
    };

    const result = await handler(message, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
    expect(result.error).toContain("Available types");
  });

  it("should return error when no issue types are available", async () => {
    mockGraphql.mockImplementation(query => {
      if (query.includes("issueTypes")) {
        return Promise.resolve({ repository: { issueTypes: { nodes: [] } } });
      }
      return Promise.resolve({});
    });

    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "Bug",
    };

    const result = await handler(message, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("No issue types are available");
  });

  it("should respect max count configuration", async () => {
    const { main } = require("./set_issue_type.cjs");
    const limitedHandler = await main({ max: 1 });

    const message1 = { type: "set_issue_type", issue_number: 1, issue_type: "Bug" };
    const message2 = { type: "set_issue_type", issue_number: 2, issue_type: "Feature" };

    const result1 = await limitedHandler(message1, {});
    expect(result1.success).toBe(true);

    const result2 = await limitedHandler(message2, {});
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count");
  });

  it("should handle API errors gracefully", async () => {
    mockGraphql.mockImplementation(query => {
      if (query.includes("issueTypes")) {
        return Promise.resolve(mockIssueTypesQuery);
      }
      if (query.includes("updateIssue")) {
        return Promise.reject(new Error("GraphQL API error"));
      }
      return Promise.resolve({});
    });

    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "Bug",
    };

    const result = await handler(message, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("GraphQL API error");
  });

  it("should handle invalid issue numbers", async () => {
    const message = {
      type: "set_issue_type",
      issue_number: -1,
      issue_type: "Bug",
    };

    const result = await handler(message, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid issue_number");
  });

  it("should handle staged mode", async () => {
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    try {
      const { main } = require("./set_issue_type.cjs");
      const stagedHandler = await main({ max: 5 });

      const message = {
        type: "set_issue_type",
        issue_number: 42,
        issue_type: "Bug",
      };

      const result = await stagedHandler(message, {});
      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
      expect(result.previewInfo.issue_number).toBe(42);
      expect(result.previewInfo.issue_type).toBe("Bug");
      // Should not call any API when staged
      expect(mockGithub.rest.issues.get).not.toHaveBeenCalled();
      expect(mockGraphql).not.toHaveBeenCalled();
    } finally {
      delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    }
  });

  it("should handle case-insensitive type matching", async () => {
    const message = {
      type: "set_issue_type",
      issue_number: 42,
      issue_type: "bug", // lowercase
    };

    const result = await handler(message, {});
    expect(result.success).toBe(true);
    // Should still resolve to the Bug type
    expect(mockGraphql).toHaveBeenCalledWith(expect.stringContaining("updateIssue"), expect.objectContaining({ typeId: bugTypeId }));
  });
});
