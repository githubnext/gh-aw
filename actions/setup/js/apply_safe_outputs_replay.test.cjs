// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

/** Environment variables managed by tests */
const TEST_ENV_VARS = ["GH_AW_RUN_URL", "GH_TOKEN", "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", "GH_AW_AGENT_OUTPUT"];

describe("apply_safe_outputs_replay", () => {
  let originalEnv;
  let originalGlobals;

  beforeEach(() => {
    originalEnv = { ...process.env };

    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      exec: global.exec,
    };

    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
    };

    global.github = {};

    global.context = {
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
    };

    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn(),
    };

    // Clear managed env vars
    for (const key of TEST_ENV_VARS) {
      delete process.env[key];
    }
  });

  afterEach(() => {
    for (const key of TEST_ENV_VARS) {
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      } else {
        delete process.env[key];
      }
    }

    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;

    vi.clearAllMocks();
  });

  describe("parseRunUrl", () => {
    it("parses a plain run ID", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      const result = parseRunUrl("23560193313");
      expect(result.runId, "should parse run ID").toBe("23560193313");
      expect(result.owner, "owner should be null for plain ID").toBeNull();
      expect(result.repo, "repo should be null for plain ID").toBeNull();
    });

    it("parses a full GitHub Actions run URL", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      const result = parseRunUrl("https://github.com/github/gh-aw/actions/runs/23560193313");
      expect(result.runId, "should parse run ID from URL").toBe("23560193313");
      expect(result.owner, "should parse owner").toBe("github");
      expect(result.repo, "should parse repo").toBe("gh-aw");
    });

    it("parses a run URL that includes a job ID", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      const result = parseRunUrl("https://github.com/github/gh-aw/actions/runs/23560193313/job/68600993738");
      expect(result.runId, "should parse run ID ignoring job ID").toBe("23560193313");
      expect(result.owner, "should parse owner").toBe("github");
      expect(result.repo, "should parse repo").toBe("gh-aw");
    });

    it("trims whitespace from the input", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      const result = parseRunUrl("  23560193313  ");
      expect(result.runId, "should trim and parse run ID").toBe("23560193313");
    });

    it("throws for an empty string", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      expect(() => parseRunUrl(""), "should throw for empty string").toThrow(/run_url is required/);
    });

    it("throws for an invalid URL format", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      expect(() => parseRunUrl("not-a-valid-url"), "should throw for invalid format").toThrow(/Cannot parse run ID/);
    });

    it("throws for a URL without a run ID", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      expect(() => parseRunUrl("https://github.com/owner/repo/actions"), "should throw for URL without run ID").toThrow(/Cannot parse run ID/);
    });
  });

  describe("buildHandlerConfigFromOutput", () => {
    it("builds config from agent output items", async () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");
      const { buildHandlerConfigFromOutput } = await import("./apply_safe_outputs_replay.cjs");

      const tmpFile = path.join(os.tmpdir(), `test-agent-output-${Date.now()}.json`);
      const agentOutput = {
        items: [
          { type: "create_issue", title: "Test issue" },
          { type: "add_comment", body: "Hello" },
          { type: "create_issue", title: "Duplicate type" },
        ],
      };
      fs.writeFileSync(tmpFile, JSON.stringify(agentOutput));

      try {
        const config = buildHandlerConfigFromOutput(tmpFile);
        expect(Object.keys(config), "should include create_issue").toContain("create_issue");
        expect(Object.keys(config), "should include add_comment").toContain("add_comment");
        expect(Object.keys(config).length, "should deduplicate types").toBe(2);
        expect(config.create_issue, "config value should be empty object").toEqual({});
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("normalizes dashes to underscores in type names", async () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");
      const { buildHandlerConfigFromOutput } = await import("./apply_safe_outputs_replay.cjs");

      const tmpFile = path.join(os.tmpdir(), `test-agent-output-${Date.now()}.json`);
      const agentOutput = {
        items: [{ type: "push-to-pull-request-branch", branch: "main" }],
      };
      fs.writeFileSync(tmpFile, JSON.stringify(agentOutput));

      try {
        const config = buildHandlerConfigFromOutput(tmpFile);
        expect(Object.keys(config), "should normalize dashes to underscores").toContain("push_to_pull_request_branch");
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("returns empty config for output with no items", async () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");
      const { buildHandlerConfigFromOutput } = await import("./apply_safe_outputs_replay.cjs");

      const tmpFile = path.join(os.tmpdir(), `test-agent-output-${Date.now()}.json`);
      fs.writeFileSync(tmpFile, JSON.stringify({ items: [] }));

      try {
        const config = buildHandlerConfigFromOutput(tmpFile);
        expect(Object.keys(config).length, "config should be empty").toBe(0);
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("returns empty config when items array is missing", async () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");
      const { buildHandlerConfigFromOutput } = await import("./apply_safe_outputs_replay.cjs");

      const tmpFile = path.join(os.tmpdir(), `test-agent-output-${Date.now()}.json`);
      fs.writeFileSync(tmpFile, JSON.stringify({}));

      try {
        const config = buildHandlerConfigFromOutput(tmpFile);
        expect(Object.keys(config).length, "config should be empty for missing items").toBe(0);
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("skips items with no type field", async () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");
      const { buildHandlerConfigFromOutput } = await import("./apply_safe_outputs_replay.cjs");

      const tmpFile = path.join(os.tmpdir(), `test-agent-output-${Date.now()}.json`);
      const agentOutput = {
        items: [{ title: "No type field" }, { type: "create_issue", title: "Has type" }],
      };
      fs.writeFileSync(tmpFile, JSON.stringify(agentOutput));

      try {
        const config = buildHandlerConfigFromOutput(tmpFile);
        expect(Object.keys(config), "should include create_issue").toContain("create_issue");
        expect(Object.keys(config).length, "should have exactly one entry").toBe(1);
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("skips items with non-string type field", async () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");
      const { buildHandlerConfigFromOutput } = await import("./apply_safe_outputs_replay.cjs");

      const tmpFile = path.join(os.tmpdir(), `test-agent-output-${Date.now()}.json`);
      const agentOutput = {
        items: [{ type: 42 }, { type: null }, { type: "add_comment" }],
      };
      fs.writeFileSync(tmpFile, JSON.stringify(agentOutput));

      try {
        const config = buildHandlerConfigFromOutput(tmpFile);
        expect(Object.keys(config), "should only include string type").toContain("add_comment");
        expect(Object.keys(config).length, "should skip non-string types").toBe(1);
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });

    it("deduplicates repeated types preserving single entry", async () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");
      const { buildHandlerConfigFromOutput } = await import("./apply_safe_outputs_replay.cjs");

      const tmpFile = path.join(os.tmpdir(), `test-agent-output-${Date.now()}.json`);
      const agentOutput = {
        items: [{ type: "add_comment" }, { type: "add_comment" }, { type: "add_comment" }, { type: "create_issue" }],
      };
      fs.writeFileSync(tmpFile, JSON.stringify(agentOutput));

      try {
        const config = buildHandlerConfigFromOutput(tmpFile);
        expect(Object.keys(config).length, "should have 2 unique types").toBe(2);
        expect(config.add_comment, "deduplicated config value should be empty object").toEqual({});
      } finally {
        fs.unlinkSync(tmpFile);
      }
    });
  });

  describe("parseRunUrl (additional edge cases)", () => {
    it("throws for null input", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      // @ts-expect-error testing invalid input
      expect(() => parseRunUrl(null), "should throw for null").toThrow(/run_url is required/);
    });

    it("throws for whitespace-only input", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      expect(() => parseRunUrl("   "), "should throw for whitespace-only input").toThrow(/Cannot parse run ID/);
    });

    it("parses run URL with query string", async () => {
      const { parseRunUrl } = await import("./apply_safe_outputs_replay.cjs");
      const result = parseRunUrl("https://github.com/github/gh-aw/actions/runs/99887766?check_suite_focus=true");
      expect(result.runId, "should parse run ID ignoring query string").toBe("99887766");
      expect(result.owner, "should parse owner").toBe("github");
      expect(result.repo, "should parse repo").toBe("gh-aw");
    });
  });
});
