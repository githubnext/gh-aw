// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import os from "os";
import fs from "fs";
import path from "path";

let parseMaxAICreditsExceededFromAuditLog;
let resolveAICreditsFailureState;

describe("ai_credits_context max_ai_credits_exceeded detection", () => {
  let tmpDir;

  beforeEach(async () => {
    vi.resetModules();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aic-test-"));
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_AIC;
    delete process.env.GH_AW_MAX_AI_CREDITS;
    delete process.env.GH_AW_AI_CREDITS_RATE_LIMIT_ERROR;
    const mod = await import("./ai_credits_context.cjs");
    const exports = mod.default || mod;
    parseMaxAICreditsExceededFromAuditLog = exports.parseMaxAICreditsExceededFromAuditLog;
    resolveAICreditsFailureState = exports.resolveAICreditsFailureState;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_AIC;
    delete process.env.GH_AW_MAX_AI_CREDITS;
    delete process.env.GH_AW_AI_CREDITS_RATE_LIMIT_ERROR;
  });

  function writeAuditLog(lines) {
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    const logPath = path.join(auditDir, "log.jsonl");
    fs.writeFileSync(logPath, lines.map(l => JSON.stringify(l)).join("\n") + "\n", "utf8");
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    return logPath;
  }

  describe("parseMaxAICreditsExceededFromAuditLog", () => {
    it("detects max_ai_credits_exceeded: true field", () => {
      writeAuditLog([{ type: "response", max_ai_credits_exceeded: true, ai_credits: 105000, max_ai_credits: 100000 }]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(true);
    });

    it("detects camelCase maxAiCreditsExceeded: true field", () => {
      writeAuditLog([{ type: "response", maxAiCreditsExceeded: true, aiCredits: 105000, maxAiCredits: 100000 }]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(true);
    });

    it("detects budget_exceeded event with hard_limit reason and forced_termination", () => {
      writeAuditLog([{ event: "budget_exceeded", reason: "hard_limit", forced_termination: true, ai_credits: 105000, max_ai_credits: 100000 }]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(true);
    });

    it("does not detect budget_exceeded event without forced_termination", () => {
      writeAuditLog([{ event: "budget_exceeded", reason: "hard_limit", forced_termination: false }]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(false);
    });

    it("does not detect budget_exceeded event without hard_limit reason", () => {
      writeAuditLog([{ event: "budget_exceeded", reason: "soft_limit", forced_termination: true }]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(false);
    });

    it("returns false when no matching signal is present", () => {
      writeAuditLog([
        { type: "request", ai_credits: 5000, max_ai_credits: 100000 },
        { type: "response", status: 200 },
      ]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(false);
    });

    it("returns false for missing audit log", () => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      expect(parseMaxAICreditsExceededFromAuditLog("/nonexistent/path/log.jsonl")).toBe(false);
    });

    it("returns false for empty audit log", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      const logPath = path.join(auditDir, "log.jsonl");
      fs.writeFileSync(logPath, "", "utf8");
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(false);
    });

    it("detects signal anywhere in a multi-entry log", () => {
      writeAuditLog([
        { type: "request", ai_credits: 5000 },
        { type: "response", status: 200 },
        { event: "budget_exceeded", reason: "hard_limit", forced_termination: true, ai_credits: 105000, max_ai_credits: 100000 },
      ]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(true);
    });

    it("does not detect ai_credits_rate_limit_error as max_ai_credits_exceeded", () => {
      writeAuditLog([{ type: "response", ai_credits_rate_limit_error: true, ai_credits: 105000, max_ai_credits: 100000 }]);
      expect(parseMaxAICreditsExceededFromAuditLog()).toBe(false);
    });
  });

  describe("resolveAICreditsFailureState maxAICreditsExceeded", () => {
    it("returns maxAICreditsExceeded: true when budget_exceeded event is present", () => {
      writeAuditLog([{ event: "budget_exceeded", reason: "hard_limit", forced_termination: true, ai_credits: 105000, max_ai_credits: 100000 }]);
      const result = resolveAICreditsFailureState();
      expect(result.maxAICreditsExceeded).toBe(true);
    });

    it("returns maxAICreditsExceeded: false when no signal is present", () => {
      writeAuditLog([{ type: "request", ai_credits: 5000 }]);
      const result = resolveAICreditsFailureState();
      expect(result.maxAICreditsExceeded).toBe(false);
    });

    it("maxAICreditsExceeded is independent of aiCreditsRateLimitError", () => {
      writeAuditLog([{ type: "response", ai_credits_rate_limit_error: true, ai_credits: 105000, max_ai_credits: 100000 }]);
      const result = resolveAICreditsFailureState();
      expect(result.aiCreditsRateLimitError).toBe(true);
      expect(result.maxAICreditsExceeded).toBe(false);
    });

    it("both flags can be true simultaneously if both signals are present", () => {
      writeAuditLog([
        { type: "response", ai_credits_rate_limit_error: true, ai_credits: 100000, max_ai_credits: 100000 },
        { event: "budget_exceeded", reason: "hard_limit", forced_termination: true, ai_credits: 105000, max_ai_credits: 100000 },
      ]);
      const result = resolveAICreditsFailureState();
      expect(result.aiCreditsRateLimitError).toBe(true);
      expect(result.maxAICreditsExceeded).toBe(true);
    });

    it("detects max-ai-credits exceeded signal from agent-stdio log", () => {
      // Set GH_AW_AGENT_OUTPUT so the derived stdio-log path resolves to tmpDir.
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
      fs.writeFileSync(path.join(tmpDir, "agent-stdio.log"), "Failed to get response from the AI model; retried 5 times. Last error: CAPIError: 429 Maximum AI credits exceeded (1002.381900 / 1000).", "utf8");
      const result = resolveAICreditsFailureState();
      expect(result.aiCredits).toBe("1002.381900");
      expect(result.maxAICredits).toBe("1000");
      expect(result.aiCreditsRateLimitError).toBe(true);
      expect(result.maxAICreditsExceeded).toBe(true);
    });

    it("detects max-ai-credits exceeded from stdio log alone (no audit rate-limit signal)", () => {
      // Audit log has a successful response (status 200); rate-limit must come only from stdio.
      writeAuditLog([{ type: "response", status: 200 }]);
      fs.writeFileSync(path.join(tmpDir, "agent-stdio.log"), "CAPIError: 429 Maximum AI credits exceeded (1002.381900 / 1000).", "utf8");
      const result = resolveAICreditsFailureState();
      expect(result.aiCreditsRateLimitError).toBe(true);
      expect(result.maxAICreditsExceeded).toBe(true);
      expect(result.aiCredits).toBe("1002.381900");
    });

    it("sets exceeded flags but leaves credit amounts empty when no parenthetical is present", () => {
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
      fs.writeFileSync(path.join(tmpDir, "agent-stdio.log"), "Fatal: Maximum AI credits exceeded.", "utf8");
      const result = resolveAICreditsFailureState();
      expect(result.aiCreditsRateLimitError).toBe(true);
      expect(result.maxAICreditsExceeded).toBe(true);
      expect(result.aiCredits).toBe("");
      expect(result.maxAICredits).toBe("");
    });
  });
});
