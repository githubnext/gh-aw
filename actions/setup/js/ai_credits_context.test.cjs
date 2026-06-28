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

    it("ignores env rate-limit signal when no AI credits evidence is present", () => {
      writeAuditLog([{ type: "response", status: 200 }]);
      process.env.GH_AW_AI_CREDITS_RATE_LIMIT_ERROR = "true";
      process.env.GH_AW_MAX_AI_CREDITS = "1000";
      const result = resolveAICreditsFailureState();
      expect(result.aiCreditsRateLimitError).toBe(false);
      expect(result.aiCredits).toBe("");
      expect(result.maxAICredits).toBe("1000");
    });
  });
});

describe("ai_credits_context unknown_model_ai_credits detection", () => {
  let tmpDir;
  let parseUnknownModelAICreditsFromAuditLog;

  beforeEach(async () => {
    vi.resetModules();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aic-unknown-test-"));
    delete process.env.GH_AW_AGENT_OUTPUT;
    const mod = await import("./ai_credits_context.cjs");
    const exports = mod.default || mod;
    parseUnknownModelAICreditsFromAuditLog = exports.parseUnknownModelAICreditsFromAuditLog;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete process.env.GH_AW_AGENT_OUTPUT;
  });

  function writeAuditLog(lines, filename = "log.jsonl") {
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    const logPath = path.join(auditDir, filename);
    fs.writeFileSync(logPath, lines.map(l => JSON.stringify(l)).join("\n") + "\n", "utf8");
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    return logPath;
  }

  it("detects unknown_model_ai_credits type in audit log", () => {
    writeAuditLog([{ type: "unknown_model_ai_credits", model: "my-custom-model" }]);
    expect(parseUnknownModelAICreditsFromAuditLog()).toBe(true);
  });

  it("detects unknown_model_ai_credits using audit.jsonl filename", () => {
    writeAuditLog([{ type: "unknown_model_ai_credits" }], "audit.jsonl");
    expect(parseUnknownModelAICreditsFromAuditLog()).toBe(true);
  });

  it("detects unknown_model_ai_credits in a multi-entry log", () => {
    writeAuditLog([
      { type: "response", status: 200 },
      { type: "unknown_model_ai_credits", model: "my-model" },
      { type: "response", status: 200 },
    ]);
    expect(parseUnknownModelAICreditsFromAuditLog()).toBe(true);
  });

  it("returns false when no unknown_model_ai_credits signal is present", () => {
    writeAuditLog([{ type: "response", status: 200 }]);
    expect(parseUnknownModelAICreditsFromAuditLog()).toBe(false);
  });

  it("returns false for missing audit log", () => {
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    expect(parseUnknownModelAICreditsFromAuditLog("/nonexistent/path/log.jsonl")).toBe(false);
  });

  it("returns false for empty audit log", () => {
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    fs.writeFileSync(path.join(auditDir, "log.jsonl"), "", "utf8");
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    expect(parseUnknownModelAICreditsFromAuditLog()).toBe(false);
  });

  it("does not detect other error types as unknown_model_ai_credits", () => {
    writeAuditLog([{ type: "ai_credits_rate_limit_error" }]);
    expect(parseUnknownModelAICreditsFromAuditLog()).toBe(false);
  });
});

