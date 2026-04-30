// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
const { ERR_CONFIG, ERR_API } = require("./error_codes.cjs");

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

const mockGithub = {
  rest: {
    issues: {
      removeLabel: vi.fn(),
    },
  },
  graphql: vi.fn(),
};

/** @type {any} */
let mockContext = {
  eventName: "issues",
  repo: { owner: "testowner", repo: "testrepo" },
  payload: {
    label: { name: "ai-label", node_id: "LA_label1" },
    issue: { number: 42 },
  },
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

/** @returns {Promise<void>} */
async function runScript() {
  const { main } = await import("./remove_trigger_label.cjs?" + Date.now());
  await main();
}

describe("remove_trigger_label", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();

    process.env.GH_AW_LABEL_NAMES = JSON.stringify(["ai-label", "bot-run"]);

    mockContext = {
      eventName: "issues",
      repo: { owner: "testowner", repo: "testrepo" },
      payload: {
        label: { name: "ai-label", node_id: "LA_label1" },
        issue: { number: 42 },
      },
    };
    global.context = mockContext;

    mockGithub.rest.issues.removeLabel.mockResolvedValue({});
    mockGithub.graphql.mockResolvedValue({});
  });

  afterEach(() => {
    delete process.env.GH_AW_LABEL_NAMES;
  });

  describe("missing configuration", () => {
    it("should fail when GH_AW_LABEL_NAMES is not set", async () => {
      delete process.env.GH_AW_LABEL_NAMES;
      await runScript();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(ERR_CONFIG));
    });

    it("should fail when GH_AW_LABEL_NAMES is invalid JSON", async () => {
      process.env.GH_AW_LABEL_NAMES = "not-valid-json";
      await runScript();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(ERR_CONFIG));
    });

    it("should fail when GH_AW_LABEL_NAMES is not an array", async () => {
      process.env.GH_AW_LABEL_NAMES = JSON.stringify({ label: "ai-label" });
      await runScript();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(ERR_CONFIG));
    });
  });

  describe("workflow_dispatch event", () => {
    it("should skip label removal for workflow_dispatch", async () => {
      global.context = { ...mockContext, eventName: "workflow_dispatch", payload: {} };
      await runScript();
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "");
    });
  });

  describe("no trigger label in payload", () => {
    it("should skip removal when payload has no label", async () => {
      global.context = { ...mockContext, payload: { issue: { number: 42 } } };
      await runScript();
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "");
    });
  });

  describe("label not in configured list", () => {
    it("should skip removal when label is not configured", async () => {
      global.context = {
        ...mockContext,
        payload: { label: { name: "random-label" }, issue: { number: 42 } },
      };
      await runScript();
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "random-label");
    });
  });

  describe("issues event", () => {
    it("should remove label from issue", async () => {
      await runScript();
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 42,
        name: "ai-label",
      });
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "ai-label");
    });

    it("should skip when issue number is missing", async () => {
      global.context = {
        ...mockContext,
        payload: { label: { name: "ai-label" } },
      };
      await runScript();
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "ai-label");
    });
  });

  describe("pull_request event", () => {
    it("should remove label from pull request", async () => {
      global.context = {
        ...mockContext,
        eventName: "pull_request",
        payload: {
          label: { name: "ai-label" },
          pull_request: { number: 99 },
        },
      };
      await runScript();
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 99,
        name: "ai-label",
      });
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "ai-label");
    });

    it("should skip when PR number is missing", async () => {
      global.context = {
        ...mockContext,
        eventName: "pull_request",
        payload: { label: { name: "ai-label" } },
      };
      await runScript();
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
    });
  });

  describe("discussion event", () => {
    it("should remove label from discussion via graphql", async () => {
      global.context = {
        ...mockContext,
        eventName: "discussion",
        payload: {
          label: { name: "ai-label", node_id: "LA_label1" },
          discussion: { node_id: "D_disc1" },
        },
      };
      await runScript();
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("removeLabelsFromLabelable"),
        expect.objectContaining({
          labelableId: "D_disc1",
          labelIds: ["LA_label1"],
        })
      );
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "ai-label");
    });

    it("should skip when discussion node_id is missing", async () => {
      global.context = {
        ...mockContext,
        eventName: "discussion",
        payload: { label: { name: "ai-label" } },
      };
      await runScript();
      expect(mockGithub.graphql).not.toHaveBeenCalled();
    });
  });

  describe("error handling", () => {
    it("should treat 404 error as already-removed (non-fatal)", async () => {
      const err = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.rest.issues.removeLabel.mockRejectedValue(err);
      await runScript();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "ai-label");
    });

    it("should warn on non-404 API errors", async () => {
      const err = Object.assign(new Error("Server error"), { status: 500 });
      mockGithub.rest.issues.removeLabel.mockRejectedValue(err);
      await runScript();
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining(ERR_API));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
