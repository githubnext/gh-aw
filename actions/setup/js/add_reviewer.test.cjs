import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
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
    pulls: {
      requestReviewers: vi.fn().mockResolvedValue({}),
    },
  },
};

const mockContext = {
  eventName: "pull_request",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    pull_request: {
      number: 123,
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("add_reviewer (Handler Factory Architecture)", () => {
  let handler;

  beforeEach(async () => {
    // Reset all mocks before each test
    vi.clearAllMocks();

    // Reset context
    global.context = mockContext;

    // Load the module and create handler
    const { main } = require("./add_reviewer.cjs");
    handler = await main({
      max: 10,
      allowed: ["user1", "user2", "copilot"],
    });
  });

  it("should return a function from main()", async () => {
    const { main } = require("./add_reviewer.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should add reviewers to PR", async () => {
    const message = {
      type: "add_reviewer",
      reviewers: ["user1", "user2"],
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.prNumber).toBe(123);
    expect(result.reviewersAdded).toEqual(["user1", "user2"]);
    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["user1", "user2"],
    });
  });

  it("should add copilot reviewer separately", async () => {
    const message = {
      type: "add_reviewer",
      reviewers: ["user1", "copilot"],
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.reviewersAdded).toEqual(["user1", "copilot"]);
    // Should be called twice - once for regular reviewers, once for copilot
    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledTimes(2);
    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["user1"],
    });
    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["copilot-pull-request-reviewer[bot]"],
    });
  });

  it("should filter by allowed reviewers", async () => {
    const message = {
      type: "add_reviewer",
      reviewers: ["user1", "user2", "unauthorized"],
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.reviewersAdded).toEqual(["user1", "user2"]);
    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["user1", "user2"],
    });
  });

  it("should enforce max count limit", async () => {
    const { main } = require("./add_reviewer.cjs");
    const limitedHandler = await main({ max: 1, allowed: ["user1", "user2"] });

    const message1 = {
      type: "add_reviewer",
      reviewers: ["user1"],
    };

    const message2 = {
      type: "add_reviewer",
      reviewers: ["user2"],
    };

    // First call should succeed
    const result1 = await limitedHandler(message1, {});
    expect(result1.success).toBe(true);

    // Second call should fail
    const result2 = await limitedHandler(message2, {});
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count");
  });

  it("should use explicit PR number from message", async () => {
    const message = {
      type: "add_reviewer",
      reviewers: ["user1"],
      pull_request_number: 456,
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.prNumber).toBe(456);
    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 456,
      reviewers: ["user1"],
    });
  });

  it("should handle missing PR context", async () => {
    // Change context to remove PR
    global.context = {
      eventName: "issues",
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        issue: {
          number: 123,
        },
      },
    };

    const message = {
      type: "add_reviewer",
      reviewers: ["user1"],
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("No PR number available");
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should handle API errors gracefully", async () => {
    mockGithub.rest.pulls.requestReviewers.mockRejectedValueOnce(new Error("API Error"));

    const message = {
      type: "add_reviewer",
      reviewers: ["user1"],
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("API Error");
  });

  it("should deduplicate reviewers", async () => {
    const message = {
      type: "add_reviewer",
      reviewers: ["user1", "user2", "user1", "user2"],
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.reviewersAdded).toEqual(["user1", "user2"]);
    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["user1", "user2"],
    });
  });

  it("should return success with empty array when no valid reviewers", async () => {
    const message = {
      type: "add_reviewer",
      reviewers: [],
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.reviewersAdded).toEqual([]);
    expect(result.message).toContain("No valid reviewers found");
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });
});