describe("ai_credits_context parseMaxAICreditsFromAuditLog", () => {
  let tmpDir;
  /** @type {(path?: string) => string} */
  let parseMaxAICreditsFromAuditLog;

  beforeEach(async () => {
    vi.resetModules();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aic-maxcredits-test-"));
    delete process.env.GH_AW_AGENT_OUTPUT;
    const mod = await import("./ai_credits_context.cjs");
    const exports = mod.default || mod;
    parseMaxAICreditsFromAuditLog = exports.parseMaxAICreditsFromAuditLog;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete process.env.GH_AW_AGENT_OUTPUT;
  });

  function writeAuditLog(lines) {
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    const logPath = path.join(auditDir, "log.jsonl");
    fs.writeFileSync(logPath, lines.map(l => JSON.stringify(l)).join("\n") + "\n", "utf8");
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    return logPath;
  }

  it("returns empty string for missing audit log", () => {
    expect(parseMaxAICreditsFromAuditLog("/nonexistent/path.jsonl")).toBe("");
  });

  it("returns empty string when no max_ai_credits field is present", () => {
    writeAuditLog([{ type: "request", ai_credits: 500 }]);
    expect(parseMaxAICreditsFromAuditLog()).toBe("");
  });

  it("parses snake_case max_ai_credits", () => {
    writeAuditLog([{ type: "response", max_ai_credits: 1000 }]);
    expect(parseMaxAICreditsFromAuditLog()).toBe("1000");
  });

  it("parses camelCase maxAiCredits", () => {
    writeAuditLog([{ type: "response", maxAiCredits: 2000 }]);
    expect(parseMaxAICreditsFromAuditLog()).toBe("2000");
  });

  it("parses nested max_ai_credits field", () => {
    writeAuditLog([{ type: "response", metadata: { max_ai_credits: 500 } }]);
    expect(parseMaxAICreditsFromAuditLog()).toBe("500");
  });

  it("returns the last non-empty value across multiple entries", () => {
    writeAuditLog([{ max_ai_credits: 1000 }, { max_ai_credits: 2000 }]);
    expect(parseMaxAICreditsFromAuditLog()).toBe("2000");
  });

  it("returns empty string for empty audit log", () => {
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    fs.writeFileSync(path.join(auditDir, "log.jsonl"), "", "utf8");
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    expect(parseMaxAICreditsFromAuditLog()).toBe("");
  });

  it("parses decimal max_ai_credits value", () => {
    writeAuditLog([{ max_ai_credits: 999.5 }]);
    expect(parseMaxAICreditsFromAuditLog()).toBe("999.5");
  });

  it("ignores non-positive max_ai_credits values", () => {
    writeAuditLog([{ max_ai_credits: -100 }]);
    expect(parseMaxAICreditsFromAuditLog()).toBe("");
  });
});

describe("ai_credits_context parseAICreditsErrorInfoFromAuditLog", () => {
  let tmpDir;
  /** @type {(path?: string) => { aiCredits: string, rateLimitError: boolean }} */
  let parseAICreditsErrorInfoFromAuditLog;

  beforeEach(async () => {
    vi.resetModules();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aic-errorinfo-test-"));
    delete process.env.GH_AW_AGENT_OUTPUT;
    const mod = await import("./ai_credits_context.cjs");
    const exports = mod.default || mod;
    parseAICreditsErrorInfoFromAuditLog = exports.parseAICreditsErrorInfoFromAuditLog;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete process.env.GH_AW_AGENT_OUTPUT;
  });

  function writeAuditLog(lines) {
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    const logPath = path.join(auditDir, "log.jsonl");
    fs.writeFileSync(logPath, lines.map(l => JSON.stringify(l)).join("\n") + "\n", "utf8");
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    return logPath;
  }

  it("returns empty defaults for missing log", () => {
    expect(parseAICreditsErrorInfoFromAuditLog("/nonexistent/path.jsonl")).toEqual({ aiCredits: "", rateLimitError: false });
  });

  it("returns empty defaults for empty log", () => {
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    fs.writeFileSync(path.join(auditDir, "log.jsonl"), "", "utf8");
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    expect(parseAICreditsErrorInfoFromAuditLog()).toEqual({ aiCredits: "", rateLimitError: false });
  });

  it("parses ai_credits value from snake_case field", () => {
    writeAuditLog([{ type: "response", ai_credits: 750 }]);
    const result = parseAICreditsErrorInfoFromAuditLog();
    expect(result.aiCredits).toBe("750");
    expect(result.rateLimitError).toBe(false);
  });

  it("parses aiCredits value from camelCase field", () => {
    writeAuditLog([{ aiCredits: 300 }]);
    const result = parseAICreditsErrorInfoFromAuditLog();
    expect(result.aiCredits).toBe("300");
  });

  it("detects ai_credits_rate_limit_error: true (snake_case)", () => {
    writeAuditLog([{ ai_credits_rate_limit_error: true }]);
    expect(parseAICreditsErrorInfoFromAuditLog().rateLimitError).toBe(true);
  });

  it("detects aiCreditsRateLimitError: true (camelCase)", () => {
    writeAuditLog([{ aiCreditsRateLimitError: true }]);
    expect(parseAICreditsErrorInfoFromAuditLog().rateLimitError).toBe(true);
  });

  it("detects rate limit from message field text pattern", () => {
    writeAuditLog([{ message: "AI credits rate limit exceeded" }]);
    expect(parseAICreditsErrorInfoFromAuditLog().rateLimitError).toBe(true);
  });

  it("detects rate limit from code field with ai_credits_limit_exceeded", () => {
    writeAuditLog([{ code: "ai_credits_limit_exceeded" }]);
    expect(parseAICreditsErrorInfoFromAuditLog().rateLimitError).toBe(true);
  });

  it("accumulates aiCredits and rateLimitError across multiple entries", () => {
    writeAuditLog([{ ai_credits: 500 }, { ai_credits_rate_limit_error: true, ai_credits: 750 }]);
    const result = parseAICreditsErrorInfoFromAuditLog();
    expect(result.aiCredits).toBe("750");
    expect(result.rateLimitError).toBe(true);
  });
});

