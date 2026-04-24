// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

// Mock globals before importing the module
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setOutput: vi.fn(),
  setFailed: vi.fn(),
};

const mockGithub = {
  rest: {
    search: {
      issuesAndPullRequests: vi.fn(),
    },
    issues: {
      create: vi.fn(),
      createComment: vi.fn(),
    },
  },
};

const mockContext = {
  repo: { owner: "test-owner", repo: "test-repo" },
};

globalThis.core = mockCore;
globalThis.github = mockGithub;
globalThis.context = mockContext;

const { HANDLER_DESCRIPTORS, handlerRegistry } = require("./missing_issue_handler_registry.cjs");

describe("missing_issue_handler_registry.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("HANDLER_DESCRIPTORS", () => {
    it("should contain exactly three descriptors", () => {
      expect(HANDLER_DESCRIPTORS).toHaveLength(3);
    });

    it("should include create_missing_data_issue descriptor", () => {
      const desc = HANDLER_DESCRIPTORS.find(d => d.handlerType === "create_missing_data_issue");
      expect(desc).toBeDefined();
      expect(desc.defaultTitlePrefix).toBe("[missing data]");
      expect(desc.itemsField).toBe("missing_data");
      expect(desc.templateListKey).toBe("missing_data_list");
      expect(desc.defaultLabels).toContain("agentic-workflows");
    });

    it("should include create_missing_tool_issue descriptor", () => {
      const desc = HANDLER_DESCRIPTORS.find(d => d.handlerType === "create_missing_tool_issue");
      expect(desc).toBeDefined();
      expect(desc.defaultTitlePrefix).toBe("[missing tool]");
      expect(desc.itemsField).toBe("missing_tools");
      expect(desc.templateListKey).toBe("missing_tools_list");
      expect(desc.defaultLabels).toContain("agentic-workflows");
    });

    it("should include create_report_incomplete_issue descriptor", () => {
      const desc = HANDLER_DESCRIPTORS.find(d => d.handlerType === "create_report_incomplete_issue");
      expect(desc).toBeDefined();
      expect(desc.defaultTitlePrefix).toBe("[incomplete]");
      expect(desc.itemsField).toBe("incomplete_signals");
      expect(desc.templateListKey).toBe("incomplete_signals_list");
      expect(desc.defaultLabels).toContain("agentic-workflows");
    });

    it("should have all required descriptor fields", () => {
      for (const desc of HANDLER_DESCRIPTORS) {
        expect(desc.handlerType).toBeTruthy();
        expect(desc.defaultTitlePrefix).toBeTruthy();
        expect(Array.isArray(desc.defaultLabels)).toBe(true);
        expect(desc.itemsField).toBeTruthy();
        expect(desc.templatePath).toBeTruthy();
        expect(desc.templateListKey).toBeTruthy();
        expect(typeof desc.buildCommentHeader).toBe("function");
        expect(typeof desc.renderCommentItem).toBe("function");
        expect(typeof desc.renderIssueItem).toBe("function");
      }
    });
  });

  describe("handlerRegistry", () => {
    it("should be a Map with three entries", () => {
      expect(handlerRegistry).toBeInstanceOf(Map);
      expect(handlerRegistry.size).toBe(3);
    });

    it("should contain a handler for create_missing_data_issue", () => {
      expect(handlerRegistry.has("create_missing_data_issue")).toBe(true);
      expect(typeof handlerRegistry.get("create_missing_data_issue")).toBe("function");
    });

    it("should contain a handler for create_missing_tool_issue", () => {
      expect(handlerRegistry.has("create_missing_tool_issue")).toBe(true);
      expect(typeof handlerRegistry.get("create_missing_tool_issue")).toBe("function");
    });

    it("should contain a handler for create_report_incomplete_issue", () => {
      expect(handlerRegistry.has("create_report_incomplete_issue")).toBe(true);
      expect(typeof handlerRegistry.get("create_report_incomplete_issue")).toBe("function");
    });

    it("should register exactly the handler types in HANDLER_DESCRIPTORS", () => {
      const registryKeys = [...handlerRegistry.keys()].sort();
      const descriptorTypes = HANDLER_DESCRIPTORS.map(d => d.handlerType).sort();
      expect(registryKeys).toEqual(descriptorTypes);
    });
  });

  describe("descriptor item renderers", () => {
    describe("create_missing_data_issue renderers", () => {
      let desc;
      beforeEach(() => {
        desc = HANDLER_DESCRIPTORS.find(d => d.handlerType === "create_missing_data_issue");
      });

      it("should render comment item with data_type, reason", () => {
        const lines = desc.renderCommentItem({ data_type: "api_key", reason: "not configured" }, 0);
        expect(lines.join("\n")).toContain("**api_key**");
        expect(lines.join("\n")).toContain("not configured");
      });

      it("should include context and alternatives when present in comment item", () => {
        const lines = desc.renderCommentItem({ data_type: "api_key", reason: "missing", context: "auth flow", alternatives: "use env var" }, 0);
        expect(lines.join("\n")).toContain("**Context:** auth flow");
        expect(lines.join("\n")).toContain("**Alternatives:** use env var");
      });

      it("should render issue item with timestamp", () => {
        const lines = desc.renderIssueItem({ data_type: "secret", reason: "not set", timestamp: "2026-01-01T00:00:00Z" }, 0);
        expect(lines.join("\n")).toContain("**Reported at:** 2026-01-01T00:00:00Z");
      });

      it("should build comment header with run URL", () => {
        const header = desc.buildCommentHeader("https://github.com/owner/repo/actions/runs/1");
        expect(header.join("\n")).toContain("## Missing Data Reported");
        expect(header.join("\n")).toContain("[workflow run](https://github.com/owner/repo/actions/runs/1)");
      });
    });

    describe("create_missing_tool_issue renderers", () => {
      let desc;
      beforeEach(() => {
        desc = HANDLER_DESCRIPTORS.find(d => d.handlerType === "create_missing_tool_issue");
      });

      it("should render comment item with tool name in backticks", () => {
        const lines = desc.renderCommentItem({ tool: "docker", reason: "not installed" }, 0);
        expect(lines.join("\n")).toContain("`docker`");
        expect(lines.join("\n")).toContain("not installed");
      });

      it("should include alternatives when present in comment item", () => {
        const lines = desc.renderCommentItem({ tool: "kubectl", reason: "unavailable", alternatives: "use manual commands" }, 0);
        expect(lines.join("\n")).toContain("**Alternatives:** use manual commands");
      });

      it("should render issue item with timestamp", () => {
        const lines = desc.renderIssueItem({ tool: "helm", reason: "missing", timestamp: "2026-02-01T00:00:00Z" }, 0);
        expect(lines.join("\n")).toContain("**Reported at:** 2026-02-01T00:00:00Z");
      });

      it("should build comment header with run URL", () => {
        const header = desc.buildCommentHeader("https://github.com/owner/repo/actions/runs/2");
        expect(header.join("\n")).toContain("## Missing Tools Reported");
        expect(header.join("\n")).toContain("[workflow run](https://github.com/owner/repo/actions/runs/2)");
      });
    });

    describe("create_report_incomplete_issue renderers", () => {
      let desc;
      beforeEach(() => {
        desc = HANDLER_DESCRIPTORS.find(d => d.handlerType === "create_report_incomplete_issue");
      });

      it("should render comment item with reason", () => {
        const lines = desc.renderCommentItem({ reason: "MCP server crashed" }, 0);
        expect(lines.join("\n")).toContain("Incomplete signal");
        expect(lines.join("\n")).toContain("MCP server crashed");
      });

      it("should include details when present in comment item", () => {
        const lines = desc.renderCommentItem({ reason: "auth failed", details: "token expired" }, 0);
        expect(lines.join("\n")).toContain("**Details:** token expired");
      });

      it("should render issue item with timestamp", () => {
        const lines = desc.renderIssueItem({ reason: "tool unavailable", timestamp: "2026-03-01T00:00:00Z" }, 0);
        expect(lines.join("\n")).toContain("**Reported at:** 2026-03-01T00:00:00Z");
      });

      it("should build comment header with run URL", () => {
        const header = desc.buildCommentHeader("https://github.com/owner/repo/actions/runs/3");
        expect(header.join("\n")).toContain("## Incomplete Run Reported");
        expect(header.join("\n")).toContain("[workflow run](https://github.com/owner/repo/actions/runs/3)");
      });
    });
  });

  describe("handler factories via registry", () => {
    it("each registered handler should be a factory function returning a message handler", async () => {
      for (const [, factory] of handlerRegistry) {
        const handler = await factory({});
        expect(typeof handler).toBe("function");
      }
    });

    it("create_missing_data_issue factory should return handler that validates missing_data field", async () => {
      const handler = await handlerRegistry.get("create_missing_data_issue")({});
      const result = await handler({ workflow_name: "Test" });
      expect(result.success).toBe(false);
      expect(result.error).toContain("missing_data");
    });

    it("create_missing_tool_issue factory should return handler that validates missing_tools field", async () => {
      const handler = await handlerRegistry.get("create_missing_tool_issue")({});
      const result = await handler({ workflow_name: "Test" });
      expect(result.success).toBe(false);
      expect(result.error).toContain("missing_tools");
    });

    it("create_report_incomplete_issue factory should return handler that validates incomplete_signals field", async () => {
      const handler = await handlerRegistry.get("create_report_incomplete_issue")({});
      const result = await handler({ workflow_name: "Test" });
      expect(result.success).toBe(false);
      expect(result.error).toContain("incomplete_signals");
    });
  });
});
