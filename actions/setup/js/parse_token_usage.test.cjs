// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const os = require("os");

const { main, TOKEN_USAGE_PATH, AGENT_USAGE_PATH } = require("./parse_token_usage.cjs");

describe("parse_token_usage", () => {
  const singleEntry = JSON.stringify({
    model: "claude-sonnet-4-6",
    provider: "anthropic",
    input_tokens: 100,
    output_tokens: 200,
    cache_read_tokens: 5000,
    cache_write_tokens: 3000,
    duration_ms: 2500,
  });

  const multiEntry = [
    JSON.stringify({ model: "claude-sonnet-4-6", provider: "anthropic", input_tokens: 100, output_tokens: 200, cache_read_tokens: 0, cache_write_tokens: 0, duration_ms: 1000 }),
    JSON.stringify({ model: "gpt-4o", provider: "openai", input_tokens: 50, output_tokens: 80, cache_read_tokens: 0, cache_write_tokens: 0, duration_ms: 500 }),
  ].join("\n");

  describe("constant paths", () => {
    test("TOKEN_USAGE_PATH points to firewall proxy log file", () => {
      expect(TOKEN_USAGE_PATH).toBe("/tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl");
    });

    test("AGENT_USAGE_PATH points to agent_usage.json", () => {
      expect(AGENT_USAGE_PATH).toBe("/tmp/gh-aw/agent_usage.json");
    });
  });

  describe("main function", () => {
    let tmpDir;
    let mockCore;
    let originalExistsSync;
    let originalStatSync;
    let originalReadFileSync;
    let originalWriteFileSync;

    beforeEach(() => {
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "parse-token-usage-test-"));

      mockCore = {
        info: vi.fn(),
        debug: vi.fn(),
        warning: vi.fn(),
        error: vi.fn(),
        setFailed: vi.fn(),
        exportVariable: vi.fn(),
        setOutput: vi.fn(),
        summary: {
          addDetails: vi.fn().mockReturnThis(),
          addRaw: vi.fn().mockReturnThis(),
          write: vi.fn().mockResolvedValue(undefined),
        },
      };

      global.core = mockCore;

      originalExistsSync = fs.existsSync;
      originalStatSync = fs.statSync;
      originalReadFileSync = fs.readFileSync;
      originalWriteFileSync = fs.writeFileSync;
    });

    afterEach(() => {
      fs.existsSync = originalExistsSync;
      fs.statSync = originalStatSync;
      fs.readFileSync = originalReadFileSync;
      fs.writeFileSync = originalWriteFileSync;
      delete global.core;
      fs.rmSync(tmpDir, { recursive: true, force: true });
    });

    test("skips summary when token usage file does not exist", async () => {
      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? false : originalExistsSync(p)));

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No token usage data found"));
      expect(mockCore.summary.addDetails).not.toHaveBeenCalled();
      expect(mockCore.summary.write).not.toHaveBeenCalled();
    });

    test("skips summary when token usage file is empty", async () => {
      const emptyFile = path.join(tmpDir, "token-usage.jsonl");
      fs.writeFileSync(emptyFile, "");

      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? true : originalExistsSync(p)));
      fs.statSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? { size: 0 } : originalStatSync(p)));

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No token usage data found"));
      expect(mockCore.summary.addDetails).not.toHaveBeenCalled();
    });

    test("writes token usage details section to summary", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? true : originalExistsSync(p)));
      fs.statSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? { size: singleEntry.length } : originalStatSync(p)));
      fs.readFileSync = vi.fn((p, enc) => (p === TOKEN_USAGE_PATH ? singleEntry : originalReadFileSync(p, enc)));
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) {
          originalWriteFileSync(agentUsageFile, data);
        } else {
          originalWriteFileSync(p, data);
        }
      });

      await main();

      expect(mockCore.summary.addDetails).toHaveBeenCalledWith("Token Usage", expect.stringContaining("| Model |"));
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Token usage summary appended"));
    });

    test("writes agent_usage.json with aggregated token totals including effective_tokens", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? true : originalExistsSync(p)));
      fs.statSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? { size: singleEntry.length } : originalStatSync(p)));
      fs.readFileSync = vi.fn((p, enc) => (p === TOKEN_USAGE_PATH ? singleEntry : originalReadFileSync(p, enc)));
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) {
          originalWriteFileSync(agentUsageFile, data);
        } else {
          originalWriteFileSync(p, data);
        }
      });

      await main();

      expect(fs.existsSync(agentUsageFile)).toBe(true);
      const agentUsage = JSON.parse(fs.readFileSync(agentUsageFile, "utf8"));
      expect(agentUsage.input_tokens).toBe(100);
      expect(agentUsage.output_tokens).toBe(200);
      expect(agentUsage.cache_read_tokens).toBe(5000);
      expect(agentUsage.cache_write_tokens).toBe(3000);
      expect(typeof agentUsage.effective_tokens).toBe("number");
    });

    test("exports effective_tokens as step output and env var when non-zero", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? true : originalExistsSync(p)));
      fs.statSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? { size: singleEntry.length } : originalStatSync(p)));
      fs.readFileSync = vi.fn((p, enc) => (p === TOKEN_USAGE_PATH ? singleEntry : originalReadFileSync(p, enc)));
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) originalWriteFileSync(agentUsageFile, data);
        else originalWriteFileSync(p, data);
      });

      await main();

      const agentUsage = JSON.parse(fs.readFileSync(agentUsageFile, "utf8"));
      if (agentUsage.effective_tokens > 0) {
        expect(mockCore.setOutput).toHaveBeenCalledWith("effective_tokens", String(agentUsage.effective_tokens));
        expect(mockCore.exportVariable).toHaveBeenCalledWith("GH_AW_EFFECTIVE_TOKENS", String(agentUsage.effective_tokens));
      }
    });

    test("handles multiple model entries", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? true : originalExistsSync(p)));
      fs.statSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? { size: multiEntry.length } : originalStatSync(p)));
      fs.readFileSync = vi.fn((p, enc) => (p === TOKEN_USAGE_PATH ? multiEntry : originalReadFileSync(p, enc)));
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) {
          originalWriteFileSync(agentUsageFile, data);
        } else {
          originalWriteFileSync(p, data);
        }
      });

      await main();

      const detailsCall = mockCore.summary.addDetails.mock.calls[0];
      expect(detailsCall[0]).toBe("Token Usage");
      expect(detailsCall[1]).toContain("claude-sonnet-4-6");
      expect(detailsCall[1]).toContain("gpt-4o");
      expect(detailsCall[1]).toContain("**Total**");

      const agentUsage = JSON.parse(fs.readFileSync(agentUsageFile, "utf8"));
      expect(agentUsage.input_tokens).toBe(150);
      expect(agentUsage.output_tokens).toBe(280);
    });

    test("calls setFailed when an error is thrown", async () => {
      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_PATH ? true : originalExistsSync(p)));
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) throw new Error("stat error");
        return originalStatSync(p);
      });

      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("stat error"));
    });
  });
});