describe("ai_credits_context resolveFirewallAuditLogPath", () => {
  let tmpDir;
  /** @type {(override?: string) => string} */
  let resolveFirewallAuditLogPath;

  beforeEach(async () => {
    vi.resetModules();
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aic-path-test-"));
    delete process.env.GH_AW_AGENT_OUTPUT;
    const mod = await import("./ai_credits_context.cjs");
    const exports = mod.default || mod;
    resolveFirewallAuditLogPath = exports.resolveFirewallAuditLogPath;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    fs.rmSync(tmpDir, { recursive: true, force: true });
    delete process.env.GH_AW_AGENT_OUTPUT;
  });

  it("returns the override path directly without checking existence", () => {
    expect(resolveFirewallAuditLogPath("/custom/path/log.jsonl")).toBe("/custom/path/log.jsonl");
  });

  it("returns derived audit path when GH_AW_AGENT_OUTPUT is set and log.jsonl exists", () => {
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    fs.writeFileSync(path.join(auditDir, "log.jsonl"), "", "utf8");
    expect(resolveFirewallAuditLogPath()).toBe(path.join(auditDir, "log.jsonl"));
  });

  it("prefers log.jsonl over audit.jsonl when both exist", () => {
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    fs.writeFileSync(path.join(auditDir, "log.jsonl"), "", "utf8");
    fs.writeFileSync(path.join(auditDir, "audit.jsonl"), "", "utf8");
    expect(resolveFirewallAuditLogPath()).toBe(path.join(auditDir, "log.jsonl"));
  });

  it("falls back to audit.jsonl when log.jsonl does not exist in derived path", () => {
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
    fs.mkdirSync(auditDir, { recursive: true });
    fs.writeFileSync(path.join(auditDir, "audit.jsonl"), "", "utf8");
    expect(resolveFirewallAuditLogPath()).toBe(path.join(auditDir, "audit.jsonl"));
  });

  it("checks the logs subdir when audit subdir has no match", () => {
    process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "output.json");
    const logsDir = path.join(tmpDir, "sandbox", "firewall", "logs");
    fs.mkdirSync(logsDir, { recursive: true });
    fs.writeFileSync(path.join(logsDir, "log.jsonl"), "", "utf8");
    expect(resolveFirewallAuditLogPath()).toBe(path.join(logsDir, "log.jsonl"));
  });

  it("returns default fallback path when GH_AW_AGENT_OUTPUT is set but no file exists", () => {
    // Use a unique tmpDir sub-path as the agent output so derived paths stay
    // within tmpDir and won't accidentally match real files elsewhere.
    const fakeTmpOutput = path.join(tmpDir, "nested", "output.json");
    process.env.GH_AW_AGENT_OUTPUT = fakeTmpOutput;
    // The derived candidates (tmpDir/nested/sandbox/...) won't exist in tmpDir,
    // but the default /tmp/gh-aw paths might — so just assert the returned path
    // is a string ending in .jsonl (the function always returns some valid path).
    const result = resolveFirewallAuditLogPath();
    expect(result).toMatch(/\.jsonl$/);
  });
});
