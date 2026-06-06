// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const os = require("os");

const {
  main,
  getReadableTokenUsagePaths,
  extractRequestId,
  readDedupedTokenUsage,
  getSummaryTitle,
  buildStepSummarySection,
  TOKEN_USAGE_AUDIT_PATH,
  TOKEN_USAGE_PATH,
  TOKEN_USAGE_PATHS,
  AGENT_USAGE_PATH,
  DEFAULT_SUMMARY_TITLE,
} = require("./parse_token_usage.cjs");

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
    test("TOKEN_USAGE_AUDIT_PATH points to firewall audit log file", () => {
      expect(TOKEN_USAGE_AUDIT_PATH).toBe("/tmp/gh-aw/sandbox/firewall-audit-logs/api-proxy-logs/token-usage.jsonl");
    });

    test("TOKEN_USAGE_PATH points to firewall proxy log file", () => {
      expect(TOKEN_USAGE_PATH).toBe("/tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl");
    });

    test("TOKEN_USAGE_PATHS includes audit and legacy paths", () => {
      expect(TOKEN_USAGE_PATHS).toEqual([TOKEN_USAGE_AUDIT_PATH, TOKEN_USAGE_PATH]);
    });

    test("AGENT_USAGE_PATH points to agent_usage.json", () => {
      expect(AGENT_USAGE_PATH).toBe("/tmp/gh-aw/agent_usage.json");
    });

    test("DEFAULT_SUMMARY_TITLE points to Token Usage", () => {
      expect(DEFAULT_SUMMARY_TITLE).toBe("Token Usage");
    });
  });

  describe("main function", () => {
    let tmpDir;
    let mockCore;
    let originalAppendFileSync;
    let originalExistsSync;
    let originalStatSync;
    let originalReadFileSync;
    let originalWriteFileSync;

    beforeEach(() => {
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "parse-token-usage-test-"));
      delete process.env.GH_AW_TOKEN_USAGE_SUMMARY_TITLE;
      process.env.GITHUB_STEP_SUMMARY = "";

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

      originalAppendFileSync = fs.appendFileSync;
      originalExistsSync = fs.existsSync;
      originalStatSync = fs.statSync;
      originalReadFileSync = fs.readFileSync;
      originalWriteFileSync = fs.writeFileSync;

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_AUDIT_PATH || p === TOKEN_USAGE_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_AUDIT_PATH || p === TOKEN_USAGE_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_AUDIT_PATH || p === TOKEN_USAGE_PATH) return "";
        return originalReadFileSync(p, enc);
      });
    });

    afterEach(() => {
      fs.appendFileSync = originalAppendFileSync;
      fs.existsSync = originalExistsSync;
      fs.statSync = originalStatSync;
      fs.readFileSync = originalReadFileSync;
      fs.writeFileSync = originalWriteFileSync;
      delete global.core;
      fs.rmSync(tmpDir, { recursive: true, force: true });
    });

    /**
     * @param {string} summaryText
     * @param {Array<[string, string, string]>} rows [alias, input, output]
     */
    function expectTokenUsageTableRows(summaryText, rows) {
      const escapeRegex = value => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      expect(summaryText).toContain("| # | Alias | Input | Output |");
      for (const [alias, input, output] of rows) {
        const aliasPattern = escapeRegex(alias);
        const inputPattern = escapeRegex(input);
        const outputPattern = escapeRegex(output);
        expect(summaryText).toMatch(new RegExp(`\\|\\s*\\d+\\s*\\|\\s*${aliasPattern}\\s*\\|\\s*${inputPattern}\\s*\\|\\s*${outputPattern}\\s*\\|`));
      }
    }

    test("skips summary when token usage file does not exist", async () => {
      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No token usage data found"));
      expect(mockCore.summary.addDetails).not.toHaveBeenCalled();
      expect(mockCore.summary.write).not.toHaveBeenCalled();
    });

    test("skips summary when token usage file is empty", async () => {
      const emptyFile = path.join(tmpDir, "token-usage.jsonl");
      fs.writeFileSync(emptyFile, "");

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: 0 };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No token usage data found"));
      expect(mockCore.summary.addDetails).not.toHaveBeenCalled();
    });

    test("writes token usage details section to summary", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: singleEntry.length };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_PATH) return singleEntry;
        if (p === TOKEN_USAGE_AUDIT_PATH) return "";
        return originalReadFileSync(p, enc);
      });
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) {
          originalWriteFileSync(agentUsageFile, data);
        } else {
          originalWriteFileSync(p, data);
        }
      });

      await main();

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("### Token Usage"), true);
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("| Alias |"), true);
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Token usage summary appended"));
    });

    test("uses custom summary title when configured", async () => {
      process.env.GH_AW_TOKEN_USAGE_SUMMARY_TITLE = "Threat Detection Token Usage";

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: singleEntry.length };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_PATH) return singleEntry;
        if (p === TOKEN_USAGE_AUDIT_PATH) return "";
        return originalReadFileSync(p, enc);
      });

      await main();

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("### Threat Detection Token Usage"), true);
    });

    test("appends token usage section to GITHUB_STEP_SUMMARY when configured", async () => {
      const stepSummaryPath = path.join(tmpDir, "step-summary.md");
      process.env.GITHUB_STEP_SUMMARY = stepSummaryPath;
      fs.appendFileSync = vi.fn((...args) => originalAppendFileSync(...args));

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: singleEntry.length };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_PATH) return singleEntry;
        if (p === TOKEN_USAGE_AUDIT_PATH) return "";
        return originalReadFileSync(p, enc);
      });

      await main();

      const stepSummary = originalReadFileSync(stepSummaryPath, "utf8");
      expect(stepSummary).toContain("### Token Usage");
      expect(stepSummary).toContain("<summary>Per-request AI credits and token totals</summary>");
      expect(stepSummary).toContain("| ΔAI Credits | AI Credits |");
      expect(fs.appendFileSync).toHaveBeenCalledWith(stepSummaryPath, expect.any(String), "utf8");
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
      expect(mockCore.summary.write).not.toHaveBeenCalled();
    });

    test("writes agent_usage.json with aggregated token totals including effective_tokens and primary_model", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: singleEntry.length };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_PATH) return singleEntry;
        if (p === TOKEN_USAGE_AUDIT_PATH) return "";
        return originalReadFileSync(p, enc);
      });
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
      expect(agentUsage.ambient_context).toBe(900);
      expect(typeof agentUsage.effective_tokens).toBe("number");
      expect(typeof agentUsage.ai_credits).toBe("number");
      // primary_model is the actual model from token-usage data (not a user alias)
      expect(agentUsage.primary_model).toBe("claude-sonnet-4-6");
    });

    test("exports effective_tokens as step output and env var when non-zero", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: singleEntry.length };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_PATH) return singleEntry;
        if (p === TOKEN_USAGE_AUDIT_PATH) return "";
        return originalReadFileSync(p, enc);
      });
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
      if (agentUsage.ai_credits > 0) {
        expect(mockCore.setOutput).toHaveBeenCalledWith("aic", agentUsage.ai_credits.toFixed(3));
        expect(mockCore.exportVariable).toHaveBeenCalledWith("GH_AW_AIC", agentUsage.ai_credits.toFixed(3));
      }
      if (agentUsage.ambient_context > 0) {
        expect(mockCore.setOutput).toHaveBeenCalledWith("ambient_context", String(agentUsage.ambient_context));
        expect(mockCore.exportVariable).toHaveBeenCalledWith("GH_AW_AMBIENT_CONTEXT", String(agentUsage.ambient_context));
      }
    });

    test("handles multiple model entries", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: multiEntry.length };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_PATH) return multiEntry;
        if (p === TOKEN_USAGE_AUDIT_PATH) return "";
        return originalReadFileSync(p, enc);
      });
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) {
          originalWriteFileSync(agentUsageFile, data);
        } else {
          originalWriteFileSync(p, data);
        }
      });

      await main();

      const summaryCall = mockCore.summary.addRaw.mock.calls[0];
      expect(summaryCall[0]).toContain("### Token Usage");
      expectTokenUsageTableRows(summaryCall[0], [
        ["sonnet46", "100", "200"],
        ["gpt40", "50", "80"],
      ]);
      expect(summaryCall[0]).toContain("**Total**");

      const agentUsage = JSON.parse(fs.readFileSync(agentUsageFile, "utf8"));
      expect(agentUsage.input_tokens).toBe(150);
      expect(agentUsage.output_tokens).toBe(280);
    });

    test("reads token usage from firewall-audit-logs path", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");

      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_AUDIT_PATH) return true;
        if (p === TOKEN_USAGE_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: multiEntry.length };
        if (p === TOKEN_USAGE_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_AUDIT_PATH) return multiEntry;
        if (p === TOKEN_USAGE_PATH) return "";
        return originalReadFileSync(p, enc);
      });
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) {
          originalWriteFileSync(agentUsageFile, data);
        } else {
          originalWriteFileSync(p, data);
        }
      });

      await main();

      const summaryCall = mockCore.summary.addRaw.mock.calls[0];
      expectTokenUsageTableRows(summaryCall[0], [
        ["sonnet46", "100", "200"],
        ["gpt40", "50", "80"],
      ]);

      const agentUsage = JSON.parse(fs.readFileSync(agentUsageFile, "utf8"));
      expect(agentUsage.input_tokens).toBe(150);
      expect(agentUsage.output_tokens).toBe(280);
    });

    test("deduplicates overlapping entries across audit and legacy token usage files", async () => {
      const agentUsageFile = path.join(tmpDir, "agent_usage.json");
      const sharedEntry = JSON.stringify({
        request_id: "req-shared",
        model: "claude-sonnet-4-6",
        provider: "anthropic",
        input_tokens: 100,
        output_tokens: 200,
        cache_read_tokens: 0,
        cache_write_tokens: 0,
        duration_ms: 1000,
      });
      const auditOnlyEntry = JSON.stringify({
        request_id: "req-audit",
        model: "claude-haiku-4-5",
        provider: "anthropic",
        input_tokens: 50,
        output_tokens: 75,
        cache_read_tokens: 0,
        cache_write_tokens: 0,
        duration_ms: 500,
      });
      const legacyOnlyEntry = JSON.stringify({
        request_id: "req-legacy",
        model: "gpt-4o",
        provider: "openai",
        input_tokens: 20,
        output_tokens: 30,
        cache_read_tokens: 0,
        cache_write_tokens: 0,
        duration_ms: 400,
      });

      const auditContent = [sharedEntry, auditOnlyEntry].join("\n");
      const legacyContent = [sharedEntry, legacyOnlyEntry].join("\n");

      fs.existsSync = vi.fn(p => (p === TOKEN_USAGE_AUDIT_PATH || p === TOKEN_USAGE_PATH ? true : originalExistsSync(p)));
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: auditContent.length };
        if (p === TOKEN_USAGE_PATH) return { size: legacyContent.length };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_AUDIT_PATH) return auditContent;
        if (p === TOKEN_USAGE_PATH) return legacyContent;
        return originalReadFileSync(p, enc);
      });
      fs.writeFileSync = vi.fn((p, data) => {
        if (p === AGENT_USAGE_PATH) {
          originalWriteFileSync(agentUsageFile, data);
        } else {
          originalWriteFileSync(p, data);
        }
      });

      await main();

      const summaryCall = mockCore.summary.addRaw.mock.calls[0];
      expectTokenUsageTableRows(summaryCall[0], [
        ["sonnet46", "100", "200"],
        ["haiku45", "50", "75"],
        ["gpt40", "20", "30"],
      ]);

      const agentUsage = JSON.parse(fs.readFileSync(agentUsageFile, "utf8"));
      expect(agentUsage.input_tokens).toBe(170);
      expect(agentUsage.output_tokens).toBe(305);
    });

    test("calls setFailed when an error is thrown", async () => {
      fs.existsSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return true;
        if (p === TOKEN_USAGE_AUDIT_PATH) return false;
        return originalExistsSync(p);
      });
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_PATH) return { size: singleEntry.length };
        if (p === TOKEN_USAGE_AUDIT_PATH) return { size: 0 };
        return originalStatSync(p);
      });
      fs.readFileSync = vi.fn((p, enc) => {
        if (p === TOKEN_USAGE_PATH) return singleEntry;
        if (p === TOKEN_USAGE_AUDIT_PATH) return "";
        return originalReadFileSync(p, enc);
      });
      fs.writeFileSync = vi.fn(p => {
        if (p === AGENT_USAGE_PATH) throw new Error("write error");
      });

      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("write error"));
    });
  });

  describe("helpers", () => {
    let originalExistsSync;
    let originalStatSync;
    let originalReadFileSync;

    beforeEach(() => {
      originalExistsSync = fs.existsSync;
      originalStatSync = fs.statSync;
      originalReadFileSync = fs.readFileSync;
      global.core = { warning: vi.fn() };
    });

    afterEach(() => {
      fs.existsSync = originalExistsSync;
      fs.statSync = originalStatSync;
      fs.readFileSync = originalReadFileSync;
      delete global.core;
    });

    test("extractRequestId reads request_id without parsing JSON", () => {
      expect(extractRequestId('{"request_id":"req-123","model":"m"}')).toBe("req-123");
      expect(extractRequestId('{"model":"m"}')).toBe("");
    });

    test("getReadableTokenUsagePaths skips failing stat path and keeps valid path", () => {
      fs.existsSync = vi.fn(p => p === TOKEN_USAGE_AUDIT_PATH || p === TOKEN_USAGE_PATH);
      fs.statSync = vi.fn(p => {
        if (p === TOKEN_USAGE_AUDIT_PATH) throw new Error("stat fail");
        if (p === TOKEN_USAGE_PATH) return { size: 42 };
        return originalStatSync(p);
      });

      const paths = getReadableTokenUsagePaths(TOKEN_USAGE_PATHS);
      expect(paths).toEqual([TOKEN_USAGE_PATH]);
    });

    test("readDedupedTokenUsage deduplicates by request_id across files", () => {
      const fileA = '{"request_id":"req-1","model":"m1","input_tokens":1}\n{"request_id":"req-2","model":"m2","input_tokens":2}';
      const fileB = '{"request_id":"req-1","model":"m1","input_tokens":1}\n{"request_id":"req-3","model":"m3","input_tokens":3}';

      fs.readFileSync = vi.fn(p => {
        if (p === TOKEN_USAGE_AUDIT_PATH) return fileA;
        if (p === TOKEN_USAGE_PATH) return fileB;
        return originalReadFileSync(p, "utf8");
      });

      const deduped = readDedupedTokenUsage([TOKEN_USAGE_AUDIT_PATH, TOKEN_USAGE_PATH]);
      expect(deduped).toContain('"request_id":"req-1"');
      expect(deduped).toContain('"request_id":"req-2"');
      expect(deduped).toContain('"request_id":"req-3"');
      expect(deduped.match(/"request_id":"req-1"/g)).toHaveLength(1);
    });

    test("getSummaryTitle returns trimmed env title", () => {
      process.env.GH_AW_TOKEN_USAGE_SUMMARY_TITLE = "  Threat Detection Token Usage  ";
      expect(getSummaryTitle()).toBe("Threat Detection Token Usage");
    });

    test("getSummaryTitle falls back to default title", () => {
      delete process.env.GH_AW_TOKEN_USAGE_SUMMARY_TITLE;
      expect(getSummaryTitle()).toBe("Token Usage");
    });

    test("buildStepSummarySection wraps markdown in a heading and details block", () => {
      const section = buildStepSummarySection("Token Usage", "| Alias |\n| --- |");
      expect(section).toContain("### Token Usage");
      expect(section).toContain("<details>");
      expect(section).toContain("<summary>Per-request AI credits and token totals</summary>");
    });
  });
});
